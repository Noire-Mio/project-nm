package transports

import (
	"net/http"
	"project-nm/pkg/transports/cores"

	"github.com/gin-gonic/gin"
)

type Trans struct {
	HCTrans     *HealthCheckTransport
	AuthTrans   *AuthTransport
	MemberTrans *MemberTransport
}

func (t *Trans) MakeHttpHandler(e *gin.Engine) http.Handler {
	t.HealthCheckAPI(e)
	t.AuthAPI(e)
	t.MemberAPI(e)
	return e
}

func (t *Trans) HealthCheckAPI(e *gin.Engine) {
	e.GET("hc", t.HCTrans.HealthCheckHandler)
}

func (t *Trans) AuthAPI(e *gin.Engine) {
	e.POST("auth/login", t.AuthTrans.Login())
}
func (t *Trans) MemberAPI(e *gin.Engine) {
	e.GET("member", t.MemberTrans.GetMember(cores.NewActionPermission(cores.ActionRead)))
}
