package contexts

import (
	grpc_client "project-nm/pkg/grpc/client"
	"project-nm/pkg/repositories"

	"gorm.io/gorm"
)

type Trade struct {
	Context
	ProjectNMGrpcClient grpc_client.IProjectNMClient
	TradeRepo           repositories.ITrade
	UserInfo            UserInfo
}

type TradeFactory struct {
	DB                  *gorm.DB
	ProjectNMGrpcClient *grpc_client.ProjectNMClient
	TradeRepoFactory    repositories.TradeFactory
}

func (f *TradeFactory) NewContext(UserInfo UserInfo) *Trade {
	dbCtx := repositories.NewGormDBContext(f.DB)
	ctx := new(Trade)
	ctx.ProjectNMGrpcClient = f.ProjectNMGrpcClient
	ctx.TradeRepo = f.TradeRepoFactory(dbCtx)
	ctx.UserInfo = UserInfo
	ctx.AddDBContexts(dbCtx)
	return ctx
}
