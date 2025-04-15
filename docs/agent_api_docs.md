# SysLens 节点端 (Agent) API 调用文档

本文档描述了 SysLens 节点端 (Agent) 如何调用其目标服务器（可能是主控端或聚合服务器）的 HTTP API 接口来上报指标数据。

## 基本信息

- **目标服务器 URL**: 由节点配置文件 (`configs/agent.yaml`) 或命令行参数决定。
  - 优先级：命令行参数 `--server` > 配置文件 `aggregator.url` (如果 `aggregator.enabled` 为 `true`) > 配置文件 `server.url` > 默认值 `http://localhost:8080`。
- **认证**:
  - 如果目标是聚合服务器 (根据 `aggregator.enabled` 和 `aggregator.url` 配置)，节点会使用 `aggregator.auth_token` 作为 Bearer Token 发送 `Authorization` 头部。
  - 如果目标是主控端，当前代码 (`internal/agent/reporter/reporter.go`) 不会自动发送 `Authorization` 头部，除非通过 `reporter.WithAuthToken` 明确设置（这在 `cmd/agent/main.go` 中仅针对聚合服务器做了设置）。主控端可能依赖其他机制（如 IP 白名单或未来实现的节点密钥）来认证直接连接的节点。
- **节点标识**: 所有请求都包含 `X-Node-ID` 头部，其值来自配置文件 `node.id` 或系统主机名。

## API 调用列表

节点端主要执行以下 API 调用：

### 1. 上报节点指标数据

- **目的**: 定期将收集到的系统指标数据发送到目标服务器。
- **触发时机**: 由 `cmd/agent/main.go` 中的定时器根据 `collection.interval` 配置定期触发，调用 `collectAndReport` 函数，最终由 `reporter.Report` 执行。
- **目标接口**: `POST /api/v1/nodes/{node_id}/metrics`
- **路径参数**:
  - `{node_id}` (string, required): 节点的唯一标识符 (来自配置文件 `node.id` 或主机名)。
- **节点请求头**:
  - `Content-Type`:
    - `application/json`: 如果数据未加密且未压缩。
    - `application/octet-stream`: 如果数据经过加密或压缩。
  - `User-Agent: SysLens-Agent`
  - `X-Node-ID` (string, required): 当前节点的 ID。
  - `Authorization: Bearer <aggregator_auth_token>` (string, optional): **仅当**目标服务器是聚合服务器且配置文件中 `aggregator.auth_token` 非空时发送。
  - `X-Encrypted: true` (optional): 如果 `security.encryption.enabled` 为 `true`。
  - `X-Compressed: gzip` (optional): 如果 `security.compression.enabled` 为 `true`。
- **节点请求体**:
  - 如果未启用加密和压缩：包含节点收集的指标数据的 JSON 对象。数据结构由 `internal/agent/collector/collector.go` 中的 `SystemStats` 定义。

    ```json
    {
      "timestamp": "2023-10-27T10:00:00Z", // 采集时间戳
      "hostname": "my-agent-node",
      "platform": "linux",
      "cpu": {
        "usage": 15.5,
        "load": [0.5, 0.6, 0.7],
        // ... 其他 CPU 指标
      },
      "memory": {
        "total": 8192,
        "used": 4096,
        "used_percent": 50.0,
        // ... 其他内存指标
      },
      "disk": {
        "/": {
          "total": 102400,
          "used": 51200,
          "used_percent": 50.0
          // ... 其他磁盘指标
        }
      },
      "network": {
        "tcp_conn_count": 150,
        "udp_conn_count": 20,
        "public_ipv4": "1.2.3.4",
        "private_ipv4": "192.168.1.100",
        "interfaces": {
          "eth0": {
            "bytes_sent": 12345678,
            "bytes_recv": 87654321,
            "upload_speed": 10240, // bytes/sec
            "download_speed": 20480 // bytes/sec
            // ... 其他接口指标
          }
        }
      }
      // ... 可能包含进程信息等其他指标
    }
    ```

  - 如果启用了加密或压缩：处理后的二进制数据。
- **预期服务器响应**:
  - `2xx` 状态码表示上报成功。
  - 非 `2xx` 状态码表示失败。节点端会根据 `server.retry_count` 和 `server.retry_interval` (或聚合服务器的相应配置) 进行重试。如果多次重试后仍然失败，错误会被记录到日志 (`logs/agent_errors.log`)，并且失败的数据可能会被缓存到本地 (`tmp/failed_reports/`)。

## 注意事项

- 节点端只会调用目标服务器的 `/api/v1/nodes/{node_id}/metrics` 接口来**发送**数据。
- 节点端**不会**主动调用接口来获取配置、注册或验证自己（这些操作通常由主控端或聚合服务器在需要时发起，或者通过其他带外机制完成）。
- 数据的加密和压缩在发送前由 `reporter.processData` 处理。
- 认证令牌 (`aggregator.auth_token`) 仅在连接到聚合服务器时使用。
