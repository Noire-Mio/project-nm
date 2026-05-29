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
	GetWithTx(tx *gorm.DB, schema string, id uint) (*entities.Member, error)
	Create(schema string, entity entities.Member) error
	UpdateMember(tx *gorm.DB, schema string, member *entities.Member) error
}

type MemberRepo struct {
	GormRepository
}

func NewMemberRepo(ctx *GormDBContext) IMember {
	repository := new(MemberRepo)
	repository.SetDBContext(ctx)
	return repository
}

// Get
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

// GetWithTx 【新增】專門提供給 Worker 事務內部使用的讀取方法，確保樂觀鎖重試能拿到最新版本號
func (repo *MemberRepo) GetWithTx(tx *gorm.DB, schema string, id uint) (*entities.Member, error) {
	var entity entities.Member
	err := tx.Table(fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())).
		Where("id = ?", id).
		First(&entity).Error

	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (repo *MemberRepo) Create(schema string, member entities.Member) error {
	return repo.DB().
		Table(fmt.Sprintf("%s.%s", schema, new(entities.Member).TableName())).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&member).Error
}

// UpdateMember 樂觀鎖更新動態 Schema 下的會員餘額與版本
func (repo *MemberRepo) UpdateMember(tx *gorm.DB, schema string, member *entities.Member) error {
	res := tx.Table(fmt.Sprintf("%s.members", schema)).
		Where("id = ? AND version = ?", member.ID, member.Version).
		Updates(map[string]interface{}{
			"balance": member.Balance,
			"version": member.Version + 1,
		})

	if res.Error != nil {
		return res.Error
	}

	// 影響行數為 0 ， 發生樂觀鎖競爭衝突
	if res.RowsAffected == 0 {
		return errors.New("OPTIMISTIC_LOCK_CONFLICT_MEMBER")
	}

	return nil
}
