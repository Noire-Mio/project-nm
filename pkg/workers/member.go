package workers

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"project-nm/pkg/database"
	"project-nm/pkg/entities"
	"project-nm/pkg/repositories"

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
	log.Println("[MQ Worker] 會員初始化非同步任務監聽已啟動...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[MQ Worker] 收到停機通知，已安全停止消費。")
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
				log.Printf("[MQ Worker] Redis Stream 讀取異常: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// 解析任務並進行資料庫落盤
			for _, stream := range streams {
				for _, message := range stream.Messages {
					userIdStr, _ := message.Values["user_id"].(string)
					username, _ := message.Values["username"].(string)
					schema, _ := message.Values["schema"].(string)

					userId, err := strconv.ParseUint(userIdStr, 10, 64)
					if err != nil {
						log.Printf("[MQ Worker] 解析使用者 ID 失敗: %v", err)
						database.RDB.XDel(ctx, streamName, message.ID)
						continue
					}

					// 建立新會員
					newMember := entities.Member{
						ID:       uint(userId),
						Username: username,
						Balance:  decimal.NewFromInt(1000),
						Version:  1, 
					}

					dbCtx := repositories.NewGormDBContext(database.DB)
					memberRepo := w.MemberRepoFactory(dbCtx)

					// 寫入指定租戶的資料庫中
					err = memberRepo.Create(schema, newMember)
					if err != nil {
						log.Printf("[MQ Worker] 非同步資料庫寫入失敗 (Schema: %s, UserID: %d): %v", schema, userId, err)
					} else {
						log.Printf("[MQ Worker] 資料庫落盤成功 (Schema: %s, UserID: %d)", schema, userId)
					}

					// 任務處理完畢自消息佇列中移除該訊息
					database.RDB.XDel(ctx, streamName, message.ID)
				}
			}
		}
	}
}
