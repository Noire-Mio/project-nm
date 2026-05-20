package migrations

import (
	"time"
)

// MigrateRecord
// 資料庫更版紀錄
type MigrateRecord struct {
	ID         uint      `gorm:"primaryKey"`
	SystemName string    `gorm:"primaryKey;Column:system_name;size:30"`    // 系統名稱
	No         string    `gorm:"primaryKey;Column:no;size:12"`             // 編號(8碼西元年月日-三碼編號 ex:20230721-001)
	Describe   string    `gorm:"Column:describe;Column:describe;size:200"` // 改版描述
	CreatedAt  time.Time `gorm:"Column:created_at"`                        // 建立時間
}

func (e *MigrateRecord) TableName() string {
	return "migrate_record"
}
