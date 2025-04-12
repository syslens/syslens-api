package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/syslens/syslens-api/internal/server/api"
	"github.com/syslens/syslens-api/internal/server/storage"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/server.yaml", "配置文件路径")
	httpAddr := flag.String("addr", "0.0.0.0:8080", "HTTP服务监听地址")
	flag.Parse()

	// 日志初始化
	log.Println("SysLens服务端启动中...")
	log.Printf("使用配置文件: %s\n", *configPath)
	log.Printf("监听地址: %s\n", *httpAddr)

	// TODO: 加载配置文件

	// 初始化存储
	metricsStorage := storage.NewMemoryStorage(1000)
	log.Println("内存存储初始化完成")

	// 初始化API服务
	metricsHandler := api.NewMetricsHandler(metricsStorage)
	router := api.SetupRoutes(metricsHandler)
	log.Println("API路由初始化完成")

	// 启动HTTP服务
	server := &http.Server{
		Addr:    *httpAddr,
		Handler: router,
	}

	// 在单独的goroutine中启动服务
	go func() {
		log.Printf("HTTP服务启动，监听: %s\n", *httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务启动失败: %v\n", err)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("服务端正在关闭...")
	// TODO: 关闭各服务
	log.Println("服务端已安全退出")
}
