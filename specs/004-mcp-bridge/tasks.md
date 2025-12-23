---

description: "Tasks for MCP Bridge implementation"
---

# Tasks：MCP Bridge（004-mcp-bridge）

**Input**: `specs/004-mcp-bridge/spec.md` / `specs/004-mcp-bridge/plan.md` / `docs/bridge/design.md`  
**Prerequisites**: 完成 contracts/ 约定后再进入实现（避免反复返工）。

> 说明：本 tasks 以用户故事分组，保证每个故事可独立交付与验证。

## Phase 1：仓库结构与脚手架（Shared）

- [x] T001 建立 Go 子项目目录 `bridge/`（module、基本 lint、构建脚本）
- [x] T002 实现配置加载骨架（Viper）：`bridge/internal/config/*`（支持 config file + env + flags）
- [x] T003 实现日志封装层（slog）：`bridge/internal/logging/*`（stderr 输出 + 脱敏）
- [x] T004 定义 WS 信封协议类型：`bridge/internal/protocol/*`（与 `contracts/ws-envelope-protocol.md` 对齐）

## Phase 2：US1（P1）Agent 在线 + tools 上报（MVP）

- [x] T010 [US1] 实现 `bridge cmd agent start`（Cobra）：解析 `agent_id/server.url/token`
- [x] T011 [US1] Agent 建立 WSS 连接、HELLO/AUTH/PING/PONG：`bridge/cmd/bridge/cmd/agent_cmd.go`
- [x] T012 [US1] Agent 管理 MCP 子进程（stdio/command）：`bridge/internal/mcpbridge/aggregator.go`
- [x] T013 [US1] tools/list 聚合与命名空间 `{server}__{tool}`：`bridge/internal/mcpbridge/aggregator.go`
- [x] T014 [US1] Agent 上报 TOOLS 给 Tunnel Gateway（WS）：`bridge/cmd/bridge/cmd/agent_cmd.go`

## Phase 3：US2（P1）工具调用 + 流式日志 + 终态可靠 + 取消

- [x] T020 [US2] 实现 Tunnel Gateway `bridge cmd gateway serve`：WSS 接入与连接表 `map[agent_id]*Conn`
- [x] T021 [US2] 实现 INVOKE 下发与 INVOKE_ACK 回收：`bridge/cmd/bridge/cmd/gateway_cmd.go`
- [x] T022 [US2] Agent 执行 tools/call：解析命名空间 -> 路由到子 MCP server -> 返回 RESULT
- [x] T023 [US2] 流式日志：MCP progress/log 通知 -> CHUNK（分片 4–16KB）
- [x] T024 [US2] 背压：CHUNK 有界队列 + dropped 计数：`bridge/internal/backpressure/bounded_bytes_channel.go`
- [x] T025 [US2] RESULT 强可靠：缓存未 ack RESULT + 重连补发（Agent 侧已实现；Gateway 可选 Redis 结果持久化）
- [x] T026 [US2] CANCEL 支持：云端下发 -> Agent 尽力取消（ctx cancel）

## Phase 4：US2（P1）后端编排与 SSE 映射

- [x] T030 [US2] 后端新增/调整 “指定 agent_id 的工具执行通道”（内部 service 层），对齐 `contracts/internal-tunnel-gateway-http.md`
- [x] T031 [US2] 后端实现 tool 执行 SSE 事件：`tool_status/tool_log/tool_result`（对齐 `contracts/frontend-sse-events.md`，新增 `GET /v1/bridge/tool-events`）
- [x] T032 [US2] 后端 agent loop 集成：LLM tool_calls -> INVOKE -> tool_result -> 继续生成（在会话发消息接口支持 `bridge_agent_id` 后启用）
- [x] T033 [US2] 后端错误分类：agent_offline/tool_not_found/invoke_timeout（透传到 `tool_result.error.code`，并在 `/v1/bridge/invoke|cancel` 统一返回标准错误体）
- [x] T034 [US2] 前端 Chat 接入 Bridge 调试面板：会话页可直接调用工具并看 CHUNK/RESULT（`frontend/components/chat/bridge-panel-client.tsx`）
- [x] T035 [US2] 前端 Chat 将工具执行作为一等公民展示（会话级选择 `agent_id`，发送消息时透传 `bridge_agent_id`，并可从回复卡片一键打开 Bridge 面板查看对应 req_id 日志）

## Phase 5：US3（P2）Web 配置下载（方案 A）

- [x] T040 [US3] 前端新增配置向导：生成 `config.yaml` 并下载（不经后端，落在 Dashboard Bridge 的 Config Tab）
- [x] T041 [US3] 前端文案与 i18n key 补齐（按项目 i18n 规范）
- [x] T042 [US3] CLI 子命令：`bridge config validate/apply/path`（写入默认路径、权限提示、脱敏日志）

## Phase 6：US4（P3）标准 MCP Server 模式（可选）

- [x] T050 [US4] Agent 支持 stdio MCP server 模式（对其他客户端暴露聚合工具，`bridge agent serve-mcp`）
- [ ] T051 [US4] 文档/示例：给 Claude Desktop/Cursor 的最小接入片段（不涉及敏感信息）

## Phase 7：可选扩展（HA）

- [ ] T060 引入云端消息系统（Redis/NATS/Kafka 任选其一）实现多实例路由与可靠队列（不影响用户侧；当前 Go Gateway 已具备 Redis Registry/Streams/PubSub/Result KV 的可选实现，但后端尚未接入发布/查询路径）
