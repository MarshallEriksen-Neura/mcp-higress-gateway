# Contract：Backend -> Frontend（SSE 事件约定）

本约定用于前端展示工具执行的实时日志与终态结果。后端仍保持现有的 SSE 输出机制（`text/event-stream`），在其上新增或统一若干事件类型。

## Event：tool_status

```json
{
  "req_id": "req_...",
  "agent_id": "aws-dev-server",
  "state": "sent|acked|running|canceled|done|error",
  "message": ""
}
```

## Event：tool_log

```json
{
  "req_id": "req_...",
  "agent_id": "aws-dev-server",
  "channel": "stdout|stderr",
  "data": "....",
  "dropped_bytes": 0,
  "dropped_lines": 0
}
```

说明:
- `dropped_*` 非 0 时，前端应提示“日志不完整（发生丢弃）”。

## Event：tool_result

```json
{
  "req_id": "req_...",
  "agent_id": "aws-dev-server",
  "ok": true,
  "exit_code": 0,
  "canceled": false,
  "result_json": {},
  "error": null
}
```

## Event：tool_error（可选）

当后端在未进入 tool 执行前就发现错误（例如 agent_offline / tool_not_found）时，发送统一错误事件。

```json
{
  "req_id": "req_...",
  "agent_id": "aws-dev-server",
  "code": "agent_offline|tool_not_found|invoke_timeout|internal_error",
  "message": ""
}
```
