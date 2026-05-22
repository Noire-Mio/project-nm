package schemas

import (
	"fmt"

	"project-nm/pkg/entities"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func CreateProductTable(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.Product{}, &entities.Transaction{}); err != nil {
		return fmt.Errorf("failed to migrate product tables: %w", err)
	}
	if err := seedDefaultProducts(db); err != nil {
		return fmt.Errorf("failed to seed default products: %w", err)
	}

	return nil
}

func seedDefaultProducts(db *gorm.DB) error {
	defaultProducts := []entities.Product{
		{ID: 1, Name: "蘋果", Stock: 100000, Price: decimal.NewFromFloat(50.0), Version: 0},
		{ID: 2, Name: "香蕉", Stock: 100000, Price: decimal.NewFromFloat(30.0), Version: 0},
		{ID: 3, Name: "西瓜", Stock: 100000, Price: decimal.NewFromFloat(150.0), Version: 0},
		{ID: 4, Name: "哈密瓜", Stock: 100000, Price: decimal.NewFromFloat(250.0), Version: 0},
	}

	for _, p := range defaultProducts {
		var existing entities.Product
		res := db.Where("id = ?", p.ID).First(&existing)

		if res.Error != nil {
			if err := db.Create(&p).Error; err != nil {
				return fmt.Errorf("failed to create product %s: %w", p.Name, err)
			}
		}
	}

	return nil
}
