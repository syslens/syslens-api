package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/server.yaml", "配置文件路径")
	flag.Parse()

	// 日志初始化
	log.Println("SysLens服务端启动中...")
	log.Printf("使用配置文件: %s\n", *configPath)

	// TODO: 加载配置文件

	// TODO: 初始化存储

	// TODO: 初始化API服务

	// TODO: 启动节点管理服务

	// TODO: 启动告警服务

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("服务端正在关闭...")
	// TODO: 关闭各服务
	log.Println("服务端已安全退出")
}
