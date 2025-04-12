package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/syslens/syslens-api/internal/agent/collector"
	"github.com/syslens/syslens-api/internal/agent/reporter"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/agent.yaml", "配置文件路径")
	serverAddr := flag.String("server", "localhost:8080", "主控服务器地址")
	interval := flag.Int("interval", 30, "数据采集间隔(秒)")
	debug := flag.Bool("debug", false, "调试模式(只打印不上报)")
	flag.Parse()

	// 日志初始化
	log.Println("SysLens节点代理启动中...")
	log.Printf("配置文件路径: %s\n", *configPath)
	log.Printf("连接到服务器: %s\n", *serverAddr)
	log.Printf("采集间隔: %d秒\n", *interval)

	// TODO: 加载配置文件

	// 初始化指标收集器
	systemCollector := collector.NewSystemCollector()
	log.Println("系统指标收集器初始化完成")

	// 如果不是调试模式，则初始化上报模块
	var metricsReporter reporter.Reporter
	if !*debug {
		// 初始化数据上报模块
		serverURL := "http://" + *serverAddr
		metricsReporter = reporter.NewHTTPReporter(serverURL)
		log.Printf("数据上报模块初始化完成，目标服务器: %s\n", serverURL)
	} else {
		log.Println("调试模式启用，将只打印收集的数据而不上报")
	}

	// 启动定时采集任务
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	go func() {
		// 立即执行一次采集
		collectAndReport(systemCollector, metricsReporter, *debug)

		// 然后按照间隔定期执行
		for range ticker.C {
			collectAndReport(systemCollector, metricsReporter, *debug)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("节点代理正在关闭...")
	ticker.Stop()
	log.Println("节点代理已安全退出")
}

// collectAndReport 收集并上报系统指标
func collectAndReport(collector collector.Collector, reporter reporter.Reporter, debugMode bool) {
	log.Println("开始采集系统指标...")

	// 收集指标
	startTime := time.Now()
	stats, err := collector.Collect()
	if err != nil {
		log.Printf("采集指标失败: %v\n", err)
		return
	}

	elapsedTime := time.Since(startTime)
	log.Printf("系统指标采集完成，耗时: %v\n", elapsedTime)

	if debugMode {
		// 调试模式，只打印关键指标
		log.Printf("CPU使用率: %.2f%%\n", stats.CPU["usage"])
		log.Printf("内存使用率: %.2f%%\n", stats.Memory.UsedPercent)
		log.Printf("收集到 %d 个磁盘分区信息\n", len(stats.Disk))
		log.Printf("收集到 %d 个网络接口信息\n", len(stats.Network.Interfaces))
		log.Printf("TCP连接数: %d, UDP连接数: %d\n",
			stats.Network.TCPConnCount, stats.Network.UDPConnCount)
		log.Printf("IP地址: 公网IPv4=%v, 内网IPv4=%v\n",
			stats.Network.PublicIPv4, stats.Network.PrivateIPv4)
		return
	}

	// 上报指标
	if reporter != nil {
		log.Println("开始上报系统指标...")
		err = reporter.Report(stats)
		if err != nil {
			log.Printf("上报指标失败: %v\n", err)
		} else {
			log.Println("系统指标上报成功")
		}
	}
}
