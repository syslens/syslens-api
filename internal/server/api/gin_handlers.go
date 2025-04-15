package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleGetAllNodesGin 获取所有节点
func (h *MetricsHandler) HandleGetAllNodesGin(c *gin.Context) {
	// 获取所有节点ID
	nodes, err := h.storage.GetAllNodes()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err, "获取节点列表失败")
		return
	}

	// 返回节点列表
	RespondWithSuccess(c, http.StatusOK, nodes)
}

// HandleGetNodeMetricsGin 获取节点指标
func (h *MetricsHandler) HandleGetNodeMetricsGin(c *gin.Context) {
	// 获取节点ID
	nodeID := c.Query("node_id")
	if nodeID == "" {
		RespondWithError(c, http.StatusBadRequest, nil, "缺少节点ID")
		return
	}

	// 解析时间范围
	startTimeStr := c.Query("start")
	endTimeStr := c.Query("end")

	var startTime, endTime time.Time
	var err error

	// 如果未提供时间范围，使用过去1小时
	if startTimeStr == "" {
		startTime = time.Now().Add(-1 * time.Hour)
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, err, "开始时间格式无效")
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, err, "结束时间格式无效")
			return
		}
	}

	// 查询指标数据
	metrics, err := h.storage.GetNodeMetrics(nodeID, startTime, endTime)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err, "获取指标数据失败")
		return
	}

	// 返回指标数据
	RespondWithSuccess(c, http.StatusOK, metrics)
}

// HandleNodeRegisterGin 处理节点注册
func (h *MetricsHandler) HandleNodeRegisterGin(c *gin.Context) {
	// 处理注册逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "节点注册成功"})
}

// HandleMetricsSubmitGin 处理节点上报的指标数据
func (h *MetricsHandler) HandleMetricsSubmitGin(c *gin.Context) {
	// 从URL参数获取节点ID
	nodeID := c.Param("node_id")
	if nodeID == "" {
		RespondWithError(c, http.StatusBadRequest, nil, "缺少节点ID")
		return
	}

	// 获取请求发送方IP和聚合服务器ID（如果有）
	remoteIP := c.ClientIP()
	aggregatorID := c.GetHeader("X-Aggregator-ID")
	source := "直接节点上报"
	if aggregatorID != "" {
		source = fmt.Sprintf("聚合服务器(%s)", aggregatorID)
	}

	// 记录请求信息
	h.logger.Info("接收到指标上报请求",
		zap.String("node_id", nodeID),
		zap.String("source", source),
		zap.String("ip", remoteIP))

	// 读取请求体
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("读取请求体失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, err, "读取请求数据失败")
		return
	}

	// 检查是否需要解密和解压缩
	isEncrypted := c.GetHeader("X-Encrypted") == "true"
	isCompressed := c.GetHeader("X-Compressed") == "gzip"

	h.logger.Debug("数据处理标记",
		zap.String("node_id", nodeID),
		zap.Bool("encrypted", isEncrypted),
		zap.Bool("compressed", isCompressed))

	// 处理数据
	startProcessing := time.Now()
	processedData, err := h.processData(body, isEncrypted, isCompressed)
	if err != nil {
		h.logger.Error("数据处理失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, err, "处理请求数据失败")
		return
	}

	// 解析处理后的数据
	var metricsData map[string]interface{}
	if err := json.Unmarshal(processedData, &metricsData); err != nil {
		h.logger.Error("JSON解析失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, err, "解析JSON数据失败")
		return
	}

	// 添加接收时间戳
	receivedAt := time.Now().Unix()
	metricsData["received_at"] = receivedAt
	h.logger.Debug("添加接收时间戳",
		zap.String("node_id", nodeID),
		zap.Int64("timestamp", receivedAt))

	// 记录关键指标（如果存在）
	if cpu, ok := metricsData["cpu"].(map[string]interface{}); ok {
		if usage, ok := cpu["usage"]; ok {
			h.logger.Debug("CPU使用率",
				zap.String("node_id", nodeID),
				zap.Any("usage", usage))
		}
	}
	if memory, ok := metricsData["memory"].(map[string]interface{}); ok {
		if used, ok := memory["used_percent"]; ok {
			h.logger.Debug("内存使用率",
				zap.String("node_id", nodeID),
				zap.Any("used_percent", used))
		}
	}

	// 存储指标数据
	startStoring := time.Now()
	if err := h.storage.StoreMetrics(nodeID, metricsData); err != nil {
		h.logger.Error("存储指标数据失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, err, "存储指标数据失败")
		return
	}
	storingTime := time.Since(startStoring)
	h.logger.Info("指标数据存储成功",
		zap.String("node_id", nodeID),
		zap.Duration("time", storingTime))

	// 返回成功
	totalTime := time.Since(startProcessing)
	h.logger.Info("指标上报处理完成",
		zap.String("node_id", nodeID),
		zap.Duration("total_time", totalTime))

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "指标数据上报成功",
		"time":    totalTime.String(),
	})
}

// HandleWebSocketGin 处理WebSocket连接
func (h *MetricsHandler) HandleWebSocketGin(c *gin.Context) {
	// WebSocket处理逻辑
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket端点"})
}

// 分组相关处理函数
// HandleGetGroupsGin 获取所有分组
func (h *MetricsHandler) HandleGetGroupsGin(c *gin.Context) {
	// 获取分组列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取分组列表"})
}

// HandleCreateGroupGin 创建分组
func (h *MetricsHandler) HandleCreateGroupGin(c *gin.Context) {
	// 创建分组逻辑
	RespondWithSuccess(c, http.StatusCreated, gin.H{"message": "创建分组成功"})
}

// HandleGetGroupGin 获取特定分组
func (h *MetricsHandler) HandleGetGroupGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 获取分组详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "获取分组详情"})
}

// HandleUpdateGroupGin 更新分组
func (h *MetricsHandler) HandleUpdateGroupGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 更新分组逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "更新分组成功"})
}

// HandleDeleteGroupGin 删除分组
func (h *MetricsHandler) HandleDeleteGroupGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 删除分组逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "删除分组成功"})
}

// HandleGetGroupNodesGin 获取分组内的节点
func (h *MetricsHandler) HandleGetGroupNodesGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 获取分组节点列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "获取分组节点列表"})
}

// 服务相关处理函数
// HandleGetServicesGin 获取所有服务
func (h *MetricsHandler) HandleGetServicesGin(c *gin.Context) {
	// 获取服务列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取服务列表"})
}

// HandleCreateServiceGin 创建服务
func (h *MetricsHandler) HandleCreateServiceGin(c *gin.Context) {
	// 创建服务逻辑
	RespondWithSuccess(c, http.StatusCreated, gin.H{"message": "创建服务成功"})
}

// HandleGetServiceGin 获取特定服务
func (h *MetricsHandler) HandleGetServiceGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 获取服务详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "获取服务详情"})
}

// HandleUpdateServiceGin 更新服务
func (h *MetricsHandler) HandleUpdateServiceGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 更新服务逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "更新服务成功"})
}

// HandleDeleteServiceGin 删除服务
func (h *MetricsHandler) HandleDeleteServiceGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 删除服务逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "删除服务成功"})
}

// HandleGetServiceNodesGin 获取服务关联的节点
func (h *MetricsHandler) HandleGetServiceNodesGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 获取服务节点列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "获取服务节点列表"})
}

// 告警规则相关处理函数
// HandleGetAlertsGin 获取所有告警规则
func (h *MetricsHandler) HandleGetAlertsGin(c *gin.Context) {
	// 获取告警规则列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取告警规则列表"})
}

// HandleCreateAlertGin 创建告警规则
func (h *MetricsHandler) HandleCreateAlertGin(c *gin.Context) {
	// 创建告警规则逻辑
	RespondWithSuccess(c, http.StatusCreated, gin.H{"message": "创建告警规则成功"})
}

// HandleGetAlertGin 获取特定告警规则
func (h *MetricsHandler) HandleGetAlertGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 获取告警规则详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "获取告警规则详情"})
}

// HandleUpdateAlertGin 更新告警规则
func (h *MetricsHandler) HandleUpdateAlertGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 更新告警规则逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "更新告警规则成功"})
}

// HandleDeleteAlertGin 删除告警规则
func (h *MetricsHandler) HandleDeleteAlertGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 删除告警规则逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "删除告警规则成功"})
}

// HandleUpdateAlertStatusGin 更新告警规则状态
func (h *MetricsHandler) HandleUpdateAlertStatusGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 更新告警规则状态逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "更新告警规则状态成功"})
}

// 通知相关处理函数
// HandleGetNotificationsGin 获取所有通知
func (h *MetricsHandler) HandleGetNotificationsGin(c *gin.Context) {
	// 获取通知列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取通知列表"})
}

// HandleGetNotificationGin 获取特定通知
func (h *MetricsHandler) HandleGetNotificationGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 获取通知详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "获取通知详情"})
}

// HandleUpdateNotificationStatusGin 更新通知状态
func (h *MetricsHandler) HandleUpdateNotificationStatusGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 更新通知状态逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "更新通知状态成功"})
}

// HandleResolveNotificationGin 解决通知
func (h *MetricsHandler) HandleResolveNotificationGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 解决通知逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "解决通知成功"})
}

// HandleDeleteNotificationGin 删除通知
func (h *MetricsHandler) HandleDeleteNotificationGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 删除通知逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "删除通知成功"})
}
