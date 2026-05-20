package contexts

import (
	"project-nm/pkg/repositories"

	"gorm.io/gorm"
)

// 營業日期
type Member struct {
	Context
	MemberRepo repositories.IMember
	UserInfo   UserInfo
}

type MemberFactory struct {
	DB                *gorm.DB
	MemberRepoFactory repositories.MemberFactory
}

func (f *MemberFactory) NewContext(UserInfo UserInfo) *Member {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(Member)
	ctx.MemberRepo = f.MemberRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo
	ctx.AddDBContexts(dbCtx)
	return ctx
}
