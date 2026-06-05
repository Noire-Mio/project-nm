package grpc

import (
	"context"

	"project-nm/pkg/endpoints"
	"project-nm/pkg/grpc/pb"
)

type ProjectNMServer struct {
	TradeEndpoint endpoints.ITradeEndpoint
	pb.UnimplementedProjectGrpcServer
}

func (s ProjectNMServer) ExecuteOrder(ctx context.Context, request *pb.TradeGrpcRequest) (*pb.TradeGrpcResponse, error) {
	return s.TradeEndpoint.ExecuteOrder(request)
}
