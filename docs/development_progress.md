# SysLens 开发进度报告 (自动生成)

**报告生成时间:** {{TIMESTAMP}}

## 概述

SysLens是一个分布式服务器监控系统，包含主控端、聚合服务器和节点代理。当前项目代码包含主控端的核心实现，以及聚合服务器和节点代理的基本框架和功能。

## 主要组件与文件分析

### 1. 主控端 (Control Plane)

**启动与核心逻辑:**

- **`cmd/server/main.go`**:
  - 应用程序入口点。
  - 解析命令行参数（配置路径、监听地址、存储类型等）。
  - 加载服务器配置 (`configs/server.yaml`)，支持环境变量替换。
  - 初始化日志系统 (标准库 log 和 zap)。
  - 初始化存储后端 (PostgreSQL, InfluxDB, Memory)。
  - 执行数据库迁移和表结构验证。
  - 初始化数据仓库 (Repositories)。
  - 初始化 API 服务 (Gin router, handlers)。
  - 应用安全配置（加密、压缩）。
  - 启动 HTTP 服务并处理优雅关闭。

**API 服务 (`internal/server/api`)**:

- **`router.go`**:
  - 使用 Gin 框架设置 API 路由。
  - 定义全局中间件 (Recovery, Logging, RequestID)。
  - 包含 `/health` 健康检查端点。
  - 按资源组织 API v1 路由 (`/api/v1`)：nodes, groups, services, alerts, notifications。
  - 注册 Swagger 文档路由。
  - 定义了节点配置获取和更新的路由 (`/nodes/configuration`, `/nodes/{node_id}/configuration`)。
- **`gin_handlers.go`**:
  - 实现了各个 API 端点的具体处理逻辑 (Gin Handlers)。
  - `HandleGetAllNodesGin`: 获取所有节点。
  - `HandleGetNodeMetricsGin`: 获取节点指标数据。
  - `HandleRegisterNodeGin`: 处理节点注册和信息更新，包含 Token 生成、哈希和加密存储，以及默认配置初始化。
  - `HandleGetNodeTokenGin`: 恢复（解密）节点认证令牌。
  - `HandleMetricsSubmitGin`: 接收节点或聚合器上报的指标数据，处理解密和解压缩。
  - `HandleGetNodeConfigurationGin`: 根据节点 Token 获取其配置，包含默认配置处理。
  - `HandleUpdateNodeConfigurationGin`: 更新指定节点的配置（需要 Token 和 NodeID）。
  - `validateNodeAuthentication`: 辅助函数，验证节点 Token。
  - `getDefaultNodeConfiguration`: 辅助函数，生成默认节点配置。
  - `validateNodeConfiguration`: 辅助函数，验证配置数据的有效性。
  - (其他 Handler 占位符): groups, services, alerts, notifications 相关接口的占位实现。
- **`handler.go`**:
  - 定义 `MetricsHandler` 结构体，包含存储、仓库、安全配置、加密服务和日志记录器的依赖。
  - 定义 `MetricsStorage` 接口，抽象指标存储操作。
  - 提供 `NewMetricsHandler` 构造函数。
  - 提供 `WithSecurityConfig`, `WithLogger`, `WithNodeRepository` 等方法用于依赖注入。
  - 包含 `processData` 方法，处理传入数据的解密和解压缩。
- **`models.go`**:
  - 定义 API 请求和响应中使用的基本数据结构，如 `Node`, `NodeMetrics`, `NodeRegisterRequest`, `StatusUpdateRequest`。
- **`response.go`**:
  - 定义标准的 API 响应结构 (`Response`, `ErrorResponse`)。
  - 提供 `RespondWithError` 和 `RespondWithSuccess` 辅助函数，用于生成统一格式的 JSON 响应。

**数据仓库 (`internal/server/repository`)**:

- **`node_repository.go`**:
  - 定义 `Node` 结构体，包含节点所有属性（ID, Name, Token, Labels, Configuration, Status 等）。
  - 定义 `NodeRepository` 接口，抽象节点数据操作。
  - `PostgresNodeRepository`: 接口的 PostgreSQL 实现。
  - 实现 CRUD 操作 (Create, GetByID, GetAll, Update, Delete)。
  - 实现按状态、分组、服务 ID 查询节点的方法。
  - `ValidateNodeToken`: 使用 `ComparePasswordAndHash` 验证提供的 Token 与存储的哈希是否匹配。
  - `UpdateConfiguration`: 更新节点的 `configuration` 字段。
  - `FindByToken`: 通过遍历所有节点并验证 Token 哈希来查找节点。
  - `scanNodes`: 辅助函数，用于从数据库行扫描节点数据，包含 JSON 字段的解析。
- **`user_repository.go`**: (推测) 用于用户账号信息的存储和管理。
- **`node_group_repository.go`**: (推测) 用于节点分组信息的存储和管理。
- **`service_repository.go`**: (推测) 用于固定服务定义的存储和管理。
- **`alerting_rule_repository.go`**: (推测) 用于告警规则的存储和管理。
- **`notification_repository.go`**: (推测) 用于告警通知记录的存储和管理。

**数据存储 (`internal/server/storage`)**:

- **`postgres.go`**:
  - 定义 `PostgresConfig` 结构体。
  - `NewPostgresDB`: 创建并配置 PostgreSQL 数据库连接（连接池、超时等）。
  - 提供数据库操作的封装方法 (ExecContext, QueryContext, QueryRowContext, BeginTx, PrepareContext)。
  - `CheckDatabaseHealth`: 检查数据库连接和查询能力。
- **`migration.go`**:
  - 定义所有数据库表的 `CREATE TABLE` SQL 语句，包括 `nodes` 表的 `configuration` 字段。
  - `MigrateDatabase`: 执行数据库迁移，使用 `schema_migrations` 表跟踪已应用的迁移。
  - `CheckTablesExist`: 检查所需表是否存在。
  - `VerifyTableColumns`: 验证表结构是否包含所有必需的列。
- **`influxdb.go`**:
  - `InfluxDBStorage`: InfluxDB 存储后端实现。
  - `NewInfluxDBStorage`: 初始化 InfluxDB 客户端、WriteAPI 和 QueryAPI。
  - `ensureInfluxDBResources`: 确保所需的组织和 Bucket 存在。
  - `StoreMetrics`: 将指标数据转换为 InfluxDB Point 格式并写入。
  - `GetNodeMetrics`, `GetAllNodes`, `GetLatestMetrics`: 实现指标查询。
- **`memory.go`**:
  - `MemoryStorage`: 基于内存的简单存储实现，主要用于测试或开发。
  - 使用 map 存储数据，有最大条目限制。

**中间件 (`internal/server/middleware`)**:

- **`logging.go`**:
  - Gin 中间件，使用 Zap 记录每个 HTTP 请求的详细信息（方法、路径、状态码、延迟等）。
  - 将 logger 实例注入 Gin 上下文。
- **`request_id.go`**:
  - Gin 中间件，为每个请求生成唯一 ID（基于时间戳）或使用 `X-Request-ID` 头。
  - 将请求 ID 注入 Gin 上下文和响应头。

### 2. 聚合服务器 (Aggregator)

**启动与核心逻辑:**

- **`cmd/aggregator/main.go`**:
  - 聚合服务器应用程序入口点。
  - 解析命令行参数 (`--config`, `--version`)。
  - 加载聚合服务器配置 (`configs/aggregator.yaml`)。
  - 初始化并启动聚合服务器实例 (`internal/aggregator/server.go`)。
  - 处理优雅关闭信号。
- **`internal/aggregator/server.go`**:
  - 定义 `Server` 结构体，包含配置、HTTP 服务器、路由、日志、连接管理、数据处理器和主控端客户端。
  - 定义 `NodeConnection` 结构体，存储连接节点的信息（ID, LastActive, Status, Verified）。
  - `NewServer`: 初始化服务器及所有依赖项（Logger, Router, Processor, ControlPlaneClient, EncryptionService）。
  - `initLogger`: 初始化 Zap 日志。
  - `initRouter`: 设置 Gin 路由，包括健康检查和 API v1 端点。
    - `/api/v1/nodes/:node_id/metrics`: 接收节点指标（带认证）。
    - `/api/v1/nodes/register`: 处理节点注册（由 Agent 调用）。
    - `/api/v1/nodes/:node_id/heartbeat`: 处理节点心跳（带认证）。
    - `/api/v1/nodes`: 获取当前连接的节点列表。
  - `authMiddleware`: Gin 中间件，检查请求的节点是否已注册和验证。
  - `Start`/`Shutdown`: 启动和关闭 HTTP 服务器及相关 goroutine。
  - `handleNodeMetrics`: 处理指标上报请求，进行数据处理并交给 `DataProcessor`。
  - `processIncomingData`: 辅助函数，处理数据的解密和解压缩。
  - `handleNodeRegister`: 处理 Agent 的注册请求，调用 `ControlPlaneClient.ValidateNode` 验证 Token。
  - `handleNodeHeartbeat`: 更新节点的最后活动时间。
  - `handleGetNodes`: 返回当前管理的节点列表。
  - `registerOrUpdateNode`: 添加或更新内部节点连接记录。
  - `updateNodeActivity`: 更新内部节点连接的最后活动时间。
  - `cleanupExpiredConnections`/`cleanupConnections`: (推测) 定期清理不活跃的节点连接。

**数据处理与转发:**

- **`internal/aggregator/processor.go`**:
  - 定义 `DataProcessor` 结构体，负责缓存和转发指标数据。
  - `NewDataProcessor`: 初始化处理器。
  - `Start`/`Shutdown`: 启动和停止指标处理 goroutine。
  - `ProcessMetrics`: 将接收到的指标数据存入内部缓存 (`metrics.data`)。
  - `processMetrics`: 定期运行的 goroutine，调用 `processMetricsData`。
  - `processMetricsData`: 获取缓存快照，处理数据（添加时间戳），并调用 `forwardMetricsToControlPlane`。
  - `forwardMetricsToControlPlane`: 将处理后的指标数据通过 HTTP POST 请求转发给主控端 API (`/api/v1/nodes/{node_id}/metrics`)，包含认证头和聚合器 ID 头 (`X-Aggregator-ID`)。

**主控端交互:**

- **`internal/aggregator/client.go`**:
  - 定义 `ControlPlaneClient` 结构体，用于和主控端 API 通信。
  - `NewControlPlaneClient`: 初始化 HTTP 客户端。
  - `RegisterNode`: (似乎未被server使用) 向主控端 API 发送节点注册信息。
  - `UpdateNodeStatus`: (似乎未被server使用) 向主控端 API 更新节点状态。
  - `GetNodeConfig`: (似乎未被server使用) 从主控端 API 获取节点配置。
  - `ValidateNode`: 向主控端的 `/api/v1/nodes/register` 发送请求，以验证 Agent 提供的 Token 是否有效。

### 3. 节点代理 (Agent)

**启动与核心逻辑:**

- **`cmd/agent/main.go`**:
  - 节点代理应用程序入口点。
  - 解析命令行参数 (`--config`, `--server`, `--interval`, `--debug`)。
  - 加载节点配置 (`configs/agent.yaml`)，支持环境变量替换。
  - 初始化错误日志记录器。
  - 初始化数据采集器 (SystemCollector 或 ParallelCollector)。
  - 初始化数据上报器 (HTTPReporter)。
    - 根据配置决定目标是主控端还是聚合服务器。
  - **注册**: 如果配置了聚合服务器，启动时调用 `attemptRegistration` 尝试向聚合服务器注册，发送节点 ID 和配置中的 `Aggregator.AuthToken`。
  - 启动定时采集和上报任务 (`collectAndReport`)。
  - 处理优雅关闭信号。
- **`internal/agent/reporter/reporter.go`**:
  - 定义 `Reporter` 接口。
  - `HTTPReporter`: 实现了 `Reporter` 接口。
    - `NewHTTPReporter`: 初始化上报器，可配置重试、超时、安全选项。
    - `Report`: 将数据序列化为 JSON，根据安全配置进行压缩和加密，然后通过 HTTP POST 发送给目标服务器 (`/api/v1/nodes/{node_id}/metrics`)。包含重试逻辑。
    - `processData`: 辅助函数，处理数据的压缩和加密。
    - **注意**: 当前上报未使用 `AgentConfig.Server.Token` 或 `AgentConfig.Aggregator.AuthToken` 进行认证，而是依赖聚合服务器或主控端的其他验证机制（如 IP 或注册状态）。

**数据采集 (`internal/agent/collector`)**:

- **`system.go`**:
  - `SystemCollector`: 顺序采集 CPU, Memory, Disk, Network, Load, Host Info 等指标。
  - 使用 `gopsutil` 库。
  - 定义 `SystemStats` 及相关子结构体。
  - 计算网络速率。
- **`parallel.go`**:
  - `ParallelCollector`: 并行版本的数据采集器，使用 goroutine 提高效率。
- **`cmd/collectors/collect_stats.go`**:
  - 独立的调试工具，用于测试数据采集器并将结果输出到 JSON 文件。

### 4. 通用工具 (`internal/common/utils`)

- **`crypto.go`**: 提供密码哈希 (bcrypt) 和随机字符串/令牌生成功能。
- **`security.go`**: 提供 AES-256-GCM 加密/解密和 Gzip 压缩/解压缩服务。

### 5. 配置模型 (`internal/config`)

- **`model.go`**: 定义 `ServerConfig`, `AgentConfig` 等核心配置结构体，对应 YAML 文件。
- **`aggregator_config.go`**: 定义 `AggregatorConfig` 结构体。

## 当前主要功能点总结

- **主控端**: 节点管理 (CRUD, 状态, 配置, Token), 指标存储 (PostgreSQL, InfluxDB), API 服务 (Gin, Swagger), 数据处理 (加解密, 压缩), 数据库迁移。
- **聚合服务器**: 接收 Agent 连接与指标, 管理连接状态, 缓存并批量转发指标到主控端, 验证 Agent Token (通过主控端)。
- **节点代理**: 系统指标采集 (顺序/并行), 指标上报 (HTTP, 重试, 加密/压缩), 向聚合服务器注册。
- **通用**: 加密/哈希工具, 安全工具, 配置模型。

## 后续开发建议

- **实现配置分发**: 修改 Agent 启动逻辑，使其通过 Token 从聚合服务器 (或主控端) 获取配置，而不是依赖本地 `agent.yaml` 的 `Collection` 等部分。聚合服务器需要添加转发配置请求的 API。
- **完善 Agent 认证**: 明确 Agent 上报指标时使用的认证方式（当前 `HTTPReporter` 未使用 Token）。
- **用户认证与授权**: 实现主控端的完整用户管理和权限控制。
- **Web 界面**: 开发前端。
- **告警系统**: 实现告警规则匹配和通知。
- **单元/集成测试**: 增加测试覆盖。
- **文档**: 完善 API 和部署文档。
