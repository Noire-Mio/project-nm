package endpoints

import (
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints/converter"
	"project-nm/pkg/endpoints/inputmodels"
	"project-nm/pkg/grpc/pb"
	"project-nm/pkg/services"
	"project-nm/pkg/transports/cores"
)

type TradeEndpoint struct {
	Service    services.ITradeService
	CtxFactory *contexts.TradeFactory
	Converter  *converter.Converter
}
type ITradeEndpoint interface {
	ExecuteOrder(request *pb.TradeGrpcRequest) (*pb.TradeGrpcResponse, error)
	ProcessOrder(userInfo *contexts.UserInfo, input []inputmodels.TradeInput) *cores.Response
}

func (e *TradeEndpoint) ExecuteOrder(request *pb.TradeGrpcRequest) (*pb.TradeGrpcResponse, error) {
	userInfo, err := convertgRPCUserInfoToUserInfo(request.GrpcUserInfo)
	if err != nil {
		return nil, err
	}
	ctx := e.CtxFactory.NewContext(userInfo)
	defer ctx.Dispose() // 釋放context

	// dto, err := e.ConvertPbToDto(request)
	// if err != nil {
	// 	return nil, err
	// }

	return nil, nil
}

func (e *TradeEndpoint) ProcessOrder(userInfo *contexts.UserInfo, input []inputmodels.TradeInput) *cores.Response {
	ctx := e.CtxFactory.NewContext(*userInfo)
	defer ctx.Dispose() // 釋放context

	// dto, err := e.ConvertPbToDto(request)
	// if err != nil {
	// 	return nil, err
	// }

	return nil
}
