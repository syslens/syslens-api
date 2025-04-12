package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// MetricsHandler 处理指标相关的API请求
type MetricsHandler struct {
	storage MetricsStorage // 指标存储接口
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

	// 解析请求体
	var metrics map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metrics); err != nil {
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

// HandleGetNodeMetrics 处理获取节点指标的请求
func (h *MetricsHandler) HandleGetNodeMetrics(w http.ResponseWriter, r *http.Request) {
	// 只接受GET请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取URL参数
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, "Missing node ID", http.StatusBadRequest)
		return
	}

	// 解析时间范围参数
	startTime := time.Now().Add(-1 * time.Hour) // 默认过去1小时
	endTime := time.Now()

	// 如果提供了开始时间参数
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}

	// 如果提供了结束时间参数
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	// 查询数据
	metrics, err := h.storage.GetNodeMetrics(nodeID, startTime, endTime)
	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
		http.Error(w, "Failed to retrieve metrics", http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id": nodeID,
		"start":   startTime.Format(time.RFC3339),
		"end":     endTime.Format(time.RFC3339),
		"metrics": metrics,
	})
}

// HandleGetAllNodes 处理获取所有节点列表的请求
func (h *MetricsHandler) HandleGetAllNodes(w http.ResponseWriter, r *http.Request) {
	// 只接受GET请求
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取所有节点
	nodes, err := h.storage.GetAllNodes()
	if err != nil {
		log.Printf("Failed to get nodes: %v", err)
		http.Error(w, "Failed to retrieve nodes", http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
	})
}
