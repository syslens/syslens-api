package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AggregatorConfig 聚合服务器配置
type AggregatorConfig struct {
	// 服务器配置
	Server struct {
		// 监听地址
		ListenAddr string `yaml:"listen_addr" json:"listen_addr"`
		// 最大连接数
		MaxConnections int `yaml:"max_connections" json:"max_connections"`
		// 连接超时时间（秒）
		ConnectionTimeout int `yaml:"connection_timeout" json:"connection_timeout"`
	} `yaml:"server" json:"server"`

	// 主控端配置
	ControlPlane struct {
		// 主控端地址
		URL string `yaml:"url" json:"url"`
		// 认证令牌
		Token string `yaml:"token" json:"token"`
		// 重试次数
		RetryCount int `yaml:"retry_count" json:"retry_count"`
		// 重试间隔（秒）
		RetryInterval int `yaml:"retry_interval" json:"retry_interval"`
	} `yaml:"control_plane" json:"control_plane"`

	// 数据处理配置
	Processing struct {
		// 批处理大小
		BatchSize int `yaml:"batch_size" json:"batch_size"`
		// 批处理间隔（毫秒）
		BatchInterval int `yaml:"batch_interval" json:"batch_interval"`
		// 数据保留时间（小时）
		RetentionHours int `yaml:"retention_hours" json:"retention_hours"`
	} `yaml:"processing" json:"processing"`

	// 安全配置 (与 Agent/Server 保持一致)
	Security SecurityConfig `yaml:"security" json:"security"`

	// 日志配置
	Log struct {
		// 日志级别
		Level string `yaml:"level" json:"level"`
		// 日志文件路径
		File string `yaml:"file" json:"file"`
		// 是否输出到控制台
		Console bool `yaml:"console" json:"console"`
	} `yaml:"log" json:"log"`
}

// DefaultAggregatorConfig 返回默认配置
func DefaultAggregatorConfig() *AggregatorConfig {
	cfg := &AggregatorConfig{}

	// 服务器默认配置
	cfg.Server.ListenAddr = "0.0.0.0:8081"
	cfg.Server.MaxConnections = 1000
	cfg.Server.ConnectionTimeout = 30

	// 主控端默认配置
	cfg.ControlPlane.URL = "http://localhost:8080"
	cfg.ControlPlane.RetryCount = 5
	cfg.ControlPlane.RetryInterval = 5

	// 数据处理默认配置
	cfg.Processing.BatchSize = 100
	cfg.Processing.BatchInterval = 1000
	cfg.Processing.RetentionHours = 24

	// 安全默认配置
	cfg.Security.Encryption.Enabled = false // 默认不启用加密
	cfg.Security.Encryption.Algorithm = "aes-256-gcm"
	cfg.Security.Compression.Enabled = false // 默认不启用压缩
	cfg.Security.Compression.Algorithm = "gzip"
	cfg.Security.Compression.Level = 6

	// 日志默认配置
	cfg.Log.Level = "info"
	cfg.Log.Console = true

	return cfg
}

// LoadAggregatorConfig 从文件加载配置
func LoadAggregatorConfig(path string) (*AggregatorConfig, error) {
	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	cfg := DefaultAggregatorConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateAggregatorConfig(cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// validateAggregatorConfig 验证配置
func validateAggregatorConfig(cfg *AggregatorConfig) error {
	// 验证服务器配置
	if cfg.Server.ListenAddr == "" {
		return fmt.Errorf("服务器监听地址不能为空")
	}

	if cfg.Server.MaxConnections <= 0 {
		return fmt.Errorf("最大连接数必须大于0")
	}

	if cfg.Server.ConnectionTimeout <= 0 {
		return fmt.Errorf("连接超时时间必须大于0")
	}

	// 验证主控端配置
	if cfg.ControlPlane.URL == "" {
		return fmt.Errorf("主控端地址不能为空")
	}

	if cfg.ControlPlane.RetryCount < 0 {
		return fmt.Errorf("重试次数不能为负数")
	}

	if cfg.ControlPlane.RetryInterval <= 0 {
		return fmt.Errorf("重试间隔必须大于0")
	}

	// 验证数据处理配置
	if cfg.Processing.BatchSize <= 0 {
		return fmt.Errorf("批处理大小必须大于0")
	}

	if cfg.Processing.BatchInterval <= 0 {
		return fmt.Errorf("批处理间隔必须大于0")
	}

	if cfg.Processing.RetentionHours <= 0 {
		return fmt.Errorf("数据保留时间必须大于0")
	}

	// 验证安全配置 (如果启用)
	if cfg.Security.Encryption.Enabled {
		if cfg.Security.Encryption.Key == "" {
			return fmt.Errorf("加密已启用，但未配置密钥(security.encryption.key)")
		}
	}

	return nil
}
