package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/syslens/syslens-api/internal/agent/collector"
)

func main() {
	// 创建系统指标收集器
	systemCollector := collector.NewSystemCollector()

	fmt.Println("正在收集系统指标...")

	// 收集指标
	stats, err := systemCollector.Collect()
	if err != nil {
		log.Fatalf("收集指标失败: %v", err)
	}

	// 将结果格式化为JSON
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		log.Fatalf("序列化数据失败: %v", err)
	}

	// 输出到控制台
	fmt.Println(string(data))

	// 也保存到文件方便检查
	os.WriteFile("system_stats.json", data, 0644)
	fmt.Println("系统指标已保存到 system_stats.json")
}
