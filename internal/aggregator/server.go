package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/syslens/syslens-api/internal/common/utils"
	"github.com/syslens/syslens-api/internal/config"
	"go.uber.org/zap"
)

// Server 聚合服务器
type Server struct {
	// 配置
	config *config.AggregatorConfig

	// HTTP服务器
	httpServer *http.Server

	// 路由引擎
	router *gin.Engine

	// 日志记录器
	logger *zap.Logger

	// 节点连接管理
	connections struct {
		sync.RWMutex
		// 节点ID -> 连接信息
		nodes map[string]*NodeConnection
	}

	// 数据处理
	processor *DataProcessor

	// 控制平面客户端
	controlPlane *ControlPlaneClient

	// 加密服务 (用于处理Agent数据)
	encryptionSvc *utils.EncryptionService

	// 上下文和取消函数
	ctx    context.Context
	cancel context.CancelFunc

	// 等待组，用于等待所有goroutine完成
	wg sync.WaitGroup
}

// NodeConnection 节点连接信息
type NodeConnection struct {
	// 节点ID
	NodeID string

	// 最后活动时间
	LastActive time.Time

	// 连接状态
	Status string

	// 连接时间
	ConnectedAt time.Time

	// 节点是否已通过验证
	Verified bool
}

// NewServer 创建新的聚合服务器
func NewServer(cfg *config.AggregatorConfig) (*Server, error) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建服务器
	s := &Server{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	// 初始化节点连接管理
	s.connections.nodes = make(map[string]*NodeConnection)

	// 初始化日志记录器
	if err := s.initLogger(); err != nil {
		return nil, fmt.Errorf("初始化日志记录器失败: %w", err)
	}

	// 初始化加密服务 (如果启用了)
	if cfg.Security.Encryption.Enabled {
		s.encryptionSvc = utils.NewEncryptionService(cfg.Security.Encryption.Algorithm)
		s.logger.Info("加密服务已初始化", zap.String("algorithm", cfg.Security.Encryption.Algorithm))
	}

	// 初始化路由
	s.initRouter()

	// 初始化HTTP服务器
	s.httpServer = &http.Server{
		Addr:         cfg.Server.ListenAddr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.Server.ConnectionTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.ConnectionTimeout) * time.Second,
		IdleTimeout:  120 * time.Second, // 空闲连接超时
		// 添加最大请求头大小限制
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// 初始化数据处理器
	s.processor = NewDataProcessor(cfg) // 移除 logger
	s.processor.logger = s.logger       // 设置 logger

	// 初始化控制平面客户端
	s.controlPlane = NewControlPlaneClient(cfg) // 移除 logger
	s.controlPlane.logger = s.logger            // 设置 logger

	return s, nil
}

// initLogger 初始化日志记录器
func (s *Server) initLogger() error {
	// 创建日志配置
	config := zap.NewProductionConfig()

	// 设置日志级别
	switch s.config.Log.Level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// 设置日志输出
	if s.config.Log.File != "" {
		config.OutputPaths = []string{s.config.Log.File}
	}

	if s.config.Log.Console {
		config.OutputPaths = append(config.OutputPaths, "stdout")
	}

	// 创建日志记录器
	logger, err := config.Build(zap.AddCallerSkip(1)) // 调整 caller skip
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}

	s.logger = logger
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter() {
	// 创建路由引擎
	s.router = gin.New()

	// 使用恢复中间件
	s.router.Use(gin.Recovery())

	// 添加请求日志中间件
	s.router.Use(func(c *gin.Context) {
		// 记录请求开始
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		s.logger.Debug("收到HTTP请求",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()))

		// 处理请求
		c.Next()

		// 记录请求完成
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		s.logger.Debug("HTTP请求完成",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency))
	})

	// 健康检查
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API路由组
	api := s.router.Group("/api/v1")
	{
		// 节点指标上报 (添加认证中间件)
		api.POST("/nodes/:node_id/metrics", s.authMiddleware(), s.handleNodeMetrics)

		// 节点注册
		api.POST("/nodes/register", s.handleNodeRegister)

		// 节点心跳 (添加认证中间件)
		api.POST("/nodes/:node_id/heartbeat", s.authMiddleware(), s.handleNodeHeartbeat)

		// 获取节点列表 (可以考虑添加管理认证)
		api.GET("/nodes", s.handleGetNodes)
	}
}

// authMiddleware 创建一个简单的认证中间件，检查节点是否已验证
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("node_id")
		if nodeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "路径缺少节点ID"})
			c.Abort()
			return
		}

		s.connections.RLock()
		conn, exists := s.connections.nodes[nodeID]
		s.connections.RUnlock()

		if !exists {
			s.logger.Warn("收到未注册节点的请求", zap.String("node_id", nodeID), zap.String("path", c.Request.URL.Path))
			// 暂时允许未注册节点的请求，但标记为未验证
			// c.JSON(http.StatusUnauthorized, gin.H{"error": "节点未注册"})
			// c.Abort()
			// return
		} else if !conn.Verified {
			s.logger.Warn("收到未验证节点的请求", zap.String("node_id", nodeID), zap.String("path", c.Request.URL.Path))
			// c.JSON(http.StatusUnauthorized, gin.H{"error": "节点未验证"})
			// c.Abort()
			// return
		}

		c.Next()
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.logger.Info("启动聚合服务器",
		zap.String("listen_addr", s.config.Server.ListenAddr))

	// 启动数据处理器
	if err := s.processor.Start(s.ctx); err != nil {
		return fmt.Errorf("启动数据处理器失败: %w", err)
	}

	// 启动控制平面客户端
	if err := s.controlPlane.Start(s.ctx); err != nil {
		return fmt.Errorf("启动控制平面客户端失败: %w", err)
	}

	// 启动HTTP服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP服务器错误", zap.Error(err))
		}
	}()

	// 启动清理过期连接的goroutine
	s.wg.Add(1)
	go s.cleanupExpiredConnections()

	s.logger.Info("聚合服务器已启动")
	return nil
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭聚合服务器...")

	// 取消上下文，通知所有goroutine退出
	s.cancel()
	s.logger.Info("已发送取消信号给所有goroutine")

	// 创建一个子上下文，用于HTTP服务器关闭
	httpCtx, httpCancel := context.WithTimeout(ctx, 15*time.Second)
	defer httpCancel()

	// 关闭HTTP服务器（给它一个单独的超时上下文）
	s.logger.Info("正在关闭HTTP服务器...")
	if err := s.httpServer.Shutdown(httpCtx); err != nil {
		s.logger.Error("关闭HTTP服务器失败", zap.Error(err))
		// 继续关闭其他资源，不要因为HTTP服务器关闭失败而中断
	} else {
		s.logger.Info("HTTP服务器已关闭")
	}

	// 关闭数据处理器
	s.logger.Info("正在关闭数据处理器...")
	if err := s.processor.Shutdown(); err != nil {
		s.logger.Error("关闭数据处理器失败", zap.Error(err))
	} else {
		s.logger.Info("数据处理器已关闭")
	}

	// 关闭控制平面客户端
	s.logger.Info("正在关闭控制平面客户端...")
	if err := s.controlPlane.Shutdown(); err != nil {
		s.logger.Error("关闭控制平面客户端失败", zap.Error(err))
	} else {
		s.logger.Info("控制平面客户端已关闭")
	}

	// 等待所有goroutine完成，设置一个合理的超时时间
	s.logger.Info("等待所有goroutine完成...")

	// 使用子上下文等待goroutine完成
	goroutineCtx, goroutineCancel := context.WithTimeout(ctx, 10*time.Second)
	defer goroutineCancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		s.logger.Info("所有goroutine已完成")
		s.logger.Info("聚合服务器已关闭")
		return nil
	case <-goroutineCtx.Done():
		s.logger.Warn("等待goroutine完成超时，服务器强制关闭")
		return fmt.Errorf("关闭超时: %w", goroutineCtx.Err())
	}
}

// cleanupExpiredConnections 清理过期连接
func (s *Server) cleanupExpiredConnections() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupConnections()
		}
	}
}

// cleanupConnections 清理过期连接
func (s *Server) cleanupConnections() {
	s.connections.Lock()
	defer s.connections.Unlock()

	now := time.Now()
	timeout := time.Duration(s.config.Server.ConnectionTimeout) * time.Second

	for nodeID, conn := range s.connections.nodes {
		if now.Sub(conn.LastActive) > timeout {
			s.logger.Info("清理过期连接", zap.String("node_id", nodeID))
			delete(s.connections.nodes, nodeID)
		}
	}
}

// handleNodeMetrics 处理节点指标上报
func (s *Server) handleNodeMetrics(c *gin.Context) {
	nodeID := c.Param("node_id")
	// 节点ID已在中间件中检查，这里理论上不会为空

	s.logger.Debug("接收到节点指标上报请求", zap.String("node_id", nodeID))

	// 更新节点活动时间 (如果节点存在)
	s.updateNodeActivity(nodeID)

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.Error("读取请求体失败", zap.String("node_id", nodeID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	// 检查是否需要解密和解压缩
	isEncrypted := c.GetHeader("X-Encrypted") == "true"
	isCompressed := c.GetHeader("X-Compressed") == "gzip"
	s.logger.Debug("处理指标数据标记",
		zap.String("node_id", nodeID),
		zap.Bool("encrypted", isEncrypted),
		zap.Bool("compressed", isCompressed))

	// 处理数据 (解密/解压缩)
	processedData, err := s.processIncomingData(body, isEncrypted, isCompressed)
	if err != nil {
		s.logger.Error("处理 Agent 数据失败", zap.String("node_id", nodeID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "处理数据失败: " + err.Error()})
		return
	}

	// 解析处理后的数据
	var metrics map[string]interface{}
	if err := json.Unmarshal(processedData, &metrics); err != nil {
		s.logger.Error("解析处理后的 JSON 数据失败", zap.String("node_id", nodeID), zap.Error(err), zap.ByteString("data", processedData[:min(len(processedData), 512)])) // 限制日志输出大小
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON数据"})
		return
	}

	s.logger.Debug("成功解析节点指标数据",
		zap.String("node_id", nodeID),
		zap.Int("metrics_size", len(metrics)))

	// 添加接收时间戳 (Aggregator接收时间)
	metrics["aggregator_received_at"] = time.Now().Unix()

	// 将指标数据传递给处理器
	s.processor.ProcessMetrics(nodeID, metrics)

	// 立即返回成功响应给Agent
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// processIncomingData 处理来自Agent的数据：解密和解压缩
func (s *Server) processIncomingData(data []byte, isEncrypted, isCompressed bool) ([]byte, error) {
	processedData := data
	var err error

	// 步骤1：解密（如果需要且配置了）
	if isEncrypted {
		if !s.config.Security.Encryption.Enabled || s.encryptionSvc == nil {
			return nil, fmt.Errorf("收到加密数据，但聚合服务器未配置解密")
		}
		startDecrypt := time.Now()
		processedData, err = s.encryptionSvc.Decrypt(processedData, s.config.Security.Encryption.Key)
		if err != nil {
			return nil, fmt.Errorf("解密失败: %w", err)
		}
		s.logger.Debug("数据解密完成", zap.Duration("duration", time.Since(startDecrypt)))
	}

	// 步骤2：解压缩（如果需要且配置了）
	if isCompressed {
		if !s.config.Security.Compression.Enabled {
			// 注意：Agent 默认启用压缩，如果 Aggregator 未配置解压，这里需要处理
			s.logger.Warn("收到压缩数据，但聚合服务器未显式启用解压缩，将尝试解压")
			// return nil, fmt.Errorf("收到压缩数据，但聚合服务器未配置解压缩")
		}
		startDecompress := time.Now()
		processedData, err = utils.DecompressData(processedData)
		if err != nil {
			return nil, fmt.Errorf("解压缩失败: %w", err)
		}
		s.logger.Debug("数据解压缩完成", zap.Duration("duration", time.Since(startDecompress)))
	}

	return processedData, nil
}

// handleNodeRegister 处理节点注册
func (s *Server) handleNodeRegister(c *gin.Context) {
	// 解析请求体
	var req struct {
		NodeID string `json:"node_id" binding:"required"`
		Token  string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("解析注册请求失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体: " + err.Error()})
		return
	}

	s.logger.Info("收到节点注册请求", zap.String("node_id", req.NodeID))

	// 验证节点 (调用主控端)
	startValidation := time.Now()
	if err := s.controlPlane.ValidateNode(req.NodeID, req.Token); err != nil {
		s.logger.Error("节点验证失败 (调用主控端)",
			zap.String("node_id", req.NodeID),
			zap.Duration("duration", time.Since(startValidation)),
			zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "节点验证失败: " + err.Error()})
		return
	}
	s.logger.Info("节点验证成功 (调用主控端)",
		zap.String("node_id", req.NodeID),
		zap.Duration("duration", time.Since(startValidation)))

	// 注册节点到聚合服务器内部管理
	s.registerOrUpdateNode(req.NodeID, true) // 标记为已验证

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"node_id": req.NodeID,
	})
}

// handleNodeHeartbeat 处理节点心跳
func (s *Server) handleNodeHeartbeat(c *gin.Context) {
	nodeID := c.Param("node_id")
	// 节点ID已在中间件中检查

	s.logger.Debug("收到节点心跳", zap.String("node_id", nodeID))

	// 更新节点活动时间
	success := s.updateNodeActivity(nodeID)
	if !success {
		// 如果节点不存在，心跳请求也认为是有效的，自动注册（但未验证）
		s.logger.Info("收到未知节点的心跳，自动注册（未验证）", zap.String("node_id", nodeID))
		s.registerOrUpdateNode(nodeID, false) // 注册但标记为未验证
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleGetNodes 处理获取节点列表
func (s *Server) handleGetNodes(c *gin.Context) {
	s.connections.RLock()
	defer s.connections.RUnlock()

	nodes := make([]map[string]interface{}, 0, len(s.connections.nodes))
	for nodeID, conn := range s.connections.nodes {
		nodes = append(nodes, map[string]interface{}{
			"node_id":      nodeID,
			"status":       conn.Status,
			"verified":     conn.Verified,
			"connected_at": conn.ConnectedAt.Format(time.RFC3339),
			"last_active":  conn.LastActive.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"nodes":  nodes,
	})
}

// registerOrUpdateNode 注册或更新节点信息
func (s *Server) registerOrUpdateNode(nodeID string, verified bool) {
	s.connections.Lock()
	defer s.connections.Unlock()

	now := time.Now()
	if conn, ok := s.connections.nodes[nodeID]; ok {
		// 更新现有节点
		conn.LastActive = now
		conn.Status = "active"
		if verified {
			conn.Verified = true // 更新验证状态
		}
		s.logger.Info("更新节点信息",
			zap.String("node_id", nodeID),
			zap.Bool("verified", conn.Verified),
			zap.String("status", conn.Status))
	} else {
		// 注册新节点
		s.connections.nodes[nodeID] = &NodeConnection{
			NodeID:      nodeID,
			Status:      "connected",
			Verified:    verified,
			ConnectedAt: now,
			LastActive:  now,
		}
		s.logger.Info("节点已注册",
			zap.String("node_id", nodeID),
			zap.Bool("verified", verified))
	}
}

// updateNodeActivity 更新节点活动时间
// 返回值表示节点是否存在
func (s *Server) updateNodeActivity(nodeID string) bool {
	s.connections.Lock()
	defer s.connections.Unlock()

	if conn, ok := s.connections.nodes[nodeID]; ok {
		conn.LastActive = time.Now()
		conn.Status = "active"
		return true
	}
	return false
}

// min 返回两个整数中较小的那个
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
