package aggregator

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

	// 初始化路由
	s.initRouter()

	// 初始化HTTP服务器
	s.httpServer = &http.Server{
		Addr:         cfg.Server.ListenAddr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.Server.ConnectionTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.ConnectionTimeout) * time.Second,
	}

	// 初始化数据处理器
	s.processor = NewDataProcessor(cfg)

	// 初始化控制平面客户端
	s.controlPlane = NewControlPlaneClient(cfg)

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
	logger, err := config.Build()
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

	// 使用日志中间件
	s.router.Use(gin.Recovery())

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
		// 节点指标上报
		api.POST("/nodes/:node_id/metrics", s.handleNodeMetrics)

		// 节点注册
		api.POST("/nodes/register", s.handleNodeRegister)

		// 节点心跳
		api.POST("/nodes/:node_id/heartbeat", s.handleNodeHeartbeat)

		// 获取节点列表
		api.GET("/nodes", s.handleGetNodes)
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

	// 关闭HTTP服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("关闭HTTP服务器失败", zap.Error(err))
	}

	// 关闭数据处理器
	if err := s.processor.Shutdown(); err != nil {
		s.logger.Error("关闭数据处理器失败", zap.Error(err))
	}

	// 关闭控制平面客户端
	if err := s.controlPlane.Shutdown(); err != nil {
		s.logger.Error("关闭控制平面客户端失败", zap.Error(err))
	}

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		s.logger.Info("聚合服务器已关闭")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("关闭超时: %w", ctx.Err())
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
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "节点ID不能为空"})
		return
	}

	// 更新节点活动时间
	s.updateNodeActivity(nodeID)

	// 解析请求体
	var metrics map[string]interface{}
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 处理指标数据
	s.processor.ProcessMetrics(nodeID, metrics)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleNodeRegister 处理节点注册
func (s *Server) handleNodeRegister(c *gin.Context) {
	// 解析请求体
	var req struct {
		NodeID string `json:"node_id" binding:"required"`
		Token  string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	// 验证节点
	if err := s.controlPlane.ValidateNode(req.NodeID, req.Token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "节点验证失败"})
		return
	}

	// 注册节点
	s.registerNode(req.NodeID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"node_id": req.NodeID,
	})
}

// handleNodeHeartbeat 处理节点心跳
func (s *Server) handleNodeHeartbeat(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "节点ID不能为空"})
		return
	}

	// 更新节点活动时间
	s.updateNodeActivity(nodeID)

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
			"connected_at": conn.ConnectedAt.Format(time.RFC3339),
			"last_active":  conn.LastActive.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"nodes":  nodes,
	})
}

// registerNode 注册节点
func (s *Server) registerNode(nodeID string) {
	s.connections.Lock()
	defer s.connections.Unlock()

	now := time.Now()
	s.connections.nodes[nodeID] = &NodeConnection{
		NodeID:      nodeID,
		Status:      "connected",
		ConnectedAt: now,
		LastActive:  now,
	}

	s.logger.Info("节点已注册", zap.String("node_id", nodeID))
}

// updateNodeActivity 更新节点活动时间
func (s *Server) updateNodeActivity(nodeID string) {
	s.connections.Lock()
	defer s.connections.Unlock()

	if conn, ok := s.connections.nodes[nodeID]; ok {
		conn.LastActive = time.Now()
		conn.Status = "active"
	} else {
		// 如果节点未注册，自动注册
		s.registerNode(nodeID)
	}
}
