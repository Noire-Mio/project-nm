package services

import (
	"errors"
	"project-nm/pkg/configs"
	"project-nm/pkg/contexts"
	"project-nm/pkg/services/dtos"
	"project-nm/pkg/utils"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

type IAuthService interface {
	Login(c *contexts.User, dto dtos.LoginDto) (*dtos.LoginResponseDto, error)
	RefreshToken(c *contexts.User, dto dtos.RefreshRequestDto) (*dtos.LoginResponseDto, error)
	Logout(c *contexts.User, dto dtos.LogoutDto) error
}

type AuthService struct{}

func (s *AuthService) Login(c *contexts.User, dto dtos.LoginDto) (*dtos.LoginResponseDto, error) {
	user, err := c.UserRepo.GetActiveUserByAccount(dto.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("帳號或密碼錯誤")
		}
		return nil, err
	}

	// 比對密碼雜湊
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dto.Password))
	if err != nil {
		return nil, errors.New("帳號或密碼錯誤")
	}

	// 透過 Repo 撈取該用戶的所有權限
	permissions, err := c.UserRepo.GetPermissionsByUserID(user.ID)
	if err != nil {
		return nil, err
	}

	permMap := make(map[string]bool)
	for _, p := range permissions {
		permMap[p.Permission] = true
	}

	// 過期時間：短效 30 分鐘，長效 7 天
	accessTokenExpireTime := time.Now().Add(30 * time.Minute)
	refreshTokenTTL := 7 * 24 * time.Hour

	// 組裝 UserInfo
	cfg := configs.GetConfig()
	userInfo := &contexts.UserInfo{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: accessTokenExpireTime.Unix(),
			Issuer:    "project-nm",
		},
		UserID:      user.ID,
		Identity:    user.Identity,
		Name:        user.Name,
		Schema:      user.Schema,
		Permissions: permMap,
		LoginAt:     time.Now().UnixNano(),
	}

	// 簽發短效 JWT (Access Token)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userInfo)
	accessTokenString, err := token.SignedString([]byte(cfg.JWTSign))
	if err != nil {
		return nil, err
	}

	clientRefreshToken := uuid.New().String()

	// 快取過期時間隨機跳動
	randomExpireTime := time.Duration(30+rand.Intn(5)) * time.Minute

	// 寫入短效快取
	_ = utils.SetUserToken(accessTokenString, userInfo, randomExpireTime)

	// 將時間戳記與長效 RefreshToken 綁定進 Redis（效期給足 7 天）
	_ = utils.BindUserLatestLoginTime(user.ID, userInfo.LoginAt, clientRefreshToken, refreshTokenTTL)

	return &dtos.LoginResponseDto{
		AccessToken:  accessTokenString,
		RefreshToken: clientRefreshToken,
	}, nil
}

func (s *AuthService) RefreshToken(c *contexts.User, dto dtos.RefreshRequestDto) (*dtos.LoginResponseDto, error) {
	// Redis 撈出該用戶全網目前唯一的最新登入時間與合法鑰匙
	latestLoginTime, errTime := utils.GetUserLatestLoginTime(dto.UserID)
	serverRefreshToken, errToken := utils.GetServerRefreshToken(dto.UserID)

	// 如果 Redis 紀錄不見了，或者前端傳來的 UUID 鑰匙跟 Redis 目前鎖定的鑰匙不符則踢回
	if errTime != nil || errToken != nil || serverRefreshToken != dto.RefreshToken {
		return nil, errors.New("憑證已失效，請重新登入")
	}

	// 去資料庫撈取該用戶當下的最新權限與狀態
	user, err := c.UserRepo.GetActiveUserByID(dto.UserID)
	if err != nil {
		return nil, errors.New("帳號已被停用或不存在")
	}

	permissions, err := c.UserRepo.GetPermissionsByUserID(dto.UserID)
	if err != nil {
		return nil, err
	}

	permMap := make(map[string]bool)
	for _, p := range permissions {
		permMap[p.Permission] = true
	}

	// 組裝全新的短效 UserInfo 快照
	cfg := configs.GetConfig()
	accessTokenExpireTime := time.Now().Add(30 * time.Minute)

	newUserInfo := &contexts.UserInfo{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: accessTokenExpireTime.Unix(),
			Issuer:    "project-nm",
		},
		UserID:      dto.UserID,
		Identity:    user.Identity,
		Name:        user.Name,
		Schema:      user.Schema,
		Permissions: permMap,
		LoginAt:     latestLoginTime,
	}

	// 簽發全新的短效 Access Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newUserInfo)
	newAccessTokenString, err := token.SignedString([]byte(cfg.JWTSign))
	if err != nil {
		return nil, err
	}

	// 寫入短效快取
	randomExpireTime := time.Duration(30+rand.Intn(5)) * time.Minute
	_ = utils.SetUserToken(newAccessTokenString, newUserInfo, randomExpireTime)

	return &dtos.LoginResponseDto{
		AccessToken:  newAccessTokenString,
		RefreshToken: dto.RefreshToken,
	}, nil
}

func (s *AuthService) Logout(c *contexts.User, dto dtos.LogoutDto) error {
	_ = utils.DeleteUserToken(dto.CurrentToken)

	_ = utils.DeleteUserLatestLoginTime(c.UserInfo.UserID)

	return nil
}
