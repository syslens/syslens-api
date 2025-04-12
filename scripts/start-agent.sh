#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 设置默认参数
NODE_ID=${NODE_ID:-""}
NODE_ENV=${NODE_ENV:-"production"}
NODE_ROLE=${NODE_ROLE:-"web"}
SERVER_URL=${SERVER_URL:-"http://localhost:8080"}
SERVER_TOKEN=${SERVER_TOKEN:-""}
COLLECTION_INTERVAL=${COLLECTION_INTERVAL:-500}
ENCRYPTION_KEY=${ENCRYPTION_KEY:-"syslens-default-security-key-change-me"}
LOG_LEVEL=${LOG_LEVEL:-"info"}
LOG_FILE=${LOG_FILE:-"logs/agent.log"}

# 导出环境变量以供配置文件使用
export NODE_ID
export NODE_ENV
export NODE_ROLE
export SERVER_URL
export SERVER_TOKEN
export COLLECTION_INTERVAL
export ENCRYPTION_KEY
export LOG_LEVEL
export LOG_FILE

# 显示配置信息
echo -e "${GREEN}启动SysLens节点代理...${NC}"
echo "服务器地址: $SERVER_URL"
echo "采集间隔: ${COLLECTION_INTERVAL}ms"
echo "环境: $NODE_ENV"
if [ -n "$NODE_ID" ]; then
    echo "节点ID: $NODE_ID"
else
    echo "节点ID: 将自动生成"
fi

# 标准化服务器URL格式
if [[ ! "$SERVER_URL" == http* ]]; then
    SERVER_URL="http://$SERVER_URL"
    echo -e "${YELLOW}已修正服务器URL格式为: $SERVER_URL${NC}"
    export SERVER_URL
fi

# 创建日志目录
if [[ "$LOG_FILE" != "" ]]; then
    LOG_DIR=$(dirname "$LOG_FILE")
    mkdir -p "$LOG_DIR"
    echo "日志将写入: $LOG_FILE"
fi

# 测试服务器连接
echo -e "${GREEN}正在测试与主控服务器的连接...${NC}"
CONNECTION_TEST=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/health" 2>/dev/null)

if [ "$CONNECTION_TEST" == "200" ]; then
    echo -e "${GREEN}服务器连接正常，状态码: $CONNECTION_TEST${NC}"
else
    echo -e "${YELLOW}警告: 无法连接到服务器 $SERVER_URL (状态码: $CONNECTION_TEST)${NC}"
    echo -e "${YELLOW}节点将尝试运行，但可能无法成功上报数据${NC}"
fi

# 运行节点代理
echo -e "${GREEN}正在启动节点代理...${NC}"

# 检测是否为开发模式
if [ -z "$GO_ENV" ] || [ "$GO_ENV" = "development" ]; then
    # 开发模式：使用go run
    go run cmd/agent/main.go
else
    # 生产模式：使用编译的二进制文件
    if [ -f "bin/agent" ]; then
        ./bin/agent
    else
        echo -e "${YELLOW}未找到编译后的二进制文件，切换到使用go run...${NC}"
        go run cmd/agent/main.go
    fi
fi 