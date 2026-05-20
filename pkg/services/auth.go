package services

import (
	"errors"
	"project-nm/pkg/configs"
	"project-nm/pkg/contexts"
	"project-nm/pkg/services/dtos"
	"project-nm/pkg/utils"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

type IAuthService interface {
	Login(c *contexts.User, dto dtos.LoginDto) (string, error)
}

type AuthService struct{}

func (s *AuthService) Login(c *contexts.User, dto dtos.LoginDto) (string, error) {
	user, err := c.UserRepo.GetActiveUserByAccount(dto.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("帳號或密碼錯誤")
		}
		return "", err
	}

	// 比對密碼雜湊
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dto.Password))
	if err != nil {
		return "", errors.New("帳號或密碼錯誤")
	}

	// 透過 Repo 撈取該用戶的所有權限
	permissions, err := c.UserRepo.GetPermissionsByUserID(user.ID)
	if err != nil {
		return "", err
	}

	permMap := make(map[string]bool)
	for _, p := range permissions {
		permMap[p.Permission] = true
	}

	// 組裝 UserInfo
	cfg := configs.GetConfig()
	expireTime := time.Now().Add(2 * time.Hour)

	userInfo := &contexts.UserInfo{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "project-nm",
		},
		UserID:      user.ID,
		Identity:    user.Identity,
		Name:        user.Name,
		Schema:      user.Schema,
		Permissions: permMap,
	}

	// 簽發 JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userInfo)
	tokenString, err := token.SignedString([]byte(cfg.JWTSign))
	if err != nil {
		return "", err
	}

	// 寫入 Redis 快取 (維持高併發快速驗證)
	randomExpireTime := time.Duration(120+rand.Intn(5)) * time.Minute
	_ = utils.SetUserToken(tokenString, userInfo, randomExpireTime)

	return tokenString, nil
}
