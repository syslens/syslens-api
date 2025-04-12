package collector

import (
	"net"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
)

// ParallelCollector 实现了一个并行收集系统指标的收集器
type ParallelCollector struct {
	// 继承SystemCollector的所有字段
	SystemCollector
}

// NewParallelCollector 创建一个新的并行收集器
func NewParallelCollector(options ...func(*SystemCollector)) *ParallelCollector {
	baseCollector := NewSystemCollector(options...)
	return &ParallelCollector{
		SystemCollector: *baseCollector,
	}
}

// Collect 并行收集系统指标
func (pc *ParallelCollector) Collect() (*SystemStats, error) {
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

	// 使用WaitGroup来并行执行收集任务
	var wg sync.WaitGroup

	// 1. 收集主机基本信息（很快，保持同步）
	if hostInfo, err := host.Info(); err == nil {
		stats.Hostname = hostInfo.Hostname
		stats.Platform = hostInfo.Platform + " " + hostInfo.PlatformVersion + " " + runtime.GOARCH
		stats.Uptime = hostInfo.Uptime
	}

	// 2. 并行收集CPU信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		pc.collectCPUInfo(stats)
	}()

	// 3. 并行收集内存信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		pc.collectMemoryInfo(stats)
	}()

	// 4. 并行收集磁盘信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		pc.collectDiskInfo(stats)
	}()

	// 5. 并行收集网络信息（最耗时的部分）
	wg.Add(1)
	go func() {
		defer wg.Done()
		pc.collectNetworkInfo(stats, now)
	}()

	// 等待所有收集任务完成
	wg.Wait()

	// 设置硬件信息
	stats.Hardware.CPUCores = stats.Hardware.CPUCores
	stats.Hardware.MemoryTotal = stats.Memory.Total
	stats.Hardware.DiskTotal = calculateTotalDiskSpace(stats.Disk)

	return stats, nil
}

// 计算磁盘总空间
func calculateTotalDiskSpace(diskStats map[string]DiskStats) uint64 {
	var total uint64 = 0
	for _, stat := range diskStats {
		total += stat.Total
	}
	return total
}

// 收集CPU信息
func (pc *ParallelCollector) collectCPUInfo(stats *SystemStats) {
	// 收集CPU使用率 - 减少等待时间从1秒到250毫秒
	if cpuPercent, err := cpu.Percent(time.Millisecond*250, false); err == nil && len(cpuPercent) > 0 {
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
}

// 收集内存信息
func (pc *ParallelCollector) collectMemoryInfo(stats *SystemStats) {
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
}

// 收集磁盘信息
func (pc *ParallelCollector) collectDiskInfo(stats *SystemStats) {
	var diskMutex sync.Mutex
	var diskWg sync.WaitGroup

	// 对每个挂载点并行收集
	for _, mountPoint := range pc.mountPoints {
		diskWg.Add(1)
		go func(mount string) {
			defer diskWg.Done()

			if diskStat, err := disk.Usage(mount); err == nil {
				// 安全更新共享的map
				diskMutex.Lock()
				stats.Disk[mount] = DiskStats{
					Total:       diskStat.Total,
					Used:        diskStat.Used,
					Free:        diskStat.Free,
					UsedPercent: diskStat.UsedPercent,
					FSType:      diskStat.Fstype,
				}
				diskMutex.Unlock()
			}
		}(mountPoint)
	}

	// 等待所有磁盘收集完成
	diskWg.Wait()
}

// 收集网络信息
func (pc *ParallelCollector) collectNetworkInfo(stats *SystemStats, now time.Time) {
	var netWg sync.WaitGroup
	var netMutex sync.Mutex

	// 1. 收集网络接口信息
	netWg.Add(1)
	go func() {
		defer netWg.Done()

		if netIOCounters, err := psnet.IOCounters(true); err == nil {
			var totalSent, totalRecv uint64
			var prevNetIOCounters map[string]psnet.IOCountersStat

			// 获取上次采集的网络数据，计算速率
			if pc.lastNetworkStats != nil {
				prevNetIOCounters = pc.lastNetworkStats
				timeDiff := now.Sub(pc.lastCollectTime).Seconds()

				if timeDiff > 0 {
					for _, netIO := range netIOCounters {
						if len(pc.interfaces) == 0 || containsString(pc.interfaces, netIO.Name) {
							prev, exists := prevNetIOCounters[netIO.Name]

							// 计算网络速率
							uploadSpeed := uint64(0)
							downloadSpeed := uint64(0)

							if exists && timeDiff > 0 {
								uploadSpeed = uint64(float64(netIO.BytesSent-prev.BytesSent) / timeDiff)
								downloadSpeed = uint64(float64(netIO.BytesRecv-prev.BytesRecv) / timeDiff)
							}

							netMutex.Lock()
							stats.Network.Interfaces[netIO.Name] = InterfaceStats{
								BytesSent:     netIO.BytesSent,
								BytesRecv:     netIO.BytesRecv,
								UploadSpeed:   uploadSpeed,
								DownloadSpeed: downloadSpeed,
							}
							netMutex.Unlock()

							totalSent += netIO.BytesSent
							totalRecv += netIO.BytesRecv
						}
					}
				}
			} else {
				// 首次采集，无法计算速率
				for _, netIO := range netIOCounters {
					if len(pc.interfaces) == 0 || containsString(pc.interfaces, netIO.Name) {
						netMutex.Lock()
						stats.Network.Interfaces[netIO.Name] = InterfaceStats{
							BytesSent:     netIO.BytesSent,
							BytesRecv:     netIO.BytesRecv,
							UploadSpeed:   0,
							DownloadSpeed: 0,
						}
						netMutex.Unlock()

						totalSent += netIO.BytesSent
						totalRecv += netIO.BytesRecv
					}
				}
			}

			netMutex.Lock()
			stats.Network.TotalSent = totalSent
			stats.Network.TotalReceived = totalRecv

			// 保存当前采集结果，用于下次计算速率
			pc.lastNetworkStats = make(map[string]psnet.IOCountersStat)
			for _, netIO := range netIOCounters {
				pc.lastNetworkStats[netIO.Name] = netIO
			}
			pc.lastCollectTime = now
			netMutex.Unlock()
		}
	}()

	// 2. 收集IP地址信息
	netWg.Add(1)
	go func() {
		defer netWg.Done()

		interfaces, err := net.Interfaces()
		if err == nil {
			var privateIPv4, publicIPv4, privateIPv6, publicIPv6 []string

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
							privateIPv4 = append(privateIPv4, ip.String())
						} else {
							publicIPv4 = append(publicIPv4, ip.String())
						}
					} else {
						// IPv6地址
						if isPrivateIP(ip) {
							privateIPv6 = append(privateIPv6, ip.String())
						} else {
							publicIPv6 = append(publicIPv6, ip.String())
						}
					}
				}
			}

			netMutex.Lock()
			stats.Network.PrivateIPv4 = privateIPv4
			stats.Network.PublicIPv4 = publicIPv4
			stats.Network.PrivateIPv6 = privateIPv6
			stats.Network.PublicIPv6 = publicIPv6
			netMutex.Unlock()
		}
	}()

	// 3. 收集TCP和UDP连接数（最耗时的部分，单独处理）
	netWg.Add(1)
	go func() {
		defer netWg.Done()

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

			netMutex.Lock()
			stats.Network.TCPConnCount = tcpCount
			stats.Network.UDPConnCount = udpCount
			netMutex.Unlock()
		}
	}()

	// 等待所有网络收集完成
	netWg.Wait()
}
