#!/bin/bash

# 启动SysLens节点代理的脚本

# 设置默认参数
SERVER_URL=${SERVER_URL:-"localhost:8080"}  # 移除默认的http://前缀
CONFIG_PATH=${CONFIG_PATH:-"configs/agent.yaml"}
INTERVAL=${INTERVAL:-500}  # 默认500毫秒
DEBUG=${DEBUG:-false}

# 添加URL规范化处理
if [[ ! "$SERVER_URL" =~ ^https?:// ]]; then
    SERVER_URL="http://$SERVER_URL"
    echo "已将服务器URL标准化为: $SERVER_URL"
fi

# 创建输出目录
mkdir -p logs

# 输出配置信息
echo "启动SysLens节点代理..."
echo "配置文件路径: $CONFIG_PATH"
echo "连接到服务器: $SERVER_URL"
echo "采集间隔: ${INTERVAL}毫秒"
if [ "$DEBUG" = "true" ]; then
    echo "调试模式: 启用"
else
    echo "调试模式: 禁用"
fi

# 检查配置文件是否存在
if [ ! -f "$CONFIG_PATH" ]; then
    echo "警告: 配置文件 $CONFIG_PATH 不存在，将使用默认配置"
    
    # 如果模板存在，则复制为配置文件
    if [ -f "configs/agent.template.yaml" ]; then
        cp configs/agent.template.yaml "$CONFIG_PATH"
        echo "已从模板创建配置文件: $CONFIG_PATH"
    fi
fi

# 此处添加连接测试
echo "测试主控端连接..."
# 使用兼容macOS的方式提取URL
BASE_URL=$(echo $SERVER_URL | sed -E 's|(https?://[^/]+).*|\1|')
if [ -z "$BASE_URL" ]; then
    BASE_URL=$SERVER_URL
fi
curl -s "${BASE_URL}/health" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "警告: 无法连接到主控端 ${BASE_URL}"
    echo "请确保主控端已启动并且URL正确"
    
    # 在调试模式下不退出，其他情况询问用户是否继续
    if [ "$DEBUG" != "true" ]; then
        read -p "是否仍然继续启动节点代理? (y/n) " answer
        if [ "$answer" != "y" ] && [ "$answer" != "Y" ]; then
            echo "节点代理启动已取消。"
            exit 1
        fi
    fi
else
    echo "主控端连接测试成功!"
fi

# 提取主机名作为节点标识
NODE_ID=${NODE_ID:-$(hostname)}
echo "节点标识: $NODE_ID"

# 启动节点代理
DEBUG_FLAG=""
if [ "$DEBUG" = "true" ]; then
    DEBUG_FLAG="--debug"
fi

go run cmd/agent/main.go \
    --config "$CONFIG_PATH" \
    --server "$SERVER_URL" \
    --interval "$INTERVAL" \
    $DEBUG_FLAG 