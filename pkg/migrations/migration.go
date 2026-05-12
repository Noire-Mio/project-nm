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

// RunMigration 執行所有遷移作業
func RunMigration(db *gorm.DB) error {
	schemas, err := getSchemas(db)
	if err != nil {
		return err
	}

	for _, schema := range schemas {
		log.Printf("[Migration] Starting migration for schema: %s", schema)

		// 切換 Session 環境
		if err := db.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema)).Error; err != nil {
			return fmt.Errorf("failed to set search_path: %w", err)
		}

		// 確保每個 Schema 都有自己的紀錄表
		if err := db.AutoMigrate(&MigrateRecord{}); err != nil {
			return err
		}

		executedRecords := loadRecords(db, "project-nm")

		fn := func(no string, describe string, up func(*gorm.DB) error) {
			if executedRecords[no] {
				return
			}
			log.Printf("[Migration] [%s] Executing: %s", no, describe)
			if err := up(db); err != nil {
				panic(fmt.Sprintf("Migration failed at %s: %v", no, err))
			}
			writeRecord(db, "project-nm", no, describe)
		}

		// --- 腳本清單 ---
		fn("20260512-001", "Initialize User Table", scripts.CreateUserTable)

		log.Printf("[Migration] Schema %s migration completed", schema)
	}

	// 恢復預設路徑避免影響後續 Session
	return db.Exec("SET search_path TO public").Error
}

func getSchemas(db *gorm.DB) ([]string, error) {
	var schemas []string
	rows, err := db.Raw(`
		SELECT nspname FROM pg_namespace 
		WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'public') 
		AND nspname NOT LIKE 'pg_temp%' 
		AND nspname NOT LIKE 'pg_toast%'
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		schemas = append(schemas, s)
	}
	return schemas, nil
}

func loadRecords(db *gorm.DB, systemName string) map[string]bool {
	var nos []string
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
	}
	db.Create(&record)
}
