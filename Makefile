.PHONY: build-server build-agent build-aggregator build-all test clean deps run-server run-agent run-aggregator docs fmt lint help migrate-up migrate-down migrate-create migrate-goto migrate-force migrate-version migrate-drop

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
	@echo "检查代码..."
	gofmt -l -s -w .
	go vet ./...
	golangci-lint run # 确保安装了 golangci-lint

# 数据库迁移 (需要安装 golang-migrate CLI)
# 从 configs/server.yaml 读取数据库配置
MIGRATIONS_PATH=migrations/postgres
CONFIG_FILE=configs/server.yaml

# 只提取包含关键字的行，具体解析在命令内部进行
PG_HOST_LINE := $(shell grep '^ *host:' $(CONFIG_FILE))
PG_PORT_LINE := $(shell grep '^ *port:' $(CONFIG_FILE))
PG_USER_LINE := $(shell grep '^ *user:' $(CONFIG_FILE))
PG_PASSWORD_LINE := $(shell grep '^ *password:' $(CONFIG_FILE))
PG_DBNAME_LINE := $(shell grep '^ *dbname:' $(CONFIG_FILE))
PG_SSLMODE_LINE := $(shell grep '^ *sslmode:' $(CONFIG_FILE))

# 定义一个可复用的命令块来获取数据库URL
# 这个块会在调用它的目标的 shell 环境中执行
define _GET_DB_URL_CMDS
PG_HOST_RAW=$$(echo "$(PG_HOST_LINE)" | sed -e 's/^ *host:[ \t]*//' -e 's/[ \t]*#.*$$//'); \
PG_PORT_RAW=$$(echo "$(PG_PORT_LINE)" | sed -e 's/^ *port:[ \t]*//' -e 's/[ \t]*#.*$$//'); \
PG_USER_RAW=$$(echo "$(PG_USER_LINE)" | sed -e 's/^ *user:[ \t]*//' -e 's/[ \t]*#.*$$//'); \
PG_PASSWORD_RAW=$$(echo "$(PG_PASSWORD_LINE)" | sed -e 's/^ *password:[ \t]*//' -e 's/[ \t]*#.*$$//'); \
PG_DBNAME_RAW=$$(echo "$(PG_DBNAME_LINE)" | sed -e 's/^ *dbname:[ \t]*//' -e 's/[ \t]*#.*$$//'); \
PG_SSLMODE_RAW=$$(echo "$(PG_SSLMODE_LINE)" | sed -e 's/^ *sslmode:[ \t]*//' -e 's/[ \t]*#.*$$//'); \
\
HOST=$${SYSLENS_POSTGRES_HOST:-$$(echo $$PG_HOST_RAW | sed -e 's/^$${SYSLENS_POSTGRES_HOST:-\(.*\)}/\1/')}; \
PORT=$${SYSLENS_POSTGRES_PORT:-$$(echo $$PG_PORT_RAW | sed -e 's/^$${SYSLENS_POSTGRES_PORT:-\(.*\)}/\1/')}; \
USER=$${SYSLENS_POSTGRES_USER:-$$(echo $$PG_USER_RAW | sed -e 's/^$${SYSLENS_POSTGRES_USER:-\(.*\)}/\1/')}; \
PASSWORD=$${SYSLENS_POSTGRES_PASSWORD:-$$(echo $$PG_PASSWORD_RAW | sed -e 's/^$${SYSLENS_POSTGRES_PASSWORD:-\(.*\)}/\1/')}; \
DBNAME=$${SYSLENS_POSTGRES_DB:-$$(echo $$PG_DBNAME_RAW | sed -e 's/^$${SYSLENS_POSTGRES_DB:-\(.*\)}/\1/')}; \
SSLMODE=$${SYSLENS_POSTGRES_SSLMODE:-$$(echo $$PG_SSLMODE_RAW | sed -e 's/^$${SYSLENS_POSTGRES_SSLMODE:-\(.*\)}/\1/')}; \
\
DATABASE_URL="postgres://$${USER}:$${PASSWORD}@$${HOST}:$${PORT}/$${DBNAME}?sslmode=$${SSLMODE}"; \
echo "正在连接: postgres://$${USER}:***@$${HOST}:$${PORT}/$${DBNAME}?sslmode=$${SSLMODE}";
endef

migrate-up:
	@echo "应用数据库迁移..."
	@$(call _GET_DB_URL_CMDS) \
	 migrate -database "$${DATABASE_URL}" -path $(MIGRATIONS_PATH) up

migrate-down:
	@echo "回滚最后一次数据库迁移..."
	@$(call _GET_DB_URL_CMDS) \
	 migrate -database "$${DATABASE_URL}" -path $(MIGRATIONS_PATH) down 1

# 创建新的迁移文件
# 用法: make migrate-create NAME=描述性名称 (例如: make migrate-create NAME=add_user_email_index)
migrate-create:
	@echo "创建新的迁移文件: $(NAME)"
	@if [ -z "$(NAME)" ]; then \
		echo "错误: 请提供迁移名称，例如: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

# 迁移到指定版本
# 用法: make migrate-goto VERSION=<版本号>
migrate-goto:
	@echo "迁移数据库到版本: $(VERSION)"
	@if [ -z "$(VERSION)" ]; then \
		echo "错误: 请提供目标版本号，例如: make migrate-goto VERSION=1"; \
		exit 1; \
	fi
	@$(call _GET_DB_URL_CMDS) \
	 migrate -database "$${DATABASE_URL}" -path $(MIGRATIONS_PATH) goto $(VERSION)

# 强制设置数据库迁移版本 (危险操作!)
# 用法: make migrate-force VERSION=<版本号>
migrate-force:
	@echo "警告: 强制设置数据库版本为 $(VERSION) (不会执行SQL)"
	@read -p "此操作可能导致数据库状态与迁移记录不一致，确定吗？[y/N] " -n 1 -r; echo; \
	 if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		 exit 1; \
	 fi
	@if [ -z "$(VERSION)" ]; then \
		echo "错误: 请提供目标版本号，例如: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@$(call _GET_DB_URL_CMDS) \
	 migrate -database "$${DATABASE_URL}" -path $(MIGRATIONS_PATH) force $(VERSION)

# 查看当前数据库迁移版本和状态
migrate-version:
	@echo "查看数据库迁移版本和状态..."
	@$(call _GET_DB_URL_CMDS) \
	 migrate -database "$${DATABASE_URL}" -path $(MIGRATIONS_PATH) version

# 删除数据库中所有内容 (极度危险操作!)
migrate-drop:
	@echo "警告: 这将删除数据库 '$(PG_DBNAME)' 中的所有表和数据！"
	@read -p "确定要删除所有内容吗？请输入 'yes' 来确认: " confirm && [ "$$confirm" = "yes" ] || exit 1
	@$(call _GET_DB_URL_CMDS) \
	 migrate -database "$${DATABASE_URL}" -path $(MIGRATIONS_PATH) drop -f

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
	@echo "  make run
	-server        - 运行主控端(开发环境)"
	@echo "  make run-agent         - 运行节点端(开发环境)"
	@echo "  make run-aggregator    - 运行聚合服务器(开发环境)"
	@echo "  make docs              - 生成文档"
	@echo "  make fmt               - 格式化代码"
	@echo "  make lint              - 检查代码"
	@echo "  make migrate-up        - 应用所有未应用的数据库迁移"
	@echo "  make migrate-down      - 回滚最后一次数据库迁移"
	@echo "  make migrate-create NAME=<name> - 创建新的数据库迁移文件"
	@echo "  make migrate-goto VERSION=<ver> - 迁移数据库到指定版本"
	@echo "  make migrate-force VERSION=<ver> - 强制设置数据库版本 (危险!)"
	@echo "  make migrate-version   - 查看数据库迁移版本和状态"
	@echo "  make migrate-drop        - 删除数据库中所有内容 (危险!)"
