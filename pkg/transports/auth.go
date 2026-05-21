package transports

import (
	"project-nm/pkg/endpoints"
	"project-nm/pkg/endpoints/inputmodels"
	"project-nm/pkg/transports/cores"

	"github.com/gin-gonic/gin"
)

type AuthTransport struct {
	Endpoint endpoints.IAuthEndpoint
}

func (t *AuthTransport) Login(permissions ...*cores.Permission) gin.HandlerFunc {
	handler := func(c *gin.Context) {

		request, ok := HandleRequestBody(c, inputmodels.LoginInput{})
		if !ok {
			return
		}

		response := t.Endpoint.Login(request.(inputmodels.LoginInput))

		cores.GenerateGinResponse(c, response)
	}
	return handler
}

func (t *AuthTransport) RefreshToken(permissions ...*cores.Permission) gin.HandlerFunc {
	handler := func(c *gin.Context) {



		request, ok := HandleRequestBody(c, inputmodels.RefreshInput{})
		if !ok {
			return
		}

		response := t.Endpoint.RefreshToken( request.(inputmodels.RefreshInput))

		cores.GenerateGinResponse(c, response)
	}
	return handler
}
