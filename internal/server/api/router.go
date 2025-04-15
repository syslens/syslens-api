package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/syslens/syslens-api/internal/server/middleware"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 配置API路由
func SetupRouter(handler *MetricsHandler, logger *zap.Logger) *gin.Engine {
	// 设置为发布模式，减少不必要的日志
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.Logging(logger))
	router.Use(middleware.RequestID())

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
		// 按功能模块组织路由
		setupNodeRoutes(api, handler)
		setupGroupRoutes(api, handler)
		setupServiceRoutes(api, handler)
		setupAlertRoutes(api, handler)
		setupNotificationRoutes(api, handler)
	}

	// 添加Swagger路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("http://localhost:8080/swagger/doc.json")))

	return router
}

// 节点相关路由
func setupNodeRoutes(rg *gin.RouterGroup, handler *MetricsHandler) {
	nodes := rg.Group("/nodes")
	{
		// 获取所有节点
		nodes.GET("", handler.HandleGetAllNodesGin)

		// 获取节点指标
		nodes.GET("/metrics", handler.HandleGetNodeMetricsGin)

		// 节点注册
		nodes.POST("/register", handler.HandleRegisterNodeGin)

		// 更新节点状态
		nodes.PUT("/status", handler.HandleUpdateNodeStatusGin)

		// 特定节点的操作
		nodeGroup := nodes.Group("/:node_id")
		{
			// 上报指标
			nodeGroup.POST("/metrics", handler.HandleMetricsSubmitGin)

			// 恢复节点令牌
			nodeGroup.GET("/token", handler.HandleRetrieveNodeTokenGin)
		}
	}

	// WebSocket 连接
	rg.GET("/ws/nodes", handler.HandleWebSocketGin)
}

// 分组相关路由
func setupGroupRoutes(rg *gin.RouterGroup, handler *MetricsHandler) {
	groups := rg.Group("/groups")
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
}

// 服务相关路由
func setupServiceRoutes(rg *gin.RouterGroup, handler *MetricsHandler) {
	services := rg.Group("/services")
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
}

// 告警规则相关路由
func setupAlertRoutes(rg *gin.RouterGroup, handler *MetricsHandler) {
	alerts := rg.Group("/alerts")
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
}

// 通知相关路由
func setupNotificationRoutes(rg *gin.RouterGroup, handler *MetricsHandler) {
	notifications := rg.Group("/notifications")
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
