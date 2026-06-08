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
	GetForUpdate(schema string, id uint) (bool, *entities.Member, error)
	Get(schema string, id uint) (bool, *entities.Member, error)
	Create(schema string, entity entities.Member) error
	UpdateMember(schema string, member *entities.Member) error
}

type MemberRepo struct {
	GormRepository
}

func NewMemberRepo(ctx *GormDBContext) IMember {
	repository := new(MemberRepo)
	repository.SetDBContext(ctx)
	return repository
}

func (repo *MemberRepo) GetForUpdate(schema string, id uint) (bool, *entities.Member, error) {
	var entity entities.Member
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())
	err := repo.DB().Table(tableName).
		Clauses(clause.Locking{Strength: "UPDATE"}).
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

// Get 獲取單一會員
func (repo *MemberRepo) Get(schema string, id uint) (bool, *entities.Member, error) {
	var entity entities.Member
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())

	err := repo.DB().
		Table(tableName).
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


// Create 建立會員
func (repo *MemberRepo) Create(schema string, member entities.Member) error {
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())

	return repo.DB().
		Table(tableName).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&member).Error
}

// UpdateMember 
func (repo *MemberRepo) UpdateMember(schema string, member *entities.Member) error {
	tableName := fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())

	res := repo.DB().Table(tableName).
		Where("id = ?", member.ID).
		Updates(map[string]interface{}{
			"balance": member.Balance,
			"version": member.Version + 1,
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}
