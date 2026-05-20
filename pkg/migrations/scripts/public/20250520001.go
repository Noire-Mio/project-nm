package public

import (
	"fmt"
	"log"
	"project-nm/pkg/entities"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 建立 User / UserPermission 表，並同時建立預設帳號
func CreateUserTable(db *gorm.DB) error {
	// 建立表結構
	if err := db.AutoMigrate(&entities.User{}, &entities.UserPermission{}); err != nil {
		return fmt.Errorf("failed to migrate user tables: %w", err)
	}

	// 建立預設帳號
	if err := seedDefaultUser(db); err != nil {
		return fmt.Errorf("failed to seed default user: %w", err)
	}

	return nil
}

// 建立預設展示帳號
func seedDefaultUser(db *gorm.DB) error {
	var count int64
	db.Table("public.users").Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("[Migration] [Seed] No user found. Creating default admin user for demo...")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	adminUser := entities.User{
		Account:  "admin",
		Password: string(hashedPassword),
		Name:     "測試帳號",
		Identity: "admin",
		Schema:   "tenant_001",
		IsActive: true,
	}

	if err := db.Table("public.users").Create(&adminUser).Error; err != nil {
		return err
	}

	demoPermissions := []entities.UserPermission{
		{UserID: adminUser.ID, Permission: "read"},
		{UserID: adminUser.ID, Permission: "write"},
	}

	if err := db.Table("public.user_permissions").Create(&demoPermissions).Error; err != nil {
		return err
	}

	log.Println("[Migration] [Seed] Default user 'admin' with password 'admin123' created successfully.")
	return nil
}
