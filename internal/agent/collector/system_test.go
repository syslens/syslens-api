package collector

import (
	"fmt"
	"testing"
	"time"
)

func TestSystemCollector(t *testing.T) {
	// 创建收集器
	sc := NewSystemCollector()

	// 执行收集
	stats, err := sc.Collect()
	if err != nil {
		t.Fatalf("收集器执行失败: %v", err)
	}

	// 基本验证
	t.Run("基本信息验证", func(t *testing.T) {
		if stats.Hostname == "" {
			t.Error("主机名为空")
		}
		if stats.Platform == "" {
			t.Error("平台信息为空")
		}
		if time.Since(stats.Timestamp) > 5*time.Second {
			t.Error("时间戳异常")
		}
	})

	// CPU信息验证
	t.Run("CPU信息验证", func(t *testing.T) {
		if usage, ok := stats.CPU["usage"]; !ok || usage < 0 || usage > 100 {
			t.Errorf("CPU使用率异常: %v", usage)
		}

		if stats.Hardware.CPUModel == "" {
			t.Error("CPU型号为空")
		}

		if stats.Hardware.CPUCores <= 0 {
			t.Errorf("CPU核心数异常: %d", stats.Hardware.CPUCores)
		}
	})

	// 内存信息验证
	t.Run("内存信息验证", func(t *testing.T) {
		if stats.Memory.Total <= 0 {
			t.Errorf("内存总量异常: %d", stats.Memory.Total)
		}

		if stats.Memory.Used > stats.Memory.Total {
			t.Errorf("内存使用量异常: 使用=%d, 总量=%d", stats.Memory.Used, stats.Memory.Total)
		}

		fmt.Printf("内存信息: 总量=%d, 使用=%d, 使用率=%.2f%%\n",
			stats.Memory.Total, stats.Memory.Used, stats.Memory.UsedPercent)
	})

	// 磁盘信息验证
	t.Run("磁盘信息验证", func(t *testing.T) {
		if len(stats.Disk) == 0 {
			t.Error("未收集到磁盘信息")
		}

		for mount, info := range stats.Disk {
			if info.Total <= 0 {
				t.Errorf("磁盘 %s 容量异常: %d", mount, info.Total)
			}

			if info.Used > info.Total {
				t.Errorf("磁盘 %s 使用量异常: 使用=%d, 总量=%d",
					mount, info.Used, info.Total)
			}

			fmt.Printf("磁盘 %s: 总量=%d, 使用=%d, 使用率=%.2f%%\n",
				mount, info.Total, info.Used, info.UsedPercent)
		}
	})

	// 网络信息验证
	t.Run("网络信息验证", func(t *testing.T) {
		if len(stats.Network.Interfaces) == 0 {
			t.Error("未收集到网络接口信息")
		}

		fmt.Printf("网络接口数量: %d\n", len(stats.Network.Interfaces))
		fmt.Printf("IP地址: 公网IPv4=%v, 内网IPv4=%v\n",
			stats.Network.PublicIPv4, stats.Network.PrivateIPv4)
	})
}
