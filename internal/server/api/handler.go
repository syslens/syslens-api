package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/syslens/syslens-api/internal/common/utils"
	"github.com/syslens/syslens-api/internal/config"
)

// MetricsHandler 处理指标相关的API请求
type MetricsHandler struct {
	storage        MetricsStorage           // 指标存储接口
	securityConfig *config.SecurityConfig   // 安全配置
	encryptionSvc  *utils.EncryptionService // 加密服务
}

// MetricsStorage 定义了指标存储接口
type MetricsStorage interface {
	StoreMetrics(nodeID string, metrics interface{}) error
	GetNodeMetrics(nodeID string, start, end time.Time) ([]interface{}, error)
	GetAllNodes() ([]string, error)
	GetLatestMetrics(nodeID string) (interface{}, error)
}

// NewMetricsHandler 创建新的指标处理器
func NewMetricsHandler(storage MetricsStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
		securityConfig: &config.SecurityConfig{
			Encryption: config.EncryptionConfig{
				Enabled:   false,
				Algorithm: "aes-256-gcm",
				Key:       "",
			},
			Compression: config.CompressionConfig{
				Enabled:   false,
				Algorithm: "gzip",
			},
		},
	}
}

// WithSecurityConfig 设置安全配置
func (h *MetricsHandler) WithSecurityConfig(secConfig *config.SecurityConfig) {
	if secConfig != nil {
		h.securityConfig = secConfig
		// 初始化加密服务
		if h.securityConfig.Encryption.Enabled {
			h.encryptionSvc = utils.NewEncryptionService(h.securityConfig.Encryption.Algorithm)
		}
	}
}

// HandleMetricsSubmit 处理节点上报的指标数据
func (h *MetricsHandler) HandleMetricsSubmit(w http.ResponseWriter, r *http.Request) {
	// 只接受POST请求
	if r.Method != http.MethodPost {
		log.Printf("[错误] 不支持的HTTP方法: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从请求中获取节点ID
	nodeID := r.Header.Get("X-Node-ID")
	if nodeID == "" {
		log.Printf("[错误] 缺少节点ID头部")
		http.Error(w, "Missing node ID", http.StatusBadRequest)
		return
	}

	// 获取请求发送方IP和聚合服务器ID（如果有）
	remoteIP := r.RemoteAddr
	aggregatorID := r.Header.Get("X-Aggregator-ID")
	source := "直接节点上报"
	if aggregatorID != "" {
		source = fmt.Sprintf("聚合服务器(%s)", aggregatorID)
	}

	log.Printf("[信息] 接收到节点指标上报请求 - 节点ID: %s, 来源: %s, IP: %s", nodeID, source, remoteIP)

	// 检查授权令牌
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		log.Printf("[调试] 授权头部存在: %s", strings.Replace(authHeader, authHeader[10:], "****", 1))
	} else {
		log.Printf("[警告] 节点指标上报缺少授权令牌 - 节点ID: %s", nodeID)
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[错误] 读取请求体失败 - 节点ID: %s, 错误: %v", nodeID, err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	log.Printf("[调试] 接收到原始请求体 - 节点ID: %s, 大小: %d字节", nodeID, len(body))

	// 检查是否需要解密和解压缩
	isEncrypted := r.Header.Get("X-Encrypted") == "true"
	isCompressed := r.Header.Get("X-Compressed") == "gzip"
	log.Printf("[调试] 数据处理标记 - 节点ID: %s, 加密: %v, 压缩: %v", nodeID, isEncrypted, isCompressed)

	// 处理数据
	startProcessing := time.Now()
	processedData, err := h.processData(body, isEncrypted, isCompressed)
	processingTime := time.Since(startProcessing)

	if err != nil {
		log.Printf("[错误] 数据处理失败 - 节点ID: %s, 耗时: %v, 错误: %v", nodeID, processingTime, err)
		http.Error(w, "Failed to process data", http.StatusBadRequest)
		return
	}
	log.Printf("[信息] 数据处理完成 - 节点ID: %s, 耗时: %v, 处理前大小: %d字节, 处理后大小: %d字节",
		nodeID, processingTime, len(body), len(processedData))

	// 解析处理后的数据
	var metrics map[string]interface{}
	if err := json.Unmarshal(processedData, &metrics); err != nil {
		log.Printf("[错误] JSON解析失败 - 节点ID: %s, 错误: %v", nodeID, err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	log.Printf("[信息] 成功解析JSON数据 - 节点ID: %s, 指标数量: %d", nodeID, len(metrics))

	// 添加接收时间戳
	receivedAt := time.Now().Unix()
	metrics["received_at"] = receivedAt
	log.Printf("[调试] 添加接收时间戳 - 节点ID: %s, 时间戳: %d", nodeID, receivedAt)

	// 记录关键指标（如果存在）
	if cpu, ok := metrics["cpu"].(map[string]interface{}); ok {
		if usage, ok := cpu["usage"]; ok {
			log.Printf("[调试] CPU使用率 - 节点ID: %s, 使用率: %v", nodeID, usage)
		}
	}
	if memory, ok := metrics["memory"].(map[string]interface{}); ok {
		if used, ok := memory["used_percent"]; ok {
			log.Printf("[调试] 内存使用率 - 节点ID: %s, 使用率: %v", nodeID, used)
		}
	}

	// 存储指标数据
	startStoring := time.Now()
	if err := h.storage.StoreMetrics(nodeID, metrics); err != nil {
		log.Printf("[错误] 存储指标数据失败 - 节点ID: %s, 错误: %v", nodeID, err)
		http.Error(w, "Failed to store metrics", http.StatusInternalServerError)
		return
	}
	storingTime := time.Since(startStoring)
	log.Printf("[信息] 指标数据存储成功 - 节点ID: %s, 耗时: %v", nodeID, storingTime)

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	}); err != nil {
		log.Printf("[警告] 写入响应失败 - 节点ID: %s, 错误: %v", nodeID, err)
	}

	// 记录整体处理时间
	totalTime := time.Since(startProcessing)
	log.Printf("[信息] 指标上报处理完成 - 节点ID: %s, 总耗时: %v", nodeID, totalTime)
}

// processData 处理数据：解密和解压缩
func (h *MetricsHandler) processData(data []byte, isEncrypted, isCompressed bool) ([]byte, error) {
	processedData := data
	var err error

	// 步骤1：解密（如果启用）
	if isEncrypted && h.securityConfig.Encryption.Enabled && h.encryptionSvc != nil {
		processedData, err = h.encryptionSvc.Decrypt(processedData, h.securityConfig.Encryption.Key)
		if err != nil {
			return nil, err
		}
	}

	// 步骤2：解压缩（如果启用）
	if isCompressed && h.securityConfig.Compression.Enabled {
		processedData, err = utils.DecompressData(processedData)
		if err != nil {
			return nil, err
		}
	}

	return processedData, nil
}

// HandleGetNodeMetrics 处理获取节点指标的请求
func (h *MetricsHandler) HandleGetNodeMetrics(w http.ResponseWriter, r *http.Request) {
	// 只接受GET请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取节点ID
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, "Missing node ID", http.StatusBadRequest)
		return
	}

	// 解析时间范围
	startTimeStr := r.URL.Query().Get("start")
	endTimeStr := r.URL.Query().Get("end")

	var startTime, endTime time.Time
	var err error

	// 如果未提供时间范围，使用过去1小时
	if startTimeStr == "" {
		startTime = time.Now().Add(-1 * time.Hour)
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			http.Error(w, "Invalid start time format", http.StatusBadRequest)
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			http.Error(w, "Invalid end time format", http.StatusBadRequest)
			return
		}
	}

	// 查询指标数据
	metrics, err := h.storage.GetNodeMetrics(nodeID, startTime, endTime)
	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	// 返回指标数据
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"metrics": metrics,
	})
}

// HandleGetAllNodes 处理获取所有节点的请求
func (h *MetricsHandler) HandleGetAllNodes(w http.ResponseWriter, r *http.Request) {
	// 只接受GET请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取所有节点ID
	nodes, err := h.storage.GetAllNodes()
	if err != nil {
		log.Printf("Failed to get nodes: %v", err)
		http.Error(w, "Failed to get nodes", http.StatusInternalServerError)
		return
	}

	// 返回节点列表
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"nodes":  nodes,
	})
}

// HandleGetNodeLatest 处理获取节点最新指标的请求
func (h *MetricsHandler) HandleGetNodeLatest(w http.ResponseWriter, r *http.Request) {
	// 只接受GET请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取节点ID
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, "Missing node ID", http.StatusBadRequest)
		return
	}

	// 获取最新指标
	metrics, err := h.storage.GetLatestMetrics(nodeID)
	if err != nil {
		log.Printf("Failed to get latest metrics: %v", err)
		http.Error(w, "Failed to get latest metrics", http.StatusInternalServerError)
		return
	}

	// 返回最新指标
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"metrics": metrics,
	})
}
