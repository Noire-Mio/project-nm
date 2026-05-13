package main

import (
	"project-nm/pkg/configs"
	"project-nm/pkg/database"
)

func main() {
	configs.SetConfig()
	systemConfig := configs.GetConfig()

	db := database.InitDatabase(systemConfig.RelationalDB)

	rdb := database.InitRedis(systemConfig.Redis)
	defer rdb.Close()

	database.InitRedisStream()

	app := InitApplication(db, rdb)

	app.Serve(db)
}
