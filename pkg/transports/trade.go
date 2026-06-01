package transports

import (
	"project-nm/pkg/endpoints"
	"project-nm/pkg/endpoints/inputmodels"
	"project-nm/pkg/transports/cores"

	"github.com/gin-gonic/gin"
)

type TradeTransport struct {
	Endpoint endpoints.ITradeEndpoint
}

func (t *TradeTransport) ProcessOrder(permissions ...*cores.Permission) gin.HandlerFunc {
	handler := func(c *gin.Context) {

		userInfo, ok := HandleBearerTokenToUserInfo(c)
		if !ok {
			return
		}

		request, ok := HandleRequestBody(c, []inputmodels.TradeInput{})
		if !ok {
			return
		}

		response := t.Endpoint.ProcessOrder(userInfo, request.([]inputmodels.TradeInput))

		cores.GenerateGinResponse(c, response)
	}
	return handler
}
