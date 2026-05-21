package repositories

import (
	"project-nm/pkg/entities"
)

type UserFactory func(ctx *GormDBContext) IUser

type IUser interface {
	GetActiveUserByAccount(account string) (*entities.User, error)
	GetPermissionsByUserID(userID uint) ([]entities.UserPermission, error)
	GetActiveUserByID(id uint) (*entities.User, error)
}

type UserRepository struct {
	GormRepository
}

// 外部呼叫
func NewUserRepo(ctx *GormDBContext) IUser {
	repository := new(UserRepository)
	repository.SetDBContext(ctx)
	return repository
}

// GetActiveUserByAccount 根據帳號找尋啟用的使用者
func (r *UserRepository) GetActiveUserByAccount(account string) (*entities.User, error) {
	var user entities.User
	err := r.DB().
		Table("public.users").
		Where("account = ? AND is_active = ?", account, true).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetPermissionsByUserID 獲取使用者的所有權限節點
func (r *UserRepository) GetPermissionsByUserID(userID uint) ([]entities.UserPermission, error) {
	var permissions []entities.UserPermission
	err := r.DB().
		Table("public.user_permissions").
		Where("user_id = ?", userID).
		Find(&permissions).Error
	return permissions, err
}

// GetActiveUserByID 確認帳號狀態
func (r *UserRepository) GetActiveUserByID(id uint) (*entities.User, error) {
	var user entities.User
	err := r.DB().
		Table("public.users").
		Where("id = ? AND is_active = ?", id, true).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
