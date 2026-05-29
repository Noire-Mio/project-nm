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
	"project-nm/pkg/utils"

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
	log.Println("[Trade Worker] 交易落盤非同步任務監聽已啟動...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[Trade Worker] 收到停機通知，安全關閉消費。")
			return

		default:
			streams, err := database.RDB.XRead(ctx, &redis.XReadArgs{
				Streams: []string{streamName, "0"},
				Count:   100,
				Block:   2 * time.Second,
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				log.Printf("[Trade Worker] Redis Stream 讀取異常: %v", err)
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

					dbCtx := repositories.NewGormDBContext(database.DB)
					tradeRepo := w.TradeRepoFactory(dbCtx)
					memberRepo := w.MemberRepoFactory(dbCtx)

					maxRetries := 5
					txSuccess := false

					for i := 0; i < maxRetries; i++ {
						err = database.DB.Transaction(func(tx *gorm.DB) error {
							product, err := tradeRepo.GetProduct(tx, uint(productID))
							if err != nil {
								return err
							}

							member, err := memberRepo.GetWithTx(tx, schema, uint(userID))
							if err != nil {
								return err
							}

							if txType == "pickup" {
								member.Balance = member.Balance.Sub(amount)
								product.Stock = product.Stock - quantity

								if member.Balance.IsNegative() || product.Stock < 0 {
									return errors.New("BUSINESS_LIMIT_EXCEEDED: 資料庫餘額或庫存不足")
								}
							} else if txType == "return" {
								member.Balance = member.Balance.Add(amount)
								product.Stock = product.Stock + quantity
							} else {
								return errors.New("INVALID_TX_TYPE: 未知的交易類型")
							}

							err = tradeRepo.UpdateProduct(tx, product)
							if err != nil {
								return err
							}

							err = memberRepo.UpdateMember(tx, schema, member)
							if err != nil {
								return err
							}

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
							return tradeRepo.CreateTransaction(tx, history)
						})

						if err == nil {
							txSuccess = true
							break
						}

						if err.Error() == "OPTIMISTIC_LOCK_CONFLICT_PRODUCT" || err.Error() == "OPTIMISTIC_LOCK_CONFLICT_MEMBER" {
							time.Sleep(5 * time.Millisecond)
							continue
						}

						break
					}

					if !txSuccess {
						log.Printf("[Trade Worker] 交易落盤重度衝突或失敗，啟動快取還原 (UserID: %d, Error: %v)", userID, err)

						memberCache, cErr := utils.GetMemberCache(schema, uint(userID))
						productStockKey := fmt.Sprintf("product:stock:%d", productID)

						if cErr == nil && memberCache != nil {
							if txType == "pickup" {
								memberCache.Balance = memberCache.Balance.Add(amount)
								database.RDB.IncrBy(ctx, productStockKey, quantity)
							} else if txType == "return" {
								memberCache.Balance = memberCache.Balance.Sub(amount)
								database.RDB.DecrBy(ctx, productStockKey, quantity)
							}
							_ = utils.SetMemberCache(schema, memberCache, 30*time.Minute)
						}
					}

					database.RDB.XDel(ctx, streamName, message.ID)
				}
			}
		}
	}
}
