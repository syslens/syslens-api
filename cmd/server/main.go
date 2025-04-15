package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/syslens/syslens-api/internal/config"
	"github.com/syslens/syslens-api/internal/server/api"
	"github.com/syslens/syslens-api/internal/server/storage"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

	// 加载配置文件
	serverConfig, err := loadConfig(*configPath)
	if err != nil {
		log.Printf("警告: 无法加载配置文件，使用默认配置: %v\n", err)
		// 创建默认配置
		serverConfig = &config.ServerConfig{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Enabled:   false,
					Algorithm: "aes-256-gcm",
					Key:       "",
				},
				Compression: config.CompressionConfig{
					Enabled:   false,
					Algorithm: "gzip",
				},
			},
		}
	}

	// 命令行参数覆盖配置文件
	if *httpAddr != "0.0.0.0:8080" {
		serverConfig.Server.HTTPAddr = *httpAddr
	}
	if *storageType != "memory" {
		serverConfig.Storage.Type = *storageType
	}

	// 初始化存储
	var metricsStorage api.MetricsStorage

	switch serverConfig.Storage.Type {
	case "influxdb":
		// 优先使用命令行参数，其次使用配置文件
		influxDBURL := *influxURL
		if influxDBURL == "http://localhost:8086" && serverConfig.Storage.InfluxDB.URL != "" {
			influxDBURL = serverConfig.Storage.InfluxDB.URL
		}

		influxDBToken := *influxToken
		if influxDBToken == "" {
			influxDBToken = serverConfig.Storage.InfluxDB.Token
		}

		influxDBOrg := *influxOrg
		if influxDBOrg == "syslens" && serverConfig.Storage.InfluxDB.Org != "" {
			influxDBOrg = serverConfig.Storage.InfluxDB.Org
		}

		influxDBBucket := *influxBucket
		if influxDBBucket == "metrics" && serverConfig.Storage.InfluxDB.Bucket != "" {
			influxDBBucket = serverConfig.Storage.InfluxDB.Bucket
		}

		if influxDBToken == "" {
			log.Fatal("InfluxDB Token不能为空")
		}
		log.Printf("初始化InfluxDB存储: %s\n", influxDBURL)
		metricsStorage = storage.NewInfluxDBStorage(influxDBURL, influxDBToken, influxDBOrg, influxDBBucket)
		log.Println("InfluxDB存储初始化完成")
	case "memory":
		fallthrough
	default:
		maxItems := 1000
		if serverConfig.Storage.Memory.MaxItems > 0 {
			maxItems = serverConfig.Storage.Memory.MaxItems
		}
		log.Println("初始化内存存储")
		metricsStorage = storage.NewMemoryStorage(maxItems)
		log.Println("内存存储初始化完成")
	}

	// 初始化API服务
	metricsHandler := api.NewMetricsHandler(metricsStorage)

	// 应用安全配置
	metricsHandler.WithSecurityConfig(&serverConfig.Security)

	// 初始化zap日志记录器
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 日志安全配置状态
	if serverConfig.Security.Encryption.Enabled {
		log.Printf("数据解密已启用，算法: %s", serverConfig.Security.Encryption.Algorithm)
	} else {
		log.Println("数据解密未启用")
	}

	if serverConfig.Security.Compression.Enabled {
		log.Printf("数据解压缩已启用，算法: %s", serverConfig.Security.Compression.Algorithm)
	} else {
		log.Println("数据解压缩未启用")
	}

	router := api.SetupRouter(metricsHandler, logger)
	log.Println("API路由初始化完成")

	// 启动HTTP服务
	server := &http.Server{
		Addr:    serverConfig.Server.HTTPAddr,
		Handler: router,
	}

	// 在单独的goroutine中启动服务
	go func() {
		log.Printf("HTTP服务启动在 %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务启动失败: %v\n", err)
		}
	}()

	// 等待中断信号
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

// loadConfig 从文件加载配置并支持环境变量替换
func loadConfig(path string) (*config.ServerConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 环境变量替换
	content := string(data)
	re := regexp.MustCompile(`\${([^}]+)}`)
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取变量名，去掉${}
		envVar := match[2 : len(match)-1]

		// 检查是否有默认值设置（格式：${ENV_VAR:-default}）
		parts := strings.SplitN(envVar, ":-", 2)
		envName := parts[0]

		// 获取环境变量值
		if val, exists := os.LookupEnv(envName); exists {
			return val
		}

		// 如果环境变量不存在但有默认值，则使用默认值
		if len(parts) > 1 {
			return parts[1]
		}

		// 保持原样
		return match
	})

	var cfg config.ServerConfig
	if err := yaml.Unmarshal([]byte(result), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
