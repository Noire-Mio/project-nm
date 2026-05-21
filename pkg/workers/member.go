package workers

import (
	"context"
	"errors"
	"log"
	"project-nm/pkg/database"
	"project-nm/pkg/entities"
	"project-nm/pkg/repositories" // 確保有引入你的 repo
	"strconv"
	"time"

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
	log.Println("[MQ Worker] 會員初始化非同步已啟動...")

	for {
		select {
		case <-ctx.Done():
			// 收到外層 WorkerManager 發送的停機訊號
			log.Println("[MQ Worker] 收到停機通知，已安全停止消費。")
			return

		default:
			// 每 2 秒如果沒有新任務，Worker 會醒來並走回 select 檢查有沒有收到關機指令！
			streams, err := database.RDB.XRead(ctx, &redis.XReadArgs{
				Streams: []string{streamName, "0"},
				Count:   1,
				Block:   2 * time.Second,
			}).Result()

			if err != nil {
				// 如果 err 是 redis.Nil 代表這 2 秒內 Stream 都沒有新包裹，這是正常的，直接繼續
				if errors.Is(err, redis.Nil) {
					continue
				}
				// 如果是連線中斷或其他致命錯誤，休息 1 秒防止日誌爆掉
				log.Printf("[MQ Worker] Redis Stream 讀取異常: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// 解析任務並寫入
			for _, stream := range streams {
				for _, message := range stream.Messages {
					userIdStr := message.Values["user_id"].(string)
					username := message.Values["username"].(string)
					schema := message.Values["schema"].(string)

					userId, _ := strconv.ParseUint(userIdStr, 10, 64)

					newMember := entities.Member{
						ID:       uint(userId),
						Username: username,
						Balance:  decimal.NewFromInt(0),
					}

					dbCtx := repositories.NewGormDBContext(database.DB)
					memberRepo := w.MemberRepoFactory(dbCtx)

					// 寫入資料庫
					_, err := memberRepo.Create(schema, newMember)
					if err != nil {
						log.Printf("[MQ Worker] 非同步初始化失敗 (Schema: %s, UserID: %d): %v", schema, userId, err)
					} else {
						log.Printf("[MQ Worker] Schema %s 初始化會員 ID: %d", schema, userId)
					}

					// 把任務刪除
					database.RDB.XDel(ctx, streamName, message.ID)
				}
			}
		}
	}
}
