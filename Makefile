.PHONY: build-server build-agent build-aggregator build-all test clean

# 变量定义
BINARY_DIR=bin
SERVER_BINARY=$(BINARY_DIR)/server
AGENT_BINARY=$(BINARY_DIR)/agent
AGGREGATOR_BINARY=$(BINARY_DIR)/aggregator
GO=go
GOFLAGS=-ldflags="-s -w"

# 检查并创建输出目录
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# 构建主控端
build-server: $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $(SERVER_BINARY) ./cmd/server

# 构建节点端
build-agent: $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $(AGENT_BINARY) ./cmd/agent

# 构建聚合服务器
build-aggregator: $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $(AGGREGATOR_BINARY) ./cmd/aggregator

# 构建所有组件
build-all: build-server build-agent build-aggregator

# 运行测试
test:
	$(GO) test -v ./...

# 安装依赖
deps:
	$(GO) mod tidy
	$(GO) mod download

# 清理构建产物
clean:
	rm -rf $(BINARY_DIR)

# 运行主控端（开发环境）
run-server:
	$(GO) run ./cmd/server

# 运行节点端（开发环境）
run-agent:
	$(GO) run ./cmd/agent

# 运行聚合服务器（开发环境）
run-aggregator:
	$(GO) run ./cmd/aggregator

# 生成文档
docs:
	@echo "生成文档..."
	@pandoc -f markdown -t docx -o PRODUCT.docx PRODUCT.md

# 格式化代码
fmt:
	$(GO) fmt ./...

# 检查代码
lint:
	gofmt -l -s -w .
	go vet ./...

# 帮助信息
help:
	@echo "以下是可用的Make命令:"
	@echo "  make build-server      - 构建主控端"
	@echo "  make build-agent       - 构建节点端"
	@echo "  make build-aggregator  - 构建聚合服务器"
	@echo "  make build-all         - 构建所有组件"
	@echo "  make test              - 运行测试"
	@echo "  make deps              - 安装依赖"
	@echo "  make clean             - 清理构建产物"
	@echo "  make run-server        - 运行主控端(开发环境)"
	@echo "  make run-agent         - 运行节点端(开发环境)"
	@echo "  make run-aggregator    - 运行聚合服务器(开发环境)"
	@echo "  make docs              - 生成文档"
	@echo "  make fmt               - 格式化代码"
	@echo "  make lint              - 检查代码" 