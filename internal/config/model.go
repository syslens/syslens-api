package config

// AgentConfig 节点代理配置结构
type AgentConfig struct {
	Node       NodeConfig       `yaml:"node"`
	Server     ServerConnection `yaml:"server"`
	Security   SecurityConfig   `yaml:"security"`
	Collection CollectionConfig `yaml:"collection"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// NodeConfig 节点信息配置
type NodeConfig struct {
	ID     string            `yaml:"id"`
	Labels map[string]string `yaml:"labels"`
}

// ServerConnection 服务器连接配置
type ServerConnection struct {
	URL           string `yaml:"url"`
	TLSVerify     bool   `yaml:"tls_verify"`
	Token         string `yaml:"token"`
	Timeout       int    `yaml:"timeout"`
	RetryCount    int    `yaml:"retry_count"`
	RetryInterval int    `yaml:"retry_interval"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	Encryption  EncryptionConfig  `yaml:"encryption"`
	Compression CompressionConfig `yaml:"compression"`
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Algorithm string `yaml:"algorithm"`
	Key       string `yaml:"key"`
}

// CompressionConfig 压缩配置
type CompressionConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Algorithm string `yaml:"algorithm"`
	Level     int    `yaml:"level"`
}

// CollectionConfig 采集配置
type CollectionConfig struct {
	Interval int              `yaml:"interval"`
	Enabled  EnabledCollector `yaml:"enabled"`
	Disk     DiskConfig       `yaml:"disk"`
	Network  NetworkConfig    `yaml:"network"`
	Process  ProcessConfig    `yaml:"process"`
}

// EnabledCollector 启用的采集项
type EnabledCollector struct {
	CPU       bool `yaml:"cpu"`
	Memory    bool `yaml:"memory"`
	Disk      bool `yaml:"disk"`
	Network   bool `yaml:"network"`
	Processes bool `yaml:"processes"`
}

// DiskConfig 磁盘采集配置
type DiskConfig struct {
	MountPoints     []string `yaml:"mount_points"`
	IncludeInactive bool     `yaml:"include_inactive"`
}

// NetworkConfig 网络采集配置
type NetworkConfig struct {
	Interfaces []string `yaml:"interfaces"`
}

// ProcessConfig 进程采集配置
type ProcessConfig struct {
	CollectAll      bool     `yaml:"collect_all"`
	TargetProcesses []string `yaml:"target_processes"`
	MaxProcesses    int      `yaml:"max_processes"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level   string `yaml:"level"`
	File    string `yaml:"file"`
	Verbose bool   `yaml:"verbose"`
}

// ServerConfig 主控端配置结构
type ServerConfig struct {
	Server    HTTPServerConfig `yaml:"server"`
	Security  SecurityConfig   `yaml:"security"`
	Storage   StorageConfig    `yaml:"storage"`
	Discovery DiscoveryConfig  `yaml:"discovery"`
	Alerting  AlertingConfig   `yaml:"alerting"`
	Logging   LoggingConfig    `yaml:"logging"`
}

// HTTPServerConfig HTTP服务器配置
type HTTPServerConfig struct {
	HTTPAddr string `yaml:"http_addr"`
	UseHTTPS bool   `yaml:"use_https"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type     string          `yaml:"type"`
	Memory   MemoryStorage   `yaml:"memory"`
	File     FileStorage     `yaml:"file"`
	InfluxDB InfluxDBStorage `yaml:"influxdb"`
}

// MemoryStorage 内存存储配置
type MemoryStorage struct {
	MaxItems int `yaml:"max_items"`
}

// FileStorage 文件存储配置
type FileStorage struct {
	DataDir     string `yaml:"data_dir"`
	RotateHours int    `yaml:"rotate_hours"`
}

// InfluxDBStorage InfluxDB存储配置
type InfluxDBStorage struct {
	URL           string `yaml:"url"`
	Token         string `yaml:"token"`
	Org           string `yaml:"org"`
	Bucket        string `yaml:"bucket"`
	RetentionDays int    `yaml:"retention_days"`
}

// DiscoveryConfig 节点发现配置
type DiscoveryConfig struct {
	NodeExpiry        int  `yaml:"node_expiry"`
	AutoRemoveExpired bool `yaml:"auto_remove_expired"`
}

// AlertingConfig 告警配置
type AlertingConfig struct {
	Enabled       bool            `yaml:"enabled"`
	CheckInterval int             `yaml:"check_interval"`
	Rules         []AlertRule     `yaml:"rules"`
	Notifiers     NotifiersConfig `yaml:"notifiers"`
}

// AlertRule 告警规则
type AlertRule struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition"`
	Duration  string `yaml:"duration"`
	Severity  string `yaml:"severity"`
}

// NotifiersConfig 通知配置
type NotifiersConfig struct {
	Email   EmailNotifier   `yaml:"email"`
	Webhook WebhookNotifier `yaml:"webhook"`
}

// EmailNotifier 邮件通知配置
type EmailNotifier struct {
	Enabled    bool     `yaml:"enabled"`
	SMTPServer string   `yaml:"smtp_server"`
	SMTPPort   int      `yaml:"smtp_port"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	From       string   `yaml:"from"`
	To         []string `yaml:"to"`
}

// WebhookNotifier Webhook通知配置
type WebhookNotifier struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}
