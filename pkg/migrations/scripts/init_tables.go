package scripts

import (
	"project-nm/pkg/entities"

	"gorm.io/gorm"
)

// CreateUserTable 建立使用者資料表
func CreateUserTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.User{})
}
