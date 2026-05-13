package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time       `json:"created_at" gorm:"Column:created_at"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"Column:updated_at"`
	Amount    decimal.Decimal `gorm:"type:decimal(16,2)"`
	Type      string          `gorm:"size:20"` // "pickup" or "return"
	Status    string          `gorm:"size:20"` // "pending", "success", "failed"
}
