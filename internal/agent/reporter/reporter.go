package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/syslens/syslens-api/internal/common/utils"
	"github.com/syslens/syslens-api/internal/config"
)

// Reporter 定义了指标上报器接口
type Reporter interface {
	Report(data interface{}) error
}

// HTTPReporter 实现了通过HTTP上报数据的Reporter
type HTTPReporter struct {
	serverURL     string        // 主控服务器URL
	nodeID        string        // 节点ID字段
	client        *http.Client  // HTTP客户端
	retryCount    int           // 重试次数
	retryInterval time.Duration // 重试间隔
	authToken     string        // 认证令牌

	securityConfig *config.SecurityConfig   // 安全配置
	encryptionSvc  *utils.EncryptionService // 加密服务
}

// NewHTTPReporter 创建一个新的HTTP上报器
func NewHTTPReporter(serverURL string, nodeID string, options ...func(*HTTPReporter)) *HTTPReporter {
	r := &HTTPReporter{
		serverURL:     serverURL,
		nodeID:        nodeID,
		retryCount:    3,
		retryInterval: 1 * time.Second,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		securityConfig: &config.SecurityConfig{
			Encryption: config.EncryptionConfig{
				Enabled:   false,
				Algorithm: "aes-256-gcm",
				Key:       "",
			},
			Compression: config.CompressionConfig{
				Enabled:   false,
				Algorithm: "gzip",
				Level:     6,
			},
		},
	}

	// 应用选项
	for _, option := range options {
		option(r)
	}

	// 初始化加密服务
	if r.securityConfig.Encryption.Enabled {
		r.encryptionSvc = utils.NewEncryptionService(r.securityConfig.Encryption.Algorithm)
	}

	return r
}

// WithRetryCount 设置重试次数
func WithRetryCount(count int) func(*HTTPReporter) {
	return func(r *HTTPReporter) {
		if count >= 0 {
			r.retryCount = count
		}
	}
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(interval time.Duration) func(*HTTPReporter) {
	return func(r *HTTPReporter) {
		if interval > 0 {
			r.retryInterval = interval
		}
	}
}

// WithTimeout 设置HTTP请求超时时间
func WithTimeout(timeout time.Duration) func(*HTTPReporter) {
	return func(r *HTTPReporter) {
		if timeout > 0 {
			r.client.Timeout = timeout
		}
	}
}

// WithSecurityConfig 设置安全配置
func WithSecurityConfig(secConfig *config.SecurityConfig) func(*HTTPReporter) {
	return func(r *HTTPReporter) {
		if secConfig != nil {
			r.securityConfig = secConfig
			// 初始化加密服务
			if r.securityConfig.Encryption.Enabled {
				r.encryptionSvc = utils.NewEncryptionService(r.securityConfig.Encryption.Algorithm)
			}
		}
	}
}

// WithAuthToken 设置认证令牌
func WithAuthToken(token string) func(*HTTPReporter) {
	return func(r *HTTPReporter) {
		r.authToken = token
	}
}

// SetAuthToken 设置认证令牌
func (r *HTTPReporter) SetAuthToken(token string) {
	r.authToken = token
}

// Report 将数据上报到服务器
func (r *HTTPReporter) Report(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("数据序列化失败: %w", err)
	}

	// 压缩和加密数据
	processedData, contentType, err := r.processData(jsonData)
	if err != nil {
		return fmt.Errorf("数据处理失败: %w", err)
	}

	// 发送数据，支持重试
	var lastErr error
	for i := 0; i <= r.retryCount; i++ {
		if i > 0 {
			// 重试前等待
			retryDelay := r.retryInterval
			log.Printf("上报重试 (%d/%d)，等待 %v 后重试...", i, r.retryCount, retryDelay)
			time.Sleep(retryDelay)
		}

		// 构建请求URL
		nodeID := r.nodeID
		if nodeID == "" {
			if hostname, err := os.Hostname(); err == nil {
				nodeID = hostname
			} else {
				nodeID = "unknown-node"
			}
		}
		url := fmt.Sprintf("%s/api/v1/nodes/%s/metrics", r.serverURL, nodeID)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(processedData))
		if err != nil {
			lastErr = fmt.Errorf("创建HTTP请求失败: %w", err)
			log.Printf("重试失败: %v", lastErr)
			continue
		}

		// 设置适当的内容类型
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("User-Agent", "SysLens-Agent")

		// 添加节点ID头部
		req.Header.Set("X-Node-ID", nodeID)

		// 添加认证令牌
		if r.authToken != "" {
			req.Header.Set("Authorization", "Bearer "+r.authToken)
		}

		// 添加数据处理标记
		if r.securityConfig.Compression.Enabled {
			req.Header.Set("X-Compressed", "gzip")
		}
		if r.securityConfig.Encryption.Enabled {
			req.Header.Set("X-Encrypted", "true")
		}

		// 添加请求上下文，带超时控制
		ctx, cancel := context.WithTimeout(context.Background(), r.client.Timeout)
		req = req.WithContext(ctx)

		startTime := time.Now()
		resp, err := r.client.Do(req)
		requestTime := time.Since(startTime)
		cancel() // 释放上下文资源

		if err != nil {
			lastErr = fmt.Errorf("HTTP请求失败 (耗时: %v): %w", requestTime, err)
			log.Printf("请求错误: %v", lastErr)
			continue
		}

		defer resp.Body.Close()

		// 读取响应内容，用于详细日志
		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("上报成功，响应码: %d，耗时: %v", resp.StatusCode, requestTime)
			return nil // 成功
		}

		lastErr = fmt.Errorf("服务器返回错误状态码: %d，响应: %s", resp.StatusCode, string(respBody))
		log.Printf("服务端错误: %v", lastErr)
	}

	// 构造详细的错误信息
	detailedErr := fmt.Errorf("数据上报失败，已重试%d次，主控节点URL: %s，最后错误: %w",
		r.retryCount, r.serverURL, lastErr)

	return detailedErr
}

// processData 处理数据：压缩和加密
func (r *HTTPReporter) processData(data []byte) ([]byte, string, error) {
	processedData := data
	contentType := "application/json"
	var err error

	// 步骤1：压缩
	if r.securityConfig.Compression.Enabled {
		processedData, err = utils.CompressData(processedData, r.securityConfig.Compression.Level)
		if err != nil {
			return nil, contentType, fmt.Errorf("压缩失败: %w", err)
		}
		contentType = "application/octet-stream"
	}

	// 步骤2：加密
	if r.securityConfig.Encryption.Enabled && r.encryptionSvc != nil {
		processedData, err = r.encryptionSvc.Encrypt(processedData, r.securityConfig.Encryption.Key)
		if err != nil {
			return nil, contentType, fmt.Errorf("加密失败: %w", err)
		}
		contentType = "application/octet-stream"
	}

	return processedData, contentType, nil
}
