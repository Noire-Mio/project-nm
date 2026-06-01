package contexts

import (
	"project-nm/pkg/repositories"

	"gorm.io/gorm"
)

type Trade struct {
	Context
	TradeRepo repositories.ITrade
	UserInfo  UserInfo
}

type TradeFactory struct {
	DB               *gorm.DB
	TradeRepoFactory repositories.TradeFactory
}

func (f *TradeFactory) NewContext(UserInfo UserInfo) *Trade {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(Trade)
	ctx.TradeRepo = f.TradeRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo
	ctx.AddDBContexts(dbCtx)
	return ctx
}
