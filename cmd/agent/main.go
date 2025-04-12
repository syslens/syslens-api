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
	flag.Parse()

	// 日志初始化
	log.Println("SysLens节点代理启动中...")
	log.Printf("连接到服务器: %s\n", *serverAddr)
	log.Printf("采集间隔: %d秒\n", *interval)

	// TODO: 加载配置文件
	log.Printf("配置文件路径: %s\n", *configPath)

	// TODO: 初始化指标收集器
	systemCollector := collector.NewSystemCollector()

	// TODO: 初始化数据上报模块
	metricsReporter := reporter.NewHTTPReporter(*serverAddr)

	// TODO: 启动定时采集任务
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	go func() {
		for range ticker.C {
			// 收集指标
			metrics, err := systemCollector.Collect()
			if err != nil {
				log.Printf("采集指标失败: %v", err)
				continue
			}

			// 上报指标
			if err := metricsReporter.Report(metrics); err != nil {
				log.Printf("上报指标失败: %v", err)
			} else {
				log.Println("成功上报节点指标")
			}
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("节点代理正在关闭...")
	ticker.Stop()
	// TODO: 执行清理操作
	log.Println("节点代理已安全退出")
}
