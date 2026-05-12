package configs

import "os"

func getConfigByDefault() Config {
	// 優先嘗試從環境變數讀取，若無則使用預設值
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")

	return Config{
		ProjectID: "project-nm",
		JWTSign:   "nm-secret-key-2026",
		RelationalDB: RelationalDB{
			Type:         "postgres",
			Host:         dbHost,
			Port:         dbPort,
			User:         "postgres",
			Password:     "password",
			Database:     "sjdb",
			SslMode:      "disable",
			MaxOpenConns: 150,
			MaxIdleConns: 50,
			MaxLifetime:  30,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     "6379",
			Password: "",
			DB:       0,
			PoolSize: 300,
		},
		Token: TokenConfig{
			AccessSecret:  "nm-access-key-32-chars",
			RefreshSecret: "nm-refresh-key-32-chars",
			AccessExpire:  3600,
			RefreshExpire: 604800,
		},
	}
}

// 輔助函式：讀取環境變數，若不存在則回傳預設值
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
