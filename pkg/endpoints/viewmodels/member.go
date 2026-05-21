package viewmodels

import (
	"github.com/shopspring/decimal"
)

type MemberView struct {
	ID      uint            `json:"id"`
	Userame string          `json:"user_name"`
	Balance decimal.Decimal `json:"balance"`
}
