package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Member struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time       `json:"created_at" gorm:"Column:created_at"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"Column:updated_at"`
	Username  string          `gorm:"uniqueIndex;size:50" json:"username"`
	Balance   decimal.Decimal `gorm:"type:decimal(16,2);default:0" json:"balance"`
	Version   int64           `gorm:"default:0" json:"version"`
}

// InitialPoints 設定會員註冊時的初始點數 (例如 1000 點)
const InitialPoints = 1000
