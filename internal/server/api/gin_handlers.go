package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/syslens/syslens-api/internal/common/utils"
	"github.com/syslens/syslens-api/internal/server/repository"
	"go.uber.org/zap"
)

// HandleGetAllNodesGin godoc
//
//	@Summary		获取所有节点
//	@Description	获取系统中所有注册的节点列表
//	@Tags			nodes
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	Response{data=[]Node}
//	@Router			/api/v1/nodes [get]
func (h *MetricsHandler) HandleGetAllNodesGin(c *gin.Context) {
	// 从节点仓库获取所有节点
	if h.nodeRepo != nil {
		nodes, err := h.nodeRepo.GetAll(c.Request.Context())
		if err != nil {
			RespondWithError(c, http.StatusInternalServerError, err, "获取节点列表失败")
			return
		}
		RespondWithSuccess(c, http.StatusOK, nodes)
		return
	}

	// 兼容旧实现，从指标存储中获取节点列表
	nodes, err := h.storage.GetAllNodes()
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err, "获取节点列表失败")
		return
	}

	// 返回节点列表
	RespondWithSuccess(c, http.StatusOK, nodes)
}

// HandleGetNodeMetricsGin godoc
//
//	@Summary		获取节点指标
//	@Description	获取指定节点的监控指标数据，支持时间范围查询
//	@Tags			nodes
//	@Accept			json
//	@Produce		json
//	@Param			node_id	query		string	true	"节点ID"
//	@Param			start	query		string	false	"开始时间（RFC3339格式）"
//	@Param			end		query		string	false	"结束时间（RFC3339格式）"
//	@Success		200		{object}	Response{data=[]NodeMetrics}
//	@Failure		400		{object}	Response
//	@Failure		500		{object}	Response
//	@Router			/api/v1/nodes/metrics [get]
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

// HandleRegisterNodeGin 注册节点
//
//	@Summary		注册节点
//	@Description	注册一个新节点或更新已存在节点的信息
//	@Tags			nodes
//	@Accept			json
//	@Produce		json
//	@Param			request	body		NodeRegisterRequest	true	"节点注册信息"
//	@Success		200		{object}	Response{data=Node}	"成功"
//	@Failure		400		{object}	Response			"请求错误"
//	@Failure		500		{object}	Response			"服务器错误"
//	@Router			/api/v1/nodes/register [post]
func (h *MetricsHandler) HandleRegisterNodeGin(c *gin.Context) {
	// 检查是否配置了节点仓库
	if h.nodeRepo == nil {
		h.logger.Error("节点仓库未配置")
		RespondWithError(c, http.StatusInternalServerError, nil, "系统配置错误，节点仓库未初始化")
		return
	}

	// 获取请求体
	var registerRequest struct {
		NodeID      string         `json:"node_id"`
		Name        string         `json:"name" binding:"required"`
		Labels      map[string]any `json:"labels"`
		Type        string         `json:"type"`
		GroupID     string         `json:"group_id"`
		ServiceID   string         `json:"service_id"`
		Description string         `json:"description"`
		AuthToken   string         `json:"auth_token"`
	}

	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		h.logger.Error("解析节点注册请求失败",
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()))
		RespondWithError(c, http.StatusBadRequest, err, "解析请求数据失败")
		return
	}

	// 记录请求信息
	h.logger.Info("接收到节点注册请求",
		zap.String("node_id", registerRequest.NodeID),
		zap.String("name", registerRequest.Name),
		zap.String("client_ip", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()))

	// 实现节点注册逻辑
	// 1. 检查节点是否已存在
	var node *repository.Node
	var err error

	ctx := c.Request.Context()
	if registerRequest.NodeID != "" {
		// 如果提供了节点ID，检查是否已存在
		node, err = h.nodeRepo.GetByID(ctx, registerRequest.NodeID)
		if err != nil {
			h.logger.Error("查询节点失败",
				zap.String("node_id", registerRequest.NodeID),
				zap.Error(err))
			RespondWithError(c, http.StatusInternalServerError, err, "查询节点信息失败")
			return
		}
	}

	// 2. 如果节点不存在，创建新节点
	if node == nil {
		// 生成节点ID（如果未提供）
		nodeID := registerRequest.NodeID
		if nodeID == "" {
			// 生成一个唯一ID，这里使用时间戳+随机字符组合
			nodeID = fmt.Sprintf("node-%d-%s", time.Now().Unix(),
				utils.GenerateRandomString(8))
		}

		// 生成认证令牌（如果未提供）
		authToken := registerRequest.AuthToken
		if authToken == "" {
			authToken = utils.GenerateRandomString(32)
		}

		// 计算令牌的哈希值（存储哈希而不是原始令牌）
		authTokenHash, err := utils.HashPassword(authToken)
		if err != nil {
			h.logger.Error("生成令牌哈希失败", zap.Error(err))
			RespondWithError(c, http.StatusInternalServerError, err, "生成安全凭证失败")
			return
		}

		// 使用系统主密钥加密令牌
		systemKey := h.securityConfig.Encryption.Key
		if systemKey == "" {
			h.logger.Warn("系统主密钥未设置，使用默认密钥")
			systemKey = "syslens-default-encryption-key-2023" // 默认密钥，建议在配置中设置更强的密钥
		}

		// 加密服务初始化
		encryptSvc := utils.NewEncryptionService("aes-256-gcm")
		encryptedTokenBytes, err := encryptSvc.Encrypt([]byte(authToken), systemKey)
		if err != nil {
			h.logger.Error("加密令牌失败", zap.Error(err))
			RespondWithError(c, http.StatusInternalServerError, err, "加密令牌失败")
			return
		}
		encryptedToken := base64.StdEncoding.EncodeToString(encryptedTokenBytes)

		// 确定节点类型
		nodeType := repository.NodeTypeAgent
		if registerRequest.Type == string(repository.NodeTypeFixedService) {
			nodeType = repository.NodeTypeFixedService
		}

		// 创建新节点实体
		now := time.Now()
		node = &repository.Node{
			ID:                 nodeID,
			Name:               registerRequest.Name,
			AuthTokenHash:      authTokenHash,
			EncryptedAuthToken: encryptedToken,
			Labels:             registerRequest.Labels,
			Type:               nodeType,
			Status:             repository.NodeStatusPending, // 初始状态为待处理
			RegisteredAt:       sql.NullTime{Time: now, Valid: true},
			LastActiveAt:       sql.NullTime{Time: now, Valid: true},
		}

		// 设置可选字段
		if registerRequest.GroupID != "" {
			node.GroupID = sql.NullString{String: registerRequest.GroupID, Valid: true}
		}
		if registerRequest.ServiceID != "" {
			node.ServiceID = sql.NullString{String: registerRequest.ServiceID, Valid: true}
		}
		if registerRequest.Description != "" {
			node.Description = sql.NullString{String: registerRequest.Description, Valid: true}
		}

		// 保存节点到数据库
		if err := h.nodeRepo.Create(ctx, node); err != nil {
			h.logger.Error("创建节点失败",
				zap.String("name", node.Name),
				zap.Error(err))
			RespondWithError(c, http.StatusInternalServerError, err, "创建节点失败")
			return
		}

		// 记录成功信息
		h.logger.Info("节点注册成功",
			zap.String("node_id", node.ID),
			zap.String("name", node.Name))

		// 返回节点信息和认证令牌
		RespondWithSuccess(c, http.StatusOK, gin.H{
			"message":    "节点注册成功",
			"node_id":    node.ID,
			"auth_token": authToken, // 仅在初始注册时返回明文令牌
		})
		return
	}

	// 3. 如果节点已存在，更新节点信息
	// 更新基本信息
	node.Name = registerRequest.Name
	if registerRequest.Labels != nil {
		node.Labels = registerRequest.Labels
	}

	// 更新可选字段
	if registerRequest.GroupID != "" {
		node.GroupID = sql.NullString{String: registerRequest.GroupID, Valid: true}
	}
	if registerRequest.ServiceID != "" {
		node.ServiceID = sql.NullString{String: registerRequest.ServiceID, Valid: true}
	}
	if registerRequest.Description != "" {
		node.Description = sql.NullString{String: registerRequest.Description, Valid: true}
	}

	// 如果节点状态是非活动，将其设为等待中
	if node.Status == repository.NodeStatusInactive {
		node.Status = repository.NodeStatusPending
	}

	// 更新最后活动时间
	now := time.Now()
	node.LastActiveAt = sql.NullTime{Time: now, Valid: true}

	// 保存更新到数据库
	if err := h.nodeRepo.Update(ctx, node); err != nil {
		h.logger.Error("更新节点失败",
			zap.String("node_id", node.ID),
			zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, err, "更新节点信息失败")
		return
	}

	// 记录成功信息
	h.logger.Info("节点信息更新成功",
		zap.String("node_id", node.ID),
		zap.String("name", node.Name))

	// 返回更新结果
	RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "节点信息更新成功",
		"node_id": node.ID,
	})
}

// HandleGetNodeTokenGin HandleRetrieveNodeToken godoc
//
//	@Summary		获取节点令牌
//	@Description	根据节点ID恢复节点的认证令牌
//	@Tags			nodes
//	@Accept			json
//	@Produce		json
//	@Param			node_id	path		string													true	"节点ID"
//	@Success		200		{object}	Response{data=object{node_id=string,auth_token=string}}	"成功"
//	@Failure		404		{object}	Response												"节点不存在"
//	@Failure		500		{object}	Response												"服务器错误"
//	@Router			/api/v1/nodes/{node_id}/token [get]
func (h *MetricsHandler) HandleGetNodeTokenGin(c *gin.Context) {
	// 获取节点ID
	nodeID := c.Param("node_id")
	if nodeID == "" {
		RespondWithError(c, http.StatusBadRequest, nil, "缺少节点ID")
		return
	}

	// 检查是否配置了节点仓库
	if h.nodeRepo == nil {
		h.logger.Error("节点仓库未配置")
		RespondWithError(c, http.StatusInternalServerError, nil, "系统配置错误，节点仓库未初始化")
		return
	}

	// 获取节点信息
	ctx := c.Request.Context()
	node, err := h.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		h.logger.Error("获取节点信息失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, err, "获取节点信息失败")
		return
	}

	// 检查节点是否存在
	if node == nil {
		h.logger.Warn("节点不存在",
			zap.String("node_id", nodeID))
		RespondWithError(c, http.StatusNotFound, nil, "节点不存在")
		return
	}

	// 检查是否有加密的令牌
	if node.EncryptedAuthToken == "" {
		h.logger.Warn("节点没有保存加密令牌",
			zap.String("node_id", nodeID))
		RespondWithError(c, http.StatusNotFound, nil, "节点没有保存令牌或令牌已丢失")
		return
	}

	// 解密令牌
	systemKey := h.securityConfig.Encryption.Key
	if systemKey == "" {
		systemKey = "syslens-default-encryption-key-2023" // 默认密钥，与加密时使用的相同
	}

	// 解码Base64
	encryptedBytes, err := base64.StdEncoding.DecodeString(node.EncryptedAuthToken)
	if err != nil {
		h.logger.Error("解码加密令牌失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, err, "解码加密令牌失败")
		return
	}

	// 解密
	encryptSvc := utils.NewEncryptionService("aes-256-gcm")
	decryptedBytes, err := encryptSvc.Decrypt(encryptedBytes, systemKey)
	if err != nil {
		h.logger.Error("解密令牌失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, err, "解密令牌失败")
		return
	}

	// 记录操作日志(敏感操作)
	h.logger.Info("节点令牌恢复请求成功",
		zap.String("node_id", nodeID),
		zap.String("client_ip", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()))

	// 返回令牌
	RespondWithSuccess(c, http.StatusOK, gin.H{
		"node_id":    nodeID,
		"auth_token": string(decryptedBytes),
		"message":    "节点令牌获取成功",
	})
}

// HandleMetricsSubmitGin godoc
//
//	@Summary		上报节点指标
//	@Description	接收并处理节点上报的监控指标数据
//	@Tags			metrics
//	@Accept			json
//	@Produce		json
//	@Param			node_id			path		string	true	"节点ID"
//	@Param			X-Encrypted		header		string	false	"是否加密(true/false)"
//	@Param			X-Compressed	header		string	false	"压缩格式(gzip)"
//	@Param			X-Aggregator-ID	header		string	false	"聚合服务器ID"
//	@Param			metrics			body		object	true	"指标数据"
//	@Success		200				{object}	object{message=string,time=string,success=bool}
//	@Failure		400				{object}	object{error=string,message=string,success=bool}
//	@Failure		500				{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/nodes/{node_id}/metrics [post]
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

// HandleWebSocketGin godoc
//
//	@Summary		WebSocket连接
//	@Description	建立WebSocket连接，用于实时监控节点指标
//	@Tags			websocket
//	@Accept			json
//	@Produce		json
//	@Param			node_id	query		string	false	"节点ID"
//	@Success		200		{object}	object{message=string,success=bool}
//	@Router			/api/v1/ws/nodes [get]
func (h *MetricsHandler) HandleWebSocketGin(c *gin.Context) {
	// WebSocket处理逻辑
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket端点"})
}

// HandleGetGroupsGin 分组相关处理函数
// HandleGetGroupsGin godoc
//
//	@Summary		获取所有分组
//	@Description	获取系统中所有节点分组
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object{data=[]object,success=bool}
//	@Router			/api/v1/groups [get]
func (h *MetricsHandler) HandleGetGroupsGin(c *gin.Context) {
	// 获取分组列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取分组列表"})
}

// HandleCreateGroupGin godoc
//
//	@Summary		创建分组
//	@Description	创建新的节点分组
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			group	body		object	true	"分组信息"
//	@Success		201		{object}	object{message=string,success=bool}
//	@Failure		400		{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/groups [post]
func (h *MetricsHandler) HandleCreateGroupGin(c *gin.Context) {
	// 创建分组逻辑
	RespondWithSuccess(c, http.StatusCreated, gin.H{"message": "创建分组成功"})
}

// HandleGetGroupGin godoc
//
//	@Summary		获取分组详情
//	@Description	获取指定分组的详细信息
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			group_id	path		string	true	"分组ID"
//	@Success		200			{object}	object{group_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/groups/{group_id} [get]
func (h *MetricsHandler) HandleGetGroupGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 获取分组详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "获取分组详情"})
}

// HandleUpdateGroupGin godoc
//
//	@Summary		更新分组
//	@Description	更新指定分组的信息
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			group_id	path		string	true	"分组ID"
//	@Param			group		body		object	true	"分组更新信息"
//	@Success		200			{object}	object{group_id=string,message=string,success=bool}
//	@Failure		400			{object}	object{error=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/groups/{group_id} [put]
func (h *MetricsHandler) HandleUpdateGroupGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 更新分组逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "更新分组成功"})
}

// HandleDeleteGroupGin godoc
//
//	@Summary		删除分组
//	@Description	删除指定的节点分组
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			group_id	path		string	true	"分组ID"
//	@Success		200			{object}	object{group_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/groups/{group_id} [delete]
func (h *MetricsHandler) HandleDeleteGroupGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 删除分组逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "删除分组成功"})
}

// HandleGetGroupNodesGin godoc
//
//	@Summary		获取分组节点
//	@Description	获取指定分组中的所有节点
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			group_id	path		string	true	"分组ID"
//	@Success		200			{object}	object{group_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/groups/{group_id}/nodes [get]
func (h *MetricsHandler) HandleGetGroupNodesGin(c *gin.Context) {
	// 获取分组ID
	groupID := c.Param("group_id")
	// 获取分组节点列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"group_id": groupID, "message": "获取分组节点列表"})
}

// HandleGetServicesGin 服务相关处理函数
// HandleGetServicesGin godoc
//
//	@Summary		获取所有服务
//	@Description	获取系统中所有注册的服务
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object{message=string,success=bool}
//	@Router			/api/v1/services [get]
func (h *MetricsHandler) HandleGetServicesGin(c *gin.Context) {
	// 获取服务列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取服务列表"})
}

// HandleCreateServiceGin godoc
//
//	@Summary		创建服务
//	@Description	创建新的服务
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Param			service	body		object	true	"服务信息"
//	@Success		201		{object}	object{message=string,success=bool}
//	@Failure		400		{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/services [post]
func (h *MetricsHandler) HandleCreateServiceGin(c *gin.Context) {
	// 创建服务逻辑
	RespondWithSuccess(c, http.StatusCreated, gin.H{"message": "创建服务成功"})
}

// HandleGetServiceGin godoc
//
//	@Summary		获取服务详情
//	@Description	获取指定服务的详细信息
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Param			service_id	path		string	true	"服务ID"
//	@Success		200			{object}	object{service_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/services/{service_id} [get]
func (h *MetricsHandler) HandleGetServiceGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 获取服务详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "获取服务详情"})
}

// HandleUpdateServiceGin godoc
//
//	@Summary		更新服务
//	@Description	更新指定服务的信息
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Param			service_id	path		string	true	"服务ID"
//	@Param			service		body		object	true	"服务更新信息"
//	@Success		200			{object}	object{service_id=string,message=string,success=bool}
//	@Failure		400			{object}	object{error=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/services/{service_id} [put]
func (h *MetricsHandler) HandleUpdateServiceGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 更新服务逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "更新服务成功"})
}

// HandleDeleteServiceGin godoc
//
//	@Summary		删除服务
//	@Description	删除指定的服务
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Param			service_id	path		string	true	"服务ID"
//	@Success		200			{object}	object{service_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/services/{service_id} [delete]
func (h *MetricsHandler) HandleDeleteServiceGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 删除服务逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "删除服务成功"})
}

// HandleGetServiceNodesGin godoc
//
//	@Summary		获取服务节点
//	@Description	获取指定服务关联的所有节点
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Param			service_id	path		string	true	"服务ID"
//	@Success		200			{object}	object{service_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/services/{service_id}/nodes [get]
func (h *MetricsHandler) HandleGetServiceNodesGin(c *gin.Context) {
	// 获取服务ID
	serviceID := c.Param("service_id")
	// 获取服务节点列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"service_id": serviceID, "message": "获取服务节点列表"})
}

// HandleGetAlertsGin 告警规则相关处理函数
// HandleGetAlertsGin godoc
//
//	@Summary		获取所有告警规则
//	@Description	获取系统中所有配置的告警规则
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object{message=string,success=bool}
//	@Router			/api/v1/alerts [get]
func (h *MetricsHandler) HandleGetAlertsGin(c *gin.Context) {
	// 获取告警规则列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取告警规则列表"})
}

// HandleCreateAlertGin godoc
//
//	@Summary		创建告警规则
//	@Description	创建新的告警规则
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Param			alert	body		object	true	"告警规则信息"
//	@Success		201		{object}	object{message=string,success=bool}
//	@Failure		400		{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/alerts [post]
func (h *MetricsHandler) HandleCreateAlertGin(c *gin.Context) {
	// 创建告警规则逻辑
	RespondWithSuccess(c, http.StatusCreated, gin.H{"message": "创建告警规则成功"})
}

// HandleGetAlertGin godoc
//
//	@Summary		获取告警规则详情
//	@Description	获取指定告警规则的详细信息
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Param			alert_id	path		string	true	"告警规则ID"
//	@Success		200			{object}	object{alert_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/alerts/{alert_id} [get]
func (h *MetricsHandler) HandleGetAlertGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 获取告警规则详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "获取告警规则详情"})
}

// HandleUpdateAlertGin godoc
//
//	@Summary		更新告警规则
//	@Description	更新指定告警规则的信息
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Param			alert_id	path		string	true	"告警规则ID"
//	@Param			alert		body		object	true	"告警规则更新信息"
//	@Success		200			{object}	object{alert_id=string,message=string,success=bool}
//	@Failure		400			{object}	object{error=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/alerts/{alert_id} [put]
func (h *MetricsHandler) HandleUpdateAlertGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 更新告警规则逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "更新告警规则成功"})
}

// HandleDeleteAlertGin godoc
//
//	@Summary		删除告警规则
//	@Description	删除指定的告警规则
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Param			alert_id	path		string	true	"告警规则ID"
//	@Success		200			{object}	object{alert_id=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/alerts/{alert_id} [delete]
func (h *MetricsHandler) HandleDeleteAlertGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 删除告警规则逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "删除告警规则成功"})
}

// HandleUpdateAlertStatusGin godoc
//
//	@Summary		更新告警规则状态
//	@Description	更新指定告警规则的启用状态
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Param			alert_id	path		string	true	"告警规则ID"
//	@Param			status		body		object	true	"告警规则状态"
//	@Success		200			{object}	object{alert_id=string,message=string,success=bool}
//	@Failure		400			{object}	object{error=string,message=string,success=bool}
//	@Failure		404			{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/alerts/{alert_id}/status [patch]
func (h *MetricsHandler) HandleUpdateAlertStatusGin(c *gin.Context) {
	// 获取告警规则ID
	alertID := c.Param("alert_id")
	// 更新告警规则状态逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"alert_id": alertID, "message": "更新告警规则状态成功"})
}

// HandleGetNotificationsGin 通知相关处理函数
// HandleGetNotificationsGin godoc
//
//	@Summary		获取所有通知
//	@Description	获取系统中所有通知记录
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	object{message=string,success=bool}
//	@Router			/api/v1/notifications [get]
func (h *MetricsHandler) HandleGetNotificationsGin(c *gin.Context) {
	// 获取通知列表逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "获取通知列表"})
}

// HandleGetNotificationGin godoc
//
//	@Summary		获取通知详情
//	@Description	获取指定通知的详细信息
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Param			notification_id	path		string	true	"通知ID"
//	@Success		200				{object}	object{notification_id=string,message=string,success=bool}
//	@Failure		404				{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/notifications/{notification_id} [get]
func (h *MetricsHandler) HandleGetNotificationGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 获取通知详情逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "获取通知详情"})
}

// HandleUpdateNotificationStatusGin godoc
//
//	@Summary		更新通知状态
//	@Description	更新指定通知的状态
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Param			notification_id	path		string	true	"通知ID"
//	@Param			status			body		object	true	"通知状态"
//	@Success		200				{object}	object{notification_id=string,message=string,success=bool}
//	@Failure		400				{object}	object{error=string,message=string,success=bool}
//	@Failure		404				{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/notifications/{notification_id}/status [patch]
func (h *MetricsHandler) HandleUpdateNotificationStatusGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 更新通知状态逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "更新通知状态成功"})
}

// HandleResolveNotificationGin godoc
//
//	@Summary		解决通知
//	@Description	将指定通知标记为已解决
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Param			notification_id	path		string	true	"通知ID"
//	@Success		200				{object}	object{notification_id=string,message=string,success=bool}
//	@Failure		404				{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/notifications/{notification_id}/resolve [post]
func (h *MetricsHandler) HandleResolveNotificationGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 解决通知逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "解决通知成功"})
}

// HandleDeleteNotificationGin godoc
//
//	@Summary		删除通知
//	@Description	删除指定的通知
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Param			notification_id	path		string	true	"通知ID"
//	@Success		200				{object}	object{notification_id=string,message=string,success=bool}
//	@Failure		404				{object}	object{error=string,message=string,success=bool}
//	@Router			/api/v1/notifications/{notification_id} [delete]
func (h *MetricsHandler) HandleDeleteNotificationGin(c *gin.Context) {
	// 获取通知ID
	notificationID := c.Param("notification_id")
	// 删除通知逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"notification_id": notificationID, "message": "删除通知成功"})
}

// HandleUpdateNodeStatusGin 更新节点状态
//
//	@Summary		更新节点状态
//	@Description	根据节点ID更新节点的状态
//	@Tags			nodes
//	@Accept			json
//	@Produce		json
//	@Param			request	body		StatusUpdateRequest	true	"状态更新请求"
//	@Success		200		{object}	Response			"成功"
//	@Failure		400		{object}	Response			"请求错误"
//	@Failure		500		{object}	Response			"服务器错误"
//	@Router			/api/v1/nodes/status [put]
func (h *MetricsHandler) HandleUpdateNodeStatusGin(c *gin.Context) {
	// 获取节点状态更新请求
	var statusUpdateRequest StatusUpdateRequest
	if err := c.ShouldBindJSON(&statusUpdateRequest); err != nil {
		RespondWithError(c, http.StatusBadRequest, err, "解析请求数据失败")
		return
	}

	// 更新节点状态逻辑
	RespondWithSuccess(c, http.StatusOK, gin.H{"message": "节点状态更新成功"})
}
