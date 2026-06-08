package repositories

import (
	"fmt"
	"project-nm/pkg/entities"

	"gorm.io/gorm/clause"
)

type TradeFactory func(ctx *GormDBContext) ITrade

type ITrade interface {
	GetProduct(schema string, id uint) (*entities.Product, error)
	GetProductsByIDs(schema string, ids []uint) ([]entities.Product, error)
	UpdateProduct(schema string, product *entities.Product) error
	CreateTransaction(schema string, history entities.Transaction) error
	GetProductsByIDsForUpdate(schema string, ids []uint) ([]entities.Product, error)
}

type TradeRepo struct {
	GormRepository
}

func NewTradeRepo(ctx *GormDBContext) ITrade {
	repository := new(TradeRepo)
	repository.SetDBContext(ctx)
	return repository
}

func (repo *TradeRepo) GetProductsByIDsForUpdate(schema string, ids []uint) ([]entities.Product, error) {
	var products []entities.Product
	if len(ids) == 0 {
		return products, nil
	}
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Product).TableName())
	err := repo.DB().Table(tableName).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).
		Find(&products).Error

	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetProduct 取單一商品
func (repo *TradeRepo) GetProduct(schema string, id uint) (*entities.Product, error) {
	var product entities.Product
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Product).TableName())
	if err := repo.DB().Table(tableName).Where("id = ?", id).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// GetProductsByIDs 批量獲取商品
func (repo *TradeRepo) GetProductsByIDs(schema string, ids []uint) ([]entities.Product, error) {
	var products []entities.Product
	if len(ids) == 0 {
		return products, nil
	}
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Product).TableName())
	if err := repo.DB().Table(tableName).Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// UpdateProduct 
func (repo *TradeRepo) UpdateProduct(schema string, product *entities.Product) error {
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Product).TableName())
	res := repo.DB().Table(tableName).
		Where("id = ?", product.ID).
		Updates(map[string]interface{}{
			"stock":   product.Stock,
			"version": product.Version + 1, 
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

// CreateTransaction 寫入歷史流水帳紀錄
func (repo *TradeRepo) CreateTransaction(schema string, history entities.Transaction) error {
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Transaction).TableName())
	return repo.DB().Table(tableName).Create(&history).Error
}
