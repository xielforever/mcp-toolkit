# Monorepo Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 初始化 Go Monorepo 工作区（go.work + 多 go.mod），为后续 mcpkit 与各 MCP 服务提供依赖隔离与本地联调能力。

**Architecture:** 使用 `go.work` 聚合 `modules/mcpkit`、`services/vcenter`、`services/ces` 三个 module；服务 module 通过 `require github.com/xielforever/mcp-toolkit/modules/mcpkit v0.0.0` 依赖共享库，并由 workspace 在本地解析源代码。

**Tech Stack:** Go 1.22、go.work

---

## Spec

- `docs/superpowers/specs/2026-04-16-01-monorepo-workspace-design.md`

## 目标目录结构（本计划完成后）

```
.
├── go.work
├── modules/
│   └── mcpkit/
│       └── go.mod
└── services/
    ├── vcenter/
    │   └── go.mod
    └── ces/
        └── go.mod
```

---

### Task 1: 初始化 go.work 与三个 module 的 go.mod

**Files:**
- Create: `go.work`
- Create: `modules/mcpkit/go.mod`
- Create: `services/vcenter/go.mod`
- Create: `services/ces/go.mod`

- [ ] **Step 1: 创建 go.work**

Create `go.work`:

```txt
go 1.22

use (
	./modules/mcpkit
	./services/vcenter
	./services/ces
)
```

- [ ] **Step 2: 创建 modules/mcpkit/go.mod**

Create `modules/mcpkit/go.mod`:

```go
module github.com/xielforever/mcp-toolkit/modules/mcpkit

go 1.22
```

- [ ] **Step 3: 创建 services/vcenter/go.mod**

Create `services/vcenter/go.mod`:

```go
module github.com/xielforever/mcp-toolkit/services/vcenter

go 1.22

require github.com/xielforever/mcp-toolkit/modules/mcpkit v0.0.0
```

- [ ] **Step 4: 创建 services/ces/go.mod**

Create `services/ces/go.mod`:

```go
module github.com/xielforever/mcp-toolkit/services/ces

go 1.22

require github.com/xielforever/mcp-toolkit/modules/mcpkit v0.0.0
```

- [ ] **Step 5: 验证工作区解析与跨 module 引用**

Run: `go env GOWORK`
Expected: `GOWORK=/workspace/go.work`

Run: `go -C modules/mcpkit list -m`
Expected: 输出 `github.com/xielforever/mcp-toolkit/modules/mcpkit`（workspace 模式下可能会同时列出 workspace 内其它 module）

Run: `go -C services/vcenter list -m`
Expected: 输出 `github.com/xielforever/mcp-toolkit/services/vcenter`（workspace 模式下可能会同时列出 workspace 内其它 module）

Run: `go -C services/ces list -m`
Expected: 输出 `github.com/xielforever/mcp-toolkit/services/ces`（workspace 模式下可能会同时列出 workspace 内其它 module）
