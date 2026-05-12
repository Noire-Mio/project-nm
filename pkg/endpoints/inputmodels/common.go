package inputmodels

type ListParam struct {
	Page    int      `json:"page" form:"page,default=1"`
	PerPage int      `json:"per_page" form:"per_page,default=20"`
	Order   []string `json:"order" form:"order" validate:"dive,no_special_chars"`
	Desc    []bool   `json:"desc" form:"desc"`
	Reverse bool     `json:"reverse" form:"reverse"`
}

type DataParam struct {
	Order []string `json:"order" form:"order" validate:"dive,no_special_chars"`
	Desc  []bool   `json:"desc" form:"desc"`
}
