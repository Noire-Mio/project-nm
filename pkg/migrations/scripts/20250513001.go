package scripts

import (
	"project-nm/pkg/entities"

	"gorm.io/gorm"
)

func CreateMemberTransactionTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Member{}, &entities.Transaction{})
}
