# Feature Specification：AI-Higress 接入 MCP Bridge（Go Agent + Tunnel Gateway）

**Feature Branch**: `004-mcp-bridge`  
**Created**: 2025-12-20  
**Status**: Draft  
**Input**:
- 通用架构：`docs/bridge/design.md`
- 项目落地：`docs/bridge/ai-higress-mcp-integration.md`

## User Scenarios & Testing

### User Story 1 - 用户无需额外依赖即可接入本地工具（Priority: P1）

用户在自己的机器/服务器上安装并启动 Bridge Agent，使用本地 `config.yaml` 配置 MCP servers，Agent 能连接云端 Tunnel Gateway 并上报可用工具列表。

**Why this priority**: 这是所有后续“网页调用工具/流式回显”的前置条件，也是用户“安装即用”的关键体验。

**Independent Test**: 不依赖前端改动；只要能看到 Agent 在线并成功上报 TOOLS，即可证明接入链路可用。

**Acceptance Scenarios**:
1. **Given** 用户准备好 `~/.ai-bridge/config.yaml`，**When** 运行 `bridge agent start`，**Then** Agent 与云端建立 WSS 连接并完成鉴权与心跳。
2. **Given** Agent 已连接，**When** Agent 上报 TOOLS，**Then** 后端能获取该 `agent_id` 的工具列表（工具名已命名空间化且无冲突）。

---

### User Story 2 - Web 端调用工具并流式回显日志（Priority: P1）

用户在 Web Chat 中触发工具调用（由 LLM tool_calls 决策或后端的“测试调用”），能实时看到 stdout/stderr 日志流，并在最终收到结构化结果；支持取消。

**Why this priority**: 这是用户感知价值最大的一条链路；你们的业务定义明确要求长耗时任务必须可流式回显。

**Independent Test**: 使用一个耗时工具（例如 sleep + 输出日志）即可独立验证 CHUNK + RESULT + CANCEL 的时序与语义。

**Acceptance Scenarios**:
1. **Given** 选择了 `agent_id` 且工具存在，**When** 发起 INVOKE，**Then** 前端在 1 秒内开始收到 `tool_log` SSE（stdout/stderr）。
2. **Given** 工具执行完成，**When** Agent 上报 RESULT，**Then** 前端收到 `tool_result` SSE，后端可将结果回填给 LLM 继续生成最终回复。
3. **Given** 工具执行中，**When** 用户点击停止，**Then** 后端下发 CANCEL，最终以单一 RESULT 收敛（可能 canceled 或已完成）。

---

### User Story 3 - Web 端配置向导生成 config.yaml（Priority: P2）

用户在网页中填写 MCP servers 配置，浏览器本地生成 `config.yaml` 并下载；后端不触碰密钥明文；用户可用 CLI 导入并写入默认路径。

**Why this priority**: 提升接入体验，同时满足“后端零触碰敏感信息”的安全边界。

**Independent Test**: 不依赖 Agent 在线；只要下载得到符合 schema 的 `config.yaml` 并能通过 `bridge config validate` 即可。

**Acceptance Scenarios**:
1. **Given** 用户在网页填写配置，**When** 点击下载，**Then** 浏览器下载到包含对应配置项的 `config.yaml`。
2. **Given** 用户下载的 `config.yaml`，**When** 执行 `bridge config validate --file ./config.yaml`，**Then** 校验通过且不会在日志输出敏感字段。

---

### User Story 4 - 其他客户端可复用 Bridge（标准 MCP Server 模式）（Priority: P3）

Bridge Agent 可选以标准 MCP server 模式运行（stdio / streamable http），供 Claude Desktop / Cursor / Copilot 等 MCP 客户端直接连接使用。

**Why this priority**: 增强生态兼容性与可维护性，但不阻塞 Web 闭环 MVP。

**Independent Test**: 用官方 MCP client（或简单的 stdio client）连接 bridge 并执行一个工具即可。

**Acceptance Scenarios**:
1. **Given** Bridge 以 stdio 模式运行，**When** MCP 客户端连接并 `tools/list`，**Then** 能列出聚合工具。
2. **Given** MCP 客户端发起 `tools/call`，**When** Bridge 路由到对应子 server，**Then** 返回结果符合 MCP 规范。

---

### Edge Cases
- Agent 离线/掉线：后端应明确返回“agent offline”，并在 UI 侧提示用户选择其他 agent 或重连。
- 工具重名/不合法：工具名必须命名空间化；遇到非法字符时做可逆映射或拒绝并给出原因。
- 日志爆量：Agent 不能因网络慢阻塞工具进程；应丢弃并上报 `dropped_bytes/lines`。
- 取消竞争态：用户取消与工具完成并发，最终必须以单一 RESULT 收敛且幂等。
- 断线重连：CHUNK 可丢；RESULT 必须可通过缓存 + RESULT_ACK 机制补发。
- 安全：任何 token/password 不得出现在日志；config 文件权限提示；默认不提供任意 shell 工具。

## Requirements

### Functional Requirements
- **FR-001**: Bridge Agent 必须可在用户侧“安装即用”，不要求用户额外安装 Redis 等基础设施依赖。
- **FR-002**: MCP SDK 固定使用 `github.com/modelcontextprotocol/go-sdk`（备选 `mark3labs/mcp-go` 仅作为兜底）。
- **FR-003**: Agent 必须支持聚合多个 MCP servers（stdio 优先，远程 SSE/HTTP 作为可选扩展）。
- **FR-004**: 工具名必须命名空间化为 `{server}__{tool}`，并保证可稳定映射回源工具名。
- **FR-005**: 必须支持流式日志回显：stdout/stderr -> CHUNK -> 后端 SSE -> 前端实时展示。
- **FR-006**: 必须支持取消：后端可对 `req_id` 下发 CANCEL；Agent 尽力取消并以 RESULT 收敛。
- **FR-007**: RESULT 必须强可靠：Agent 在收到 RESULT_ACK 前缓存终态，并在重连后补发未确认结果；云端以 `req_id` 幂等去重。
- **FR-008**: 方案 A（网页下载 config.yaml）必须做到后端不触碰敏感信息明文。
- **FR-009**: 日志必须默认写 stderr 并脱敏（不输出 token/password；必要时输出“已脱敏/字段存在”级别信息）。

### Key Entities
- **Agent**: 用户的一台边缘机器/服务器实例；由 `agent_id` 唯一标识。
- **MCPServerConfig**: 单个子 MCP server 的连接信息（command/args/env 或 url）。
- **ToolDescriptor**: 命名空间化后的工具定义（供 LLM 与前端展示）。
- **InvokeRequest / InvokeAck / Chunk / Result**: 一次工具调用的生命周期消息（以 `req_id` 关联）。

## Success Criteria

### Measurable Outcomes
- **SC-001**: 用户可在无 Redis 的前提下完成端到端闭环：Agent 在线 -> tools 上报 -> Web 发起调用 -> 看到流式日志 -> 收到终态结果。
- **SC-002**: 在工具持续输出日志的情况下，前端能持续收到 SSE；即使发生丢弃，系统可通过 `dropped_*` 可观测提示“日志不完整”。
- **SC-003**: 任意一次工具调用在断线重连后仍能最终拿到 RESULT（通过缓存/ACK/补发），且不会重复执行导致重复副作用（req_id 幂等去重）。
