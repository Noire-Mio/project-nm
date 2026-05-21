package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"project-nm/pkg/contexts"
	"project-nm/pkg/database"
	"strconv"
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

// GetUserLatestLoginTime 獲取該用戶目前在全網最新登入的時間戳記
func GetUserLatestLoginTime(userID uint) (int64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:latest_login_time:%d", userID)

	// 🎯 標準改動：先拿字串，再轉為 int64，避免 go-redis 方法誤用噴錯
	val, err := database.RDB.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	loginAt, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return loginAt, nil
}

// GetUserToken 獲取 Token 對應的用戶快取資訊
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

// DeleteUserToken 登出或被剔除時，使 Token 立即失效
func DeleteUserToken(token string) error {
	ctx := context.Background()
	key := "token:" + token
	return database.RDB.Del(ctx, key).Err()
}

// BindUserLatestLoginTime 同時綁定時間戳記與長效 Refresh Token
func BindUserLatestLoginTime(userID uint, loginAt int64, refreshToken string, expiration time.Duration) error {
	ctx := context.Background()

	// 儲存最新合法時間戳記（你原本寫的）
	timeKey := fmt.Sprintf("user:latest_login_time:%d", userID)
	_ = database.RDB.Set(ctx, timeKey, loginAt, expiration).Err()

	// 儲存這台裝置獨佔的最新長效 Refresh Token
	refreshKey := fmt.Sprintf("user:refresh_token:%d", userID)
	return database.RDB.Set(ctx, refreshKey, refreshToken, expiration).Err()
}

// GetServerRefreshToken 獲取該用戶目前全網最新、合法的長效鑰匙
func GetServerRefreshToken(userID uint) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:refresh_token:%d", userID)
	return database.RDB.Get(ctx, key).Result()
}
