package storage

import (
	"sync"
	"time"
)

// MemoryStorage 提供基于内存的指标存储实现
type MemoryStorage struct {
	// 存储结构：map[nodeID][]MetricsEntry
	data     map[string][]MetricsEntry
	mutex    sync.RWMutex
	maxItems int // 每个节点最大存储条数
}

// MetricsEntry 表示一条指标记录
type MetricsEntry struct {
	Timestamp time.Time
	Data      interface{}
}

// NewMemoryStorage 创建一个新的内存存储实例
func NewMemoryStorage(maxItems int) *MemoryStorage {
	if maxItems <= 0 {
		maxItems = 1000 // 默认每个节点最多存储1000条记录
	}

	return &MemoryStorage{
		data:     make(map[string][]MetricsEntry),
		maxItems: maxItems,
	}
}

// StoreMetrics 存储节点指标数据
func (s *MemoryStorage) StoreMetrics(nodeID string, metrics interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 创建条目
	entry := MetricsEntry{
		Timestamp: time.Now(),
		Data:      metrics,
	}

	// 如果节点不存在，初始化切片
	if _, exists := s.data[nodeID]; !exists {
		s.data[nodeID] = []MetricsEntry{}
	}

	// 添加新条目
	s.data[nodeID] = append(s.data[nodeID], entry)

	// 如果超出最大存储量，移除最旧的条目
	if len(s.data[nodeID]) > s.maxItems {
		s.data[nodeID] = s.data[nodeID][len(s.data[nodeID])-s.maxItems:]
	}

	return nil
}

// GetNodeMetrics 获取指定节点在时间范围内的指标
func (s *MemoryStorage) GetNodeMetrics(nodeID string, start, end time.Time) ([]interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if entries, exists := s.data[nodeID]; exists {
		var result []interface{}
		for _, entry := range entries {
			if (entry.Timestamp.Equal(start) || entry.Timestamp.After(start)) &&
				(entry.Timestamp.Equal(end) || entry.Timestamp.Before(end)) {
				result = append(result, entry.Data)
			}
		}
		return result, nil
	}

	return []interface{}{}, nil // 返回空结果而不是错误
}

// GetAllNodes 获取所有节点ID列表
func (s *MemoryStorage) GetAllNodes() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	nodes := make([]string, 0, len(s.data))
	for node := range s.data {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// GetLatestMetrics 获取指定节点的最新指标
func (s *MemoryStorage) GetLatestMetrics(nodeID string) (interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if entries, exists := s.data[nodeID]; exists && len(entries) > 0 {
		// 返回最新的条目
		return entries[len(entries)-1].Data, nil
	}

	return nil, nil // 如果没有数据，返回nil
}
