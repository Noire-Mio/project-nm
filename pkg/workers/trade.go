package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"project-nm/pkg/database"
	"project-nm/pkg/entities"
	"project-nm/pkg/repositories"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TradeWorker struct {
	TradeRepoFactory  repositories.TradeFactory
	MemberRepoFactory repositories.MemberFactory
}

func NewTradeWorker(tradeFactory repositories.TradeFactory, memberFactory repositories.MemberFactory) *TradeWorker {
	return &TradeWorker{
		TradeRepoFactory:  tradeFactory,
		MemberRepoFactory: memberFactory,
	}
}

func (w *TradeWorker) Start(ctx context.Context) {
	streamName := "stream:trade_tasks"
	log.Printf("[INFO] TradeWorker execution loop started. Target stream: %s", streamName)

	// 使用時間戳指針 $ 起跑，確保常駐消費只關注最新任務，切斷歷史訊息無限循環
	lastIdx := "$"

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] TradeWorker received shutdown signal. Terminating consumption loop cleanly.")
			return

		default:
			streams, err := database.RDB.XRead(ctx, &redis.XReadArgs{
				Streams: []string{streamName, lastIdx},
				Count:   100,
				Block:   2 * time.Second,
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				log.Printf("[ERROR] Failed to read from Redis Stream: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					userIdStr, _ := message.Values["user_id"].(string)
					productIdStr, _ := message.Values["product_id"].(string)
					quantityStr, _ := message.Values["quantity"].(string)
					amountStr, _ := message.Values["amount"].(string)
					txType, _ := message.Values["type"].(string)
					schema, _ := message.Values["schema"].(string)

					userID, _ := strconv.ParseUint(userIdStr, 10, 64)
					productID, _ := strconv.ParseUint(productIdStr, 10, 64)
					quantity, _ := strconv.ParseInt(quantityStr, 10, 64)
					amount, _ := decimal.NewFromString(amountStr)

					// 啟動資料庫本地事務，藉由硬碟強隔離性完成資產物理落盤
					err = database.DB.Transaction(func(tx *gorm.DB) error {
						txCtx := repositories.NewGormDBContext(tx)
						tradeRepo := w.TradeRepoFactory(txCtx)
						memberRepo := w.MemberRepoFactory(txCtx)

						// 悲觀鎖（FOR UPDATE）強行排隊，阻斷非同步時間差內的改價與庫存衝突
						products, err := tradeRepo.GetProductsByIDsForUpdate(schema, []uint{uint(productID)})
						if err != nil {
							return err
						}
						if len(products) == 0 {
							return fmt.Errorf("PRODUCT_NOT_FOUND: ProductID %d", productID)
						}
						product := &products[0]

						exists, member, err := memberRepo.GetForUpdate(schema, uint(userID))
						if err != nil {
							return err
						}
						if !exists {
							return fmt.Errorf("MEMBER_NOT_FOUND: UserID %d", userID)
						}

						if txType == "pickup" {
							member.Balance = member.Balance.Sub(amount)
							product.Stock = product.Stock - quantity

							// 終端水位校驗：防範資料庫數值穿透
							if member.Balance.IsNegative() || product.Stock < 0 {
								return errors.New("BUSINESS_LIMIT_EXCEEDED: Insufficient balance or stock in database")
							}
						} else if txType == "return" {
							member.Balance = member.Balance.Add(amount)
							product.Stock = product.Stock + quantity
						} else {
							return errors.New("INVALID_TX_TYPE: Unsupported order transaction type")
						}

						if err := tradeRepo.UpdateProduct(schema, product); err != nil {
							return err
						}
						if err := memberRepo.UpdateMember(schema, member); err != nil {
							return err
						}

						// 依據 MQ 包內攜帶的售價歷史快照，實打實地寫入歷史流水帳
						history := entities.Transaction{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							MemberID:  uint(userID),
							ProductID: uint(productID),
							Quantity:  quantity,
							Amount:    amount,
							Type:      txType,
							Status:    "success",
						}

						return tradeRepo.CreateTransaction(schema, history)
					})

					// 核心硬化修復：極端狀況落盤失敗，反向同步還原全快取陣列（包含獨立原子餘額計數器）
					if err != nil {
						log.Printf("[WARNING] Transaction commit failed. Cache compensation triggered. UserID: %d, Error: %v", userID, err)

						memberBalanceKey := fmt.Sprintf("cache:member:balance:%s:%d", schema, userID)
						productStockKey := fmt.Sprintf("cache:product:stock:%s:%d", schema, productID)

						if txType == "pickup" {
							_ = database.RDB.IncrBy(ctx, memberBalanceKey, amount.IntPart()).Err()
							_ = database.RDB.IncrBy(ctx, productStockKey, quantity).Err()
						} else if txType == "return" {
							_ = database.RDB.DecrBy(ctx, memberBalanceKey, amount.IntPart()).Err()
							_ = database.RDB.DecrBy(ctx, productStockKey, quantity).Err()
						}
					}

					// 成功處理完成，物理劃銷任務包，並移動指針游標防止重複消費
					database.RDB.XDel(ctx, streamName, message.ID)
					lastIdx = message.ID
				}
			}
		}
	}
}
