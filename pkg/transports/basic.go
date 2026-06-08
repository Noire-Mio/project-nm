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
	TradeTrans  *TradeTransport
}

func (t *Trans) MakeHttpHandler(e *gin.Engine) http.Handler {
	t.HealthCheckAPI(e)
	t.AuthAPI(e)
	t.MemberAPI(e)
	t.TradeAPI(e)
	return e
}

func (t *Trans) HealthCheckAPI(e *gin.Engine) {
	e.GET("hc", t.HCTrans.HealthCheckHandler)
}

func (t *Trans) AuthAPI(e *gin.Engine) {
	e.POST("sessions", t.AuthTrans.Login())
	e.DELETE("sessions", t.AuthTrans.Logout())
	e.PUT("sessions/refresh", t.AuthTrans.RefreshToken())

}
func (t *Trans) MemberAPI(e *gin.Engine) {
	e.GET("member", t.MemberTrans.GetMember(cores.NewActionPermission(cores.ActionRead)))
	e.GET("member-mq", t.MemberTrans.GetMemberMQ(cores.NewActionPermission(cores.ActionRead)))
}

func (t *Trans) TradeAPI(e *gin.Engine) {
	e.POST("trade", t.TradeTrans.ProcessOrder())
}
