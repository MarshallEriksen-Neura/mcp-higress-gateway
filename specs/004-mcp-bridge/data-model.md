# Data Model：MCP Bridge（概念模型）

本 feature 的“数据模型”主要用于协议、状态机与持久化边界的对齐；MVP 阶段不强制引入新的数据库表。

## 核心标识

- `user_id`: 归属用户（后端鉴权后可得）
- `agent_id`: 一台用户机器/服务器的唯一标识（用户侧配置）
- `req_id`: 一次工具调用的全局幂等键（UUID/ULID 均可），贯穿 INVOKE/CHUNK/RESULT/CANCEL/ACK
- `conn_session_id`: Agent 与 Tunnel Gateway 的一次 WS 会话 id（重连会变化，主要用于诊断与安全绑定）

## 实体与结构

### Agent

表示一台在线或离线的边缘节点。

关键属性（概念）:
- `agent_id: string`
- `label: string`（可选）
- `status: online|offline`
- `connected_at: time`（可选）
- `last_seen_at: time`（可选）
- `capabilities: {streaming: bool, transports: [...]}`（可选）

### MCPServerConfig

Agent 本地的子 MCP server 配置。

类型:
- `stdio`：`command + args + env`
- `remote`（扩展）：`url + headers`

关键属性（概念）:
- `name: string`（命名空间前缀）
- `type: stdio|sse|http`（按 SDK transport 支持情况）

### ToolDescriptor（命名空间化工具）

Agent 聚合后对外呈现的工具定义。

关键属性（概念）:
- `name: string`（命名空间化：`{server}__{tool}`）
- `description: string`
- `input_schema: jsonschema`
- `source: {server_name, original_tool_name}`

### InvokeRequest

后端发起的一次工具调用。

关键属性（概念）:
- `req_id: string`
- `agent_id: string`
- `tool_name: string`（命名空间化）
- `arguments: object`
- `timeout_ms: int`
- `stream: bool`

### InvokeAck

Agent 确认已接收/拒绝执行。

关键属性（概念）:
- `req_id`
- `accepted: bool`
- `reason?: string`

### Chunk（流式日志分片）

关键属性（概念）:
- `req_id`
- `channel: stdout|stderr`
- `data: string|bytes`（建议 Base64 或直接 utf-8 文本）
- `seq?: int`（可选，便于排序/诊断）
- `dropped_bytes?: int`
- `dropped_lines?: int`

### Result（终态结果）

关键属性（概念）:
- `req_id`
- `ok: bool`
- `exit_code?: int`
- `result_json?: object`（MCP tools/call 的结构化输出）
- `error?: {message, code?, details?}`
- `canceled?: bool`

### ResultAck

云端确认已接收/持久化终态（Agent 可释放缓存）。

关键属性（概念）:
- `req_id`

## 状态机（req_id 维度）

后端侧（概念）:
- `PENDING` -> `SENT` -> `ACKED` -> `STREAMING` -> `DONE|ERROR|CANCELED`

Agent 侧（概念）:
- `RECEIVED` -> `ACKED` -> `RUNNING` -> `DONE` -> `WAIT_RESULT_ACK` -> `CLOSED`

约束:
- CHUNK 可丢；RESULT 不可丢（依赖 RESULT_ACK 与重连补发）。
- req_id 必须幂等：云端对重复 RESULT 只接受首个终态并忽略后续重复。
