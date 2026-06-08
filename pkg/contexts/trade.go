package contexts

import (
	"project-nm/pkg/repositories"

	"gorm.io/gorm"
)

type Trade struct {
	Context
	TradeRepo  repositories.ITrade
	MemberRepo repositories.IMember
	UserInfo   UserInfo
}

type TradeFactory struct {
	DB                *gorm.DB
	TradeRepoFactory  repositories.TradeFactory
	MemberRepoFactory repositories.MemberFactory
}

func (f *TradeFactory) NewContext(UserInfo UserInfo) *Trade {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(Trade)
	ctx.TradeRepo = f.TradeRepoFactory(dbCtx)
	ctx.MemberRepo = f.MemberRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo
	ctx.AddDBContexts(dbCtx)
	return ctx
}
