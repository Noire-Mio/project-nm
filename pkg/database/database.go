package database

import (
	"fmt"
	"log"
	"project-nm/pkg/configs" // 確保此路徑與您的 go.mod 一致
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase 初始化資料庫連線並配置連線池
func InitDatabase(cfg configs.RelationalDB) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Taipei",
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.SslMode)

	// 開啟連線
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 高併發下關閉詳細日誌以提升效能
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 取得底層 sql.DB 物件以配置連線池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// 連線池參數優化
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)                                // 最大連線數
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)                                // 保持空閒連線，減少建立連線的延遲
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Minute) // 連線生命週期

	// 驗證連線是否可用
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	log.Printf("[project-nm] Database connection pool initialized (MaxOpen: %d)", cfg.MaxOpenConns)

	DB = db
	return db
}
