package configs

// getConfigByDefault 傳回符合 project-nm 高併發需求的預設配置
func getConfigByDefault() Config {
	return Config{
		ProjectID: "project-nm",
		JWTSign:   "nm-secret-key-2026", // 建議之後由環境變數注入
		RelationalDB: RelationalDB{
			Type:         "postgres",
			Host:         "localhost",
			Port:         "5432",
			User:         "postgres",
			Password:     "password",
			Database:     "sjdb",
			SslMode:      "disable",
			MaxOpenConns: 150, // 針對 QPS 2000 預設 150 個連線
			MaxIdleConns: 50,  // 保持 50 個熱連線
			MaxLifetime:  30,  // 每 30 分鐘回收連線
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
			PoolSize: 300, // 支撐高併發的連線池大小
		},
	}
}
