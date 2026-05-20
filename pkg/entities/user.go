package entities

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Account  string `gorm:"type:varchar(50);not null;uniqueIndex"`      // 登入帳號
	Password string `gorm:"type:varchar(255);not null"`                 // 密碼雜湊
	Name     string `gorm:"type:varchar(50);not null"`                  // 姓名
	Identity string `gorm:"type:varchar(20);not null;default:'staff'"`  // 身分別 (admin/staff)
	Schema   string `gorm:"type:varchar(50);not null;default:'public'"` // 該用戶所屬的租戶 Schema
	IsActive bool   `gorm:"type:boolean;not null;default:true"`         // 帳號是否啟用
}

func (m *User) TableName() string {
	return "users"
}

// UserPermission 存放使用者的權限 (例如：member:read, member:write)
type UserPermission struct {
	gorm.Model
	UserID     uint   `gorm:"not null;index"`
	Permission string `gorm:"type:varchar(50);not null"` // 權限名稱
}

func (m *UserPermission) TableName() string {
	return "user_permissions"
}
