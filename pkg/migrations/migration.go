package migrations

import (
	"fmt"
	"log"
	"project-nm/pkg/migrations/scripts"
	"time"

	"gorm.io/gorm"
)

// MigrateRecord 紀錄更版歷史 
type MigrateRecord struct {
	ID         uint   `gorm:"primaryKey"`
	SystemName string `gorm:"size:50;index"`
	No         string `gorm:"size:20;index"`
	Describe   string `gorm:"size:250"`
	CreatedAt  time.Time
}

func RunMigration(db *gorm.DB) error {
	// 初始化 Public Schema 
	log.Println("[Migration] --- Starting Public Schema Migration ---")
	if err := db.Exec(`SET search_path TO "public"`).Error; err != nil {
		return fmt.Errorf("failed to set public search_path: %w", err)
	}

	// 確保 Public 有紀錄表
	if err := db.AutoMigrate(&MigrateRecord{}); err != nil {
		return err
	}

	// 執行 腳本 
	execute(db, "20260512-001", "Initialize User Table", scripts.CreateUserTable)

	// 初始化 Tenant Schema 
	targetSchemas := []string{"tenant_001"}

	for _, schema := range targetSchemas {
		log.Printf("[Migration] --- Starting Tenant Schema: %s ---", schema)

		// 如果不存在自動建立 Schema
		createSchemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema)
		if err := db.Exec(createSchemaSQL).Error; err != nil {
			return fmt.Errorf("failed to create schema %s: %w", schema, err)
		}

		// 切換該 Schema 環境
		if err := db.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema)).Error; err != nil {
			return fmt.Errorf("failed to set search_path to %s: %w", schema, err)
		}

		// 確保該 Schema 內有自己的紀錄表
		if err := db.AutoMigrate(&MigrateRecord{}); err != nil {
			return err
		}

		// 執行業務腳本
		execute(db, "20260513-001", "Initialize Member & Transaction Table", scripts.CreateMemberTransactionTable)

		log.Printf("[Migration] Schema %s completed", schema)
	}

	// 最後將 search_path 切回 public，避免影響後續連線
	return db.Exec("SET search_path TO public").Error
}

// execute 封裝判斷與執行邏輯
func execute(db *gorm.DB, no string, describe string, up func(*gorm.DB) error) {
	// 讀取該 Schema 下已執行的紀錄
	executedRecords := loadRecords(db, "project-nm")

	if executedRecords[no] {
		return
	}

	log.Printf("[Migration] [%s] Executing: %s", no, describe)

	// 執行傳入的腳本函式
	if err := up(db); err != nil {
		log.Fatalf("[Migration] CRITICAL FAILURE at %s: %v", no, err)
	}

	// 寫入執行紀錄
	writeRecord(db, "project-nm", no, describe)
}

func loadRecords(db *gorm.DB, systemName string) map[string]bool {
	var nos []string
	// 由於已經 SET search_path，這裡會讀取目前 Schema 的 migrate_records
	db.Model(&MigrateRecord{}).Where("system_name = ?", systemName).Pluck("no", &nos)

	recordMap := make(map[string]bool)
	for _, no := range nos {
		recordMap[no] = true
	}
	return recordMap
}

func writeRecord(db *gorm.DB, systemName, no, describe string) {
	record := MigrateRecord{
		SystemName: systemName,
		No:         no,
		Describe:   describe,
		CreatedAt:  time.Now(),
	}
	db.Create(&record)
}
