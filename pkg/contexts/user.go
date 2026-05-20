package contexts

import (
	"project-nm/pkg/repositories"

	"gorm.io/gorm"
)

type User struct {
	Context
	UserRepo repositories.IUser
	UserInfo UserInfo
}

type UserFactory struct {
	DB              *gorm.DB
	UserRepoFactory repositories.UserFactory
}

func (f *UserFactory) NewContext(UserInfo UserInfo) *User {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(User)
	ctx.UserRepo = f.UserRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo
	ctx.AddDBContexts(dbCtx)
	return ctx
}

func (f *UserFactory) NewAnonymousContext() *User {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(User)
	ctx.UserRepo = f.UserRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo{}
	ctx.AddDBContexts(dbCtx)
	return ctx
}
