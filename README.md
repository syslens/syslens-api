# SysLens - 服务器监控系统

SysLens是一个分布式服务器监控系统，由主控端和节点端组成，可实时监控和分析多服务器环境的系统指标。

![版本](https://img.shields.io/badge/版本-1.1.0-blue) ![开发工具](https://img.shields.io/badge/IDE-VSCode-green) ![Go](https://img.shields.io/badge/Go-1.24+-success) ![Gin](https://img.shields.io/badge/Gin-1.10+-blueviolet) ![InfluxDB](https://img.shields.io/badge/InfluxDB-2.x-informational) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-blue) ![Docker](https://img.shields.io/badge/Docker-blue)

[![GitHub stars](https://img.shields.io/github/stars/syslens/syslens-api)](https://github.com/syslens/syslens-api/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/syslens/syslens-api)](https://github.com/syslens/syslens-api/network/members)
[![GitHub issues](https://img.shields.io/github/issues/syslens/syslens-api)](https://github.com/syslens/syslens-api/issues)

## 访问量

![Profile views counter](https://komarev.com/ghpvc/?username=syslens&style=flat&color=blue) 
[![Visitor Count](https://profile-counter.glitch.me/syslens/syslens-api/count.svg)](https://github.com/syslens/syslens-api)

## 功能特点

- **毫秒级监控**：支持500毫秒以上的高频数据采集
- **自动资源初始化**：自动创建所需的InfluxDB组织和存储桶
- **分布式架构**：主控/节点分离，可监控大规模服务器集群
- **多指标采集**：CPU、内存、磁盘、网络、进程等全方位监控
- **连接健康检查**：启动前自动检测主控连接状态
- **可扩展存储**：支持内存存储和InfluxDB时序数据库
- **告警通知**：可配置的告警规则和通知渠道

## 数据传输安全

SysLens提供了强大的数据传输安全机制，确保监控数据在传输过程中的安全性和高效性：

### 数据加密

节点上报数据时支持加密传输，防止敏感监控数据被截获和篡改：

- **多种加密算法**：支持AES-256-GCM等高强度加密标准
- **密钥管理**：支持通过配置文件或环境变量设置加密密钥
- **选择性启用**：可根据安全需求灵活开启或关闭加密功能

#### 加密实现细节

加密功能由`internal/common/utils/security.go`中的`EncryptionService`实现：

- **加密过程**：
  1. 使用AES-256-GCM算法进行加密，提供认证加密功能
  2. 生成随机nonce确保加密安全性
  3. 对加密后的数据进行Base64编码便于HTTP传输
  4. 自动处理密钥长度（不足自动填充，过长则截断为32字节）

- **加密调用流程**：
  1. `reporter.Report` → 序列化数据为JSON
  2. `reporter.processData` → 处理数据（先压缩后加密）
  3. `utils.EncryptionService.Encrypt` → 执行具体加密操作
  4. 通过HTTP POST发送加密数据，设置`Content-Type: application/octet-stream`

- **解密过程**：
  1. 服务端`api.MetricsHandler.HandleMetricsSubmit` → 接收加密数据
  2. `api.MetricsHandler.processData` → 处理数据（先解密后解压）
  3. `utils.EncryptionService.Decrypt` → 执行具体解密操作

### 数据压缩

为了减少网络带宽占用并提高传输效率，系统支持数据压缩功能：

- **压缩算法**：支持Gzip等通用压缩算法
- **可调压缩级别**：支持1-9级压缩，可根据CPU资源和网络状况调整
- **高频采集优化**：特别适合高频采集场景下的数据传输

#### 压缩实现细节

压缩功能由`internal/common/utils/security.go`中的`CompressData`和`DecompressData`函数实现：

- **压缩过程**：
  1. 使用gzip.NewWriterLevel创建指定压缩级别的压缩器
  2. 将数据写入压缩缓冲区
  3. 返回压缩后的字节数组

- **压缩调用流程**：
  1. `reporter.Report` → 序列化数据为JSON
  2. `reporter.processData` → 处理数据（先压缩）
  3. `utils.CompressData` → 执行具体压缩操作
  4. 如果还需加密，则对压缩后的数据进行加密

- **解压过程**：
  1. 服务端接收到数据
  2. 如果数据已加密，先解密
  3. `utils.DecompressData` → 使用gzip.NewReader创建解压缩器
  4. 读取解压缩后的原始数据

### 安全配置示例

节点端配置文件中的安全设置示例：

```yaml
security:
  # 数据加密配置
  encryption:
    enabled: true          # 启用加密
    algorithm: aes-256-gcm # 加密算法
    key: "${ENCRYPTION_KEY:-default_dev_key}"  # 从环境变量读取密钥，有默认值
  
  # 数据压缩配置
  compression:
    enabled: true          # 启用压缩
    algorithm: gzip        # 压缩算法
    level: 6               # 压缩级别(1-9)，数字越大压缩率越高但CPU消耗也越大
```

### 安全处理流程

节点数据上报和服务端处理的完整流程：

1. **节点端初始化**：

   ```go
   // 创建HTTPReporter时传入安全配置
   httpReporter := reporter.NewHTTPReporter(
       serverURL, 
       nodeID,
       reporter.WithSecurityConfig(&agentConfig.Security)
   )
   ```

2. **数据处理流程**：

   ```go
   // 节点端：processData处理数据顺序
   func (r *HTTPReporter) processData(data []byte) ([]byte, string, error) {
       // 1. 压缩（如果启用）
       if r.securityConfig.Compression.Enabled {
           processedData, err = utils.CompressData(processedData, r.securityConfig.Compression.Level)
       }
       
       // 2. 加密（如果启用）
       if r.securityConfig.Encryption.Enabled && r.encryptionSvc != nil {
           processedData, err = r.encryptionSvc.Encrypt(processedData, r.securityConfig.Encryption.Key)
       }
   }
   
   // 服务端：反向处理数据顺序
   func (h *MetricsHandler) processData(data []byte, isEncrypted, isCompressed bool) ([]byte, error) {
       // 1. 解密（如果启用）
       if isEncrypted && h.securityConfig.Encryption.Enabled && h.encryptionSvc != nil {
           processedData, err = h.encryptionSvc.Decrypt(processedData, h.securityConfig.Encryption.Key)
       }
       
       // 2. 解压缩（如果启用）
       if isCompressed && h.securityConfig.Compression.Enabled {
           processedData, err = utils.DecompressData(processedData)
       }
   }
   ```

3. **安全标志传递**：
   - 节点端通过HTTP头部告知服务端数据是否经过加密/压缩：

     ```yaml
     X-Data-Encrypted: true
     X-Data-Compressed: true
     ```

   - 服务端根据这些头部标志确定如何处理数据

### 安全最佳实践

- 生产环境中始终启用加密功能，特别是当监控数据通过公网传输时
- 通过环境变量注入密钥，避免在配置文件中明文存储
- 根据网络带宽和监控频率权衡压缩级别
- 定期更换加密密钥提高安全性
- 同时启用HTTPS（TLS）为传输层提供额外保护

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
│   ├── collectors/         # 辅助工具
│   │   └── collect_stats.go # 系统指标收集测试工具
│   └── test/               # 测试命令工具目录(预留)
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
├── test/                   # 集成测试与测试工具(预留)
├── tmp/                    # 临时文件目录(不纳入版本控制)
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
- **重试设置**: 上报失败后的重试次数和间隔时间
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
  
# 设置重试参数
server:
  retry_count: 5        # 最大重试5次
  retry_interval: 2     # 每次重试间隔2秒
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

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=syslens/syslens-api&type=Date)](https://www.star-history.com/#syslens/syslens-api&Date)
