package database

import (
	"context"
	"fmt"
	"log"
	"project-nm/pkg/configs"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

// InitRedis 初始化 Redis 連線池
func InitRedis(cfg configs.RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize, // 設定連線池大小
		// 高併發下的超時設定
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 使用 Context 進行連線測試
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Printf("[project-nm] Redis connection pool initialized (PoolSize: %d)", cfg.PoolSize)

	RDB = rdb
	return rdb
}

// PoolSize: 300：在高併發環境中，如果連線池太小，新的請求會因為拿不到 Redis 連線而進入等待狀態（Wait Queue），這會導致 API 響應時間大幅增加。

// ctx, cancel：在初始化時使用帶超時的 Context 執行 Ping。這確保了如果 Redis 伺服器掛掉，程式會在 5 秒內報錯退出，而不是無限期卡住。
