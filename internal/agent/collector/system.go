package collector

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemStats 包含系统各项指标数据
type SystemStats struct {
	Timestamp time.Time              `json:"timestamp"`
	Hostname  string                 `json:"hostname"`
	Platform  string                 `json:"platform"`
	CPU       map[string]float64     `json:"cpu"`
	Memory    map[string]interface{} `json:"memory"`
	Disk      map[string]interface{} `json:"disk"`
	Network   map[string]interface{} `json:"network"`
}

// Collector 系统指标收集器接口
type Collector interface {
	Collect() (*SystemStats, error)
}

// SystemCollector 实现了Collector接口的系统指标收集器
type SystemCollector struct {
	// 可配置的采集选项
	mountPoints []string
	interfaces  []string
}

// NewSystemCollector 创建新的系统指标收集器
func NewSystemCollector(options ...func(*SystemCollector)) *SystemCollector {
	sc := &SystemCollector{
		mountPoints: []string{"/"},
		interfaces:  []string{}, // 空切片表示收集所有网络接口
	}

	// 应用可选配置
	for _, option := range options {
		option(sc)
	}

	return sc
}

// WithMountPoints 设置要监控的挂载点
func WithMountPoints(mounts []string) func(*SystemCollector) {
	return func(sc *SystemCollector) {
		sc.mountPoints = mounts
	}
}

// WithInterfaces 设置要监控的网络接口
func WithInterfaces(ifaces []string) func(*SystemCollector) {
	return func(sc *SystemCollector) {
		sc.interfaces = ifaces
	}
}

// Collect 采集系统指标
func (sc *SystemCollector) Collect() (*SystemStats, error) {
	stats := &SystemStats{
		Timestamp: time.Now(),
		CPU:       make(map[string]float64),
		Memory:    make(map[string]interface{}),
		Disk:      make(map[string]interface{}),
		Network:   make(map[string]interface{}),
	}

	// 获取主机信息
	hostInfo, err := host.Info()
	if err == nil {
		stats.Hostname = hostInfo.Hostname
		stats.Platform = hostInfo.Platform + " " + hostInfo.PlatformVersion + " " + runtime.GOARCH
	}

	// 获取CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		stats.CPU["usage"] = cpuPercent[0]
	}

	// 获取CPU负载
	loadAvg, err := load.Avg()
	if err == nil {
		stats.CPU["load1"] = loadAvg.Load1
		stats.CPU["load5"] = loadAvg.Load5
		stats.CPU["load15"] = loadAvg.Load15
	}

	// 获取内存信息
	memStat, err := mem.VirtualMemory()
	if err == nil {
		stats.Memory["total"] = memStat.Total
		stats.Memory["used"] = memStat.Used
		stats.Memory["free"] = memStat.Free
		stats.Memory["used_percent"] = memStat.UsedPercent
	}

	// 获取交换分区信息
	swapStat, err := mem.SwapMemory()
	if err == nil {
		stats.Memory["swap_total"] = swapStat.Total
		stats.Memory["swap_used"] = swapStat.Used
		stats.Memory["swap_used_percent"] = swapStat.UsedPercent
	}

	// 获取磁盘信息
	for _, mountPoint := range sc.mountPoints {
		diskStat, err := disk.Usage(mountPoint)
		if err == nil {
			stats.Disk[mountPoint] = map[string]interface{}{
				"total":        diskStat.Total,
				"used":         diskStat.Used,
				"free":         diskStat.Free,
				"used_percent": diskStat.UsedPercent,
			}
		}
	}

	// 获取网络信息
	netIOCounters, err := net.IOCounters(true)
	if err == nil {
		for _, netIO := range netIOCounters {
			if len(sc.interfaces) == 0 || containsString(sc.interfaces, netIO.Name) {
				stats.Network[netIO.Name] = map[string]interface{}{
					"bytes_sent":   netIO.BytesSent,
					"bytes_recv":   netIO.BytesRecv,
					"packets_sent": netIO.PacketsSent,
					"packets_recv": netIO.PacketsRecv,
					"errin":        netIO.Errin,
					"errout":       netIO.Errout,
					"dropin":       netIO.Dropin,
					"dropout":      netIO.Dropout,
				}
			}
		}
	}

	return stats, nil
}

// containsString 检查字符串数组是否包含指定字符串
func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
