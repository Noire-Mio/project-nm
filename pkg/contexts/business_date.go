package contexts

import (
	"project-nm/pkg/repositories"

	"gorm.io/gorm"
)

// 營業日期
type BusinessDate struct {
	Context
	BusinessDateRepo repositories.IBusinessDate
	UserInfo         UserInfo
}

type BusinessDateFactory struct {
	DB                      *gorm.DB
	BusinessDateRepoFactory repositories.BusinessDateFactory
}

func (f *BusinessDateFactory) NewContext(UserInfo UserInfo) *BusinessDate {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(BusinessDate)
	ctx.BusinessDateRepo = f.BusinessDateRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo
	ctx.AddDBContexts(dbCtx)
	return ctx
}
