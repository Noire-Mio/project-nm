package transports

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Trans struct {
	HCTrans *HealthCheckTransport
}

func (t *Trans) MakeHttpHandler(e *gin.Engine) http.Handler {
	t.HealthCheckAPI(e)
	return e
}

func (t *Trans) HealthCheckAPI(e *gin.Engine) {
	e.GET("hc", t.HCTrans.HealthCheckHandler)
}
