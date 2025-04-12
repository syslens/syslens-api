#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 检查InfluxDB令牌是否已设置
if [ -z "$INFLUXDB_TOKEN" ]; then
    echo -e "${YELLOW}警告: 未设置INFLUXDB_TOKEN环境变量，将使用配置文件中的值${NC}"
fi

# 设置默认参数
INFLUXDB_URL=${INFLUXDB_URL:-"http://localhost:8086"}
INFLUXDB_ORG=${INFLUXDB_ORG:-"syslens"}
INFLUXDB_BUCKET=${INFLUXDB_BUCKET:-"metrics"}
STORAGE_TYPE=${STORAGE_TYPE:-"influxdb"}
ENCRYPTION_KEY=${ENCRYPTION_KEY:-"syslens-default-security-key-change-me"}

# 导出环境变量以供配置文件使用
export INFLUXDB_URL
export INFLUXDB_TOKEN
export INFLUXDB_ORG
export INFLUXDB_BUCKET
export STORAGE_TYPE
export ENCRYPTION_KEY

# 显示配置信息
echo -e "${GREEN}启动SysLens主控服务...${NC}"
echo "存储类型: $STORAGE_TYPE"
if [ "$STORAGE_TYPE" = "influxdb" ]; then
    echo "InfluxDB URL: $INFLUXDB_URL"
    echo "InfluxDB 组织: $INFLUXDB_ORG"
    echo "InfluxDB 存储桶: $INFLUXDB_BUCKET"
    
    # 使用***显示部分令牌，增强安全性
    if [ -n "$INFLUXDB_TOKEN" ]; then
        TOKEN_LENGTH=${#INFLUXDB_TOKEN}
        VISIBLE_PART=${INFLUXDB_TOKEN:0:4}
        HIDDEN_PART=$(printf "%*s" $((TOKEN_LENGTH-4)) | tr ' ' '*')
        echo "InfluxDB Token: $VISIBLE_PART$HIDDEN_PART"
    else
        echo -e "${YELLOW}InfluxDB Token: 未设置，使用配置文件中的值${NC}"
    fi
fi

# 创建输出目录
mkdir -p logs

# 运行服务端
echo -e "${GREEN}正在启动主控服务...${NC}"

# 检测是否为开发模式
if [ -z "$GO_ENV" ] || [ "$GO_ENV" = "development" ]; then
    # 开发模式：使用go run
    go run cmd/server/main.go
else
    # 生产模式：使用编译的二进制文件
    if [ -f "bin/server" ]; then
        ./bin/server
    else
        echo -e "${YELLOW}未找到编译后的二进制文件，切换到使用go run...${NC}"
        go run cmd/server/main.go
    fi
fi 