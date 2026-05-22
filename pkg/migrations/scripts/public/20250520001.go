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
// func seedDefaultUser(db *gorm.DB) error {
// 	var count int64
// 	db.Table("public.users").Count(&count)
// 	if count > 0 {
// 		return nil
// 	}

// 	log.Println("[Migration] [Seed] No user found. Creating default admin user for demo...")

// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
// 	if err != nil {
// 		return fmt.Errorf("failed to hash password: %w", err)
// 	}

// 	adminUser := entities.User{
// 		Account:  "admin",
// 		Password: string(hashedPassword),
// 		Name:     "測試帳號",
// 		Identity: "admin",
// 		Schema:   "tenant_001",
// 		IsActive: true,
// 	}

// 	if err := db.Table("public.users").Create(&adminUser).Error; err != nil {
// 		return err
// 	}

// 	demoPermissions := []entities.UserPermission{
// 		{UserID: adminUser.ID, Permission: "read"},
// 		{UserID: adminUser.ID, Permission: "write"},
// 	}

// 	if err := db.Table("public.user_permissions").Create(&demoPermissions).Error; err != nil {
// 		return err
// 	}

// 	log.Println("[Migration] [Seed] Default user 'admin' with password 'admin123' created successfully.")
// 	return nil
// }

// 建立 2000 個測試帳號
func seedDefaultUser(db *gorm.DB) error {
	var count int64
	db.Table("public.users").Count(&count)
	if count > 0 {
		log.Println("[Migration] [Seed] Users already exist. Skipping seed.")
		return nil
	}

	totalUsers := 2000
	log.Printf("[Migration] [Seed] No users found. Generating %d default users for multi-tenant concurrent demo...", totalUsers)

	// 密碼雜湊
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	passwordStr := string(hashedPassword)

	// 準備切片（Slice）收集大量資料
	users := make([]entities.User, totalUsers)

	// 組裝 2000 筆用戶資料
	for i := 0; i < totalUsers; i++ {
		idNum := i + 1
		
		// 隨機分流租戶：tenant_001, tenant_002, tenant_003
		schema := fmt.Sprintf("tenant_%03d", (i%3)+1)

		users[i] = entities.User{
			Account:  fmt.Sprintf("admin%04d", idNum), 
			Password: passwordStr,
			Name:     fmt.Sprintf("測試帳號%04d", idNum), 
			Identity: "admin",
			Schema:   schema,
			IsActive: true,
		}
	}

	
	
	// 使用 GORM CreateInBatches 執行批量寫入 ，每 100 筆打包成一條 SQL 語句發送
	log.Println("[Migration] [Seed] Inserting users into database in batches...")
	if err := db.Table("public.users").CreateInBatches(&users, 100).Error; err != nil {
		return fmt.Errorf("failed to batch insert users: %w", err)
	}

	permissions := make([]entities.UserPermission, 0, totalUsers*2)
	for _, user := range users {
		permissions = append(permissions,
			entities.UserPermission{UserID: user.ID, Permission: "read"},
			entities.UserPermission{UserID: user.ID, Permission: "write"},
		)
	}

	// 權限表也採用批量寫入
	log.Println("[Migration] [Seed] Inserting user permissions in batches...")
	if err := db.Table("public.user_permissions").CreateInBatches(&permissions, 200).Error; err != nil {
		return fmt.Errorf("failed to batch insert permissions: %w", err)
	}

	log.Printf("[Migration] [Seed] Successfully generated %d admin users (admin0001~admin2000) and %d permissions across tenant_001~003!", totalUsers, len(permissions))
	return nil
}