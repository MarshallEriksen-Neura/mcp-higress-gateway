# Implementation Plan：MCP Bridge（Go Agent + Tunnel Gateway）落地

**Branch**: `004-mcp-bridge` | **Date**: 2025-12-20 | **Spec**: `specs/004-mcp-bridge/spec.md`  
**Input**: `docs/bridge/design.md` + `docs/bridge/ai-higress-mcp-integration.md`

## Summary

本计划将设计文档落地为可迭代交付的实现路径：
- Go Bridge Agent 负责 MCP 聚合/路由/流式日志与 RESULT 可靠终态；
- Go Tunnel Gateway 负责云端 WSS 连接层（单实例 MVP 不引入 Redis）；
- FastAPI 后端负责 LLM 编排（agent loop）与 SSE 给前端；
- Web 前端只展示：工具日志流、工具结果、取消按钮与配置下载（方案 A）。

## Technical Context

**Language/Version**:
- Backend: Python 3.12 + FastAPI
- Frontend: Next.js（App Router）+ Tailwind + shadcn/ui
- Bridge: Go 1.21+（`log/slog`）

**Primary Dependencies**:
- Go CLI: `github.com/spf13/cobra` + `github.com/spf13/viper`
- Go MCP: `github.com/modelcontextprotocol/go-sdk`（固定）
- Go WS: `nhooyr.io/websocket` 或 `github.com/gorilla/websocket`

**Storage**: MVP 不新增强依赖（无 Redis）。沿用现有 DB/Redis（若已存在）仅用于项目已有能力，不作为 Bridge MVP 必需项。  
**Testing**:
- Backend: pytest（按仓库规范，AI 不运行测试）
- Bridge: go test（仅在需要时增加）
- Frontend: 现有 e2e/单测体系（按项目既有方式）

**Target Platform**:
- Agent: Windows/macOS/Linux（x86/ARM），单二进制分发
- Tunnel Gateway: Linux server（云端部署）

**Project Type**: Web app + 新增 Go 子项目（同仓可拆分）  
**Performance Goals**:
- 日志流延迟：工具输出后 1s 内前端可见（网络允许情况下）
- 背压：大日志不阻塞工具进程，允许丢弃并可观测

**Constraints**:
- 用户侧零额外依赖（不要求安装 Redis）
- 敏感信息不进后端/日志（方案 A + 脱敏日志）

**Scale/Scope**:
- MVP: 单实例 Tunnel Gateway（无 Redis/消息总线）
- 扩展: K8s 多实例/HA 再引入消息系统（Redis/NATS/Kafka 之一）

## Constitution Check

- 代码质量：Go/Python 代码保持小而清晰的职责划分；敏感字段不写日志。
- 测试：关键路径补齐最小集成测试（至少覆盖 tool 执行的 SSE 事件序列）。
- 体验一致性：前端展示与错误码/错误消息统一；取消/重试可解释。
- 性能：避免阻塞 I/O；长连接资源可控；背压策略明确。
- 安全：最小权限工具；配置文件权限提示；禁止默认提供任意 shell 工具。

## Project Structure

### Documentation（本 feature）

```text
specs/004-mcp-bridge/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── internal-tunnel-gateway-http.md
│   ├── ws-envelope-protocol.md
│   └── frontend-sse-events.md
└── tasks.md
```

### Source Code（仓库根）

```text
backend/
  app/
    ...（新增：bridge/tunnel client + chat loop 集成 + SSE events）
frontend/
  ...（新增：配置下载页 + chat 工具日志展示）
bridge/                 # 新增（Go module）
  cmd/
    agent/
    tunnel-gateway/
  internal/
    config/
    protocol/
    mcp/
    backpressure/
    reliable/
docs/bridge/
  design.md
  ai-higress-mcp-integration.md
```

**Structure Decision**: 采用“同仓双项目”结构：Go Bridge 可独立 module，便于后续拆仓；后端/前端仅做集成点改动。
