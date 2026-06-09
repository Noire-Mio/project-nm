package endpoints

import (
	"fmt"
	"net/http"
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints/converter"
	"project-nm/pkg/endpoints/inputmodels"
	"project-nm/pkg/grpc/pb"
	"project-nm/pkg/services"
	"project-nm/pkg/services/dtos"
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
	defer ctx.Dispose()

	if len(request.Items) == 0 {
		return nil, fmt.Errorf("INVALID_GRPC_REQUEST: 結帳商品清單不能為空")
	}

	tradeDTOs := make([]dtos.TradeDto, 0, len(request.Items))
	for _, item := range request.Items {
		tradeDTOs = append(tradeDTOs, dtos.TradeDto{
			ProductID: uint(item.ProductId),
			Quantity:  item.Quantity,
			Type:      item.Type,
		})
	}

	tx, err := e.Service.ExecuteOrderV2(ctx, tradeDTOs)
	if err != nil {
		return nil, fmt.Errorf("CORE_TRANSACTION_ERROR: 核心扣款程序失敗: %w", err)
	}

	return &pb.TradeGrpcResponse{
		MemberId: uint32(tx.MemberID),
		Status:   tx.Status,
	}, nil
}

func (e *TradeEndpoint) ProcessOrder(userInfo *contexts.UserInfo, input []inputmodels.TradeInput) *cores.Response {
	ctx := e.CtxFactory.NewContext(*userInfo)
	defer ctx.Dispose() // 釋放context

	tradeDTOs := make([]dtos.TradeDto, 0, len(input))
	for _, in := range input {
		if in.Quantity <= 0 {
			return NewErrorResponse(http.StatusInternalServerError, fmt.Errorf("INVALID_QUANTITY: 商品 ID %d 的數量必須大於 0", in.ProductID))
		}

		tradeDTOs = append(tradeDTOs, dtos.TradeDto{
			ProductID: in.ProductID,
			Quantity:  in.Quantity,
			Type:      in.Type,
		})
	}

	response, err := e.Service.ProcessOrder(ctx, tradeDTOs)
	if err != nil {
		return NewErrorResponse(http.StatusInternalServerError, err)
	}

	return cores.NewResponse(http.StatusOK, response)
}
