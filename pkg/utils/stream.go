package utils

import (
	"context"
	"project-nm/pkg/database"

	"github.com/redis/go-redis/v9"
)

// ProduceMessage 將訊息發送到 Redis Stream
func ProduceMessage(streamName string, data map[string]interface{}) (string, error) {
	ctx := context.Background()
	
	// XAdd 將資料寫入 Stream，"*" 代表由 Redis 自動產生訊息 ID
	id, err := database.RDB.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: data,
	}).Result()
	
	return id, err
}

// CreateConsumerGroup 建立消費者組 (通常在啟動時執行一次)
func CreateConsumerGroup(streamName, groupName string) {
	ctx := context.Background()
	// 從 Stream 的開頭 ($ 代表最新消息, 0 代表從頭開始) 建立組
	database.RDB.XGroupCreateMkStream(ctx, streamName, groupName, "0")
}


// XAdd: 這是生產者的核心動作。在高併發 API 請求中，我們將非同步任務（例如：發送註冊郵件、日誌分析）寫入 Stream 後立即回應用戶。

// XGroupCreateMkStream: 這建立了消費者組。它的優點是即使有多個 nm-app 實例，Redis 也會確保一條訊息只會被其中一個實例處理，達成負載平衡。

// 效能考量: 因為我們已經在 InitRedis 中設定了 PoolSize: 300，這足以支撐生產者頻繁的寫入操作。