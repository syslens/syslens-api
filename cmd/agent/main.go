package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/syslens/syslens-api/internal/agent/collector"
	"github.com/syslens/syslens-api/internal/agent/reporter"
	"github.com/syslens/syslens-api/internal/config"
	"gopkg.in/yaml.v3"
)

// 全局错误日志记录器
var errorLogger *log.Logger

const maxRegisterRetries = 3
const registerRetryInterval = 5 * time.Second

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/agent.yaml", "配置文件路径")
	serverAddr := flag.String("server", "localhost:8080", "主控服务器地址")
	interval := flag.Int("interval", 500, "数据采集间隔(毫秒)")
	debug := flag.Bool("debug", false, "调试模式(只打印不上报)")
	flag.Parse()

	// 创建日志目录
	os.MkdirAll("logs", 0755)

	// 初始化错误日志文件
	errorLogFile := "logs/agent_errors.log"
	errFile, err := os.OpenFile(errorLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("警告: 无法创建错误日志文件: %v，错误将只输出到标准输出", err)
		errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	} else {
		// 同时输出到文件和控制台
		multiWriter := io.MultiWriter(os.Stderr, errFile)
		errorLogger = log.New(multiWriter, "[ERROR] ", log.LstdFlags)
		log.Printf("错误日志将同时记录到: %s", errorLogFile)
	}

	// 日志初始化
	log.Println("SysLens节点代理启动中...")
	log.Printf("配置文件路径: %s\n", *configPath)
	log.Printf("连接到服务器: %s\n", *serverAddr)
	log.Printf("采集间隔: %d毫秒\n", *interval)

	// 加载配置文件
	agentConfig, err := loadConfig(*configPath)
	if err != nil {
		errorLogger.Printf("无法加载配置文件，使用默认配置: %v\n", err)
		// 创建默认配置
		agentConfig = &config.AgentConfig{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Enabled:   false,
					Algorithm: "aes-256-gcm",
					Key:       "",
				},
				Compression: config.CompressionConfig{
					Enabled:   false,
					Algorithm: "gzip",
					Level:     6,
				},
			},
		}
	}

	// 命令行参数覆盖配置文件
	if *interval > 0 {
		agentConfig.Collection.Interval = *interval
	}

	// 初始化指标收集器
	// systemCollector := collector.NewSystemCollector()
	systemCollector := collector.NewParallelCollector(
		collector.WithMountPoints(agentConfig.Collection.Disk.MountPoints),
		collector.WithInterfaces(agentConfig.Collection.Network.Interfaces),
	)
	log.Println("系统指标收集器初始化完成(并行收集模式)")
	log.Printf("监控磁盘挂载点: %v", agentConfig.Collection.Disk.MountPoints)
	log.Printf("监控网络接口: %v", agentConfig.Collection.Network.Interfaces)

	// 如果不是调试模式，则初始化上报模块
	var metricsReporter reporter.Reporter
	var serverURL string
	var nodeID string
	var agentToken string

	if !*debug {
		// 初始化数据上报模块
		// var serverURL string // 移除内部声明
		// 优先使用命令行参数，否则使用配置文件
		if *serverAddr != "localhost:8080" {
			// 检查serverAddr是否已包含协议前缀
			if strings.HasPrefix(*serverAddr, "http://") || strings.HasPrefix(*serverAddr, "https://") {
				serverURL = *serverAddr
			} else {
				serverURL = "http://" + *serverAddr
			}
		} else if agentConfig.Aggregator.Enabled && agentConfig.Aggregator.URL != "" {
			// 如果启用了聚合服务器，优先使用聚合服务器地址
			serverURL = agentConfig.Aggregator.URL
			agentToken = agentConfig.Aggregator.AuthToken // 获取用于注册的 token
		} else if agentConfig.Server.URL != "" {
			serverURL = agentConfig.Server.URL
			// agentToken = agentConfig.Server.Token // 如果直连主控，理论上也需要token，配置文件里目前没有
		} else {
			serverURL = "http://localhost:8080"
			// agentToken 保持空
		}

		// 获取主机名作为节点ID
		// var nodeID string // 移除内部声明
		nodeID = agentConfig.Node.ID
		if nodeID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				hostname = "unknown-node"
				log.Println("警告: 无法获取主机名，使用 'unknown-node' 作为节点ID")
			}
			nodeID = hostname
			log.Printf("未在配置中指定节点ID，使用主机名: %s", nodeID)
		} else {
			log.Printf("使用配置中的节点ID: %s", nodeID)
		}

		// 创建HTTP上报器并附加安全配置
		httpReporter := reporter.NewHTTPReporter(
			serverURL,
			nodeID,
			reporter.WithRetryCount(agentConfig.Server.RetryCount),
			reporter.WithRetryInterval(time.Duration(agentConfig.Server.RetryInterval)*time.Second),
			reporter.WithTimeout(time.Duration(getAppropriateTimeout(agentConfig, serverURL))*time.Second),
			reporter.WithSecurityConfig(&agentConfig.Security),
		)

		// 如果启用了聚合服务器，设置认证令牌 (这个是用于上报指标的，注册时用 agentToken)
		// if agentConfig.Aggregator.Enabled && agentConfig.Aggregator.AuthToken != "" {
		// 	httpReporter.SetAuthToken(agentConfig.Aggregator.AuthToken)
		// }

		metricsReporter = httpReporter
		log.Printf("数据上报模块初始化完成，目标服务器: %s\n", serverURL)

		// 日志安全配置状态
		if agentConfig.Security.Encryption.Enabled {
			log.Printf("数据加密已启用，算法: %s", agentConfig.Security.Encryption.Algorithm)
		} else {
			log.Println("数据加密未启用")
		}

		if agentConfig.Security.Compression.Enabled {
			log.Printf("数据压缩已启用，算法: %s, 级别: %d", agentConfig.Security.Compression.Algorithm, agentConfig.Security.Compression.Level)
		} else {
			log.Println("数据压缩未启用")
		}

		// --- 添加注册逻辑 ---
		registrationSuccessful := false
		if agentConfig.Aggregator.Enabled {
			if agentToken != "" {
				log.Printf("聚合服务器已启用，开始注册节点 %s 到 %s...", nodeID, serverURL)
				err := attemptRegistration(serverURL, nodeID, agentToken)
				if err != nil {
					errorLogger.Printf("向聚合服务器注册失败 (重试 %d 次后): %v", maxRegisterRetries, err)
					log.Printf("警告: 向聚合服务器注册失败，上报请求可能被拒绝。错误: %v", err)
					// registrationSuccessful remains false
				} else {
					log.Printf("节点 %s 成功注册到聚合服务器 %s", nodeID, serverURL)
					registrationSuccessful = true
				}
			} else {
				log.Println("警告: 聚合服务器已启用，但未在配置中找到 aggregator.auth_token，无法执行注册。节点将标记为未验证。")
				// registrationSuccessful remains false
			}
		} else {
			log.Println("聚合服务器未启用，直接连接主控端，跳过聚合器注册。")
			registrationSuccessful = true // Assume direct connection is allowed for now
		}
		log.Printf("节点注册状态: %v", registrationSuccessful) // Log final registration status
		// --- 注册逻辑结束 ---

	} else {
		log.Println("调试模式启用，将只打印收集的数据而不上报")
		// 调试模式也需要 nodeID
		nodeID = agentConfig.Node.ID
		if nodeID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				hostname = "unknown-node"
			}
			nodeID = hostname
		}
	}

	// 设置实际的采集间隔
	collectionInterval := time.Duration(agentConfig.Collection.Interval) * time.Millisecond
	log.Printf("采用实际采集间隔: %v\n", collectionInterval)

	// 启动定时采集任务
	ticker := time.NewTicker(collectionInterval)
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

	// 关闭错误日志文件
	if errFile != nil {
		errFile.Close()
	}
}

// attemptRegistration 尝试向聚合服务器注册 Agent，带重试逻辑
func attemptRegistration(aggregatorURL, nodeID, token string) error {
	var lastErr error
	for i := 0; i <= maxRegisterRetries; i++ {
		if i > 0 {
			log.Printf("注册重试 (%d/%d)，等待 %v 后重试...", i, maxRegisterRetries, registerRetryInterval)
			time.Sleep(registerRetryInterval)
		}
		log.Printf("尝试注册 (第 %d 次)...", i+1)
		err := registerAgentWithAggregator(aggregatorURL, nodeID, token)
		if err == nil {
			log.Printf("注册成功 (第 %d 次尝试)", i+1)
			return nil // 成功
		}
		log.Printf("注册尝试 %d 失败: %v", i+1, err)
		lastErr = err
	}
	return fmt.Errorf("注册失败，已重试 %d 次: %w", maxRegisterRetries, lastErr)
}

// registerAgentWithAggregator 执行单次注册尝试
func registerAgentWithAggregator(aggregatorURL, nodeID, token string) error {
	registerURL := fmt.Sprintf("%s/api/v1/nodes/register", aggregatorURL)
	payload := map[string]string{
		"node_id": nodeID,
		"token":   token,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化注册负载失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // 注册超时
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", registerURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建注册请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SysLens-Agent/Register")

	client := &http.Client{Timeout: 15 * time.Second}
	log.Printf("发送注册请求到 %s for node %s", registerURL, nodeID)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("注册请求错误 for node %s: %v", nodeID, err)
		return fmt.Errorf("发送注册请求失败: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("收到注册响应 for node %s: Status %d", nodeID, resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("注册失败响应体 for node %s: %s", nodeID, string(respBody))
		return fmt.Errorf("注册请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil // 注册成功
}

// loadConfig 从文件加载配置并支持环境变量替换
func loadConfig(path string) (*config.AgentConfig, error) {
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

	var cfg config.AgentConfig
	if err := yaml.Unmarshal([]byte(result), &cfg); err != nil {
		return nil, err
	}

	// 确保关键配置有默认值
	ensureDefaultConfig(&cfg)

	return &cfg, nil
}

// ensureDefaultConfig 确保关键配置项有合理的默认值
func ensureDefaultConfig(cfg *config.AgentConfig) {
	// 确保磁盘挂载点配置
	if len(cfg.Collection.Disk.MountPoints) == 0 {
		cfg.Collection.Disk.MountPoints = []string{"/"}
	}

	// 确保网络接口配置（空切片表示所有接口）
	if cfg.Collection.Network.Interfaces == nil {
		cfg.Collection.Network.Interfaces = []string{}
	}

	// 确保采集间隔合理
	if cfg.Collection.Interval <= 0 {
		cfg.Collection.Interval = 500 // 默认500毫秒
	}

	// 确保重试设置合理
	if cfg.Server.RetryCount <= 0 {
		cfg.Server.RetryCount = 3
	}

	if cfg.Server.RetryInterval <= 0 {
		cfg.Server.RetryInterval = 1
	}

	// 确保超时时间合理
	if cfg.Server.Timeout <= 0 {
		cfg.Server.Timeout = 10
	}
}

// collectAndReport 收集并上报系统指标
func collectAndReport(collector collector.Collector, reporter reporter.Reporter, debugMode bool) {
	collectTime := time.Now().Format("2006-01-02 15:04:05")
	log.Println("开始采集系统指标...")

	// 收集指标
	startTime := time.Now()
	stats, err := collector.Collect()
	if err != nil {
		errorLogger.Printf("采集指标失败: %v\n", err)
		return
	}

	elapsedTime := time.Since(startTime)
	log.Printf("系统指标采集完成，耗时: %v\n", elapsedTime)

	if debugMode {
		// 调试模式，只打印关键指标
		log.Printf("CPU使用率: %.2f%%\n", stats.CPU["usage"])
		log.Printf("内存使用率: %.2f%%\n", stats.Memory.UsedPercent)

		// 磁盘信息
		log.Printf("收集到 %d 个磁盘分区信息\n", len(stats.Disk))
		for mountPoint, diskInfo := range stats.Disk {
			log.Printf("  - 挂载点: %s, 使用率: %.2f%%, 总空间: %.2f GB\n",
				mountPoint,
				diskInfo.UsedPercent,
				float64(diskInfo.Total)/(1024*1024*1024))
		}

		// 网络信息
		log.Printf("收集到 %d 个网络接口信息\n", len(stats.Network.Interfaces))
		for iface, netInfo := range stats.Network.Interfaces {
			log.Printf("  - 接口: %s, 上传速度: %.2f KB/s, 下载速度: %.2f KB/s\n",
				iface,
				float64(netInfo.UploadSpeed)/1024,
				float64(netInfo.DownloadSpeed)/1024)
		}

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
			// 详细记录上报失败信息
			errMsg := fmt.Sprintf("上报失败 [时间点: %s] - 错误: %v", collectTime, err)
			errorLogger.Printf("%s\n", errMsg)

			// 提取更具体的错误信息
			if strings.Contains(err.Error(), "connection refused") {
				errorLogger.Println("原因: 主控服务器可能未启动或无法访问")
			} else if strings.Contains(err.Error(), "timeout") {
				errorLogger.Println("原因: 连接主控服务器超时")
			} else if strings.Contains(err.Error(), "no such host") {
				errorLogger.Println("原因: 无法解析主控服务器主机名")
			}

			// 在标准日志中也输出简要信息
			log.Printf("上报失败: %v - 详细错误已记录到错误日志", err)
			log.Println("将继续采集数据，即使上报失败")

			// 保存失败数据到本地缓存文件
			saveFailedReportData(stats, collectTime)
		} else {
			log.Printf("系统指标上报成功 [时间点: %s]\n", collectTime)
		}
	}
}

// saveFailedReportData 将上报失败的数据保存到本地文件
func saveFailedReportData(stats *collector.SystemStats, timestamp string) {
	// 创建缓存目录
	cacheDir := "tmp/failed_reports"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		errorLogger.Printf("创建缓存目录失败: %v", err)
		return
	}

	// 生成文件名，使用时间戳确保唯一性
	safeTime := strings.ReplaceAll(timestamp, ":", "-")
	filename := filepath.Join(cacheDir, fmt.Sprintf("metrics_%s.json", safeTime))

	// 序列化数据
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		errorLogger.Printf("序列化失败数据失败: %v", err)
		return
	}

	// 写入文件
	if err := os.WriteFile(filename, data, 0644); err != nil {
		errorLogger.Printf("保存失败数据到文件失败: %v", err)
		return
	}

	errorLogger.Printf("已保存上报失败的数据到: %s", filename)
}

// getAppropriateTimeout 获取合适的超时时间
func getAppropriateTimeout(agentConfig *config.AgentConfig, serverURL string) int {
	// 如果启用了聚合服务器且URL匹配聚合服务器地址，使用聚合服务器的超时配置
	if agentConfig.Aggregator.Enabled && strings.Contains(serverURL, strings.TrimPrefix(agentConfig.Aggregator.URL, "${AGGREGATOR_URL:-")) {
		log.Printf("使用聚合服务器超时配置: %d秒", agentConfig.Aggregator.Timeout)
		return agentConfig.Aggregator.Timeout
	}

	// 否则使用主控服务器的超时配置
	log.Printf("使用主控服务器超时配置: %d秒", agentConfig.Server.Timeout)
	return agentConfig.Server.Timeout
}
