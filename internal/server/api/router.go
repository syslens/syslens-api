package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRouter 配置API路由
func SetupRouter(handler *MetricsHandler, logger *zap.Logger) *gin.Engine {
	// 设置为发布模式，减少不必要的日志
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(LoggingMiddleware(logger))

	// 健康检查路由
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API路由组
	api := router.Group("/api/v1")
	{
		// 节点相关路由
		nodes := api.Group("/nodes")
		{
			// 获取所有节点
			nodes.GET("", handler.HandleGetAllNodesGin)

			// 获取节点指标
			nodes.GET("/metrics", handler.HandleGetNodeMetricsGin)

			// 节点注册
			nodes.POST("/register", handler.HandleNodeRegisterGin)

			// 特定节点的操作
			nodeGroup := nodes.Group("/:node_id")
			{
				// 上报指标
				nodeGroup.POST("/metrics", handler.HandleMetricsSubmitGin)
			}
		}

		// 分组相关路由
		groups := api.Group("/groups")
		{
			// 获取所有分组
			groups.GET("", handler.HandleGetGroupsGin)

			// 创建分组
			groups.POST("", handler.HandleCreateGroupGin)

			// 特定分组的操作
			groupID := groups.Group("/:group_id")
			{
				// 获取特定分组
				groupID.GET("", handler.HandleGetGroupGin)

				// 更新分组
				groupID.PUT("", handler.HandleUpdateGroupGin)

				// 删除分组
				groupID.DELETE("", handler.HandleDeleteGroupGin)

				// 获取分组内的节点
				groupID.GET("/nodes", handler.HandleGetGroupNodesGin)
			}
		}

		// 服务相关路由
		services := api.Group("/services")
		{
			// 获取所有服务
			services.GET("", handler.HandleGetServicesGin)

			// 创建服务
			services.POST("", handler.HandleCreateServiceGin)

			// 特定服务的操作
			serviceID := services.Group("/:service_id")
			{
				// 获取特定服务
				serviceID.GET("", handler.HandleGetServiceGin)

				// 更新服务
				serviceID.PUT("", handler.HandleUpdateServiceGin)

				// 删除服务
				serviceID.DELETE("", handler.HandleDeleteServiceGin)

				// 获取服务关联的节点
				serviceID.GET("/nodes", handler.HandleGetServiceNodesGin)
			}
		}

		// 告警规则相关路由
		alerts := api.Group("/alerts")
		{
			// 获取所有告警规则
			alerts.GET("", handler.HandleGetAlertsGin)

			// 创建告警规则
			alerts.POST("", handler.HandleCreateAlertGin)

			// 特定告警规则的操作
			alertID := alerts.Group("/:alert_id")
			{
				// 获取特定告警规则
				alertID.GET("", handler.HandleGetAlertGin)

				// 更新告警规则
				alertID.PUT("", handler.HandleUpdateAlertGin)

				// 删除告警规则
				alertID.DELETE("", handler.HandleDeleteAlertGin)

				// 启用/禁用告警规则
				alertID.PATCH("/status", handler.HandleUpdateAlertStatusGin)
			}
		}

		// 通知相关路由
		notifications := api.Group("/notifications")
		{
			// 获取所有通知
			notifications.GET("", handler.HandleGetNotificationsGin)

			// 特定通知的操作
			notificationID := notifications.Group("/:notification_id")
			{
				// 获取特定通知
				notificationID.GET("", handler.HandleGetNotificationGin)

				// 更新通知状态
				notificationID.PATCH("/status", handler.HandleUpdateNotificationStatusGin)

				// 解决通知
				notificationID.POST("/resolve", handler.HandleResolveNotificationGin)

				// 删除通知
				notificationID.DELETE("", handler.HandleDeleteNotificationGin)
			}
		}
	}

	// WebSocket 连接
	router.GET("/api/v1/ws/nodes", handler.HandleWebSocketGin)

	return router
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 记录请求开始
		logger.Debug("收到HTTP请求",
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
		logger.Debug("HTTP请求完成",
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

// ErrorResponse 统一的错误响应结构
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// RespondWithError 返回统一格式的错误响应
func RespondWithError(c *gin.Context, statusCode int, err error, message string) {
	errMsg := "未知错误"
	if err != nil {
		errMsg = err.Error()
	}

	c.JSON(statusCode, ErrorResponse{
		Error:   errMsg,
		Code:    statusCode,
		Message: message,
	})
}

// RespondWithSuccess 返回统一格式的成功响应
func RespondWithSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"status": "success",
		"data":   data,
	})
}
