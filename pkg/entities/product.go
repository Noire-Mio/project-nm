package entities

import "github.com/shopspring/decimal"

type Product struct {
	ID      uint            `gorm:"primaryKey" json:"id"`
	Name    string          `gorm:"size:50;uniqueIndex" json:"name"`
	Stock   int64           `gorm:"default:0" json:"stock"`          // 庫存數量
	Price   decimal.Decimal `gorm:"type:decimal(16,2)" json:"price"` // 商品單價
	Version int64           `gorm:"default:0" json:"version"`        // 商品樂觀鎖
}
func (m *Product) TableName() string {
	return "product"
}