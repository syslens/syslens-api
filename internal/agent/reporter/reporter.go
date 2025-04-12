package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Reporter 定义了指标上报器接口
type Reporter interface {
	Report(data interface{}) error
}

// HTTPReporter 实现了通过HTTP上报数据的Reporter
type HTTPReporter struct {
	serverURL     string        // 主控服务器URL
	client        *http.Client  // HTTP客户端
	retryCount    int           // 重试次数
	retryInterval time.Duration // 重试间隔
}

// NewHTTPReporter 创建一个新的HTTP上报器
func NewHTTPReporter(serverURL string, options ...func(*HTTPReporter)) *HTTPReporter {
	r := &HTTPReporter{
		serverURL:     serverURL,
		retryCount:    3,
		retryInterval: 5 * time.Second,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// 应用选项
	for _, option := range options {
		option(r)
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

// Report 将数据上报到服务器
func (r *HTTPReporter) Report(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("数据序列化失败: %w", err)
	}

	// 发送数据，支持重试
	var lastErr error
	for i := 0; i <= r.retryCount; i++ {
		if i > 0 {
			// 重试前等待
			time.Sleep(r.retryInterval)
		}

		req, err := http.NewRequest("POST", r.serverURL+"/api/v1/metrics", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "SysLens-Agent")

		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP请求失败: %w", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // 成功
		}

		lastErr = fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	return fmt.Errorf("数据上报失败，已重试%d次: %w", r.retryCount, lastErr)
}
