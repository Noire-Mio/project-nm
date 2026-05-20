package utils

import (
	"context"
	"encoding/json"
	"project-nm/pkg/contexts"
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

// SetUserToken 將登入成功的 UserInfo 快取至 Redis，並設定 Token 的有效期限
func SetUserToken(token string, userInfo *contexts.UserInfo, expiration time.Duration) error {
	ctx := context.Background()

	// 將結構體序列化成 JSON 字串
	data, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}
	key := "token:" + token
	return database.RDB.Set(ctx, key, string(data), expiration).Err()
}

// GetUserToken
func GetUserToken(token string) (*contexts.UserInfo, error) {
	ctx := context.Background()
	key := "token:" + token

	val, err := database.RDB.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var userInfo contexts.UserInfo
	err = json.Unmarshal([]byte(val), &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// DeleteUserToken 登出時，使 Token 立即失效
func DeleteUserToken(token string) error {
	ctx := context.Background()
	key := "token:" + token
	return database.RDB.Del(ctx, key).Err()
}


