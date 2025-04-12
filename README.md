# SysLens - 服务器监控系统

SysLens是一个分布式服务器监控系统，由主控端和节点端组成，可实时监控和分析多服务器环境的系统指标。

## 功能特点

- **毫秒级监控**：支持500毫秒以上的高频数据采集
- **自动资源初始化**：自动创建所需的InfluxDB组织和存储桶
- **分布式架构**：主控/节点分离，可监控大规模服务器集群
- **多指标采集**：CPU、内存、磁盘、网络、进程等全方位监控
- **连接健康检查**：启动前自动检测主控连接状态
- **可扩展存储**：支持内存存储和InfluxDB时序数据库
- **告警通知**：可配置的告警规则和通知渠道

## 系统架构

系统由两个主要组件构成：

- **主控端(Control Plane)**: 接收、存储、分析和展示来自所有节点的监控数据
- **节点端(Node Agent)**: 部署在每台被监控服务器上，负责收集本地系统指标并上报至主控端

## 目录结构

项目采用标准Go项目布局，结构如下：

```bash
syslens-api/
├── cmd/                    # 可执行文件入口
│   ├── server/             # 主控端入口
│   │   └── main.go
│   ├── agent/              # 节点端入口
│   │   └── main.go
│   └── collectors/         # 辅助工具
│       └── collect_stats.go # 系统指标收集测试工具
├── internal/               # 内部私有代码
│   ├── agent/              # 节点端核心逻辑
│   │   ├── collector/      # 系统指标收集
│   │   └── reporter/       # 数据上报模块
│   ├── server/             # 主控端核心逻辑
│   │   ├── api/            # HTTP/gRPC API接口
│   │   └── storage/        # 数据存储层
│   ├── discovery/          # 节点注册与发现
│   ├── telemetry/          # 指标、日志、追踪
│   ├── alerting/           # 告警规则与通知
│   ├── dashboard/          # 数据可视化接口
│   ├── common/             # 公共代码
│   │   ├── models/         # 数据模型定义
│   │   └── utils/          # 通用工具函数
│   └── config/             # 配置处理
├── pkg/                    # 可被外部项目引用的库
│   └── metrics/            # 通用指标定义与处理
├── api/                    # API契约定义
│   └── proto/              # Protobuf定义(gRPC)
├── web/                    # 前端资源
│   ├── static/             # 静态资源文件
│   └── templates/          # HTML模板
├── configs/                # 配置文件模板
│   ├── server.yaml         # 主控端配置
│   └── agent.yaml          # 节点端配置
├── deployments/            # 部署配置
│   ├── docker/             # Docker相关配置
│   └── kubernetes/         # K8s资源定义
├── scripts/                # 构建和辅助脚本
├── migrations/             # 数据库迁移脚本
├── docs/                   # 项目文档
│   ├── architecture.md     # 架构设计文档
│   └── api.md              # API使用文档
├── test/                   # 测试资源与工具
├── go.mod                  # Go模块定义
├── go.sum                  # 依赖版本锁定
└── README.md               # 项目说明文档
```

## 功能模块说明

### 节点端(Agent)功能

- **指标收集器(collector)**: 收集CPU、内存、磁盘、网络等系统指标
- **上报模块(reporter)**: 将收集的指标定期上报至主控端

### 主控端(Server)功能

- **API服务**: 提供HTTP/gRPC接口，接收节点上报数据，响应查询请求
- **存储层**: 管理监控数据的存储与检索
- **节点管理**: 维护节点注册信息，监控节点状态
- **告警系统**: 基于规则触发告警，支持多种通知方式
- **可视化接口**: 为前端仪表盘提供数据支持

## 监控指标

系统采集的核心指标包括但不限于：

- **CPU**: 使用率、负载、核心数等
- **内存**: 总量、使用量、使用率、交换分区状态
- **磁盘**: 使用率、I/O状态、读写速度
- **网络**: 流量、连接数、带宽使用情况，公网/内网IP地址
- **进程**: 重要进程资源占用情况
- **系统信息**: 主机名、平台、运行时间等

## 技术栈

- **后端**: Go语言 (1.18+)
- **通信**: HTTP REST API
- **数据存储**: InfluxDB 2.x 时序数据库
- **配置格式**: YAML
- **监控组件**: gopsutil、shirou/gopsutil
- **前端**: (可选)基于Web的仪表盘界面

## 使用教程

### 系统要求

- Go 1.16+
- 支持Linux、macOS、Windows操作系统
- （可选）Docker 和 Docker Compose 用于容器化部署

### 安装步骤

#### 方式一：从源码构建

1. 克隆代码仓库：

```bash
git clone https://github.com/syslens/syslens-api.git
cd syslens-api
```

2. 安装依赖：

```bash
go mod tidy
go mod download
```

3. 构建项目：

```bash
# 使用Makefile构建
make build-all

# 或使用构建脚本
./scripts/build.sh --all
```

构建完成后，二进制文件将生成在`bin/`目录下：

- `bin/server`: 主控端可执行文件
- `bin/agent`: 节点端可执行文件

#### 方式二：使用Docker

1. 使用Docker Compose构建和运行：

```bash
cd deployments/docker
docker-compose up -d
```

这将同时启动主控端和节点端服务。

### 配置说明

#### 初始配置

在首次运行前，请从模板创建配置文件:

```bash
cp configs/server.template.yaml configs/server.yaml
cp configs/agent.template.yaml configs/agent.yaml
```

然后根据您的环境编辑这些文件。

#### 主控端配置

配置文件位于`configs/server.yaml`，主要配置项包括：

- **监听地址**: 默认为`0.0.0.0:8080`
- **存储设置**: 可选内存、文件或InfluxDB时序数据库
- **告警规则**: 可配置各类资源的告警阈值
- **通知方式**: 支持邮件、Webhook等

InfluxDB相关配置：

```yaml
storage:
  type: "influxdb"
  influxdb:
    url: "http://localhost:8086"
    token: "your-influxdb-token"
    org: "syslens"
    bucket: "metrics"
    retention_days: 30
```

#### 节点端配置

配置文件位于`configs/agent.yaml`，主要配置项包括：

- **服务器地址**: 主控端的连接地址
- **节点标识**: 节点的唯一标识和标签
- **采集间隔**: 数据采集的时间间隔（毫秒）
- **采集项目**: 可启用或禁用特定资源的监控

示例配置修改：

```yaml
# 设置节点标签
node:
  labels:
    environment: "production"
    role: "database"

# 修改采集间隔为500毫秒
collection:
  interval: 500
```

### 构建与运行工具

SysLens提供了两种运行和构建方式，分别适用于不同场景：

#### Makefile vs 启动脚本

| 工具类型 | 特点 | 适用场景 |
|---------|------|---------|
| **Makefile** | 简洁命令、直接调用、无额外检查 | 开发环境、快速测试 |
| **脚本(scripts/)** | 错误处理、环境变量配置、健康检查 | 生产环境、自动化部署 |

两者并不冲突，而是针对不同需求设计的互补工具：

- **Makefile命令**：

  ```bash
  make build-all    # 构建所有组件
  make run-server   # 运行主控端(无初始化和检查)
  make run-agent    # 运行节点端(无连接测试)
  ```

- **启动脚本**：

  ```bash
  ./scripts/start-server-with-influxdb.sh  # 包含InfluxDB初始化
  ./scripts/start-agent.sh                 # 包含连接测试和URL规范化
  ./scripts/build.sh --all                 # 高级构建选项
  ```

**建议**：开发测试时使用Makefile，生产部署时使用脚本。

### 运行服务

#### 使用启动脚本（推荐）

1. 运行主控端（需要设置InfluxDB令牌）：

```bash
export INFLUXDB_TOKEN=your-token-here
./scripts/start-server-with-influxdb.sh
```

2. 运行节点端（可指定主控端地址和采集间隔）：

```bash
SERVER_URL=http://192.168.1.100:8080 INTERVAL=1000 ./scripts/start-agent.sh
```

#### 直接运行

1. 运行主控端：

```bash
./bin/server -config configs/server.yaml -storage influxdb -influx-token your-token-here
```

2. 运行节点端（在每台需要监控的服务器上）：

```bash
./bin/agent -config configs/agent.yaml -server http://主控端IP:8080 -interval 500
```

### 访问监控数据

启动服务后，可通过以下方式访问监控数据：

1. **API接口**：
   - 获取所有节点列表：`GET http://主控端IP:8080/api/v1/nodes`
   - 获取节点指标数据：`GET http://主控端IP:8080/api/v1/nodes/metrics?node_id=<节点ID>`

2. **Web界面**（如果已实现）：
   在浏览器中访问 `http://主控端IP:8080`

### 常见操作示例

#### 查看节点实时状态

```bash
curl -X GET http://主控端IP:8080/api/v1/nodes/metrics?node_id=web-server-01
```

#### 查询特定时间范围的数据

```bash
curl -X GET "http://主控端IP:8080/api/v1/nodes/metrics?node_id=web-server-01&start=2023-06-01T10:00:00Z&end=2023-06-01T11:00:00Z"
```

### 实用工具

SysLens提供了一些实用工具来帮助您进行开发、测试和故障排查：

#### 系统指标收集测试

可以使用独立工具收集并查看系统指标，方便测试：

```bash
go run cmd/collectors/collect_stats.go
```

此工具会收集当前系统的所有指标并输出为JSON格式，同时保存到`tmp/system_stats.json`文件中。

### 故障排除

1. **节点无法连接主控端**
   - 启动脚本会自动进行连接测试，显示警告信息
   - 检查网络连接和防火墙设置
   - 确认主控端地址配置正确，注意URL格式
   - 查看节点日志: `cat logs/agent.log`

2. **数据采集异常**
   - 检查节点机器的权限设置
   - 使用收集测试工具验证: `go run cmd/collectors/collect_stats.go`
   - 查看详细日志：修改`configs/agent.yaml`中的日志级别为`debug`
   - 重启节点代理：`kill -SIGTERM <进程ID>` 然后重新启动

3. **主控端未接收数据**
   - 检查API服务是否正常运行：`curl http://主控端IP:8080/health`
   - 检查节点ID是否正确发送（X-Node-ID头部）
   - 查看服务日志：`cat logs/server.log`
   - 检查InfluxDB连接：`curl -I http://localhost:8086/ping`

4. **InfluxDB数据问题**
   - 主控端会自动初始化InfluxDB资源
   - 确认令牌权限正确: 需要读写权限
   - 在InfluxDB UI中正确调整时间范围查询数据
   - 检查错误日志: `grep "InfluxDB" logs/server.log`

### 更新和升级

1. 拉取最新代码：

```bash
git pull origin main
```

2. 重新构建并部署：

```bash
make build-all
# 停止旧服务，启动新服务
```

### 配置告警规则

编辑`configs/server.yaml`中的告警配置部分，添加所需的告警规则，例如：

```yaml
alerting:
  rules:
    - name: "high-cpu-usage"
      condition: "cpu.usage > 90"
      duration: "5m"
      severity: "warning"
```

重启主控端服务使配置生效。
