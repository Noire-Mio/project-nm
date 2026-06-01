package endpoints

import (
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints/viewmodels"
	"project-nm/pkg/grpc/pb"
	"project-nm/pkg/transports/cores"
)

// 共用錯誤回應
func NewErrorResponse(statusCode int, err error) *cores.Response {
	return cores.NewResponse(statusCode, viewmodels.ErrorResponse{
		Code:    statusCode,
		Message: err.Error(),
	})
}

func convertgRPCUserInfoToUserInfo(userInfo *pb.GRPCUserInfo) (contexts.UserInfo, error) {
	if userInfo == nil {
		return contexts.UserInfo{}, nil
	}
	return contexts.UserInfo{
		UserID:   uint(userInfo.UserId),
		Identity: userInfo.Identity,
		Name:     userInfo.Name,
		Schema:   userInfo.Schema,
	}, nil
}
