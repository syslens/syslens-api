#!/bin/bash

# 启动InfluxDB服务器的脚本

# 设置默认参数
CONFIG_PATH=${CONFIG_PATH:-"configs/server.yaml"}
INFLUX_URL=${INFLUX_URL:-"http://localhost:8086"}
INFLUX_ORG=${INFLUX_ORG:-"syslens"}
INFLUX_BUCKET=${INFLUX_BUCKET:-"metrics"}
HTTP_ADDR=${HTTP_ADDR:-"0.0.0.0:8080"}

# 检查InfluxDB token
if [ -z "$INFLUXDB_TOKEN" ]; then
    echo "错误: 环境变量INFLUXDB_TOKEN未设置"
    echo "请设置InfluxDB token: export INFLUXDB_TOKEN=your_token_here"
    exit 1
fi

# 检查配置文件是否存在
if [ ! -f "$CONFIG_PATH" ]; then
    echo "警告: 配置文件 $CONFIG_PATH 不存在，将使用默认配置"
    
    # 如果模板存在，则复制为配置文件
    if [ -f "configs/server.template.yaml" ]; then
        cp configs/server.template.yaml "$CONFIG_PATH"
        echo "已从模板创建配置文件: $CONFIG_PATH"
    fi
fi

# 检查InfluxDB连接
curl -s "$INFLUX_URL/ping" > /dev/null
if [ $? -ne 0 ]; then
    echo "警告: 无法连接到InfluxDB服务器 $INFLUX_URL"
    echo "请确保InfluxDB已启动并可访问"
    # 不退出，因为可能是临时网络问题
fi

# 创建输出目录
mkdir -p logs

# 输出配置信息
echo "启动SysLens服务器..."
echo "配置文件: $CONFIG_PATH"
echo "存储类型: InfluxDB"
echo "InfluxDB URL: $INFLUX_URL"
echo "InfluxDB 组织: $INFLUX_ORG"
echo "InfluxDB Bucket: $INFLUX_BUCKET"
echo "HTTP监听地址: $HTTP_ADDR"

# 启动服务器
go run cmd/server/main.go \
    --config "$CONFIG_PATH" \
    --storage influxdb \
    --influx-url "$INFLUX_URL" \
    --influx-token "$INFLUXDB_TOKEN" \
    --influx-org "$INFLUX_ORG" \
    --influx-bucket "$INFLUX_BUCKET" \
    --addr "$HTTP_ADDR" 