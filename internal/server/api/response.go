package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse 统一的成功响应结构
type SuccessResponse struct {
	Status    string      `json:"status"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id,omitempty"`
}

// RespondWithValidationError 返回验证错误响应
func RespondWithValidationError(c *gin.Context, message string, details interface{}) {
	// 从上下文获取请求ID
	requestID, _ := c.Get("request_id")
	requestIDStr, _ := requestID.(string)

	c.JSON(http.StatusBadRequest, gin.H{
		"error":      "参数验证失败",
		"code":       http.StatusBadRequest,
		"message":    message,
		"details":    details,
		"request_id": requestIDStr,
	})
}

// RespondWithNotFound 返回资源不存在响应
func RespondWithNotFound(c *gin.Context, resourceType, resourceID string) {
	// 从上下文获取请求ID
	requestID, _ := c.Get("request_id")
	requestIDStr, _ := requestID.(string)

	message := resourceType
	if resourceID != "" {
		message += " (ID: " + resourceID + ")"
	}
	message += " 不存在"

	c.JSON(http.StatusNotFound, gin.H{
		"error":      "资源不存在",
		"code":       http.StatusNotFound,
		"message":    message,
		"request_id": requestIDStr,
	})
}
