package configs

import (
	"os"
)

var config *Config

// RelationalDB 儲存資料庫連線與連線池參數
type RelationalDB struct {
	Type         string
	Host         string
	Port         string
	User         string
	Password     string
	Database     string
	SslMode      string
	MaxOpenConns int // 最大連線數，QPS 2000 關鍵參數
	MaxIdleConns int // 最大閒置連線數
	MaxLifetime  int // 連線存活時間 (分鐘)
}

// RedisConfig 針對高併發快取設計
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

type TokenConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpire  int // 單位：秒
	RefreshExpire int // 單位：秒
}

// Config 專案總合配置
type Config struct {
	ProjectID    string
	JWTSign      string
	RelationalDB RelationalDB
	Redis        RedisConfig
	Token        TokenConfig
}

// SetConfig 初始化配置
func SetConfig() {
	// 檢查並設定環境模式
	envMode := os.Getenv("ENV_MODE")
	if envMode == "" {
		os.Setenv("ENV_MODE", "DEV")
	}

	// 取得預設配置
	resp := getConfigByDefault()
	config = &resp
}

// GetConfig 提供外部取得配置的入口 (Getter)
func GetConfig() Config {
	if config == nil {
		panic("Config has not been initialized. Call SetConfig() first.")
	}
	return *config
}
