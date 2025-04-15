# SysLens 聚合服务器 -> 主控端 API 文档

本文档描述了 SysLens 聚合服务器 (Aggregator) 调用主控端 (Control Plane) HTTP API 的接口。

## 基本信息

- **主控端基路径**: 从聚合服务器配置文件 (`configs/aggregator.yaml` 或 `configs/aggregator.template.yaml`) 的 `control_plane.url` 字段获取。
- **认证**: 所有请求都需要通过 `Authorization: Bearer <token>` 头部进行认证。令牌从聚合服务器配置文件的 `control_plane.token` 字段获取。
- **聚合服务器标识**: 聚合服务器在请求中通常会包含 `X-Aggregator-ID` 头部来标识自己。

## API 调用列表

以下是聚合服务器向主控端发起的 API 调用：

### 1. 转发节点指标数据

- **目的**: 将从节点收集并处理后的指标数据批量转发给主控端进行存储和分析。
- **触发时机**: 由 `internal/aggregator/processor.go` 中的 `processMetricsData` 函数定时触发，调用 `forwardMetricsToControlPlane`。
- **主控端接口**: `POST /api/v1/nodes/{node_id}/metrics`
- **路径参数**:
  - `{node_id}` (string, required): 被转发指标数据的原始节点ID。
- **聚合服务器请求头**:
  - `Content-Type: application/json`
  - `Authorization: Bearer <control_plane_token>`
  - `X-Node-ID` (string, required): 原始节点的ID。
  - `X-Aggregator-ID` (string, required): 发起请求的聚合服务器的ID (例如: "aggregator-1")。
- **聚合服务器请求体**: 包含处理后指标数据的 JSON 对象。聚合服务器可能会添加额外的元数据，如 `processed_at` 时间戳。

  ```json
  {
    "processed_at": 1678886400,
    "received_at": 1678886399,
    "cpu": {
      "usage": 55.1
      // ... 其他 CPU 指标
    },
    "memory": {
      "used_percent": 60.5
      // ... 其他内存指标
    }
    // ... 其他原始指标
  }
  ```

- **预期主控端响应**:
  - `2xx` 状态码表示成功。
  - 非 `2xx` 状态码表示失败，聚合服务器会记录错误日志。

### 2. 验证节点令牌

- **目的**: 在节点尝试通过聚合服务器注册时，验证节点提供的 `node_id` 和 `token` 是否有效。
- **触发时机**: 当聚合服务器收到节点的注册请求 (`POST /api/v1/nodes/register`) 时，由 `internal/aggregator/server.go` 中的 `handleNodeRegister` 函数调用 `controlPlane.ValidateNode`。
- **主控端接口**: `POST /api/v1/nodes/validate`
- **聚合服务器请求头**:
  - `Content-Type: application/json`
  - `Authorization: Bearer <control_plane_token>`
- **聚合服务器请求体**: 包含需要验证的节点 ID 和令牌的 JSON 对象。

  ```json
  {
    "node_id": "agent-node-123",
    "token": "agent-provided-token"
  }
  ```

- **预期主控端响应**:
  - `200 OK` 状态码表示验证成功。
  - 非 `200 OK` 状态码（特别是 `401 Unauthorized`）表示验证失败，聚合服务器将拒绝节点的注册请求。

### 3. (潜在的) 注册节点到主控端

- **目的**: 主动将聚合服务器自身管理的节点信息（如果需要）注册或更新到主控端。
- **触发时机**: 在当前代码 `internal/aggregator/client.go` 中存在 `RegisterNode` 函数，但未在主要流程 (`processor.go`, `server.go`) 中被调用。如果未来启用，可能会在节点连接或状态变化时触发。
- **主控端接口**: `POST /api/v1/nodes/{node_id}`
- **路径参数**:
  - `{node_id}` (string, required): 要注册或更新的节点ID。
- **聚合服务器请求头**:
  - `Content-Type: application/json`
  - `Authorization: Bearer <control_plane_token>`
- **聚合服务器请求体**: 包含节点详细信息的 JSON 对象。

  ```json
  {
    "status": "active",
    "labels": {"managed_by": "aggregator-1"}
    // ... 其他节点信息
  }
  ```

- **预期主控端响应**:
  - `200 OK` 状态码表示成功。
  - 非 `200 OK` 状态码表示失败。

### 4. (潜在的) 更新节点状态

- **目的**: 更新主控端上记录的特定节点的状态。
- **触发时机**: 在当前代码 `internal/aggregator/client.go` 中存在 `UpdateNodeStatus` 函数，但未在主要流程中被调用。如果未来启用，可能会在聚合服务器检测到节点状态变化时触发。
- **主控端接口**: `PUT /api/v1/nodes/{node_id}/status`
- **路径参数**:
  - `{node_id}` (string, required): 要更新状态的节点ID。
- **聚合服务器请求头**:
  - `Content-Type: application/json`
  - `Authorization: Bearer <control_plane_token>`
- **聚合服务器请求体**: 包含新状态的 JSON 对象。

  ```json
  {
    "status": "inactive" 
  }
  ```

- **预期主控端响应**:
  - `200 OK` 状态码表示成功。
  - 非 `200 OK` 状态码表示失败。

### 5. (潜在的) 获取节点配置

- **目的**: 从主控端获取特定节点的配置信息。
- **触发时机**: 在当前代码 `internal/aggregator/client.go` 中存在 `GetNodeConfig` 函数，但未在主要流程中被调用。如果未来启用，可能在聚合服务器需要为节点提供特定配置时触发。
- **主控端接口**: `GET /api/v1/nodes/{node_id}/config`
- **路径参数**:
  - `{node_id}` (string, required): 要获取配置的节点ID。
- **聚合服务器请求头**:
  - `Authorization: Bearer <control_plane_token>`
- **预期主控端响应**:
  - `200 OK` 状态码，响应体为包含节点配置的 JSON 对象。
  - 非 `200 OK` 状态码表示失败（如 `404 Not Found`）。
