# SysLens Golang 代码技术规范

本文档定义了 SysLens 项目中 Golang 代码的编写规范，旨在确保代码质量、可读性和一致性。

## 目录

- [代码风格](#代码风格)
- [命名规范](#命名规范)
- [项目结构](#项目结构)
- [错误处理](#错误处理)
- [日志规范](#日志规范)
- [测试规范](#测试规范)
- [文档规范](#文档规范)
- [性能优化](#性能优化)
- [安全规范](#安全规范)

## 代码风格

### 基本规范

- 使用 `gofmt` 或 `goimports` 自动格式化代码
- 使用 4 个空格缩进，不使用制表符
- 行长度控制在 100 字符以内
- 使用 `go vet` 和 `golint` 检查代码问题
- 遵循 [Effective Go](https://golang.org/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) 中的建议

### 导入规范

- 导入包按标准库、第三方库、内部包的顺序分组
- 每组之间用空行分隔
- 使用绝对路径导入内部包
- 避免使用相对路径导入

```go
import (
    "fmt"
    "io"
    "os"
    
    "github.com/gin-gonic/gin"
    "github.com/spf13/viper"
    
    "github.com/syslens/syslens-api/internal/common"
    "github.com/syslens/syslens-api/internal/models"
)
```

### 注释规范

- 所有导出的函数、类型、变量必须有注释
- 注释以被注释项的名称开头，以句点结尾
- 包注释应放在包声明之前，描述包的功能和用途
- 使用 `// TODO(username):` 格式标记待办事项

```go
// Package collector provides functionality for collecting system metrics.
package collector

// CPUCollector collects CPU-related metrics from the system.
type CPUCollector struct {
    // ...
}

// Collect gathers CPU metrics and returns them as a map.
// Returns an error if collection fails.
func (c *CPUCollector) Collect() (map[string]float64, error) {
    // ...
}
```

## 命名规范

### 包名

- 使用小写单词，不使用下划线或混合大小写
- 包名应该是简短、有意义的名词
- 避免使用与标准库或常用第三方库相同的名称

### 变量名

- 使用驼峰命名法（camelCase）
- 简短但有描述性
- 避免使用单字母变量名（除了循环计数器）
- 布尔变量名应该是 `is`、`has`、`can` 等开头

```go
var (
    isRunning bool
    hasError  bool
    canRetry  bool
    maxRetries int
    serverURL string
)
```

### 常量名

- 使用驼峰命名法（camelCase）或全大写加下划线（UPPER_SNAKE_CASE）
- 对于包级常量，推荐使用全大写加下划线

```go
const (
    DefaultPort = 8080
    MaxConnections = 100
    
    // 包级常量使用全大写加下划线
    MAX_RETRY_COUNT = 3
    DEFAULT_TIMEOUT = 30
)
```

### 函数名

- 使用驼峰命名法（camelCase）
- 函数名应该是动词或动词短语
- 对于导出的函数，首字母大写

```go
func GetUserByID(id string) (*User, error) {
    // ...
}

func validateConfig(config *Config) error {
    // ...
}
```

### 方法名

- 使用驼峰命名法（camelCase）
- 方法名应该是动词或动词短语
- 对于导出的方法，首字母大写

```go
func (s *Server) Start() error {
    // ...
}

func (c *Client) connect() error {
    // ...
}
```

### 接口名

- 使用驼峰命名法（CamelCase）
- 接口名通常是名词或名词短语
- 对于只有一个方法的接口，通常以 "er" 结尾

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type MetricsCollector interface {
    Collect() (map[string]float64, error)
}
```

### 结构体名

- 使用驼峰命名法（CamelCase）
- 结构体名通常是名词或名词短语

```go
type User struct {
    ID        string
    Name      string
    Email     string
    CreatedAt time.Time
}
```

## 项目结构

SysLens 项目遵循标准的 Go 项目布局，主要目录结构如下：

```
syslens-api/
├── .github/                 # GitHub 相关配置
│   ├── workflows/           # GitHub Actions 工作流配置
│   └── ISSUE_TEMPLATE/      # Issue 模板
├── api/                     # API 契约定义
│   ├── proto/               # Protocol Buffers 定义
│   ├── openapi/             # OpenAPI/Swagger 定义
│   └── graphql/             # GraphQL schema 定义
├── build/                   # 构建相关配置和脚本
│   ├── ci/                  # CI 配置
│   ├── package/             # 打包脚本
│   └── docker/              # Docker 构建文件
├── cmd/                     # 可执行文件入口
│   ├── server/              # 主控端入口
│   │   └── main.go
│   ├── agent/               # 节点端入口
│   │   └── main.go
│   ├── aggregator/          # 聚合服务器入口
│   │   └── main.go
│   └── collectors/          # 辅助工具
│       └── collect_stats.go
├── configs/                 # 配置文件模板
│   ├── server.yaml          # 主控端配置
│   ├── agent.yaml           # 节点端配置
│   └── aggregator.yaml      # 聚合服务器配置
├── deployments/             # 部署配置
│   ├── docker/              # Docker 相关配置
│   │   ├── Dockerfile.server
│   │   ├── Dockerfile.agent
│   │   └── docker-compose.yml
│   ├── kubernetes/          # K8s 资源定义
│   │   ├── server/
│   │   ├── agent/
│   │   └── aggregator/
│   └── terraform/           # 基础设施即代码
├── docs/                    # 项目文档
│   ├── architecture.md      # 架构设计文档
│   ├── api.md               # API 使用文档
│   ├── deployment.md        # 部署指南
│   └── development.md       # 开发指南
├── examples/                # 使用示例
│   ├── basic/               # 基础使用示例
│   ├── advanced/            # 高级功能示例
│   └── integration/         # 集成示例
├── internal/                # 内部私有代码
│   ├── agent/               # 节点端核心逻辑
│   │   ├── collector/       # 系统指标收集
│   │   ├── reporter/        # 数据上报模块
│   │   └── config/          # 节点配置
│   ├── aggregator/          # 聚合服务器逻辑
│   │   ├── collector/       # 节点数据收集
│   │   ├── processor/       # 数据处理器
│   │   └── forwarder/       # 数据转发器
│   ├── server/              # 主控端核心逻辑
│   │   ├── api/             # HTTP/gRPC API 接口
│   │   ├── auth/            # 认证与授权
│   │   ├── handler/         # 请求处理器
│   │   └── middleware/      # 中间件
│   ├── common/              # 公共代码
│   │   ├── models/          # 数据模型定义
│   │   ├── utils/           # 通用工具函数
│   │   └── constants/       # 常量定义
│   ├── alerting/            # 告警规则与通知
│   │   ├── engine/          # 告警引擎
│   │   ├── rules/           # 告警规则
│   │   └── notifier/        # 通知器
│   ├── dashboard/           # 数据可视化接口
│   │   ├── api/             # 仪表盘 API
│   │   └── renderer/        # 数据渲染器
│   ├── discovery/           # 节点注册与发现
│   │   ├── registry/        # 节点注册表
│   │   └── health/          # 健康检查
│   ├── telemetry/           # 指标、日志、追踪
│   │   ├── metrics/         # 指标收集
│   │   ├── logger/          # 日志处理
│   │   └── tracer/          # 分布式追踪
│   ├── storage/             # 数据存储层
│   │   ├── influxdb/        # InfluxDB 存储
│   │   ├── postgres/        # PostgreSQL 存储
│   │   └── cache/           # 缓存层
│   └── config/              # 配置处理
│       ├── loader/          # 配置加载器
│       └── validator/       # 配置验证器
├── migrations/              # 数据库迁移脚本
│   ├── influxdb/            # InfluxDB 迁移
│   └── postgres/            # PostgreSQL 迁移
├── pkg/                     # 可被外部项目引用的库
│   ├── client/              # 客户端库
│   ├── models/              # 公共数据模型
│   └── utils/               # 公共工具函数
├── scripts/                 # 构建和辅助脚本
│   ├── build.sh             # 构建脚本
│   ├── deploy.sh            # 部署脚本
│   └── test.sh              # 测试脚本
├── test/                    # 集成测试与测试工具
│   ├── integration/         # 集成测试
│   ├── benchmark/           # 性能测试
│   └── fixtures/            # 测试数据
├── tools/                   # 开发工具和辅助脚本
│   ├── lint/                # 代码检查工具
│   ├── mock/                # 模拟生成器
│   └── proto/               # Protocol Buffers 工具
├── web/                     # 前端资源
│   ├── static/              # 静态资源文件
│   ├── templates/           # HTML 模板
│   └── src/                 # 前端源代码
├── .gitignore               # Git 忽略文件
├── .golangci.yml            # golangci-lint 配置
├── Dockerfile               # 主 Dockerfile
├── Makefile                 # Make 构建脚本
├── README.md                # 项目说明文档
├── CONTRIBUTING.md          # 贡献指南
├── LICENSE                  # 许可证
├── go.mod                   # Go 模块定义
└── go.sum                   # 依赖版本锁定
```

### 目录说明

#### 核心目录

1. **`cmd/`**：包含项目的主要入口点
   - `server/`：主控端服务入口
   - `agent/`：节点端服务入口
   - `aggregator/`：聚合服务器入口
   - `collectors/`：辅助工具入口

2. **`internal/`**：私有应用程序和库代码
   - `agent/`：节点端核心逻辑
   - `aggregator/`：聚合服务器逻辑
   - `server/`：主控端核心逻辑
   - `common/`：公共代码
   - `alerting/`：告警规则与通知
   - `dashboard/`：数据可视化接口
   - `discovery/`：节点注册与发现
   - `telemetry/`：指标、日志、追踪
   - `storage/`：数据存储层
   - `config/`：配置处理

3. **`pkg/`**：可以被外部应用程序使用的库代码
   - `client/`：客户端库
   - `models/`：公共数据模型
   - `utils/`：公共工具函数

4. **`api/`**：API 协议定义、OpenAPI/Swagger 规范等
   - `proto/`：Protocol Buffers 定义
   - `openapi/`：OpenAPI/Swagger 定义
   - `graphql/`：GraphQL schema 定义

#### 配置和部署

1. **`configs/`**：配置文件模板或默认配置
   - `server.yaml`：主控端配置
   - `agent.yaml`：节点端配置
   - `aggregator.yaml`：聚合服务器配置

2. **`deployments/`**：IaaS、PaaS、系统和容器编排部署配置和模板
   - `docker/`：Docker 相关配置
   - `kubernetes/`：K8s 资源定义
   - `terraform/`：基础设施即代码

3. **`build/`**：打包和持续集成
   - `ci/`：CI 配置
   - `package/`：打包脚本
   - `docker/`：Docker 构建文件

#### 文档和测试

1. **`docs/`**：设计和用户文档
   - `architecture.md`：架构设计文档
   - `api.md`：API 使用文档
   - `deployment.md`：部署指南
   - `development.md`：开发指南

2. **`test/`**：额外的外部测试应用程序和测试数据
   - `integration/`：集成测试
   - `benchmark/`：性能测试
   - `fixtures/`：测试数据

3. **`examples/`**：应用程序或公共库的示例
   - `basic/`：基础使用示例
   - `advanced/`：高级功能示例
   - `integration/`：集成示例

#### 工具和脚本

1. **`scripts/`**：各种构建、安装、分析等脚本
   - `build.sh`：构建脚本
   - `deploy.sh`：部署脚本
   - `test.sh`：测试脚本

2. **`tools/`**：项目工具和辅助脚本
   - `lint/`：代码检查工具
   - `mock/`：模拟生成器
   - `proto/`：Protocol Buffers 工具

3. **`.github/`**：GitHub 相关配置
   - `workflows/`：GitHub Actions 工作流配置
   - `ISSUE_TEMPLATE/`：Issue 模板

### 目录组织原则

1. **按功能组织**：
   - 每个目录都有明确的职责
   - 相关功能放在同一个包中

2. **分层设计**：
   - 清晰的分层架构
   - 避免循环依赖

3. **可扩展性**：
   - 结构设计合理，便于添加新功能
   - 支持未来扩展

4. **符合 Go 社区规范**：
   - 遵循 Go 项目的最佳实践
   - 便于其他 Go 开发者理解

### 最佳实践建议

1. **包设计**：
   - 保持包的粒度适中，避免过大或过小
   - 每个包应该有明确的职责
   - 避免循环依赖

2. **代码组织**：
   - 相关的功能放在同一个包中
   - 使用 `internal` 目录保护私有代码
   - 公共代码放在 `pkg` 目录中

3. **测试组织**：
   - 单元测试与源代码放在同一个包中
   - 集成测试放在 `test` 目录中
   - 使用表驱动测试测试多种情况

4. **文档组织**：
   - 每个包都应该有一个 README.md 文件
   - API 文档放在 `docs` 目录中
   - 使用 godoc 格式的注释

5. **构建与部署**：
   - 使用 Makefile 简化构建过程
   - 提供 Docker 和 Kubernetes 部署配置
   - 使用 CI/CD 自动化构建和部署

## 错误处理

### 错误返回

- 函数应返回错误，而不是忽略它们
- 使用 `errors.New()` 或 `fmt.Errorf()` 创建错误
- 对于可恢复的错误，返回 `nil` 而不是错误
- 对于不可恢复的错误，记录日志并返回错误

```go
func (s *Service) Process(data []byte) error {
    if len(data) == 0 {
        return errors.New("empty data")
    }
    
    if err := s.validate(data); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // 处理数据...
    return nil
}
```

### 错误包装

- 使用 `fmt.Errorf()` 和 `%w` 包装错误，保留原始错误
- 添加上下文信息，使错误更有意义
- 避免重复包装错误

```go
func (s *Service) GetUser(id string) (*User, error) {
    user, err := s.db.FindUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to find user %s: %w", id, err)
    }
    return user, nil
}
```

### 错误检查

- 使用 `if err != nil` 检查错误
- 对于特定类型的错误，使用 `errors.Is()` 或 `errors.As()`
- 避免使用 `_` 忽略错误，除非有充分的理由

```go
func (s *Service) Connect() error {
    conn, err := s.db.Connect()
    if err != nil {
        if errors.Is(err, sql.ErrConnDone) {
            // 处理特定错误...
        }
        return fmt.Errorf("database connection failed: %w", err)
    }
    s.conn = conn
    return nil
}
```

## 日志规范

### 日志级别

- 使用适当的日志级别：DEBUG、INFO、WARN、ERROR、FATAL
- 生产环境默认使用 INFO 级别
- 开发环境可以使用 DEBUG 级别获取更多信息

### 日志格式

- 使用结构化日志，包含时间戳、级别、组件、消息和上下文
- 使用 JSON 格式便于解析和分析
- 包含请求 ID 或跟踪 ID 以便追踪请求流

```go
// 使用 zap 日志库
logger.Info("user logged in",
    zap.String("user_id", user.ID),
    zap.String("ip", ip),
    zap.Duration("duration", duration),
)
```

### 日志内容

- 日志消息应该是描述性的，包含足够的上下文
- 避免记录敏感信息（密码、令牌等）
- 对于错误日志，包含错误详情和堆栈跟踪
- 使用结构化字段而不是字符串拼接

```go
// 不推荐
logger.Error("Failed to process request: " + err.Error())

// 推荐
logger.Error("failed to process request",
    zap.Error(err),
    zap.String("request_id", reqID),
)
```

## 测试规范

### 单元测试

- 为每个包编写单元测试
- 测试文件名以 `_test.go` 结尾
- 测试函数以 `Test` 开头
- 使用 `testing` 包和 `testify` 库编写测试
- 测试覆盖率应达到 80% 以上

```go
func TestUserService_GetUser(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        want    *User
        wantErr bool
    }{
        {
            name:    "valid user",
            id:      "user1",
            want:    &User{ID: "user1", Name: "Test User"},
            wantErr: false,
        },
        {
            name:    "non-existent user",
            id:      "nonexistent",
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := NewUserService()
            got, err := s.GetUser(tt.id)
            if (err != nil) != tt.wantErr {
                t.Errorf("UserService.GetUser() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 表驱动测试

- 使用表驱动测试测试多种情况
- 为每个测试用例提供清晰的名称和描述
- 使用 `t.Run()` 运行子测试

### 测试辅助函数

- 使用 `testify` 库的断言函数
- 创建辅助函数简化测试设置和清理
- 使用 `testing.T.Cleanup()` 注册清理函数

```go
func setupTestDB(t *testing.T) *DB {
    db, err := NewTestDB()
    if err != nil {
        t.Fatalf("failed to create test database: %v", err)
    }
    
    t.Cleanup(func() {
        if err := db.Close(); err != nil {
            t.Errorf("failed to close test database: %v", err)
        }
    })
    
    return db
}
```

### 模拟和存根

- 使用接口和依赖注入便于测试
- 使用 `gomock` 或 `testify/mock` 创建模拟对象
- 避免在测试中使用真实的外部服务

```go
// 定义接口
type UserRepository interface {
    FindByID(id string) (*User, error)
}

// 在测试中创建模拟
mockRepo := new(MockUserRepository)
mockRepo.On("FindByID", "user1").Return(&User{ID: "user1", Name: "Test User"}, nil)

service := NewUserService(mockRepo)
user, err := service.GetUser("user1")
assert.NoError(t, err)
assert.Equal(t, "Test User", user.Name)
```

## 文档规范

### 代码注释

- 所有导出的函数、类型、变量必须有注释
- 注释应解释"为什么"而不仅仅是"是什么"
- 使用 `godoc` 格式的注释

### API 文档

- 使用 Swagger/OpenAPI 规范记录 API
- 为每个 API 端点提供详细的描述、参数和响应
- 包含请求和响应示例

### README 文件

- 每个包都应该有一个 README.md 文件
- README 应包含包的概述、用法示例和注意事项
- 包含指向 godoc 的链接

## 性能优化

### 内存管理

- 避免不必要的内存分配
- 使用对象池减少 GC 压力
- 预分配切片和映射的容量
- 使用 `sync.Pool` 重用对象

```go
// 预分配切片容量
users := make([]User, 0, 100)

// 使用对象池
var userPool = sync.Pool{
    New: func() interface{} {
        return &User{}
    },
}

func getUser() *User {
    return userPool.Get().(*User)
}

func putUser(user *User) {
    userPool.Put(user)
}
```

### 并发处理

- 使用 goroutine 处理并发任务
- 使用 channel 进行 goroutine 间通信
- 使用 `sync.WaitGroup` 等待 goroutine 完成
- 使用 `context.Context` 控制 goroutine 生命周期

```go
func ProcessItems(ctx context.Context, items []Item) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(items))
    
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            
            select {
            case <-ctx.Done():
                errChan <- ctx.Err()
                return
            default:
                if err := processItem(item); err != nil {
                    errChan <- err
                    return
                }
            }
        }(item)
    }
    
    // 等待所有 goroutine 完成
    go func() {
        wg.Wait()
        close(errChan)
    }()
    
    // 收集错误
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### 数据库操作

- 使用连接池管理数据库连接
- 使用预处理语句减少 SQL 注入风险
- 使用事务确保数据一致性
- 批量操作减少数据库往返

```go
func (s *Service) BatchInsertUsers(ctx context.Context, users []User) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    stmt, err := tx.PrepareContext(ctx, "INSERT INTO users (id, name, email) VALUES (?, ?, ?)")
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()
    
    for _, user := range users {
        if _, err := stmt.ExecContext(ctx, user.ID, user.Name, user.Email); err != nil {
            return fmt.Errorf("failed to insert user %s: %w", user.ID, err)
        }
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```

## 安全规范

### 输入验证

- 验证所有用户输入
- 使用参数化查询防止 SQL 注入
- 限制输入长度和类型
- 使用白名单而非黑名单验证

```go
func (s *Service) CreateUser(ctx context.Context, user *User) error {
    // 验证输入
    if err := s.validateUser(user); err != nil {
        return fmt.Errorf("invalid user data: %w", err)
    }
    
    // 使用参数化查询
    _, err := s.db.ExecContext(ctx, 
        "INSERT INTO users (name, email) VALUES (?, ?)",
        user.Name, user.Email)
    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }
    
    return nil
}

func (s *Service) validateUser(user *User) error {
    if user == nil {
        return errors.New("user is nil")
    }
    
    if len(user.Name) == 0 || len(user.Name) > 100 {
        return errors.New("name must be between 1 and 100 characters")
    }
    
    if !isValidEmail(user.Email) {
        return errors.New("invalid email format")
    }
    
    return nil
}
```

### 认证与授权

- 使用安全的认证机制（如 JWT、OAuth2）
- 实现基于角色的访问控制（RBAC）
- 验证所有 API 请求的权限
- 使用 HTTPS 加密传输

```go
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        
        claims, err := h.authService.ValidateToken(token)
        if err != nil {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }
        
        // 将用户信息添加到请求上下文
        ctx := context.WithValue(r.Context(), "user", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

### 敏感数据处理

- 不在日志中记录敏感信息
- 使用环境变量或安全的配置管理系统存储密钥
- 加密存储敏感数据
- 实现数据访问审计

```go
// 使用环境变量存储敏感信息
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    return errors.New("API_KEY environment variable is not set")
}

// 加密敏感数据
func (s *Service) StoreCredentials(ctx context.Context, userID string, credentials *Credentials) error {
    encryptedData, err := s.encryptionService.Encrypt(credentials.Password)
    if err != nil {
        return fmt.Errorf("failed to encrypt credentials: %w", err)
    }
    
    // 存储加密后的数据
    return s.db.StoreEncryptedCredentials(ctx, userID, encryptedData)
}
```

## 工具与最佳实践

### 代码质量工具

- 使用 `golangci-lint` 进行代码检查
- 使用 `go vet` 检查常见错误
- 使用 `goimports` 管理导入
- 使用 `gofmt` 格式化代码

### CI/CD 集成

- 在 CI 流程中运行测试和代码检查
- 使用 GitHub Actions 或 GitLab CI 自动化构建和测试
- 实现自动化部署流程
- 使用 Docker 容器化应用

### 版本控制

- 使用语义化版本控制（SemVer）
- 在每次提交前运行测试和代码检查
- 使用分支策略（如 Git Flow）管理开发流程
- 编写有意义的提交消息

### 依赖管理

- 使用 Go Modules 管理依赖
- 定期更新依赖以修复安全漏洞
- 锁定依赖版本以确保构建一致性
- 使用 `go mod tidy` 清理未使用的依赖

## 总结

遵循本规范将有助于保持 SysLens 项目代码的高质量、可维护性和一致性。所有团队成员应熟悉并遵循这些规范，以确保项目的长期成功。
