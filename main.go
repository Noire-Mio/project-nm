package main

import (
	"project-nm/pkg/configs"
	"project-nm/pkg/database"
	"time"
)

func main() {
	configs.SetConfig()
	systemConfig := configs.GetConfig()

	// 初始化資料庫
	migrateDb := database.InitDatabase(systemConfig.RelationalDB)
	mainDb := database.InitDatabase(systemConfig.RelationalDB)

	sqlDB, _ := mainDb.DB()
	sqlDB.SetMaxOpenConns(systemConfig.RelationalDB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(systemConfig.RelationalDB.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(systemConfig.RelationalDB.MaxLifetime) * time.Minute)

	// 初始化 Redis
	rdb := database.InitRedis(systemConfig.Redis)
	defer rdb.Close()

	// 傳入 mainDb 與 rdb 初始化 App
	app := InitApplication(mainDb, rdb)
	app.Serve(migrateDb)
}
