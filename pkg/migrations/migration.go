package migrations

import (
	"fmt"
	"log"
	"project-nm/pkg/migrations/scripts/public"
	"project-nm/pkg/migrations/scripts/schemas"
	"time"

	"gorm.io/gorm"
)

func RunMigration(db *gorm.DB) error {
	log.Println("[Migration] --- Starting Public Schema Migration ---")
	if err := db.Exec(`SET search_path TO "public"`).Error; err != nil {
		return fmt.Errorf("failed to set public search_path: %w", err)
	}
	if err := db.AutoMigrate(&MigrateRecord{}); err != nil {
		return err
	}
	execute(db, "20260513-001", "建立User", public.CreateUserTable)

	// 初始化 Tenant Schema
	targetSchemas := []string{"tenant_001", "tenant_002", "tenant_003"}
	for _, schema := range targetSchemas {
		log.Printf("[Migration] --- Starting Tenant Schema: %s ---", schema)

		createSchemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema)
		if err := db.Exec(createSchemaSQL).Error; err != nil {
			return fmt.Errorf("failed to create schema %s: %w", schema, err)
		}

		if err := db.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema)).Error; err != nil {
			return fmt.Errorf("failed to set search_path to %s: %w", schema, err)
		}

		if err := db.AutoMigrate(&MigrateRecord{}); err != nil {
			return err
		}

		execute(db, "20260513-001", "建立會員表和交易表", schemas.CreateMemberTransactionTable)
		execute(db, "20260522-001", "建立商品表", schemas.CreateProductTable)

		log.Printf("[Migration] Schema %s completed", schema)
	}

	return db.Exec("SET search_path TO public").Error
}

// execute 封裝判斷與執行邏輯
func execute(db *gorm.DB, no string, describe string, up func(*gorm.DB) error) {
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
	// 讀取目前 Schema 的 migrate_records
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
