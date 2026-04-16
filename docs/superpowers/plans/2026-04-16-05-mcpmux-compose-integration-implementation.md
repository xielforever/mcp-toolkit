# mcpmux Compose Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提供 `deploy/mcpmux` 的 docker-compose 接入示例，展示 mcpmux 如何以 HTTP/SSE 接入 vCenter MCP 服务并透传 `Authorization: Bearer` token。

**Architecture:** compose 中包含 `vcenter` 服务容器与 `mcpmux` 容器（mcpmux 镜像/配置字段保持占位但必须明确 token 与后端 URL/transport 的关键项）。通过 `.env.example` 统一注入 `MCP_SERVICE_TOKEN` 并在 README 说明联调步骤。

**Tech Stack:** Docker Compose、dotenv、HTTP/SSE

---

## Spec

- `docs/superpowers/specs/2026-04-16-05-mcpmux-compose-integration-design.md`

## Dependencies

- vCenter 服务可构建/可运行：`docs/superpowers/plans/2026-04-16-04-vcenter-service-implementation.md`
- 鉴权约定：`docs/superpowers/plans/2026-04-16-03-authn-authz-implementation.md`

---

### Task 1: 创建 deploy/mcpmux 目录与环境变量示例

**Files:**
- Create: `deploy/mcpmux/.env.example`

- [ ] **Step 1: 写 .env.example（仅示例，不包含真实凭据）**

Create `deploy/mcpmux/.env.example`:

```dotenv
MCP_SERVICE_TOKEN=change-me
VCENTER_URL=https://vcenter.example/sdk
VCENTER_USERNAME=admin
VCENTER_PASSWORD=change-me
VCENTER_INSECURE=false
VCENTER_ENABLE_DANGEROUS_OPS=false
```

---

### Task 2: 创建 docker-compose.yaml（mcpmux 镜像与配置占位）

**Files:**
- Create: `deploy/mcpmux/docker-compose.yaml`

- [ ] **Step 1: 写 compose 文件（确保 token 注入到 vcenter，mcpmux 同步持有 token）**

Create `deploy/mcpmux/docker-compose.yaml`:

```yaml
services:
  vcenter:
    image: vcenter-mcp-server:local
    environment:
      HTTP_LISTEN_ADDR: 0.0.0.0:8080
      MCP_SERVICE_TOKEN: ${MCP_SERVICE_TOKEN}
      VCENTER_URL: ${VCENTER_URL}
      VCENTER_USERNAME: ${VCENTER_USERNAME}
      VCENTER_PASSWORD: ${VCENTER_PASSWORD}
      VCENTER_INSECURE: ${VCENTER_INSECURE}
      VCENTER_ENABLE_DANGEROUS_OPS: ${VCENTER_ENABLE_DANGEROUS_OPS}
    ports:
      - "18080:8080"

  mcpmux:
    image: mcpmux:replace-me
    environment:
      MCP_SERVICE_TOKEN: ${MCP_SERVICE_TOKEN}
    depends_on:
      - vcenter
    ports:
      - "19090:9090"
```

---

### Task 3: 输出 README 联调说明（明确后端配置关键项）

**Files:**
- Create: `deploy/mcpmux/README.md`

- [ ] **Step 1: 写 README（包含 mcpmux 后端注册要点与验证步骤）**

Create `deploy/mcpmux/README.md`:

```md
# mcpmux integration (docker-compose)

## Backend registration (concept)

- backend name: `vcenter`
- backend base URL: `http://vcenter:8080`
- transport: HTTP/SSE
- headers:
  - `Authorization: Bearer ${MCP_SERVICE_TOKEN}`

If your deployment uses a base path (e.g. `MCP_HTTP_BASE_PATH=/vcenter`), set backend URL to:

- `http://vcenter:8080/vcenter`

## Smoke checks

- `GET http://localhost:18080/healthz`
- `GET http://localhost:18080/readyz`

Auth checks:

- `GET http://localhost:18080/sse` -> 401
- `GET http://localhost:18080/sse` with `Authorization: Bearer wrong` -> 403

When mcpmux is correctly configured, verify:

- `tools/list` returns `vcenter.*` tools
- `tools/call` can call `vcenter.list_inventory` (needs valid vCenter credentials)
```

