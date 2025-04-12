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
	storageType := flag.String("storage", "memory", "存储类型: memory, influxdb")
	influxURL := flag.String("influx-url", "http://localhost:8086", "InfluxDB URL")
	influxToken := flag.String("influx-token", "", "InfluxDB Token")
	influxOrg := flag.String("influx-org", "syslens", "InfluxDB Organization")
	influxBucket := flag.String("influx-bucket", "metrics", "InfluxDB Bucket")
	flag.Parse()

	// 日志初始化
	log.Println("SysLens服务端启动中...")
	log.Printf("使用配置文件: %s\n", *configPath)
	log.Printf("监听地址: %s\n", *httpAddr)
	log.Printf("存储类型: %s\n", *storageType)

	// TODO: 加载配置文件

	// 初始化存储
	var metricsStorage api.MetricsStorage

	switch *storageType {
	case "influxdb":
		if *influxToken == "" {
			log.Fatal("InfluxDB Token不能为空")
		}
		log.Printf("初始化InfluxDB存储: %s\n", *influxURL)
		metricsStorage = storage.NewInfluxDBStorage(*influxURL, *influxToken, *influxOrg, *influxBucket)
		log.Println("InfluxDB存储初始化完成")
	case "memory":
		fallthrough
	default:
		log.Println("初始化内存存储")
		metricsStorage = storage.NewMemoryStorage(1000)
		log.Println("内存存储初始化完成")
	}

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

	// 关闭存储连接
	if influxStorage, ok := metricsStorage.(*storage.InfluxDBStorage); ok {
		influxStorage.Close()
		log.Println("InfluxDB连接已关闭")
	}

	log.Println("服务端已安全退出")
}
