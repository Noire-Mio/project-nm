package inputmodels

type LoginInput struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshInput struct {
	UserID       uint   `json:"user_id" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}
