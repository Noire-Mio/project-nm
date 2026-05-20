package transports

import (
	"project-nm/pkg/endpoints"
	"project-nm/pkg/transports/cores"

	"github.com/gin-gonic/gin"
)

type MemberTransport struct {
	Endpoint endpoints.IMemberEndpoint
}

func (t *MemberTransport) GetMember(permissions ...*cores.Permission) gin.HandlerFunc {
	handler := func(c *gin.Context) {
		userInfo, ok := HandleBearerTokenToUserInfo(c)
		if !ok {
			return
		}

		if !CheckPermissions(c, permissions) {
			return
		}

		response := t.Endpoint.GetMember(userInfo)
		cores.GenerateGinResponse(c, response)
	}
	return handler
}
