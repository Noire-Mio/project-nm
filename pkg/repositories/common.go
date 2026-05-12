package repositories

import (
	"fmt"

	"gorm.io/gorm"
)

// Editing mode (編修模式)
type EditedMode uint

type MultiSchemaMdl struct {
	BusinessDateSchema string
}

func WithSchema(schema string, tableName string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Table(fmt.Sprintf("s%s.%s", schema, tableName))
	}
}
