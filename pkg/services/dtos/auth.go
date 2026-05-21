package dtos

type LoginDto struct {
	Account  string
	Password string
}

type RefreshRequestDto struct {
	UserID       uint   `json:"user_id" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LoginResponseDto struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LogoutDto struct {
	CurrentToken string `json:"current_token" binding:"required"`
}
