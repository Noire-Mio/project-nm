package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time       `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"column:updated_at"`
	MemberID  uint            `gorm:"index" json:"member_id"`    
	ProductID uint            `gorm:"index" json:"product_id"`   
	Quantity  int64           `gorm:"default:1" json:"quantity"` 
	Amount    decimal.Decimal `gorm:"type:decimal(16,2)" json:"amount"`
	Type      string          `gorm:"size:20" json:"type"`   // "pickup" 或 "return"
	Status    string          `gorm:"size:20" json:"status"` // "pending", "success", "failed"
}

func (m *Transaction) TableName() string {
	return "transaction"
}
