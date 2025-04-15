package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logging 日志中间件
// 记录HTTP请求的开始、结束和处理时间，并将日志记录器添加到上下文
func Logging(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 将日志记录器添加到上下文
		c.Set("logger", logger)

		// 记录请求开始
		logger.Info("收到HTTP请求",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()))

		// 处理请求
		c.Next()

		// 计算处理时间
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// 判断请求是否成功
		statusText := "成功"
		if statusCode >= 400 {
			statusText = "失败"
		}

		// 记录请求完成
		logger.Info("HTTP请求完成",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", duration),
			zap.String("result", statusText))

		// 记录慢请求
		if duration > 500*time.Millisecond {
			logger.Warn("慢请求",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Duration("latency", duration))
		}
	}
}
