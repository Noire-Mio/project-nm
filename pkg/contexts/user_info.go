package contexts

import "github.com/golang-jwt/jwt"

type UserInfo struct {
	jwt.StandardClaims
	UserID      uint            `json:"user_id"`     // 使用者 ID
	Identity    string          `json:"identity"`    // 身分別 (例如: admin, staff)
	Name        string          `json:"name"`        // 使用者姓名
	Schema      string          `json:"schema"`      // 租戶資料庫 Schema 名稱
	Permissions map[string]bool `json:"permissions"` // 權限清單
}
