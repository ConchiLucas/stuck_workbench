package http

import (
	"github.com/gin-gonic/gin"
	stdhttp "net/http"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data, "error": nil})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": APIError{Code: code, Message: message}})
}

func writeInternalError(c *gin.Context) {
	writeError(c, stdhttp.StatusInternalServerError, "internal_error", "服务暂时不可用")
}
