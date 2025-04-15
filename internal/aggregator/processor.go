package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/syslens/syslens-api/internal/config"
	"go.uber.org/zap"
)

// DataProcessor 数据处理器
type DataProcessor struct {
	// 配置
	config *config.AggregatorConfig

	// 日志记录器
	logger *zap.Logger

	// 指标缓存
	metrics struct {
		sync.RWMutex
		// 节点ID -> 指标数据
		data map[string]map[string]interface{}
	}

	// 上下文和取消函数
	ctx    context.Context
	cancel context.CancelFunc

	// 等待组，用于等待所有goroutine完成
	wg sync.WaitGroup
}

// NewDataProcessor 创建新的数据处理器
func NewDataProcessor(cfg *config.AggregatorConfig) *DataProcessor {
	p := &DataProcessor{
		config: cfg,
	}

	// 初始化指标缓存
	p.metrics.data = make(map[string]map[string]interface{})

	// 创建日志记录器
	p.logger = zap.NewNop() // 使用空日志记录器，实际日志由服务器管理

	return p
}

// Start 启动数据处理器
func (p *DataProcessor) Start(ctx context.Context) error {
	// 创建一个可取消的上下文
	p.ctx, p.cancel = context.WithCancel(ctx)

	p.logger.Debug("启动数据处理器")

	// 启动指标处理goroutine
	p.wg.Add(1)
	go p.processMetrics()

	return nil
}

// Shutdown 关闭数据处理器
func (p *DataProcessor) Shutdown() error {
	if p.cancel != nil {
		p.cancel()
	}

	// 等待所有goroutine完成
	p.wg.Wait()

	return nil
}

// ProcessMetrics 处理节点指标
func (p *DataProcessor) ProcessMetrics(nodeID string, metrics map[string]interface{}) {
	p.metrics.Lock()
	defer p.metrics.Unlock()

	// 更新指标数据
	p.metrics.data[nodeID] = metrics

	p.logger.Debug("处理节点指标",
		zap.String("node_id", nodeID),
		zap.Any("metrics", metrics))
}

// processMetrics 处理指标数据
func (p *DataProcessor) processMetrics() {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Duration(p.config.Processing.BatchInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.processMetricsData()
		}
	}
}

// processMetricsData 处理指标数据
func (p *DataProcessor) processMetricsData() {
	// 复制当前数据，避免长时间持有锁
	p.metrics.Lock()
	metricsSnapshot := make(map[string]map[string]interface{})
	for nodeID, data := range p.metrics.data {
		// 深拷贝指标数据
		metricsCopy := make(map[string]interface{})
		for k, v := range data {
			metricsCopy[k] = v
		}
		metricsSnapshot[nodeID] = metricsCopy
	}
	p.metrics.Unlock()

	// 处理每个节点的指标数据
	for nodeID, metrics := range metricsSnapshot {
		// 添加处理时间戳
		metrics["processed_at"] = time.Now().Unix()

		// 转发数据到主控平面 - 这里应该调用controlPlaneClient的方法
		if err := p.forwardMetricsToControlPlane(nodeID, metrics); err != nil {
			p.logger.Error("转发指标数据到主控平面失败",
				zap.String("node_id", nodeID),
				zap.Error(err))
		} else {
			p.logger.Debug("成功转发指标数据到主控平面",
				zap.String("node_id", nodeID))
		}
	}
}

// forwardMetricsToControlPlane 将指标数据转发到主控平面
func (p *DataProcessor) forwardMetricsToControlPlane(nodeID string, metrics map[string]interface{}) error {
	// 创建一个子上下文，设置5秒超时
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/nodes/%s/metrics", p.config.ControlPlane.URL, nodeID)
	p.logger.Info("准备向主控平面转发指标数据",
		zap.String("node_id", nodeID),
		zap.String("url", url),
		zap.Int("metrics_count", len(metrics)))

	// 构建请求体
	body, err := json.Marshal(metrics)
	if err != nil {
		p.logger.Error("序列化指标数据失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return fmt.Errorf("序列化指标数据失败: %v", err)
	}
	p.logger.Debug("序列化的请求体大小",
		zap.String("node_id", nodeID),
		zap.Int("body_size_bytes", len(body)))

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		p.logger.Error("创建HTTP请求失败",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.ControlPlane.Token))
	req.Header.Set("X-Node-ID", nodeID)
	req.Header.Set("X-Aggregator-ID", "aggregator-1") // 可以设置聚合服务器的ID
	p.logger.Debug("HTTP请求头设置完成",
		zap.String("node_id", nodeID),
		zap.Strings("headers", []string{
			"Content-Type: application/json",
			"Authorization: Bearer ****",
			"X-Node-ID: " + nodeID,
			"X-Aggregator-ID: aggregator-1",
		}))

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 5 * time.Second, // 降低超时时间
		Transport: &http.Transport{
			MaxIdleConns:          100,              // 最大空闲连接数
			IdleConnTimeout:       90 * time.Second, // 空闲连接超时时间
			TLSHandshakeTimeout:   5 * time.Second,  // TLS握手超时
			ExpectContinueTimeout: 1 * time.Second,  // Expect: 100-continue超时
			DisableKeepAlives:     false,            // 启用连接复用
			MaxConnsPerHost:       10,               // 每个主机的最大连接数
		},
	}

	// 记录开始时间
	startTime := time.Now()
	p.logger.Info("开始发送HTTP请求到主控平面",
		zap.String("node_id", nodeID),
		zap.String("url", url),
		zap.Time("start_time", startTime))

	// 发送请求
	resp, err := client.Do(req)

	// 记录请求耗时
	requestTime := time.Since(startTime)
	p.logger.Info("HTTP请求完成",
		zap.String("node_id", nodeID),
		zap.Duration("elapsed", requestTime),
		zap.Error(err))

	if err != nil {
		p.logger.Error("向主控平面发送请求失败",
			zap.String("node_id", nodeID),
			zap.Duration("elapsed", requestTime),
			zap.Error(err))
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error("读取响应体失败",
			zap.String("node_id", nodeID),
			zap.Int("status_code", resp.StatusCode),
			zap.Error(err))
	} else {
		p.logger.Info("收到主控平面响应",
			zap.String("node_id", nodeID),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(respBody)))
	}

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logger.Error("主控平面返回错误状态码",
			zap.String("node_id", nodeID),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(respBody)))
		return fmt.Errorf("主控平面返回错误状态码: %d", resp.StatusCode)
	}

	p.logger.Info("成功向主控平面转发指标数据",
		zap.String("node_id", nodeID),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("total_time", requestTime))
	return nil
}

// GetNodeMetrics 获取节点指标
func (p *DataProcessor) GetNodeMetrics(nodeID string) (map[string]interface{}, error) {
	p.metrics.RLock()
	defer p.metrics.RUnlock()

	metrics, ok := p.metrics.data[nodeID]
	if !ok {
		return nil, fmt.Errorf("节点 %s 的指标数据不存在", nodeID)
	}

	return metrics, nil
}

// GetAllNodesMetrics 获取所有节点的指标
func (p *DataProcessor) GetAllNodesMetrics() map[string]map[string]interface{} {
	p.metrics.RLock()
	defer p.metrics.RUnlock()

	// 创建副本
	metrics := make(map[string]map[string]interface{})
	for nodeID, data := range p.metrics.data {
		metrics[nodeID] = data
	}

	return metrics
}
