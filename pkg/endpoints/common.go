package endpoints

import (
	"project-nm/pkg/endpoints/viewmodels"
	"project-nm/pkg/transports/cores"
)

// 共用錯誤回應
func NewErrorResponse(statusCode int, err error) *cores.Response {
	return cores.NewResponse(statusCode, viewmodels.ErrorResponse{
		Code:    statusCode,
		Message: err.Error(),
	})
}
