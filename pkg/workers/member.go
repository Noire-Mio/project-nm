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
)

type MemberInitWorker struct {
	MemberRepoFactory repositories.MemberFactory
}

func NewMemberInitWorker(factory repositories.MemberFactory) *MemberInitWorker {
	return &MemberInitWorker{
		MemberRepoFactory: factory,
	}
}

func (w *MemberInitWorker) Start(ctx context.Context) {
	streamName := "stream:member_init_tasks"
	log.Printf("[INFO] MemberInitWorker execution loop started. Target stream: %s", streamName)

	// 使用時間戳 $ 作為起跑線，確保只消費啟動後進入隊列的新任務
	lastIdx := "$"

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] MemberInitWorker received shutdown signal. Terminating consumption loop cleanly.")
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
					username, _ := message.Values["username"].(string)
					schema, _ := message.Values["schema"].(string)

					userId, err := strconv.ParseUint(userIdStr, 10, 64)
					if err != nil {
						log.Printf("[ERROR] Parameter parsing failed. MessageID: %s, Error: %v", message.ID, err)
						database.RDB.XDel(ctx, streamName, message.ID)
						continue
					}

					initialBalance := decimal.NewFromInt(1000)
					newMember := entities.Member{
						ID:       uint(userId),
						Username: username,
						Balance:  initialBalance,
						Version:  1,
					}

					dbCtx := repositories.NewGormDBContext(database.DB)
					memberRepo := w.MemberRepoFactory(dbCtx)

					err = memberRepo.Create(schema, newMember)
					if err != nil {
						log.Printf("[ERROR] Database transaction failed. Schema: %s, UserID: %d, Error: %v", schema, userId, err)
						// 發生主鍵衝突代表硬碟數據早已就位，屬於正常現象，容錯放行
						if !errors.Is(err, errors.New("DUPLICATE_KEY")) {
							continue
						}
					}

					// 修正關鍵點：使用 SetNX（不存在才建立）。若網關早一步初始化且正在交易消耗，工人絕不覆蓋
					memberBalanceKey := fmt.Sprintf("cache:member:balance:%s:%d", schema, userId)
					_ = database.RDB.SetNX(ctx, memberBalanceKey, initialBalance.String(), 0).Err()

					_ = utils.SetMemberCache(schema, &newMember, 30*time.Minute)

					log.Printf("[INFO] Member initialization completed. Schema: %s, UserID: %d, Assets: %s", schema, userId, initialBalance.String())

					database.RDB.XDel(ctx, streamName, message.ID)

					// 移動游標指針，防止重複消費
					lastIdx = message.ID
				}
			}
		}
	}
}
