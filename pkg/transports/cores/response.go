package cores

import (
	"github.com/gin-gonic/gin"
)

// 響應
type Response struct {
	StatusCode int
	Body       interface{}
}

// 新的錯誤響應
func NewErrorResponse(statusCode int, message string, errors interface{}) *Response {
	body := map[string]interface{}{
		"code":    statusCode,
		"message": message,
		"errors":  errors,
	}
	return &Response{StatusCode: statusCode, Body: body}
}

// 新的響應
func NewResponse(statusCode int, body interface{}) *Response {
	return &Response{
		StatusCode: statusCode,
		Body:       body,
	}
}

// 生成 GinResponse
func GenerateGinResponse(c *gin.Context, resp *Response) {
	switch resp.Body.(type) {
	case string:
		c.String(resp.StatusCode, resp.Body.(string))
	case nil:
		c.Status(resp.StatusCode)
	default:
		c.JSON(resp.StatusCode, resp.Body)
	}
}
