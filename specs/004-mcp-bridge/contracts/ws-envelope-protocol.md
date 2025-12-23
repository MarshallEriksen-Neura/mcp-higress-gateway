# Contract：Tunnel WS 信封协议（项目内部 v1）

本协议用于 Agent ⇄ Tunnel Gateway 的隧道层通信（WSS）。它是项目内部契约（internal），不承诺第三方稳定性，但需要在后端/前端/Agent/Tunnel 之间保持一致。

## 信封格式（JSON）

```json
{
  "v": 1,
  "type": "INVOKE",
  "agent_id": "aws-dev-server",
  "req_id": "req_...",
  "conn_session_id": "ws_...",
  "seq": 123,
  "ts": 1712345678,
  "payload": {}
}
```

字段语义:
- `agent_id`: 路由目标
- `req_id`: 工具调用幂等键
- `conn_session_id`: 单次 WS 会话 id（诊断/安全绑定）
- `seq`: 连接内序号（观测/排错）

## 消息类型（v1）

### HELLO（Agent -> Gateway）
payload（示例）:
```json
{
  "agent_meta": {"os":"linux","arch":"amd64","hostname":"..."},
  "resume": {"pending_result_req_ids": ["req_..."]}
}
```

### AUTH（Agent -> Gateway）
payload（示例）:
```json
{"token":"...","device_fingerprint":"..."}
```

### PING / PONG（双向）
payload（示例）:
```json
{"ts":1712345678}
```

### TOOLS（Agent -> Gateway）
payload（示例）:
```json
{"tools":[{"name":"fs__readFile","description":"...","input_schema":{}}]}
```

### INVOKE（Gateway -> Agent）
payload（示例）:
```json
{
  "tool": {"name":"fs__readFile","args":{"path":"/tmp/a.txt"}},
  "timeout_ms": 60000,
  "stream": {"enabled": true}
}
```

### INVOKE_ACK（Agent -> Gateway）
payload（示例）:
```json
{"accepted": true}
```

### CHUNK（Agent -> Gateway）
payload（示例）:
```json
{
  "stream_id":"s1",
  "channel":"stdout",
  "data":"...",
  "dropped_bytes":0,
  "dropped_lines":0
}
```

### RESULT（Agent -> Gateway）
payload（示例）:
```json
{
  "ok": true,
  "exit_code": 0,
  "result_json": {"content":[{"type":"text","text":"..."}]},
  "canceled": false
}
```

### RESULT_ACK（Gateway -> Agent）
payload（示例）:
```json
{}
```

### CANCEL（Gateway -> Agent）
payload（示例）:
```json
{"reason":"user_cancel"}
```

### CANCEL_ACK（Agent -> Gateway）
payload（示例）:
```json
{"will_cancel": true, "reason": ""}
```

## 可靠性约束

- CHUNK：尽力而为（可丢），但应通过 `dropped_*` 可观测。
- RESULT：强可靠（不可丢）。Agent 在收到 RESULT_ACK 前必须缓存，重连后可补发。
- 幂等：云端对 `req_id` 维度去重（重复 RESULT 仅接受首次终态）。
