package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Member struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
	Username  string          `gorm:"uniqueIndex;size:50" json:"username"`
	Balance   decimal.Decimal `gorm:"type:decimal(16,2);default:0" json:"balance"`
	Version   int64           `gorm:"default:0" json:"version"`
}

func (m *Member) TableName() string {
	return "member"
}
