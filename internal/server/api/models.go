package api

// Response 通用响应结构
type Response struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message,omitempty" example:"操作成功"`
	Error   string      `json:"error,omitempty" example:"错误信息"`
	Data    interface{} `json:"data,omitempty"`
}

// Node 节点信息
type Node struct {
	ID           string            `json:"id" example:"node-123456"`
	Name         string            `json:"name" example:"web-server-01"`
	IP           string            `json:"ip" example:"192.168.1.100"`
	Status       string            `json:"status" example:"online"`
	LastReported string            `json:"last_reported" example:"2023-06-01T15:30:45Z"`
	System       SystemInfo        `json:"system,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	OS           string `json:"os" example:"linux"`
	HostName     string `json:"hostname" example:"web-server-01"`
	Platform     string `json:"platform" example:"Ubuntu"`
	Architecture string `json:"architecture" example:"x86_64"`
	Version      string `json:"version" example:"20.04"`
}

// NodeMetrics 节点指标数据
type NodeMetrics struct {
	NodeID    string                 `json:"node_id" example:"node-123456"`
	Timestamp string                 `json:"timestamp" example:"2023-06-01T15:30:45Z"`
	CPU       map[string]interface{} `json:"cpu,omitempty"`
	Memory    map[string]interface{} `json:"memory,omitempty"`
	Disk      map[string]interface{} `json:"disk,omitempty"`
	Network   map[string]interface{} `json:"network,omitempty"`
}

// NodeRegisterRequest 节点注册请求
type NodeRegisterRequest struct {
	Name     string            `json:"name" example:"web-server-01"`
	IP       string            `json:"ip" example:"192.168.1.100"`
	System   SystemInfo        `json:"system"`
	Tags     map[string]string `json:"tags,omitempty"`
	Features []string          `json:"features,omitempty" example:"['cpu','memory','disk']"`
}

// StatusUpdateRequest 节点状态更新请求
type StatusUpdateRequest struct {
	NodeID string `json:"node_id" example:"node-123456"`
	Status string `json:"status" example:"offline"`
	Reason string `json:"reason,omitempty" example:"系统维护中"`
}
