package utils

import (
	"context"
	"project-nm/pkg/database" 
	"time"

	"github.com/redis/go-redis/v9"
)

// SetCache 設置快取 
func SetCache(key string, value string, expiration time.Duration) error {
	return database.RDB.Set(context.Background(), key, value, expiration).Err()
}

// GetCache 獲取快取
func GetCache(key string) (string, error) {
	return database.RDB.Get(context.Background(), key).Result()
}

// PushToStream 將資料推入 Redis Stream
func PushToStream(streamName string, data map[string]interface{}) error {
	return database.RDB.XAdd(context.Background(), &redis.XAddArgs{
		Stream: streamName,
		Values: data,
	}).Err()
}
