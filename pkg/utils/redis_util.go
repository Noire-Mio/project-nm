package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"project-nm/pkg/contexts"
	"project-nm/pkg/database"
	"project-nm/pkg/entities"
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

	// 先拿字串，再轉為 int64，避免 go-redis 方法誤用噴錯
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

// DeleteUserLatestLoginTime 登出時，徹底清除該用戶的最新登入時間與 RefreshToken 紀錄
func DeleteUserLatestLoginTime(userID uint) error {
	ctx := context.Background()

	timeKey := fmt.Sprintf("user:latest_login_time:%d", userID)
	refreshKey := fmt.Sprintf("user:refresh_token:%d", userID)

	// 使用 Redis 的 Del 同時刪除多個 Key，減少網路往返（RTT）
	return database.RDB.Del(ctx, timeKey, refreshKey).Err()
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

// GetMemberCache 獲取會員資料快取
func GetMemberCache(schema string, userID uint) (*entities.Member, error) {
	ctx := context.Background()
	key := fmt.Sprintf("member:cache:%s:%d", schema, userID)

	val, err := database.RDB.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var member entities.Member
	err = json.Unmarshal([]byte(val), &member)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// SetMemberCache 設置會員資料快取
func SetMemberCache(schema string, member *entities.Member, expiration time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("member:cache:%s:%d", schema, member.ID)

	data, err := json.Marshal(member)
	if err != nil {
		return err
	}

	return database.RDB.Set(ctx, key, string(data), expiration).Err()
}

// DeleteMemberCache 刪除會員資料快取
func DeleteMemberCache(schema string, userID uint) error {
	ctx := context.Background()
	key := fmt.Sprintf("member:cache:%s:%d", schema, userID)
	return database.RDB.Del(ctx, key).Err()
}
