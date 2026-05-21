package endpoints

import (
	"fmt"
	"net/http"
	"project-nm/pkg/contexts"
	"project-nm/pkg/endpoints/converter"
	"project-nm/pkg/endpoints/inputmodels"
	"project-nm/pkg/endpoints/viewmodels"
	"project-nm/pkg/services"
	"project-nm/pkg/services/dtos"
	"project-nm/pkg/transports/cores"
)

type AuthEndpoint struct {
	Service    services.IAuthService
	CtxFactory *contexts.UserFactory
	Converter  *converter.Converter
}
type IAuthEndpoint interface {
	Login(input inputmodels.LoginInput) *cores.Response
	RefreshToken(input inputmodels.RefreshInput) *cores.Response
	Logout(userInfo *contexts.UserInfo, input inputmodels.LogoutInput) *cores.Response
}

// Login
// @Summary 登入
// @Description 登入
// @ID login
// @Tags User
// @Param Authorization header string true "Bearer Token"
// @Success 204 "Success."
// @Router  /login [post]
func (e *AuthEndpoint) Login(input inputmodels.LoginInput) *cores.Response {
	ctx := e.CtxFactory.NewAnonymousContext()
	defer ctx.Dispose() // 釋放context

	if input.Account == "" || input.Password == "" {
		return NewErrorResponse(http.StatusBadRequest, fmt.Errorf("account and password are required"))
	}
	auth, err := e.Service.Login(ctx, dtos.LoginDto(input))
	if err != nil {
		return NewErrorResponse(http.StatusInternalServerError, err)
	}

	respBody := viewmodels.LoginView{
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
	}

	return cores.NewResponse(http.StatusOK, respBody)
}

func (e *AuthEndpoint) RefreshToken(input inputmodels.RefreshInput) *cores.Response {
	ctx := e.CtxFactory.NewAnonymousContext()
	defer ctx.Dispose() // 釋放context

	auth, err := e.Service.RefreshToken(ctx, dtos.RefreshRequestDto(input))
	if err != nil {
		return NewErrorResponse(http.StatusInternalServerError, err)
	}

	respBody := viewmodels.LoginView{
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
	}

	return cores.NewResponse(http.StatusOK, respBody)
}

func (e *AuthEndpoint) Logout(userInfo *contexts.UserInfo, input inputmodels.LogoutInput) *cores.Response {
	ctx := e.CtxFactory.NewContext(*userInfo)
	defer ctx.Dispose() 

	err := e.Service.Logout(ctx, dtos.LogoutDto(input))
	if err != nil {
		return NewErrorResponse(http.StatusInternalServerError, err)
	}

	return cores.NewResponse(http.StatusOK, nil)
}
