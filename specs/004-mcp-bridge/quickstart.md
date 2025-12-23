# Quickstart：MCP Bridge MVP（无 Redis）

本 Quickstart 目标是验证最小闭环：Agent 在线 -> tools 上报 -> Web/后端触发一次工具调用 -> 前端看到流式日志 -> 收到最终结果。

> 说明：这里的“无 Redis”指 **MVP 不把 Redis 作为必需依赖**。项目现有后端若已经依赖 Redis（例如会话/缓存），不在本文范围内强制移除。

## 1. 本地准备（用户侧）

1) 下载/生成配置文件（方案 A）
- 在网页配置向导中填写 MCP servers
- 下载 `config.yaml`

2) 导入配置（写入默认路径）
- `bridge config apply --file ./config.yaml`
- 可选：`bridge config validate --file ./config.yaml`

3) 启动 Agent
- `bridge agent start`

期望现象:
- 控制台（stderr）显示已连接云端、已上报 tools。

## 2. 云端启动（开发态，单实例）

1) 启动 Tunnel Gateway（Go）
- `bridge gateway serve --listen :8088`

2) 启动后端（FastAPI）
- 按仓库现有方式启动（例如 `uvicorn` / `apiproxy`）

3) 启动前端（Next.js）
- 按仓库现有方式启动（`bun dev` / `next dev`）

## 3. 验证一次工具调用（流式）

- 打开 Web Chat
- 选择目标 `agent_id`（若 MVP 暂时固定，也应在 UI/后端明确绑定）
- 触发一次会产生持续输出的工具调用

期望现象:
- 前端持续收到 `tool_log`（stdout/stderr）
- 结束后收到 `tool_result`
- 点击“停止”能触发取消并最终收敛到单一 `tool_result`

## 4. 建议的测试命令（由人类开发者运行）

后端:
- `pytest`

前端（若有）:
- `bun test`
- `bun run test:e2e`

Go:
- `go test ./...`

> 注意：AI 助手不运行测试；请你运行后把结果贴回来，我可以继续迭代。
