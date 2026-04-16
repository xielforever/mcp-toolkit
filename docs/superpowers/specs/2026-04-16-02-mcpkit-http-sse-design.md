# 2026-04-16-02 MCPKit HTTP/SSE Design

## 目标

在 `modules/mcpkit` 中提供可复用的 MCP HTTP/SSE 运行框架，供各 `services/*` 快速注册 tools 并以统一的对外协议暴露，便于 `mcpmux` 或其他 MCP 客户端接入。

约束：

- 传输协议采用 MCP 标准 HTTP/SSE
- 鉴权方案在独立文档定义：`2026-04-16-03-authn-authz-design.md`

## 组件边界

`modules/mcpkit` 提供：

- HTTP server 启动与路由装配
- MCP 会话管理（SSE 连接生命周期）
- MCP 消息收发（messages endpoint）
- Tool 注册与调用分发（由服务层提供 tool 集合）
- 标准化错误与响应

服务模块提供：

- Tool 定义（name/description/input schema）
- Tool handler（业务执行）

## 端点约定

默认端点（可通过配置前缀化，避免与反向代理路径冲突）：

- `GET /healthz`
- `GET /readyz`
- `GET /sse`：建立 SSE 连接
- `POST /messages`：客户端向服务发送 MCP messages

服务必须支持在反向代理或网关下运行，推荐保留配置项：

- `MCP_HTTP_BASE_PATH`（默认空），例如 `/vcenter`，则端点变为 `/vcenter/sse`、`/vcenter/messages`

## 会话模型

### 会话标识

- 当客户端请求 `GET /sse` 时，服务创建会话并分配 `session_id`
- 服务通过 SSE 事件向客户端发送“握手/初始化”消息，包含 `session_id`（具体字段按 MCP 标准）
- 客户端后续向 `POST /messages` 发送消息时携带 `session_id`（Header 或 JSON 字段，按 MCP 标准）

### 生命周期

- SSE 连接断开时会话进入过期状态
- 会话缓存保留短 TTL（例如 1-5 分钟）以允许网络抖动恢复
- TTL 后回收会话资源

### 并发与顺序

- 单会话内消息处理顺序按到达顺序串行，避免同一会话并发导致上下文错乱
- 不同会话之间可并行处理
- 服务可配置并发上限以保护后端（例如 `MCP_MAX_CONCURRENCY`）

## 消息处理与分发

### 输入消息类型

服务需支持 MCP 标准消息类型（以“最小可用”为起点）：

- 初始化/能力协商（如 `initialize`）
- 工具列举（如 `tools/list`）
- 工具调用（如 `tools/call`）

### Tool 注册接口（mcpkit 对服务提供）

服务通过 `mcpkit` 注册 tools：

- tool 元信息：`name`、`description`、`inputSchema`
- handler：`func(ctx, input) (output, error)`

Tool 命名建议：`<service>.<action>`（例如 `vcenter.list_inventory`）。

### 输出与错误映射

- 成功响应按 MCP 标准封装
- 失败响应映射为 MCP error（error code + message + optional data）
- `mcpkit` 负责把常见错误分类：
  - 参数错误（400）
  - 鉴权失败（401/403）
  - 下游依赖错误（502/503）
  - 超时（504）
  - 未知错误（500）

## 健康检查

`/healthz`：进程存活检查（不依赖外部依赖）

`/readyz`：服务就绪检查（建议包含必要依赖的轻量探测；例如 vCenter 可选做一次登录或只检查配置完备）

## 可观测性（最小化约定）

- Request ID：若网关传入（如 `X-Request-Id`），则贯穿日志与响应；否则服务生成
- 日志：不记录 secrets（token、密码、AK/SK）
- 指标：至少暴露请求计数、错误计数、延迟分布（具体实现后续计划阶段确定）

## 安全边界（与鉴权文档的衔接）

`mcpkit` 提供鉴权中间件挂载点：

- 对 `/sse` 与 `/messages` 强制鉴权
- 健康检查默认不鉴权（可选开关）

鉴权细节见：

- `2026-04-16-03-authn-authz-design.md`
