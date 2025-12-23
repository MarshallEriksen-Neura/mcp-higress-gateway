# Contract：Backend ⇄ Tunnel Gateway（MVP 无 Redis，内部 HTTP API）

本契约用于 MVP 阶段（Tunnel Gateway 单实例）时，后端通过内网 HTTP 向 Tunnel Gateway 下发 INVOKE/CANCEL，并查询在线 Agent 与工具列表。

> 注意：这是项目内部 API（internal），不面向第三方直接开放。

## 认证

MVP 可先使用内网隔离 + 共享密钥 header（示例）：
- `X-Internal-Token: <token>`

后续可升级为 mTLS 或服务网格鉴权。

## 1) 列出在线 Agent

`GET /internal/bridge/agents`

Response 200:
```json
{
  "agents": [
    {"agent_id":"aws-dev-server","label":"AWS Dev Server","status":"online","last_seen_at":1712345678}
  ]
}
```

## 2) 获取某 Agent 的工具列表

`GET /internal/bridge/agents/{agent_id}/tools`

Response 200:
```json
{
  "agent_id":"aws-dev-server",
  "tools":[{"name":"fs__readFile","description":"...","input_schema":{}}]
}
```

## 3) 下发工具调用（INVOKE）

`POST /internal/bridge/invoke`

Request:
```json
{
  "req_id": "req_...",
  "agent_id": "aws-dev-server",
  "tool_name": "fs__readFile",
  "arguments": {"path":"/tmp/a.txt"},
  "timeout_ms": 60000,
  "stream": true
}
```

Response 202（已投递/排队）:
```json
{"req_id":"req_...","status":"accepted"}
```

Response 409（重复 req_id）:
```json
{"error":"duplicate_req_id"}
```

Response 404（agent 不在线）:
```json
{"error":"agent_offline"}
```

## 4) 取消工具调用（CANCEL）

`POST /internal/bridge/cancel`

Request:
```json
{"req_id":"req_...","agent_id":"aws-dev-server","reason":"user_cancel"}
```

Response 202:
```json
{"req_id":"req_...","status":"sent"}
```

## 5) 事件回传（CHUNK/RESULT）

MVP 推荐由后端主动“订阅”事件：
- 方案 A：Tunnel Gateway 向后端发 SSE（`GET /internal/bridge/events?agent_id=...`）
- 方案 B：Tunnel Gateway 向后端发 Webhook（简单但不适合流式）

具体实现可在阶段 1 选择其一；但对前端的输出统一由后端 SSE 承担（见 `frontend-sse-events.md`）。
