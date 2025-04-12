package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从请求中获取节点ID
	nodeID := r.Header.Get("X-Node-ID")
	if nodeID == "" {
		http.Error(w, "Missing node ID", http.StatusBadRequest)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// 检查是否需要解密和解压缩
	isEncrypted := r.Header.Get("X-Encrypted") == "true"
	isCompressed := r.Header.Get("X-Compressed") == "gzip"

	// 处理数据
	processedData, err := h.processData(body, isEncrypted, isCompressed)
	if err != nil {
		log.Printf("数据处理失败: %v", err)
		http.Error(w, "Failed to process data", http.StatusBadRequest)
		return
	}

	// 解析处理后的数据
	var metrics map[string]interface{}
	if err := json.Unmarshal(processedData, &metrics); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 添加接收时间戳
	metrics["received_at"] = time.Now().Unix()

	// 存储指标数据
	if err := h.storage.StoreMetrics(nodeID, metrics); err != nil {
		log.Printf("Failed to store metrics: %v", err)
		http.Error(w, "Failed to store metrics", http.StatusInternalServerError)
		return
	}

	// 返回成功
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
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
