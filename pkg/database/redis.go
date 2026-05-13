package database

import (
	"context"
	"fmt"
	"log"
	"project-nm/pkg/configs"
	"time"

	"github.com/redis/go-redis/v9"
)

// RDB 全域 Redis 用戶端，供其他套件直接調用
var RDB *redis.Client

// 定義 Stream 相關常數，確保全專案一致
const (
	MemberTxStream = "member_tx_stream"
	MemberTxGroup  = "member_tx_group"
)

// InitRedis 初始化 Redis 連線池
func InitRedis(cfg configs.RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,

		// 高併發下的超時設定，防止單一請求卡死連線池
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		// 連線閒置管理
		MinIdleConns: 10,
	})

	// 使用 Context 進行連線測試
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("[project-nm] Failed to connect to Redis: %v", err)
	}

	log.Printf("[project-nm] Redis connection pool initialized (PoolSize: %d)", cfg.PoolSize)

	RDB = rdb
	return rdb
}

// InitRedisStream 初始化 Redis Stream 和消費者群組
// 確保通道已經準備好了
func InitRedisStream() {
	if RDB == nil {
		log.Fatal("[project-nm] Redis 尚未初始化，無法建立 Stream")
	}

	ctx := context.Background()

	// 嘗試建立消費者群組
	// 如果 member_tx_stream 不存在，Redis 會自動建立它
	// "0" 代表從 Stream 的最開頭開始讀取訊息
	err := RDB.XGroupCreateMkStream(ctx, MemberTxStream, MemberTxGroup, "0").Err()

	if err != nil {
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			log.Println("[project-nm] Redis Stream Group already exists, skipping...")
			return
		}
		log.Printf("[project-nm] Failed to create Redis Stream Group: %v", err)
	} else {
		log.Println("[project-nm] Redis Stream and Consumer Group initialized successfully")
	}
}
