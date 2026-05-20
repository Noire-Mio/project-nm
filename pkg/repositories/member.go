package repositories

import (
	"errors"
	"fmt"
	"project-nm/pkg/entities"

	"gorm.io/gorm"
)

// 內部宣告介面
type MemberFactory func(ctx *GormDBContext) IMember

type IMember interface {
	Get(schema string, id uint) (bool, *entities.Member, error)
	Create(schema string, entity entities.Member) (*entities.Member, error)
}

type MemberRepo struct {
	GormRepository
}

// 外部呼叫
func NewMemberRepo(ctx *GormDBContext) IMember {
	repository := new(MemberRepo)
	repository.SetDBContext(ctx)
	return repository
}

func (repo *MemberRepo) Get(schema string, id uint) (bool, *entities.Member, error) {
	var entity entities.Member
	err := repo.DB().
		Table(fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())).
		Where("id = ?", id).
		First(&entity).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	// 找到
	return true, &entity, nil
}

func (repo *MemberRepo) Create(schema string, entity entities.Member) (*entities.Member, error) {
	err := repo.DB().
		Table(fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())).
		Create(&entity).Error

	return &entity, err
}
