# Research：MCP Bridge 技术调研与决策

本文件记录本 feature 的关键技术选型与“为什么”，用于后续维护与替换成本评估。

## MCP SDK

结论：固定使用官方 Go SDK。

- 首选：`github.com/modelcontextprotocol/go-sdk`
  - 原因：官方维护、覆盖 MCP client/server/session 的核心模型；支持 stdio/command；并包含 HTTP/SSE/streamable 相关实现（视接入的 MCP server transport 使用）。
  - 关键能力：工具调用基于 `context.Context` 支持取消；便于与本项目的 CANCEL 语义对齐。
- 备选：`github.com/mark3labs/mcp-go`
  - 使用条件：仅在官方 SDK 缺失某个必需 transport/特性且短期无法绕过时启用。
  - 风险：双栈维护成本上升（需避免）。

## CLI / 配置 / 日志

- CLI：Cobra（`github.com/spf13/cobra`）
- 配置：Viper（`github.com/spf13/viper`）
  - 推荐优先级：flags > env > config > defaults
  - ENV 前缀：`AI_BRIDGE_`
- 日志：默认 `log/slog`
  - 约束：日志默认写 stderr；stdout 用于业务/管道/协议数据（避免混淆与污染）。
  - 必须脱敏：任何 token/password 不出日志。

## Tunnel Gateway 与“无 Redis MVP”

用户侧体验要求“安装即用，不额外安装 Redis”。因此：

- MVP 选择：Tunnel Gateway 单实例部署（内存连接表）
  - 后端（FastAPI）通过内网 HTTP/gRPC 直接向 Tunnel Gateway 下发 INVOKE/CANCEL
  - Agent 端通过 RESULT_ACK 做终态可靠交付（缓存/重传/去重）
  - 适用于：早期验证、单租户/低规模、或自托管一键部署场景

- 扩展阶段（HA）再引入消息系统：
  - Redis（Registry + Streams + PubSub）或 NATS/Kafka
  - 目的：跨实例路由、可靠队列、回传解耦
  - 注意：这是云端部署复杂度，不影响用户侧安装体验

## Transport 范围（MCP 子服务）

Aggregator 首期优先级：
1) 本地 MCP 子进程（stdio）：最稳定、生态最成熟（npx/python/docker 都可）
2) 远程 MCP（HTTP/SSE/streamable）：作为扩展项（依赖具体 server 是否提供）

## 命名空间与兼容性

- 工具命名空间：`{server}__{tool}`
- 原因：
  - 避免工具重名
  - 避免部分上游/客户端对 tool name 字符集限制（保守使用字母/数字/下划线）

## 风险与回避

- 风险：日志量过大导致阻塞工具进程
  - 回避：Agent 端“有界队列 + 丢弃计数 + 小分片 CHUNK”
- 风险：取消/完成竞争态
  - 回避：以单一 RESULT 收敛；req_id 幂等去重；CANCEL 仅是“请求取消”
