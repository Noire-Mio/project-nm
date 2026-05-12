package main

import (
	"project-nm/pkg/configs"
	"project-nm/pkg/database"
)

func main() {
	configs.SetConfig()
	cfg := configs.GetConfig()

	// 初始化資料庫
	database.InitDatabase(cfg.RelationalDB)

	// 初始化 Redis
	database.InitRedis(cfg.Redis)

	// 啟動 Gin...
}
