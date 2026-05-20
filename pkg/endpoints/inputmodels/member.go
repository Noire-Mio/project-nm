package inputmodels

type GetMember struct {
	ID uint `json:"id" form:"id" validate:"required"`
}
