package configs

import (
	"os"
)

var config *Config

// RelationalDB
type RelationalDB struct {
	Type          string
	Host          string
	Port          string
	User          string
	Password      string
	Database      string
	DefaultSchema string // 新增：支援多 Schema 的初始預設值
	SslMode       string
	MaxOpenConns  int // 最大連線數，QPS 2000 關鍵參數
	MaxIdleConns  int // 最大閒置連線數
	MaxLifetime   int // 連線存活時間 (分鐘)
}

// RedisConfig
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int // 針對 QPS 2000 建議設為 200-500
}

// TokenConfig
type TokenConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpire  int // 單位：秒
	RefreshExpire int // 單位：秒
}

// Config 專案總合配置
type Config struct {
	ProjectID    string
	ServerPort   string 
	JWTSign      string
	RelationalDB RelationalDB
	Redis        RedisConfig
	Token        TokenConfig
}

// SetConfig 初始化配置
func SetConfig() {
	// 檢查並設定環境模式 (DEV/PROD/TEST)
	envMode := os.Getenv("ENV_MODE")
	if envMode == "" {
		envMode = "DEV"
		os.Setenv("ENV_MODE", envMode)
	}

	// 取得基礎配置
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

// Helper function: getEnv 讓 default.go 可以更輕鬆地讀取環境變數
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// DefaultSchema 欄位：
// 這是為了應對您的多 Schema 需求。在 PostgreSQL 中，我們通常會有一個 public 或特定的初始 Schema。將其放入配置中，可以讓 pkg/database 在連線後執行 SET search_path TO ... 的初始化動作。

// GetEnv 封裝：
// 我新增了一個輔助函數。這樣在您的 default.go 檔案中，您就可以寫出像 Host: GetEnv("DB_HOST", "localhost") 這樣的程式碼，讓 Docker Compose 注入的變數（如 DB_HOST=postgres）能生效。

// 安全性（Panic）：
// 保留了 panic 檢查。在高併發系統中，若配置未載入就嘗試讀取，會導致不可預期的 Null 指標錯誤，及早拋出錯誤有助於除錯。

// 如何導入
// 覆蓋檔案：將上述程式碼貼入 pkg/configs/config.go。

// 更新 default.go：確保您的 default.go 也同步更新，填入 DefaultSchema 的數值（例如 "public"）。

// 資料庫初始化：接下來我們將在 pkg/database 中使用這個新欄位來設定連線。
