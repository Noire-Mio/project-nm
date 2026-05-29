package repositories

import (
	"errors"
	"project-nm/pkg/entities"

	"gorm.io/gorm"
)

type TradeFactory func(ctx *GormDBContext) ITrade

type ITrade interface {
	GetProduct(tx *gorm.DB, id uint) (*entities.Product, error)
	GetProductsByIDs(tx *gorm.DB, ids []uint) ([]entities.Product, error) 
	UpdateProduct(tx *gorm.DB, product *entities.Product) error
	CreateTransaction(tx *gorm.DB, history entities.Transaction) error
}

type TradeRepo struct {
	GormRepository
}

func NewTradeRepo(ctx *GormDBContext) ITrade {
	repository := new(TradeRepo)
	repository.SetDBContext(ctx)
	return repository
}

// GetProduct 取單一商品
func (repo *TradeRepo) GetProduct(tx *gorm.DB, id uint) (*entities.Product, error) {
	var product entities.Product
	if err := tx.Where("id = ?", id).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// GetProductsByIDs
func (repo *TradeRepo) GetProductsByIDs(tx *gorm.DB, ids []uint) ([]entities.Product, error) {
	var products []entities.Product
	if err := tx.Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// UpdateProduct 用樂觀鎖更新商品庫存與版本
func (repo *TradeRepo) UpdateProduct(tx *gorm.DB, product *entities.Product) error {
	res := tx.Model(&entities.Product{}).
		Where("id = ? AND version = ?", product.ID, product.Version).
		Updates(map[string]interface{}{
			"stock":   product.Stock,
			"version": product.Version + 1,
		})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return errors.New("OPTIMISTIC_LOCK_CONFLICT_PRODUCT")
	}

	return nil
}

// CreateTransaction 寫入歷史流水帳紀錄
func (repo *TradeRepo) CreateTransaction(tx *gorm.DB, history entities.Transaction) error {
	return tx.Create(&history).Error
}
