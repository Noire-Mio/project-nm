package repositories

import (
	"errors"
	"fmt"
	"project-nm/pkg/entities"

	"gorm.io/gorm"
)

// 內部宣告介面
type BusinessDateFactory func(ctx *GormDBContext) IBusinessDate

type IBusinessDate interface {
	GetBusinessDate(schema string) (entity *entities.BusinessDate, err error)
}

type BusinessDateRepo struct {
	GormRepository
}

// 外部呼叫
func NewBusinessDateRepo(ctx *GormDBContext) IBusinessDate {
	repository := new(BusinessDateRepo)
	repository.SetDBContext(ctx)
	return repository
}

func (repo *BusinessDateRepo) GetBusinessDate(schemaMdl string) (*entities.BusinessDate, error) {
	var entity entities.BusinessDate
	err := repo.DB().
		Table(fmt.Sprintf("s%s.%s", schemaMdl, new(entities.BusinessDate).TableName())).
		Where("start_date IS NOT NULL AND end_date IS NULL").
		First(&entity).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entity, err
}
