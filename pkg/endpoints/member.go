package endpoints

import (
	"net/http"
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints/converter"
	"project-nm/pkg/services"
	"project-nm/pkg/transports/cores"
)

type MemberEndpoint struct {
	Service    services.IMemberService
	CtxFactory *contexts.MemberFactory
	Converter  *converter.Converter
}
type IMemberEndpoint interface {
	GetMember(userInfo *contexts.UserInfo) *cores.Response
}

// GetMember
// @Summary 取得單筆會員
// @Description 取得單筆會員
// @ID get-member
// @Tags Member
// @Param Authorization header string true "Bearer Token"
// @Success 204 "Success."
// @Failure 400 {object} viewmodels.Error "Bad Request."
// @Failure 401 {object} viewmodels.Error "Unauthorized."
// @Failure 403 {object} viewmodels.Error "Forbidden."
// @Failure 404 {object} viewmodels.Error "Not found."
// @Failure 500 {object} viewmodels.Error "Internal error."
// @Router  /member/:id [get]
func (e *MemberEndpoint) GetMember(userInfo *contexts.UserInfo) *cores.Response {
	ctx := e.CtxFactory.NewContext(*userInfo)
	defer ctx.Dispose() // 釋放context

	Member, err := e.Service.GetMember(ctx)
	if err != nil {
		return NewErrorResponse(http.StatusInternalServerError, err)
	}

	return cores.NewResponse(http.StatusOK, Member)
}
