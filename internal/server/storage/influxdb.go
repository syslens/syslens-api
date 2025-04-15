package storage

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// InfluxDBStorage 提供基于InfluxDB的指标存储实现
type InfluxDBStorage struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	queryAPI api.QueryAPI
	org      string
	bucket   string
}

// NewInfluxDBStorage 创建新的InfluxDB存储实例
func NewInfluxDBStorage(url, token, org, bucket string) *InfluxDBStorage {
	// 创建InfluxDB客户端
	client := influxdb2.NewClient(url, token)

	// 初始化资源（组织和存储桶）
	if err := ensureInfluxDBResources(client, org, bucket); err != nil {
		log.Printf("初始化InfluxDB资源时出错: %v", err)
		// 继续执行，因为错误可能是资源已存在
	}

	// 获取写入和查询API
	writeAPI := client.WriteAPI(org, bucket)
	queryAPI := client.QueryAPI(org)

	// 返回存储实例
	return &InfluxDBStorage{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
		org:      org,
		bucket:   bucket,
	}
}

// ensureInfluxDBResources 确保必要的InfluxDB资源已存在
func ensureInfluxDBResources(client influxdb2.Client, orgName, bucketName string) error {
	ctx := context.Background()

	// 获取组织API
	orgAPI := client.OrganizationsAPI()

	// 尝试直接写入数据以触发自动创建资源
	log.Printf("尝试初始化InfluxDB资源: 组织=%s, 存储桶=%s", orgName, bucketName)

	// 写入一条测试点，如果组织和存储桶不存在，将会自动创建
	writeAPI := client.WriteAPI(orgName, bucketName)
	p := influxdb2.NewPoint(
		"system",
		map[string]string{"test": "setup"},
		map[string]interface{}{"value": 1},
		time.Now(),
	)
	writeAPI.WritePoint(p)
	writeAPI.Flush()

	// 检查是否成功创建
	_, err := orgAPI.FindOrganizationByName(ctx, orgName)
	if err != nil {
		return fmt.Errorf("确认组织创建失败: %w", err)
	}

	log.Printf("InfluxDB资源初始化成功")
	return nil
}

// StoreMetrics 存储节点指标数据
func (s *InfluxDBStorage) StoreMetrics(nodeID string, metrics interface{}) error {
	// 转换metrics为map
	metricsMap, ok := metrics.(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的指标格式: 期望map[string]interface{}, 实际为%T", metrics)
	}

	// 提取时间戳，默认为当前时间
	timestamp := time.Now()
	if ts, ok := metricsMap["timestamp"].(time.Time); ok {
		timestamp = ts
	}

	// 提取主机名和平台信息作为标签
	tags := map[string]string{
		"node_id": nodeID,
	}

	if hostname, ok := metricsMap["hostname"].(string); ok {
		tags["hostname"] = hostname
	}

	if platform, ok := metricsMap["platform"].(string); ok {
		tags["platform"] = platform
	}

	// 创建CPU指标点
	if cpu, ok := metricsMap["cpu"].(map[string]interface{}); ok {
		for key, value := range cpu {
			p := influxdb2.NewPoint(
				"cpu",
				tags,
				map[string]interface{}{key: value},
				timestamp,
			)
			s.writeAPI.WritePoint(p)
		}
	}

	// 创建内存指标点
	if memory, ok := metricsMap["memory"].(map[string]interface{}); ok {
		p := influxdb2.NewPoint(
			"memory",
			tags,
			memory,
			timestamp,
		)
		s.writeAPI.WritePoint(p)
	}

	// 创建磁盘指标点
	if disk, ok := metricsMap["disk"].(map[string]interface{}); ok {
		for mountPoint, info := range disk {
			if diskInfo, ok := info.(map[string]interface{}); ok {
				diskTags := make(map[string]string)
				for k, v := range tags {
					diskTags[k] = v
				}
				diskTags["mount_point"] = mountPoint

				p := influxdb2.NewPoint(
					"disk",
					diskTags,
					diskInfo,
					timestamp,
				)
				s.writeAPI.WritePoint(p)
			}
		}
	}

	// 创建网络指标点
	if network, ok := metricsMap["network"].(map[string]interface{}); ok {
		// 总体网络统计
		netStats := make(map[string]interface{})

		// 提取除interfaces外的字段
		for key, value := range network {
			if key != "interfaces" {
				netStats[key] = value
			}
		}

		// 创建总体网络指标点
		if len(netStats) > 0 {
			p := influxdb2.NewPoint(
				"network",
				tags,
				netStats,
				timestamp,
			)
			s.writeAPI.WritePoint(p)
		}

		// 创建每个接口的网络指标点
		if interfaces, ok := network["interfaces"].(map[string]interface{}); ok {
			for iface, info := range interfaces {
				if ifaceInfo, ok := info.(map[string]interface{}); ok {
					ifaceTags := make(map[string]string)
					for k, v := range tags {
						ifaceTags[k] = v
					}
					ifaceTags["interface"] = iface

					p := influxdb2.NewPoint(
						"network_interface",
						ifaceTags,
						ifaceInfo,
						timestamp,
					)
					s.writeAPI.WritePoint(p)
				}
			}
		}
	}

	// 异步提交
	s.writeAPI.Flush()

	// 记录详细的写入信息
	var metricsTypes []string
	pointCounts := make(map[string]int)

	if metricsMap["cpu"] != nil {
		metricsTypes = append(metricsTypes, "cpu")
		pointCounts["cpu"] = 1
	}
	if metricsMap["memory"] != nil {
		metricsTypes = append(metricsTypes, "memory")
		pointCounts["memory"] = 1
	}
	if diskMap, ok := metricsMap["disk"].(map[string]interface{}); ok {
		metricsTypes = append(metricsTypes, "disk")
		if partitions, hasParts := diskMap["partitions"].([]interface{}); hasParts {
			pointCounts["disk"] = len(partitions)
		} else {
			pointCounts["disk"] = 1
		}
	}
	if networkMap, ok := metricsMap["network"].(map[string]interface{}); ok {
		metricsTypes = append(metricsTypes, "network")
		pointCounts["network"] = 1
		if interfaces, hasIfaces := networkMap["interfaces"].(map[string]interface{}); hasIfaces {
			pointCounts["network_interfaces"] = len(interfaces)
		}
	}

	log.Printf("[信息] InfluxDB写入开始 - 节点: %s, 时间戳: %s, 指标类型: %v",
		nodeID,
		timestamp.Format(time.RFC3339),
		strings.Join(metricsTypes, ", "))

	// 记录写入的点数
	totalPoints := 0
	for _, count := range pointCounts {
		totalPoints += count
	}

	log.Printf("[详细] InfluxDB写入点数统计 - 节点: %s, 总点数: %d, 详情: %v",
		nodeID,
		totalPoints,
		pointCounts)

	// 捕获异步写入错误
	errChan := make(chan error, 1)

	go func() {
		for err := range s.writeAPI.Errors() {
			errChan <- err
			log.Printf("[错误] InfluxDB写入失败 - 节点: %s, 错误: %v", nodeID, err)
		}
	}()

	// 等待一小段时间，让异步错误有机会被捕获
	select {
	case err := <-errChan:
		return fmt.Errorf("InfluxDB写入错误: %w", err)
	case <-time.After(100 * time.Millisecond):
		// 继续执行
	}

	log.Printf("[信息] InfluxDB写入成功 - 节点: %s, 数据点: %d, 指标类型: %v",
		nodeID,
		totalPoints,
		strings.Join(metricsTypes, ", "))

	return nil
}

// GetNodeMetrics 获取指定节点在时间范围内的指标
func (s *InfluxDBStorage) GetNodeMetrics(nodeID string, start, end time.Time) ([]interface{}, error) {
	// 构建Flux查询语句
	query := fmt.Sprintf(`
from(bucket: "%s")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => r["node_id"] == "%s")
  |> pivot(rowKey:["_time"], columnKey: ["_measurement"], valueColumn: "_value")
`,
		s.bucket,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		nodeID,
	)

	// 执行查询
	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询InfluxDB失败: %w", err)
	}
	defer result.Close()

	// 处理结果
	var metrics []interface{}
	for result.Next() {
		record := result.Record()
		metricMap := make(map[string]interface{})

		// 提取时间戳
		metricMap["timestamp"] = record.Time()

		// 提取所有字段
		for k, v := range record.Values() {
			if k != "_time" && k != "node_id" && k != "_measurement" {
				metricMap[k] = v
			}
		}

		metrics = append(metrics, metricMap)
	}

	// 检查查询过程中是否有错误
	if result.Err() != nil {
		return nil, fmt.Errorf("处理查询结果时出错: %w", result.Err())
	}

	return metrics, nil
}

// GetAllNodes 获取所有节点ID列表
func (s *InfluxDBStorage) GetAllNodes() ([]string, error) {
	// 使用Flux查询获取所有唯一的node_id
	query := fmt.Sprintf(`
import "distinct"

from(bucket: "%s")
  |> range(start: -7d)
  |> distinct(column: "node_id")
  |> group()
`,
		s.bucket,
	)

	// 执行查询
	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询InfluxDB失败: %w", err)
	}
	defer result.Close()

	// 提取唯一的节点ID
	var nodes []string
	for result.Next() {
		if nodeID, ok := result.Record().ValueByKey("node_id").(string); ok {
			nodes = append(nodes, nodeID)
		}
	}

	// 检查查询过程中是否有错误
	if result.Err() != nil {
		return nil, fmt.Errorf("处理查询结果时出错: %w", result.Err())
	}

	return nodes, nil
}

// GetLatestMetrics 获取指定节点的最新指标
func (s *InfluxDBStorage) GetLatestMetrics(nodeID string) (interface{}, error) {
	// 构建Flux查询语句获取最新数据
	query := fmt.Sprintf(`
from(bucket: "%s")
  |> range(start: -5m)
  |> filter(fn: (r) => r["node_id"] == "%s")
  |> last()
  |> pivot(rowKey:["_time"], columnKey: ["_measurement"], valueColumn: "_value")
`,
		s.bucket,
		nodeID,
	)

	// 执行查询
	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询InfluxDB失败: %w", err)
	}
	defer result.Close()

	// 处理结果
	latestMetrics := make(map[string]interface{})
	if result.Next() {
		record := result.Record()

		// 提取时间戳
		latestMetrics["timestamp"] = record.Time()

		// 提取所有字段
		for k, v := range record.Values() {
			if k != "_time" && k != "node_id" && k != "_measurement" {
				latestMetrics[k] = v
			}
		}
	} else {
		return nil, nil // 没有找到数据
	}

	// 检查查询过程中是否有错误
	if result.Err() != nil {
		return nil, fmt.Errorf("处理查询结果时出错: %w", result.Err())
	}

	return latestMetrics, nil
}

// Close 关闭InfluxDB连接
func (s *InfluxDBStorage) Close() {
	s.writeAPI.Flush()
	s.client.Close()
}
