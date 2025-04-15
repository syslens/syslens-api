# SysLens 主控端 API 文档

本文档描述了 SysLens 主控端 (Control Plane) 提供的 HTTP API 接口。

## 基本信息

- **基路径**: `/api/v1`
- **认证**: 部分接口需要认证，主要通过 `Authorization: Bearer <token>` 头部进行验证（例如聚合服务器上报指标时）。节点上报时，可能需要通过 `X-Node-ID` 和内部机制进行识别和验证。

## API 接口列表

### 健康检查

- **路径**: `/health`
- **方法**: `GET`
- **描述**: 检查主控端服务的健康状态。
- **认证**: 无
- **参数**: 无
- **成功响应 (200 OK)**:

  ```json
  {
    "status": "ok",
    "time": "2023-10-27T10:00:00Z"
  }
  ```

### 指标处理 (Metrics)

#### 1. 节点/聚合服务器上报指标

- **路径**: `/api/v1/nodes/{node_id}/metrics`
- **方法**: `POST`
- **描述**: 接收来自节点或聚合服务器上报的系统指标数据。
- **认证**: 需要聚合服务器或节点的有效认证。聚合服务器通过 `Authorization` 头部发送令牌；节点可能通过其他方式（如内部令牌或 IP 识别）。
- **路径参数**:
  - `{node_id}` (string, required): 上报数据的节点唯一标识符。
- **请求头**:
  - `Content-Type: application/json` 或 `application/octet-stream` (如果数据经过加密/压缩)
  - `X-Node-ID` (string, required): 发送数据的节点ID。
  - `Authorization: Bearer <token>` (string, optional): 聚合服务器使用的认证令牌。
  - `X-Aggregator-ID` (string, optional): 标识请求是否来自聚合服务器及其ID。
  - `X-Encrypted: true` (optional): 标识请求体是否已加密。
  - `X-Compressed: gzip` (optional): 标识请求体是否已压缩。
- **请求体**: 包含节点指标数据的 JSON 对象。如果启用了加密或压缩，则为二进制数据流。

  ```json
  {
    "timestamp": 1621234567890,
    "hostname": "web-server-01",
    "platform": "linux",
    "cpu": {
      "usage": 45.2,
      "cores": 8,
      "load": [1.2, 1.5, 1.8]
    },
    "memory": {
      "total": 16384,
      "used": 8192,
      "free": 8192,
      "usage_percent": 50.0
    },
    // ... 其他指标 (disk, network, etc.)
  }
  ```

- **成功响应 (200 OK)**:

  ```json
  {
    "status": "success"
  }
  ```

- **失败响应**:
  - `400 Bad Request`: 请求格式错误、缺少节点ID、JSON解析失败、数据处理失败等。
  - `401 Unauthorized`: 聚合服务器令牌无效。
  - `405 Method Not Allowed`: 使用了非 POST 方法。
  - `500 Internal Server Error`: 存储指标数据失败。

#### 2. 查询节点指标

- **路径**: `/api/v1/nodes/metrics`
- **方法**: `GET`
- **描述**: 查询指定节点在特定时间范围内的指标数据。
- **认证**: 可能需要用户认证（未在代码中明确体现，取决于整体认证策略）。
- **查询参数**:
  - `node_id` (string, required): 要查询的节点ID。
  - `start` (string, optional): 开始时间 (RFC3339格式，例如 `2023-10-26T10:00:00Z`)。如果省略，默认为1小时前。
  - `end` (string, optional): 结束时间 (RFC3339格式)。如果省略，默认为当前时间。
- **成功响应 (200 OK)**:

  ```json
  {
    "status": "success",
    "metrics": [
      {
        "timestamp": "2023-10-27T09:59:00Z",
        "cpu_usage": 15.5,
        "memory_used_percent": 45.0
        // ... 其他指标
      },
      {
        "timestamp": "2023-10-27T10:00:00Z",
        "cpu_usage": 16.2,
        "memory_used_percent": 45.5
        // ... 其他指标
      }
    ]
  }
  ```

- **失败响应**:
  - `400 Bad Request`: 缺少 `node_id` 参数或时间格式无效。
  - `405 Method Not Allowed`: 使用了非 GET 方法。
  - `500 Internal Server Error`: 查询存储失败。

#### 3. 获取所有节点列表

- **路径**: `/api/v1/nodes`
- **方法**: `GET`
- **描述**: 获取所有已注册或上报过数据的节点ID列表。
- **认证**: 可能需要用户认证。
- **参数**: 无
- **成功响应 (200 OK)**:

  ```json
  {
    "status": "success",
    "nodes": [
      "node-web-01",
      "node-db-01",
      "aggregator-proxy-node"
    ]
  }
  ```

- **失败响应**:
  - `405 Method Not Allowed`: 使用了非 GET 方法。
  - `500 Internal Server Error`: 查询存储失败。

### 节点管理 (Node Management)

*(注意: 以下接口在 `internal/server/server.go` 中定义，但部分可能未完全实现或路由)*

#### 1. 节点注册

- **路径**: `/api/v1/nodes/register`
- **方法**: `POST`
- **描述**: 处理新节点的注册请求。 *(具体实现细节需参考 `handleNodeRegister` 函数)*
- **认证**: 需要节点提供有效的凭证（如密钥）。
- **请求体**: 包含节点信息的 JSON 对象。

  ```json
  {
    "node_id": "new-node-123",
    "labels": {"env": "staging", "role": "app"},
    "type": "non-fixed",
    "auth_key": "generated-or-provided-key"
    // ... 其他注册信息
  }
  ```

- **成功响应 (200 OK)**:

  ```json
  {
    "status": "registered",
    "node_id": "new-node-123",
    "message": "Node registered successfully"
    // ... 可能包含分配的配置或令牌
  }
  ```

- **失败响应**:
  - `400 Bad Request`: 请求体无效。
  - `401 Unauthorized`: 认证失败。
  - `409 Conflict`: 节点ID已存在。
  - `500 Internal Server Error`: 注册过程中发生内部错误。

#### 2. 获取节点信息

- **路径**: `/api/v1/nodes/{node_id}` (推测路径，需要确认路由实现)
- **方法**: `GET`
- **描述**: 获取指定节点的详细信息（状态、标签、类型等）。 *(具体实现细节需参考 `handleGetNode` 函数)*
- **认证**: 可能需要用户认证。
- **路径参数**:
  - `{node_id}` (string, required): 要查询的节点ID。
- **成功响应 (200 OK)**:

  ```json
  {
    "status": "success",
    "node": {
      "id": "node-web-01",
      "labels": {"env": "production", "region": "us-east-1"},
      "type": "fixed-service",
      "status": "active",
      "last_active": "2023-10-27T10:30:00Z",
      "registered_at": "2023-10-20T14:00:00Z",
      "group_id": "group-webservers",
      "service_id": "service-frontend"
    }
  }
  ```

- **失败响应**:
  - `404 Not Found`: 节点不存在。

#### 3. 更新节点信息

- **路径**: `/api/v1/nodes/{node_id}` (推测路径，需要确认路由实现)
- **方法**: `PUT` / `PATCH`
- **描述**: 更新指定节点的信息（如标签、类型等）。 *(具体实现细节需参考 `handleUpdateNode` 函数)*
- **认证**: 可能需要用户认证。
- **路径参数**:
  - `{node_id}` (string, required): 要更新的节点ID。
- **请求体**: 包含要更新的节点属性的 JSON 对象。
- **成功响应 (200 OK)**:

  ```json
  {
    "status": "updated",
    "node_id": "node-web-01"
  }
  ```

- **失败响应**:
  - `400 Bad Request`: 请求体无效。
  - `404 Not Found`: 节点不存在。

#### 4. 删除节点

- **路径**: `/api/v1/nodes/{node_id}` (推测路径，需要确认路由实现)
- **方法**: `DELETE`
- **描述**: 删除指定节点。 *(具体实现细节需参考 `handleDeleteNode` 函数)*
- **认证**: 可能需要用户认证。
- **路径参数**:
  - `{node_id}` (string, required): 要删除的节点ID。
- **成功响应 (200 OK / 204 No Content)**:

  ```json
  {
    "status": "deleted",
    "node_id": "node-web-01"
  }
  ```

- **失败响应**:
  - `404 Not Found`: 节点不存在。

### 节点分组管理 (Group Management)

*(注意: 以下接口在 `internal/server/server.go` 中定义，但路由和实现可能不完整)*

- **路径**: `/api/v1/groups`
- **方法**: `GET`, `POST`
- **描述**: 获取所有分组 (`GET`) 或创建新分组 (`POST`)。 *(参考 `handleGetGroups`, `handleCreateGroup`)*
- **认证**: 需要用户认证。

- **路径**: `/api/v1/groups/{group_id}`
- **方法**: `GET`, `PUT`, `DELETE`
- **描述**: 获取 (`GET`)、更新 (`PUT`) 或删除 (`DELETE`) 指定分组。 *(实现需参考 `handleGroupOperations` 分发逻辑)*
- **认证**: 需要用户认证。

### 固定服务管理 (Service Management)

*(注意: 以下接口在 `internal/server/server.go` 中定义，但路由和实现可能不完整)*

- **路径**: `/api/v1/services`
- **方法**: `GET`, `POST`
- **描述**: 获取所有固定服务 (`GET`) 或创建新服务 (`POST`)。 *(参考 `handleGetServices`, `handleCreateService`)*
- **认证**: 需要用户认证。

- **路径**: `/api/v1/services/{service_id}`
- **方法**: `GET`, `PUT`, `DELETE`
- **描述**: 获取 (`GET`)、更新 (`PUT`) 或删除 (`DELETE`) 指定服务。 *(实现需参考 `handleServiceOperations` 分发逻辑)*
- **认证**: 需要用户认证。

### WebSocket 通信

- **路径**: `/api/v1/ws/nodes`
- **方法**: `GET` (用于升级到 WebSocket)
- **描述**: 建立 WebSocket 连接，用于实时接收节点指标或发送控制命令。
- **认证**: 连接时需要在查询参数中提供 `node_id` 和 `token`。
- **查询参数**:
  - `node_id` (string, required): 连接节点的ID。
  - `token` (string, required): 用于认证的令牌。
- **消息格式**: JSON
  - **服务器推送**: `{"type": "metrics", "data": {...}}`, `{"type": "ping", ...}`
  - **客户端发送**: `{"type": "pong", ...}`, `{"command": "set_interval", "interval": 1000}`
- **注意**: WebSocket 的具体实现细节请参考 `handleWebSocket` 和 `processWebSocketData` 函数。
