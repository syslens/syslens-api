package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/syslens/syslens-api/internal/aggregator"
	"github.com/syslens/syslens-api/internal/config"
)

var (
	configFile = flag.String("config", "", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println("SysLens Aggregator v1.0.0")
		return
	}

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建上下文，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理，用于优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 初始化并启动聚合服务器
	server, err := aggregator.NewServer(cfg)
	if err != nil {
		log.Fatalf("初始化聚合服务器失败: %v", err)
	}

	// 启动服务器
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("聚合服务器启动失败: %v", err)
			cancel()
		}
	}()

	log.Println("聚合服务器已启动，等待节点连接...")

	// 等待信号
	select {
	case sig := <-sigChan:
		log.Printf("收到信号 %v，开始优雅关闭...", sig)
	case <-ctx.Done():
		log.Println("上下文取消，开始优雅关闭...")
	}

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("聚合服务器关闭失败: %v", err)
	} else {
		log.Println("聚合服务器已成功关闭")
	}
}

// loadConfig 加载配置文件
func loadConfig() (*config.AggregatorConfig, error) {
	// 如果未指定配置文件，尝试使用默认路径
	if *configFile == "" {
		// 尝试多个可能的配置文件位置
		possiblePaths := []string{
			"configs/aggregator.yaml",
			"configs/aggregator.template.yaml",
			filepath.Join(os.Getenv("HOME"), ".syslens", "aggregator.yaml"),
			"/etc/syslens/aggregator.yaml",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				*configFile = path
				break
			}
		}

		// 如果仍未找到配置文件，使用默认配置
		if *configFile == "" {
			log.Println("未找到配置文件，使用默认配置")
			return config.DefaultAggregatorConfig(), nil
		}
	}

	log.Printf("使用配置文件: %s", *configFile)
	return config.LoadAggregatorConfig(*configFile)
}
