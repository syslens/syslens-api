package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID 请求ID中间件
// 为每个请求生成唯一ID并添加到上下文和响应头
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取请求ID
		requestID := c.GetHeader("X-Request-ID")

		// 如果请求头中没有请求ID，则生成一个
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 添加到上下文
		c.Set("request_id", requestID)

		// 添加到响应头
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求ID
// 使用时间戳和随机字符串生成唯一ID
func generateRequestID() string {
	// 简单实现，可以使用更复杂的算法如UUID
	return time.Now().Format("20060102150405") + "-" +
		time.Now().Format("000000000")
}
