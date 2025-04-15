package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/syslens/syslens-api/internal/config"
	"github.com/syslens/syslens-api/internal/server/api"
	"github.com/syslens/syslens-api/internal/server/storage"
	"go.uber.org/zap"
)

// ControlPlaneServer 控制平面服务器
type ControlPlaneServer struct {
	// 配置
	config *config.ServerConfig

	// HTTP服务器
	httpServer *http.Server

	// 路由
	router *http.ServeMux

	// 存储
	storage api.MetricsStorage

	// 日志记录器
	logger *zap.Logger

	// 节点管理
	nodes struct {
		sync.RWMutex
		// 节点ID -> 节点信息
		data map[string]*NodeInfo
	}

	// 节点分组
	groups struct {
		sync.RWMutex
		// 分组ID -> 分组信息
		data map[string]*GroupInfo
	}

	// 固定服务
	services struct {
		sync.RWMutex
		// 服务ID -> 服务信息
		data map[string]*ServiceInfo
	}

	// 上下文和取消函数
	ctx    context.Context
	cancel context.CancelFunc

	// 等待组，用于等待所有goroutine完成
	wg sync.WaitGroup

	// WebSocket连接管理
	connections struct {
		sync.RWMutex
		// 节点ID -> WebSocket连接
		data map[string]*websocket.Conn
	}

	// WebSocket升级器
	upgrader websocket.Upgrader
}

// NodeInfo 节点信息
type NodeInfo struct {
	// 节点ID
	ID string

	// 节点标签
	Labels map[string]string

	// 节点类型（固定服务节点或非固定节点）
	Type string

	// 节点状态
	Status string

	// 最后活动时间
	LastActive time.Time

	// 注册时间
	RegisteredAt time.Time

	// 所属分组ID
	GroupID string

	// 关联的服务ID（如果是固定服务节点）
	ServiceID string
}

// GroupInfo 分组信息
type GroupInfo struct {
	// 分组ID
	ID string

	// 分组名称
	Name string

	// 分组描述
	Description string

	// 分组类型（地区、功能、环境等）
	Type string

	// 创建时间
	CreatedAt time.Time

	// 节点ID列表
	NodeIDs []string
}

// ServiceInfo 固定服务信息
type ServiceInfo struct {
	// 服务ID
	ID string

	// 服务名称
	Name string

	// 服务描述
	Description string

	// 服务类型
	Type string

	// 创建时间
	CreatedAt time.Time

	// 节点ID列表（按优先级排序）
	NodeIDs []string

	// 节点优先级映射
	NodePriorities map[string]int
}

// NewControlPlaneServer 创建新的控制平面服务器
func NewControlPlaneServer(cfg *config.ServerConfig, logger *zap.Logger) (*ControlPlaneServer, error) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建服务器
	s := &ControlPlaneServer{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 在生产环境中应该检查来源
			},
		},
	}

	// 初始化节点管理
	s.nodes.data = make(map[string]*NodeInfo)

	// 初始化分组管理
	s.groups.data = make(map[string]*GroupInfo)

	// 初始化服务管理
	s.services.data = make(map[string]*ServiceInfo)

	// 初始化日志记录器
	if err := s.initLogger(); err != nil {
		return nil, fmt.Errorf("初始化日志记录器失败: %w", err)
	}

	// 初始化存储
	if err := s.initStorage(); err != nil {
		return nil, fmt.Errorf("初始化存储失败: %w", err)
	}

	// 初始化路由
	s.initRouter()

	// 初始化HTTP服务器
	s.httpServer = &http.Server{
		Addr:         cfg.Server.HTTPAddr,
		Handler:      s.router,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}

	// 初始化WebSocket连接管理
	s.connections.data = make(map[string]*websocket.Conn)

	return s, nil
}

// initLogger 初始化日志记录器
func (s *ControlPlaneServer) initLogger() error {
	// 创建日志配置
	config := zap.NewProductionConfig()

	// 设置日志级别
	switch s.config.Logging.Level {
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
	if s.config.Logging.File != "" {
		config.OutputPaths = []string{s.config.Logging.File}
	}

	// 创建日志记录器
	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}

	s.logger = logger
	return nil
}

// initStorage 初始化存储
func (s *ControlPlaneServer) initStorage() error {
	switch s.config.Storage.Type {
	case "influxdb":
		// 初始化InfluxDB存储
		influxStorage := storage.NewInfluxDBStorage(
			s.config.Storage.InfluxDB.URL,
			s.config.Storage.InfluxDB.Token,
			s.config.Storage.InfluxDB.Org,
			s.config.Storage.InfluxDB.Bucket,
		)
		s.storage = influxStorage
		s.logger.Info("已初始化InfluxDB存储")
	case "memory":
		fallthrough
	default:
		// 初始化内存存储
		maxItems := 1000
		if s.config.Storage.Memory.MaxItems > 0 {
			maxItems = s.config.Storage.Memory.MaxItems
		}
		memoryStorage := storage.NewMemoryStorage(maxItems)
		s.storage = memoryStorage
		s.logger.Info("已初始化内存存储")
	}

	return nil
}

// initRouter 初始化路由
func (s *ControlPlaneServer) initRouter() {
	// 创建路由
	s.router = http.NewServeMux()

	// 创建指标处理器
	metricsHandler := api.NewMetricsHandler(s.storage)

	// 应用安全配置
	metricsHandler.WithSecurityConfig(&s.config.Security)

	// 设置路由
	apiRouter := api.SetupRoutes(metricsHandler)

	// 添加节点管理API
	s.router.HandleFunc("/api/v1/nodes/register", s.handleNodeRegister)

	// 将指标处理API委托给apiRouter
	s.router.Handle("/api/v1/nodes/", apiRouter)

	// 其他管理API
	s.router.HandleFunc("/api/v1/groups", s.handleGroupOperations)
	s.router.HandleFunc("/api/v1/services", s.handleServiceOperations)
	s.router.HandleFunc("/api/v1/ws/nodes", s.handleWebSocket)

	// 健康检查
	s.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().Format(time.RFC3339))
	})

	// 记录所有注册的路由，帮助调试
	s.logger.Info("已注册API路由",
		zap.String("/api/v1/nodes/{nodeID}/metrics", "节点指标上报"),
		zap.String("/api/v1/nodes/metrics", "获取节点指标"),
		zap.String("/api/v1/nodes", "获取所有节点"),
		zap.String("/api/v1/nodes/register", "节点注册"),
		zap.String("/api/v1/groups", "节点分组管理"),
		zap.String("/api/v1/services", "服务管理"),
		zap.String("/api/v1/ws/nodes", "WebSocket连接"),
		zap.String("/health", "健康检查"))
}

// Start 启动服务器
func (s *ControlPlaneServer) Start() error {
	s.logger.Info("启动控制平面服务器",
		zap.String("listen_addr", s.config.Server.HTTPAddr))

	// 启动HTTP服务器
	go func() {
		var err error
		if s.config.Server.UseHTTPS && s.config.Server.CertFile != "" && s.config.Server.KeyFile != "" {
			err = s.httpServer.ListenAndServeTLS(s.config.Server.CertFile, s.config.Server.KeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP服务器错误", zap.Error(err))
		}
	}()

	// 启动节点清理goroutine
	s.wg.Add(1)
	go s.cleanupExpiredNodes()

	s.logger.Info("控制平面服务器已启动")
	return nil
}

// Shutdown 关闭服务器
func (s *ControlPlaneServer) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭控制平面服务器...")

	// 取消上下文，通知所有goroutine退出
	s.cancel()

	// 关闭HTTP服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("关闭HTTP服务器失败", zap.Error(err))
	}

	// 关闭存储连接
	if influxStorage, ok := s.storage.(*storage.InfluxDBStorage); ok {
		influxStorage.Close()
		s.logger.Info("InfluxDB连接已关闭")
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
		s.logger.Info("控制平面服务器已关闭")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("关闭超时: %w", ctx.Err())
	}
}

// cleanupExpiredNodes 清理过期节点
func (s *ControlPlaneServer) cleanupExpiredNodes() {
	defer s.wg.Done()

	// 如果未启用自动清理，则退出
	if !s.config.Discovery.AutoRemoveExpired {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupNodes()
		}
	}
}

// cleanupNodes 清理过期节点
func (s *ControlPlaneServer) cleanupNodes() {
	s.nodes.Lock()
	defer s.nodes.Unlock()

	now := time.Now()
	expiry := time.Duration(s.config.Discovery.NodeExpiry) * time.Minute

	for nodeID, node := range s.nodes.data {
		if now.Sub(node.LastActive) > expiry {
			s.logger.Info("清理过期节点", zap.String("node_id", nodeID))
			delete(s.nodes.data, nodeID)

			// 从分组中移除节点
			if node.GroupID != "" {
				s.removeNodeFromGroup(nodeID, node.GroupID)
			}

			// 从服务中移除节点
			if node.ServiceID != "" {
				s.removeNodeFromService(nodeID, node.ServiceID)
			}
		}
	}
}

// removeNodeFromGroup 从分组中移除节点
func (s *ControlPlaneServer) removeNodeFromGroup(nodeID, groupID string) {
	s.groups.Lock()
	defer s.groups.Unlock()

	if group, ok := s.groups.data[groupID]; ok {
		for i, id := range group.NodeIDs {
			if id == nodeID {
				group.NodeIDs = append(group.NodeIDs[:i], group.NodeIDs[i+1:]...)
				break
			}
		}
	}
}

// removeNodeFromService 从服务中移除节点
func (s *ControlPlaneServer) removeNodeFromService(nodeID, serviceID string) {
	s.services.Lock()
	defer s.services.Unlock()

	if service, ok := s.services.data[serviceID]; ok {
		for i, id := range service.NodeIDs {
			if id == nodeID {
				service.NodeIDs = append(service.NodeIDs[:i], service.NodeIDs[i+1:]...)
				delete(service.NodePriorities, nodeID)
				break
			}
		}
	}
}

// handleNodeRegister 处理节点注册
func (s *ControlPlaneServer) handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	// 只接受POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		NodeID string            `json:"node_id"`
		Labels map[string]string `json:"labels"`
		Type   string            `json:"type"`
		Token  string            `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证节点ID
	if req.NodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	// 验证节点类型
	if req.Type != "fixed" && req.Type != "non-fixed" {
		http.Error(w, "Invalid node type", http.StatusBadRequest)
		return
	}

	// 验证令牌（在实际应用中，应该使用更安全的认证机制）
	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusUnauthorized)
		return
	}

	// 注册节点
	s.nodes.Lock()
	defer s.nodes.Unlock()

	now := time.Now()
	s.nodes.data[req.NodeID] = &NodeInfo{
		ID:           req.NodeID,
		Labels:       req.Labels,
		Type:         req.Type,
		Status:       "active",
		LastActive:   now,
		RegisteredAt: now,
		GroupID:      "",
		ServiceID:    "",
	}

	s.logger.Info("节点已注册", zap.String("node_id", req.NodeID), zap.String("type", req.Type))

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"node_id": req.NodeID,
	})
}

// handleNodeOperations 处理节点操作
func (s *ControlPlaneServer) handleNodeOperations(w http.ResponseWriter, r *http.Request) {
	// 提取节点ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	nodeID := parts[3]

	// 根据HTTP方法处理不同的操作
	switch r.Method {
	case http.MethodGet:
		s.handleGetNode(w, r, nodeID)
	case http.MethodPut:
		s.handleUpdateNode(w, r, nodeID)
	case http.MethodDelete:
		s.handleDeleteNode(w, r, nodeID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetNode 处理获取节点信息
func (s *ControlPlaneServer) handleGetNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	s.nodes.RLock()
	defer s.nodes.RUnlock()

	node, ok := s.nodes.data[nodeID]
	if !ok {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"node":   node,
	})
}

// handleUpdateNode 处理更新节点信息
func (s *ControlPlaneServer) handleUpdateNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	s.nodes.Lock()
	defer s.nodes.Unlock()

	node, ok := s.nodes.data[nodeID]
	if !ok {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	// 解析请求体
	var req struct {
		Labels map[string]string `json:"labels"`
		Status string            `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 更新节点信息
	if req.Labels != nil {
		node.Labels = req.Labels
	}

	if req.Status != "" {
		node.Status = req.Status
	}

	node.LastActive = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// handleDeleteNode 处理删除节点
func (s *ControlPlaneServer) handleDeleteNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	s.nodes.Lock()
	defer s.nodes.Unlock()

	node, ok := s.nodes.data[nodeID]
	if !ok {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	// 从分组中移除节点
	if node.GroupID != "" {
		s.removeNodeFromGroup(nodeID, node.GroupID)
	}

	// 从服务中移除节点
	if node.ServiceID != "" {
		s.removeNodeFromService(nodeID, node.ServiceID)
	}

	// 删除节点
	delete(s.nodes.data, nodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// handleGroupOperations 处理分组操作
func (s *ControlPlaneServer) handleGroupOperations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetGroups(w, r)
	case http.MethodPost:
		s.handleCreateGroup(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetGroups 处理获取分组列表
func (s *ControlPlaneServer) handleGetGroups(w http.ResponseWriter, r *http.Request) {
	s.groups.RLock()
	defer s.groups.RUnlock()

	groups := make([]*GroupInfo, 0, len(s.groups.data))
	for _, group := range s.groups.data {
		groups = append(groups, group)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"groups": groups,
	})
}

// handleCreateGroup 处理创建分组
func (s *ControlPlaneServer) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证分组名称
	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// 生成分组ID
	groupID := fmt.Sprintf("group-%d", time.Now().UnixNano())

	// 创建分组
	s.groups.Lock()
	defer s.groups.Unlock()

	s.groups.data[groupID] = &GroupInfo{
		ID:          groupID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		CreatedAt:   time.Now(),
		NodeIDs:     []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"group":  s.groups.data[groupID],
	})
}

// handleServiceOperations 处理服务操作
func (s *ControlPlaneServer) handleServiceOperations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetServices(w, r)
	case http.MethodPost:
		s.handleCreateService(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetServices 处理获取服务列表
func (s *ControlPlaneServer) handleGetServices(w http.ResponseWriter, r *http.Request) {
	s.services.RLock()
	defer s.services.RUnlock()

	services := make([]*ServiceInfo, 0, len(s.services.data))
	for _, service := range s.services.data {
		services = append(services, service)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"services": services,
	})
}

// handleCreateService 处理创建服务
func (s *ControlPlaneServer) handleCreateService(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证服务名称
	if req.Name == "" {
		http.Error(w, "Service name is required", http.StatusBadRequest)
		return
	}

	// 生成服务ID
	serviceID := fmt.Sprintf("service-%d", time.Now().UnixNano())

	// 创建服务
	s.services.Lock()
	defer s.services.Unlock()

	s.services.data[serviceID] = &ServiceInfo{
		ID:             serviceID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		CreatedAt:      time.Now(),
		NodeIDs:        []string{},
		NodePriorities: make(map[string]int),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"service": s.services.data[serviceID],
	})
}

// handleWebSocket 处理WebSocket连接
func (s *ControlPlaneServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("升级WebSocket连接失败", zap.Error(err))
		return
	}

	// 获取节点ID
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		s.logger.Error("缺少节点ID")
		conn.Close()
		return
	}

	// 保存连接
	s.connections.Lock()
	s.connections.data[nodeID] = conn
	s.connections.Unlock()

	// 处理连接
	go s.handleWebSocketConnection(nodeID, conn)
}

// handleWebSocketConnection 处理WebSocket连接
func (s *ControlPlaneServer) handleWebSocketConnection(nodeID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		s.connections.Lock()
		delete(s.connections.data, nodeID)
		s.connections.Unlock()
	}()

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(time.Second * 30))

	for {
		// 读取消息
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("读取WebSocket消息失败", zap.Error(err))
			return
		}

		// 处理消息
		switch messageType {
		case websocket.TextMessage:
			s.processWebSocketData(nodeID, data)
		case websocket.PingMessage:
			// 发送pong响应
			if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				s.logger.Error("发送pong响应失败", zap.Error(err))
				return
			}
		}
	}
}

// processWebSocketData 处理WebSocket数据
func (s *ControlPlaneServer) processWebSocketData(nodeID string, data []byte) {
	// 解析数据
	var message map[string]interface{}
	if err := json.Unmarshal(data, &message); err != nil {
		s.logger.Error("解析WebSocket数据失败", zap.Error(err))
		return
	}

	// 处理不同类型的消息
	messageType, ok := message["type"].(string)
	if !ok {
		s.logger.Error("WebSocket消息类型无效")
		return
	}

	switch messageType {
	case "metrics":
		// 处理指标数据
		if metrics, ok := message["data"].(map[string]interface{}); ok {
			// 存储指标数据
			if err := s.storage.StoreMetrics(nodeID, metrics); err != nil {
				s.logger.Error("存储指标数据失败", zap.Error(err))
			}
		}
	case "command":
		// 处理命令
		if command, ok := message["command"].(string); ok {
			s.logger.Info("收到命令", zap.String("node_id", nodeID), zap.String("command", command))
			// 处理不同类型的命令
			switch command {
			case "get_config":
				// 获取节点配置
				s.sendNodeConfig(nodeID)
			default:
				s.logger.Warn("未知命令", zap.String("command", command))
			}
		}
	default:
		s.logger.Warn("未知消息类型", zap.String("type", messageType))
	}
}

// sendNodeConfig 发送节点配置
func (s *ControlPlaneServer) sendNodeConfig(nodeID string) {
	// 获取节点配置
	s.nodes.RLock()
	node, ok := s.nodes.data[nodeID]
	s.nodes.RUnlock()

	if !ok {
		s.logger.Error("节点不存在", zap.String("node_id", nodeID))
		return
	}

	// 构建配置
	config := map[string]interface{}{
		"node_id": node.ID,
		"labels":  node.Labels,
		"type":    node.Type,
	}

	// 如果节点属于分组，添加分组配置
	if node.GroupID != "" {
		s.groups.RLock()
		if group, ok := s.groups.data[node.GroupID]; ok {
			config["group"] = map[string]interface{}{
				"id":   group.ID,
				"name": group.Name,
				"type": group.Type,
			}
		}
		s.groups.RUnlock()
	}

	// 如果节点是固定服务节点，添加服务配置
	if node.ServiceID != "" {
		s.services.RLock()
		if service, ok := s.services.data[node.ServiceID]; ok {
			config["service"] = map[string]interface{}{
				"id":   service.ID,
				"name": service.Name,
				"type": service.Type,
			}
		}
		s.services.RUnlock()
	}

	// 构建消息
	message := map[string]interface{}{
		"type":    "config",
		"data":    config,
		"version": time.Now().Unix(),
	}

	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("序列化配置消息失败", zap.Error(err))
		return
	}

	// 获取WebSocket连接
	s.connections.RLock()
	conn, ok := s.connections.data[nodeID]
	s.connections.RUnlock()

	if !ok {
		s.logger.Error("节点WebSocket连接不存在", zap.String("node_id", nodeID))
		return
	}

	// 发送配置
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		s.logger.Error("发送节点配置失败", zap.Error(err))
		return
	}

	s.logger.Info("节点配置已发送", zap.String("node_id", nodeID))
}
