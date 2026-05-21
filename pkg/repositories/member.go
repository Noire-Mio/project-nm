package repositories

import (
	"errors"
	"fmt"
	"project-nm/pkg/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MemberFactory func(ctx *GormDBContext) IMember

type IMember interface {
	Get(schema string, id uint) (bool, *entities.Member, error)
	Create(schema string, entity entities.Member) (*entities.Member, error)
}

type MemberRepo struct {
	GormRepository
}

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
	return true, &entity, nil
}

func (repo *MemberRepo) Create(schema string, member entities.Member) (*entities.Member, error) {
	err := repo.DB().
		Table(fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&member).Error

	if err != nil {
		return nil, err
	}
	err = repo.DB().
		Table(fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())).
		First(&member, member.ID).Error
	if err != nil {
		return nil, err
	}

	return &member, nil
}
