package transports

import (
	"github.com/gin-gonic/gin"
)

type HealthCheckTransport struct {
}

func (t *HealthCheckTransport) HealthCheckHandler(c *gin.Context) {
	c.String(200, "Health Check Complete!")
}
