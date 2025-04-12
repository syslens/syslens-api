package collector

import (
	"net"
	"runtime"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
)

// SystemStats 包含系统各项指标数据
type SystemStats struct {
	// 基本信息
	Timestamp   time.Time `json:"timestamp"`
	CurrentTime string    `json:"current_time"`
	Hostname    string    `json:"hostname"`
	Platform    string    `json:"platform"`
	Uptime      uint64    `json:"uptime"`

	// 硬件参数
	Hardware HardwareInfo `json:"hardware"`

	// 系统负载
	LoadAvg LoadAvgStats `json:"load_avg"`

	// 资源使用情况
	CPU     map[string]float64   `json:"cpu"`
	Memory  MemoryStats          `json:"memory"`
	Disk    map[string]DiskStats `json:"disk"`
	Network NetworkStats         `json:"network"`
}

// HardwareInfo 包含硬件信息
type HardwareInfo struct {
	CPUModel    string   `json:"cpu_model"`
	CPUCores    int      `json:"cpu_cores"`
	MemoryTotal uint64   `json:"memory_total"`
	DiskTotal   uint64   `json:"disk_total"`
	GPUModel    []string `json:"gpu_model,omitempty"`
	GPUMemory   []uint64 `json:"gpu_memory,omitempty"`
}

// LoadAvgStats 包含系统负载信息
type LoadAvgStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// MemoryStats 包含内存使用信息
type MemoryStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapPercent float64 `json:"swap_percent"`
}

// DiskStats 包含磁盘使用信息
type DiskStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	FSType      string  `json:"fstype"`
}

// NetworkStats 包含网络使用信息
type NetworkStats struct {
	Interfaces    map[string]InterfaceStats `json:"interfaces"`
	PublicIPv4    []string                  `json:"public_ipv4"`
	PublicIPv6    []string                  `json:"public_ipv6"`
	PrivateIPv4   []string                  `json:"private_ipv4"`
	PrivateIPv6   []string                  `json:"private_ipv6"`
	TotalSent     uint64                    `json:"total_sent"`
	TotalReceived uint64                    `json:"total_received"`
	TCPConnCount  int                       `json:"tcp_connections"`
	UDPConnCount  int                       `json:"udp_connections"`
}

// InterfaceStats 包含网络接口信息
type InterfaceStats struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesRecv     uint64 `json:"bytes_recv"`
	UploadSpeed   uint64 `json:"upload_speed"`
	DownloadSpeed uint64 `json:"download_speed"`
}

// Collector 系统指标收集器接口
type Collector interface {
	Collect() (*SystemStats, error)
}

// SystemCollector 实现了Collector接口的系统指标收集器
type SystemCollector struct {
	// 可配置的采集选项
	mountPoints      []string
	interfaces       []string
	lastNetworkStats map[string]psnet.IOCountersStat
	lastCollectTime  time.Time
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
	now := time.Now()
	stats := &SystemStats{
		Timestamp:   now,
		CurrentTime: now.Format(time.RFC3339),
		CPU:         make(map[string]float64),
		Disk:        make(map[string]DiskStats),
		Network: NetworkStats{
			Interfaces:  make(map[string]InterfaceStats),
			PublicIPv4:  []string{},
			PublicIPv6:  []string{},
			PrivateIPv4: []string{},
			PrivateIPv6: []string{},
		},
	}

	// 收集主机基本信息
	if hostInfo, err := host.Info(); err == nil {
		stats.Hostname = hostInfo.Hostname
		stats.Platform = hostInfo.Platform + " " + hostInfo.PlatformVersion + " " + runtime.GOARCH
		stats.Uptime = hostInfo.Uptime
	}

	// 收集硬件信息
	stats.Hardware = sc.collectHardwareInfo()

	// 收集CPU使用率
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		stats.CPU["usage"] = cpuPercent[0]
	}

	// 收集CPU详细信息
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		stats.Hardware.CPUModel = cpuInfo[0].ModelName
		stats.Hardware.CPUCores = len(cpuInfo)
	}

	// 收集系统负载
	if loadAvg, err := load.Avg(); err == nil {
		stats.LoadAvg = LoadAvgStats{
			Load1:  loadAvg.Load1,
			Load5:  loadAvg.Load5,
			Load15: loadAvg.Load15,
		}
	}

	// 收集内存信息
	if memStat, err := mem.VirtualMemory(); err == nil {
		stats.Memory.Total = memStat.Total
		stats.Memory.Used = memStat.Used
		stats.Memory.Free = memStat.Free
		stats.Memory.UsedPercent = memStat.UsedPercent
	}

	// 收集交换分区信息
	if swapStat, err := mem.SwapMemory(); err == nil {
		stats.Memory.SwapTotal = swapStat.Total
		stats.Memory.SwapUsed = swapStat.Used
		stats.Memory.SwapPercent = swapStat.UsedPercent
	}

	// 收集磁盘信息
	for _, mountPoint := range sc.mountPoints {
		if diskStat, err := disk.Usage(mountPoint); err == nil {
			stats.Disk[mountPoint] = DiskStats{
				Total:       diskStat.Total,
				Used:        diskStat.Used,
				Free:        diskStat.Free,
				UsedPercent: diskStat.UsedPercent,
				FSType:      diskStat.Fstype,
			}
		}
	}

	// 计算总磁盘容量
	var totalDiskSpace uint64 = 0
	for _, diskStat := range stats.Disk {
		totalDiskSpace += diskStat.Total
	}
	stats.Hardware.DiskTotal = totalDiskSpace

	// 收集网络接口信息和统计
	if netIOCounters, err := psnet.IOCounters(true); err == nil {
		var totalSent, totalRecv uint64
		var prevNetIOCounters map[string]psnet.IOCountersStat

		// 获取上次采集的网络数据，计算速率
		if sc.lastNetworkStats != nil {
			prevNetIOCounters = sc.lastNetworkStats
			timeDiff := now.Sub(sc.lastCollectTime).Seconds()

			if timeDiff > 0 {
				for _, netIO := range netIOCounters {
					if len(sc.interfaces) == 0 || containsString(sc.interfaces, netIO.Name) {
						prev, exists := prevNetIOCounters[netIO.Name]

						// 计算网络速率
						uploadSpeed := uint64(0)
						downloadSpeed := uint64(0)

						if exists && timeDiff > 0 {
							uploadSpeed = uint64(float64(netIO.BytesSent-prev.BytesSent) / timeDiff)
							downloadSpeed = uint64(float64(netIO.BytesRecv-prev.BytesRecv) / timeDiff)
						}

						stats.Network.Interfaces[netIO.Name] = InterfaceStats{
							BytesSent:     netIO.BytesSent,
							BytesRecv:     netIO.BytesRecv,
							UploadSpeed:   uploadSpeed,
							DownloadSpeed: downloadSpeed,
						}

						totalSent += netIO.BytesSent
						totalRecv += netIO.BytesRecv
					}
				}
			}
		} else {
			// 首次采集，无法计算速率
			for _, netIO := range netIOCounters {
				if len(sc.interfaces) == 0 || containsString(sc.interfaces, netIO.Name) {
					stats.Network.Interfaces[netIO.Name] = InterfaceStats{
						BytesSent:     netIO.BytesSent,
						BytesRecv:     netIO.BytesRecv,
						UploadSpeed:   0,
						DownloadSpeed: 0,
					}

					totalSent += netIO.BytesSent
					totalRecv += netIO.BytesRecv
				}
			}
		}

		stats.Network.TotalSent = totalSent
		stats.Network.TotalReceived = totalRecv

		// 保存当前采集结果，用于下次计算速率
		sc.lastNetworkStats = make(map[string]psnet.IOCountersStat)
		for _, netIO := range netIOCounters {
			sc.lastNetworkStats[netIO.Name] = netIO
		}
		sc.lastCollectTime = now
	}

	// 收集IP地址信息
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}

				ip := ipnet.IP
				if ip.IsLoopback() {
					continue
				}

				if ip.To4() != nil {
					// IPv4地址
					if isPrivateIP(ip) {
						stats.Network.PrivateIPv4 = append(stats.Network.PrivateIPv4, ip.String())
					} else {
						stats.Network.PublicIPv4 = append(stats.Network.PublicIPv4, ip.String())
					}
				} else {
					// IPv6地址
					if isPrivateIP(ip) {
						stats.Network.PrivateIPv6 = append(stats.Network.PrivateIPv6, ip.String())
					} else {
						stats.Network.PublicIPv6 = append(stats.Network.PublicIPv6, ip.String())
					}
				}
			}
		}
	}

	// 收集TCP和UDP连接数
	if connections, err := psnet.Connections("all"); err == nil {
		tcpCount := 0
		udpCount := 0

		for _, conn := range connections {
			if conn.Type == syscall.SOCK_STREAM {
				tcpCount++
			} else if conn.Type == syscall.SOCK_DGRAM {
				udpCount++
			}
		}

		stats.Network.TCPConnCount = tcpCount
		stats.Network.UDPConnCount = udpCount
	}

	return stats, nil
}

// collectHardwareInfo 收集硬件信息
func (sc *SystemCollector) collectHardwareInfo() HardwareInfo {
	var info HardwareInfo

	// 内存总量
	if memStat, err := mem.VirtualMemory(); err == nil {
		info.MemoryTotal = memStat.Total
	}

	// 这里移除了GPU相关的代码，因为gopsutil没有提供GPU模块
	// 如果需要GPU信息，可以考虑使用其他库如nvml

	return info
}

// isPrivateIP 检查IP是否为私有地址
func isPrivateIP(ip net.IP) bool {
	// IPv4私有地址范围
	privateCIDRBlocks := []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 - 链路本地地址
		"127.0.0.0/8",    // RFC1122 - 环回地址
	}

	// IPv6私有地址范围
	if ip.To4() == nil {
		return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
	}

	// 检查IPv4地址是否在私有范围内
	for _, block := range privateCIDRBlocks {
		_, ipnet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
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
