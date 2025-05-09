# SysLens - 新一代分布式服务器监控系统

SysLens是一个强大的分布式服务器监控系统，由主控端和节点端组成，专为企业级大规模服务器集群设计，提供实时监控和深度分析能力。

## 产品概述与愿景

SysLens旨在提供一站式的服务器监控解决方案，通过直观的Web界面和强大的后台分析引擎，帮助运维团队快速发现和解决系统性能问题，提高基础设施的可靠性和稳定性。

## 核心功能特点

- 毫秒级监控：支持500毫秒以上的高频数据采集
- 自动资源初始化：自动创建所需的数据库资源
- 分布式架构：主控/节点分离，可无限扩展监控规模
- 多指标采集：CPU、内存、磁盘、网络、进程等全方位监控
- Web管理界面：直观的数据可视化和节点管理
- 一键节点部署：自动生成专属安装命令，快速添加监控节点
- 节点专属密钥：每个节点拥有唯一密钥，确保通信安全
- 集中配置管理：节点从主控端获取配置，简化管理
- 节点分组管理：支持按地区、功能等多维度对节点进行分组
- 固定服务节点：支持定义固定服务，可在不同机器间迁移
- 可扩展存储：支持多种数据库后端
- 告警通知系统：可自定义的告警规则和多渠道通知

## 产品架构与技术设计

### 系统架构

系统由两个主要组件和Web前端构成：

- 主控端(Control Plane)：核心管理系统，提供Web界面和API服务
- 节点端(Node Agent)：轻量级采集组件，部署在被监控服务器上
- Web前端：响应式设计的管理界面，支持多种数据可视化方式

### 数据存储架构

SysLens采用双层数据存储架构：

1. 时序数据存储：采用InfluxDB存储所有监控指标数据
2. 结构化数据存储：采用PostgreSQL存储节点信息、用户账号、密钥管理、节点分组等结构化数据

### 节点管理与安全设计

每个节点与主控端的通信基于独特的密钥体系：

1. 节点注册与密钥分配：

- 在Web界面创建新节点，系统自动生成唯一密钥
- 密钥使用AES-256加密并存储在PostgreSQL数据库中
- Web界面生成包含密钥的一键安装命令

2. 节点认证流程：

- 节点首次连接时使用分配的密钥进行认证
- 认证成功后，节点从主控端获取完整配置
- 后续通信继续使用该密钥进行加密和身份验证

3. 配置下发机制：

- 节点配置集中存储在主控端
- 首次连接和配置变更时自动下发到节点
- 支持针对节点组批量更新配置

4. 节点类型与分组：

- 支持固定服务节点：定义固定服务，可在不同机器间迁移
- 支持非固定节点：临时或动态变化的节点
- 节点分组：支持按地区、功能、环境等多维度对节点进行分组
- 分组管理：可在分组中添加或删除节点，支持批量操作

### 前端设计

Web前端提供丰富的功能：

- 多维度数据展示：仪表盘、图表、列表等多种展示形式
- 节点管理控制台：添加、编辑、删除节点
- 节点分组管理：按地区、功能、环境等维度分组管理
- 固定服务管理：定义和管理固定服务，支持服务迁移
- 告警规则配置：可视化的告警规则编辑器
- 系统设置界面：全局配置和个人偏好设置

## 功能模块详细说明

### 主控端(Server)功能

#### Web管理界面

- 仪表盘：自定义布局的监控视图
- 节点管理：添加、配置和管理监控节点
- 节点分组：创建和管理节点分组
- 固定服务：定义和管理固定服务
- 告警中心：查看和处理系统告警
- 报告生成：创建和导出性能报告
- 系统设置：全局配置和个人偏好设置

#### API服务

- 数据接收API：接收节点上报数据
- 查询API：提供数据查询服务
- 节点管理API：处理节点注册和配置
- 分组管理API：处理节点分组的创建、更新和删除
- 固定服务API：处理固定服务的定义和迁移
- 用户认证API：处理用户登录和权限

#### 存储层

- 时序数据管理：监控指标的存储与检索
- 结构化数据管理：节点信息、用户数据、配置、分组等
- 数据清理策略：自动清理过期数据

#### 节点管理模块

- 节点注册：处理新节点注册请求
- 密钥管理：生成、存储和验证节点密钥
- 配置下发：将配置推送到节点
- 状态监控：跟踪节点健康状态
- 分组管理：处理节点的分组和分组操作
- 固定服务管理：处理固定服务的定义和迁移

#### 告警系统

- 规则引擎：基于规则触发告警
- 通知管理：邮件、Webhook、短信等多渠道通知
- 告警聚合：智能聚合相似告警

### 节点端(Agent)功能

- 指标收集器：收集系统各项指标
- 本地缓存：暂存无法立即上报的数据
- 配置同步：从主控端获取完整配置
- 自我诊断：监控自身状态并报告
- 服务标识：标识节点是否为固定服务节点

## 节点部署流程

SysLens提供简化的节点部署方式，通过以下步骤快速将服务器纳入监控：

1. 创建节点：

- 在Web界面的"节点管理"中点击"添加节点"
- 填写节点基本信息（名称、描述、角色等）
- 选择节点类型（固定服务节点或非固定节点）
- 选择节点所属分组（可选）
- 点击"创建"完成节点创建

2. 获取安装命令：

- 系统自动生成包含专属密钥的安装命令
- 示例：`curl -sSL https://syslens-server/install.sh | bash -s -- --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`

3. 执行安装：

- 在目标服务器上执行安装命令
- 节点代理自动安装并启动
- 首次通信使用token进行身份验证
- 验证成功后从主控端获取完整配置

4. 验证连接：

- Web界面显示节点状态变为"在线"
- 开始接收和显示该节点的监控数据

## 节点分组与固定服务管理

### 节点分组

SysLens支持多种维度的节点分组，便于管理和监控：

1. 创建分组：

- 在Web界面的"节点分组"中点击"创建分组"
- 填写分组信息（名称、描述、类型等）
- 选择分组类型（地区、功能、环境等）
- 点击"创建"完成分组创建

2. 管理分组：

- 添加节点到分组：选择节点，添加到指定分组
- 从分组移除节点：从分组中选择节点并移除
- 批量操作：支持批量添加或移除节点
- 分组配置：可为分组设置特定的监控配置

3. 分组视图：

- 分组列表：显示所有分组及其节点数量
- 分组详情：显示分组内所有节点及其状态
- 分组指标：显示分组内节点的聚合指标

### 固定服务节点

SysLens支持固定服务节点的定义和管理，便于服务迁移：

1. 定义固定服务：

- 在Web界面的"固定服务"中点击"创建服务"
- 填写服务信息（名称、描述、类型等）
- 设置服务的关键指标和告警规则
- 点击"创建"完成服务定义

2. 分配节点：

- 为固定服务分配一个或多个节点
- 设置节点优先级（主节点、备用节点等）
- 配置节点间的故障转移规则

3. 服务迁移：

- 当节点故障时，可将服务迁移到其他节点
- 支持手动迁移和自动故障转移
- 迁移过程中保持服务监控的连续性

4. 服务视图：

- 服务状态：显示服务的运行状态和健康度
- 服务指标：显示服务的关键性能指标
- 节点关联：显示服务关联的节点及其状态

## 技术架构

### 后端技术栈

- 编程语言：Go 1.18+
- Web框架：Gin/Echo
- API通信：REST API + WebSocket
- 时序数据库：InfluxDB 2.x
- 结构化数据库：PostgreSQL
- 缓存系统：Redis(可选)
- 消息队列：NATS (可选，用于大规模部署)

### 前端技术栈

- 框架：React/Vue.js
- UI组件：Ant Design/Element UI
- 数据可视化：ECharts/D3.js
- 状态管理：Redux/Vuex
- 通信：Axios + WebSocket

## 数据安全

SysLens提供全面的数据安全保障：

### 传输安全

- 节点通信加密：每个节点使用专属密钥进行通信加密
- HTTPS加密：所有Web界面和API通信使用TLS加密
- 数据压缩：支持高效的数据压缩以减少带宽占用

### 认证与授权

- 节点认证：基于密钥的节点身份验证
- 用户认证：支持本地账号和OAuth2/OIDC集成
- 细粒度权限：基于角色的访问控制(RBAC)

### 密钥管理

- 密钥生成：使用密码学安全的随机生成器
- 密钥存储：加密存储所有敏感信息
- 自动轮换：支持密钥定期自动轮换(可选)

## 使用教程

### 系统要求

- 主控端：2核CPU，4GB内存，50GB存储空间
- 节点端：最小化资源占用，支持几乎所有Linux发行版
- 支持环境：Linux、macOS、Windows

### 主控端安装

#### 方式一：Docker部署（推荐）

```bash
# 拉取并启动SysLens主控端
docker-compose -f deployments/docker/docker-compose.yml up -d
```

#### 方式二：从源码构建

```bash
git clone https://github.com/syslens/syslens-api.git
cd syslens-api
make build-all
./scripts/start-server-with-db.sh
```

### 访问Web界面

安装完成后，访问主控端Web界面：

```
http://<主控端IP>:8080
```

初次访问需要创建管理员账号。

### 节点管理

1. 登录Web界面
2. 导航到"节点管理" > "添加节点"
3. 填写节点信息并创建
4. 复制生成的安装命令
5. 在目标服务器上执行该命令

### 节点分组管理

1. 创建分组：
   - 导航到"节点分组" > "创建分组"
   - 填写分组信息并选择分组类型
   - 点击"创建"完成分组创建

2. 添加节点到分组：
   - 在节点列表中选择节点
   - 点击"添加到分组"并选择目标分组
   - 或直接在分组详情页添加节点

3. 分组配置：
   - 在分组详情页设置分组特定的监控配置
   - 配置分组级别的告警规则

### 固定服务管理

1. 创建固定服务：
   - 导航到"固定服务" > "创建服务"
   - 填写服务信息并设置关键指标
   - 点击"创建"完成服务定义

2. 分配节点：
   - 在服务详情页点击"分配节点"
   - 选择节点并设置优先级
   - 配置故障转移规则

3. 服务迁移：
   - 在服务详情页点击"迁移服务"
   - 选择目标节点并确认迁移

### 配置管理

所有配置均可通过Web界面进行管理：

- 全局设置：数据保留策略、通知渠道等
- 节点配置：采集项目、采集频率、上报设置等
- 分组配置：分组特定的监控设置
- 服务配置：固定服务的关键指标和告警规则
- 告警规则：资源阈值、触发条件、通知方式

## 常见问题与故障排除

### 节点安装失败

- 检查安装命令是否完整正确
- 确保服务器可以访问主控端
- 检查服务器防火墙设置

### 节点认证失败

- 验证节点密钥是否正确
- 检查主控端日志查找详细错误
- 在Web界面重新生成安装命令并重试

### 服务迁移失败

- 检查目标节点是否满足服务要求
- 验证节点间网络连接是否正常
- 检查服务配置是否兼容

### 数据显示异常

- 检查时区设置是否一致
- 验证数据库连接是否正常
- 调整采集间隔或聚合设置

## 开源协议

SysLens使用Apache 2.0许可证开源。

## 社区与支持

- GitHub仓库：[https://github.com/syslens/syslens-api](https://github.com/syslens/syslens-api)
- 文档网站：[https://docs.syslens.io](https://docs.syslens.io)
- 问题反馈：[https://github.com/syslens/syslens-api/issues](https://github.com/syslens/syslens-api/issues)

## 节点实时数据展示通信方案

SysLens采用高效的实时数据通信机制，确保节点监控数据的实时性和低延迟展示：

### 通信架构设计

系统支持两种实时数据通信方式，可根据不同场景灵活选择：

1. **WebSocket长连接**（推荐）：

- 建立持久连接，服务器可主动推送数据
- 低延迟，减少网络开销
- 支持双向通信，客户端可发送控制命令
- 适合需要高频率更新的场景（如毫秒级监控）

2. **HTTP轮询**（备选方案）：

- 客户端定期请求最新数据
- 实现简单，兼容性好
- 可配置轮询间隔，平衡实时性和服务器负载
- 适合更新频率较低的场景

### WebSocket实现细节

1. **连接建立**：

- 客户端通过`ws://<主控端IP>:8080/api/v1/ws/nodes`建立WebSocket连接
- 连接时携带节点ID和认证令牌：`ws://<主控端IP>:8080/api/v1/ws/nodes?node_id=node-001&token=xxx`
- 服务器验证连接后，开始推送该节点的实时数据

2. **数据推送格式**：

```json
{
  "type": "metrics",
  "node_id": "node-001",
  "timestamp": 1621234567890,
  "data": {
    "cpu": {
      "usage": 45.2,
      "cores": 8,
      "load": [1.2, 1.5, 1.8]
    },
    "memory": {
      "total": 16384,
      "used": 8192,
      "free": 8192,
      "usage": 50.0
    },
    "network": {
      "interfaces": [
        {
          "name": "eth0",
          "bytes_sent": 1024000,
          "bytes_recv": 2048000,
          "packets_sent": 1000,
          "packets_recv": 2000
        }
      ]
    },
    "disk": {
      "partitions": [
        {
          "device": "/dev/sda1",
          "mountpoint": "/",
          "total": 512000,
          "used": 256000,
          "free": 256000,
          "usage": 50.0
        }
      ]
    }
  }
}
```

3. **控制命令**：

客户端可发送控制命令调整数据推送：

```json
{
  "command": "set_interval",
  "interval": 1000  // 毫秒
}
```

4. **心跳机制**：

- 服务器每30秒发送一次心跳消息
- 客户端需在60秒内响应心跳，否则连接将被关闭
- 心跳消息格式：`{"type": "ping", "timestamp": 1621234567890}`

### HTTP轮询实现细节

1. **数据获取接口**：

- 端点：`GET /api/v1/nodes/{node_id}/metrics`
- 支持查询参数：`?interval=1000&metrics=cpu,memory,network`
- 响应格式与WebSocket相同

2. **轮询优化**：

- 支持条件请求（If-None-Match/If-Modified-Since）
- 服务器返回304状态码表示数据未变化
- 客户端可根据数据变化情况动态调整轮询间隔

3. **批量获取**：

- 支持一次请求多个节点的数据：`GET /api/v1/nodes/metrics?node_ids=node-001,node-002`
- 减少HTTP请求数量，提高效率

### 前端实现示例

```javascript
// WebSocket连接示例
class NodeMetricsClient {
  constructor(nodeId, token, onData) {
    this.nodeId = nodeId;
    this.token = token;
    this.onData = onData;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
  }

  connect() {
    const url = `ws://${window.location.hostname}:8080/api/v1/ws/nodes?node_id=${this.nodeId}&token=${this.token}`;
    this.ws = new WebSocket(url);
    
    this.ws.onopen = () => {
      console.log('WebSocket连接已建立');
      this.reconnectAttempts = 0;
    };
    
    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'metrics') {
        this.onData(data.data);
      } else if (data.type === 'ping') {
        this.ws.send(JSON.stringify({ type: 'pong', timestamp: Date.now() }));
      }
    };
    
    this.ws.onclose = () => {
      console.log('WebSocket连接已关闭');
      this.reconnect();
    };
    
    this.ws.onerror = (error) => {
      console.error('WebSocket错误:', error);
    };
  }

  reconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`尝试重新连接 (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      setTimeout(() => this.connect(), 2000 * this.reconnectAttempts);
    } else {
      console.error('达到最大重连次数，请刷新页面重试');
    }
  }

  setInterval(interval) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        command: 'set_interval',
        interval: interval
      }));
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// 使用示例
const client = new NodeMetricsClient('node-001', 'auth-token', (data) => {
  // 更新UI显示节点数据
  updateNodeMetricsUI(data);
});

client.connect();

// 设置数据更新间隔为1秒
client.setInterval(1000);

// 组件卸载时断开连接
function cleanup() {
  client.disconnect();
}
```

### 性能优化策略

1. **数据压缩**：

- WebSocket消息使用gzip压缩
- 减少网络传输量，提高传输效率

2. **增量更新**：

- 只传输发生变化的数据字段
- 大幅减少数据传输量

3. **批量处理**：

- 服务器端对短时间内的多次更新进行合并
- 减少推送频率，降低服务器和客户端负载

4. **自适应频率**：

- 根据数据变化率动态调整推送频率
- 稳定数据降低频率，波动数据提高频率

5. **数据缓存**：

- 客户端缓存历史数据
- 支持时间范围查询和趋势分析

### 选择建议

- **高频率监控场景**（毫秒级）：使用WebSocket
- **多节点同时监控**：使用WebSocket，减少连接数
- **简单场景或兼容性要求高**：使用HTTP轮询
- **移动端或弱网络环境**：使用HTTP轮询，更可靠
- **大规模部署**：根据节点数量和网络条件选择，可混合使用
