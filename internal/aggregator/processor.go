package aggregator

import (
	"context"
	"fmt"
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
	p.ctx = ctx

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
	p.metrics.RLock()
	defer p.metrics.RUnlock()

	// 处理每个节点的指标数据
	for nodeID, metrics := range p.metrics.data {
		// 这里可以添加指标数据的处理逻辑
		// 例如：数据聚合、告警检测等

		p.logger.Debug("处理节点指标数据",
			zap.String("node_id", nodeID),
			zap.Any("metrics", metrics))
	}
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
