# 2026-04-16-05 mcpmux Compose Integration Design

## 目标

提供一份可运行的 docker-compose 示例，展示如何让外部 `mcpmux` 通过 HTTP/SSE 接入本仓库的 MCP 服务，并完成鉴权 token 透传与健康检查。

说明：

- `mcpmux` 不在本仓库实现
- 本文档仅定义接入示例的结构与约定；具体 `mcpmux` 镜像、配置字段以使用方实际版本为准

## 目录位置

`deploy/mcpmux/`

建议包含：

- `docker-compose.yaml`
- `.env.example`
- `README.md`（联调说明）
- `mcpmux-config.yaml`（如 `mcpmux` 需要独立配置文件）

## 运行拓扑

- `mcpmux` 作为网关容器
- `vcenter` MCP 服务作为后端容器
- 两者在同一 docker network 内通过服务名访问

## 鉴权与透传

### 目标行为

- `mcpmux` 调用后端 `/sse` 与 `/messages` 时，携带 `Authorization: Bearer <token>`
- 后端服务从环境变量 `MCP_SERVICE_TOKEN` 读取 token 并校验

### compose 侧的建议

- 在 `.env` 中维护 token
- `mcpmux` 与后端同时引用同一个 token 值（确保一致）

## 健康检查

- 后端服务提供：
  - `GET /healthz`
  - `GET /readyz`
- compose 对后端可配置 healthcheck（curl/wget）
- `mcpmux` 如支持后端健康探测，可配置以 `/readyz` 为准

## 接入路由约定

建议每个服务在 `mcpmux` 中以“服务名”注册，并将 base URL 指向：

- `http://vcenter:8080`（示例）

若启用 `MCP_HTTP_BASE_PATH=/vcenter`，则 `mcpmux` 后端配置中需使用：

- `http://vcenter:8080/vcenter`

## 示例配置结构（概念性）

由于不同版本 `mcpmux` 的配置字段可能不同，建议在 `deploy/mcpmux/README.md` 中以“占位”方式表达核心要素：

- 后端名：`vcenter`
- 后端 URL：`http://vcenter:8080`
- transport：HTTP/SSE
- headers：`Authorization: Bearer ${MCP_SERVICE_TOKEN}`

## 联调流程（建议）

- 启动 compose（包含 `mcpmux` + `vcenter`）
- 验证：
  - `vcenter` 的 `/healthz`、`/readyz`
  - `mcpmux` 是否能列出并调用 `vcenter.*` tools
- 失败排查：
  - 401/403：检查 token 一致性与 header 透传
  - 5xx：检查 vCenter 配置与网络可达性
