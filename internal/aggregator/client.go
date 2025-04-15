package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/syslens/syslens-api/internal/config"
	"go.uber.org/zap"
)

// ControlPlaneClient 控制平面客户端
type ControlPlaneClient struct {
	// 配置
	config *config.AggregatorConfig

	// HTTP客户端
	client *http.Client

	// 日志记录器
	logger *zap.Logger

	// 上下文和取消函数
	ctx    context.Context
	cancel context.CancelFunc
}

// NewControlPlaneClient 创建新的控制平面客户端
func NewControlPlaneClient(cfg *config.AggregatorConfig) *ControlPlaneClient {
	c := &ControlPlaneClient{
		config: cfg,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}

	// 创建日志记录器
	c.logger = zap.NewNop() // 使用空日志记录器，实际日志由服务器管理

	return c
}

// Start 启动控制平面客户端
func (c *ControlPlaneClient) Start(ctx context.Context) error {
	c.ctx = ctx
	return nil
}

// Shutdown 关闭控制平面客户端
func (c *ControlPlaneClient) Shutdown() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// RegisterNode 向控制平面注册节点
func (c *ControlPlaneClient) RegisterNode(nodeID string, info map[string]interface{}) error {
	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/nodes/%s", c.config.ControlPlane.URL, nodeID)

	// 构建请求体
	body, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("序列化节点信息失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.ControlPlane.Token))

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("注册节点失败，状态码: %d", resp.StatusCode)
	}

	c.logger.Info("节点注册成功",
		zap.String("node_id", nodeID),
		zap.Any("info", info))

	return nil
}

// UpdateNodeStatus 更新节点状态
func (c *ControlPlaneClient) UpdateNodeStatus(nodeID string, status map[string]interface{}) error {
	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/nodes/%s/status", c.config.ControlPlane.URL, nodeID)

	// 构建请求体
	body, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("序列化节点状态失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(c.ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.ControlPlane.Token))

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("更新节点状态失败，状态码: %d", resp.StatusCode)
	}

	c.logger.Debug("节点状态更新成功",
		zap.String("node_id", nodeID),
		zap.Any("status", status))

	return nil
}

// GetNodeConfig 获取节点配置
func (c *ControlPlaneClient) GetNodeConfig(nodeID string) (map[string]interface{}, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/nodes/%s/config", c.config.ControlPlane.URL, nodeID)

	// 创建请求
	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.ControlPlane.Token))

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取节点配置失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应体
	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("解析响应体失败: %v", err)
	}

	return config, nil
}

// ValidateNode 向控制平面验证节点令牌
func (c *ControlPlaneClient) ValidateNode(nodeID string, token string) error {
	// 注意：我们复用主控端的注册接口进行验证
	// 主控端需要实现: 如果节点已存在且令牌匹配，返回成功；如果节点不存在，尝试注册（可能需要更复杂的逻辑或特定验证接口）
	// 或者主控端可以专门提供 /validate 接口
	url := fmt.Sprintf("%s/api/v1/nodes/register", c.config.ControlPlane.URL)

	payload := map[string]string{
		"node_id": nodeID,
		"token":   token, // Agent 提供的令牌
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化验证负载失败: %w", err)
	}

	c.logger.Info("向主控平面发送节点验证请求",
		zap.String("url", url),
		zap.String("node_id", nodeID))

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second) // 增加验证超时
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建验证请求失败: %w", err)
	}

	// 使用聚合服务器与主控端之间的令牌进行认证
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.ControlPlane.Token))
	req.Header.Set("X-Aggregator-ID", "aggregator-1") // 标识是聚合服务器发起的验证

	startTime := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.Error("发送节点验证请求失败",
			zap.String("node_id", nodeID),
			zap.Duration("duration", duration),
			zap.Error(err))
		return fmt.Errorf("发送验证请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		c.logger.Info("主控平面节点验证成功",
			zap.String("node_id", nodeID),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", duration))
		return nil // 成功
	}

	c.logger.Error("主控平面节点验证失败",
		zap.String("node_id", nodeID),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration),
		zap.String("response", string(respBody)))

	return fmt.Errorf("节点验证失败，主控端返回状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
}
