#!/bin/bash

# 构建脚本 - 用于构建SysLens组件

# 设置错误时退出
set -e

# 项目根目录
PROJECT_ROOT=$(dirname "$(dirname "$(readlink -f "$0")")")
cd "$PROJECT_ROOT"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 显示帮助信息
function show_help {
    echo -e "${YELLOW}SysLens构建脚本${NC}"
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -a, --all       构建所有组件"
    echo "  -s, --server    只构建主控端"
    echo "  -n, --agent     只构建节点端"
    echo "  -d, --docker    构建Docker镜像"
    echo "  -h, --help      显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 --all        构建所有组件"
    echo "  $0 --docker     使用Docker构建所有组件"
}

# 构建主控端
function build_server {
    echo -e "${GREEN}构建主控端...${NC}"
    go build -o bin/server ./cmd/server
    echo -e "${GREEN}主控端构建完成: bin/server${NC}"
}

# 构建节点端
function build_agent {
    echo -e "${GREEN}构建节点端...${NC}"
    go build -o bin/agent ./cmd/agent
    echo -e "${GREEN}节点端构建完成: bin/agent${NC}"
}

# 构建Docker镜像
function build_docker {
    echo -e "${GREEN}构建Docker镜像...${NC}"
    
    echo -e "${GREEN}构建主控端镜像...${NC}"
    docker build -t syslens/server -f deployments/docker/Dockerfile.server .
    
    echo -e "${GREEN}构建节点端镜像...${NC}"
    docker build -t syslens/agent -f deployments/docker/Dockerfile.agent .
    
    echo -e "${GREEN}Docker镜像构建完成${NC}"
    docker images | grep syslens
}

# 解析命令行参数
if [ $# -eq 0 ]; then
    show_help
    exit 1
fi

# 创建输出目录
mkdir -p bin

# 处理参数
while [ "$1" != "" ]; do
    case $1 in
        -a | --all )          build_server
                              build_agent
                              ;;
        -s | --server )       build_server
                              ;;
        -n | --agent )        build_agent
                              ;;
        -d | --docker )       build_docker
                              ;;
        -h | --help )         show_help
                              exit
                              ;;
        * )                   show_help
                              exit 1
    esac
    shift
done

echo -e "${GREEN}构建完成${NC}" 