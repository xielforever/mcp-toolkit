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
- Create: `modules/mcpkit/mcpkit.go`
- Create: `services/vcenter/placeholder.go`
- Create: `services/vcenter/workspace_test.go`
- Create: `services/ces/placeholder.go`

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

- [ ] **Step 1.1: 添加 workspace replace（用于 go work sync 验证闭环）**

在未对外发布 tags 的阶段，`services/*` 对 `modules/mcpkit` 的 `require v0.0.0` 会导致 `go work sync` 尝试从远端拉取该版本。为确保本地闭环，引入 versioned replace（后续发布真实版本后可移除该 replace）。

将以下内容追加到 `go.work`：

```txt
replace github.com/xielforever/mcp-toolkit/modules/mcpkit v0.0.0 => ./modules/mcpkit
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

- [ ] **Step 5: 创建最小可编译包（用于验证闭环）**

Create `modules/mcpkit/mcpkit.go`:

```go
package mcpkit

const Module = "mcpkit"
```

Create `services/vcenter/placeholder.go`:

```go
package vcenter

const Module = "vcenter"
```

Create `services/ces/placeholder.go`:

```go
package ces

const Module = "ces"
```

- [ ] **Step 6: 创建跨 module require 验证测试（vcenter -> mcpkit）**

Create `services/vcenter/workspace_test.go`:

```go
package vcenter_test

import (
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit"
)

func TestWorkspace_RequireResolves(t *testing.T) {
	if mcpkit.Module != "mcpkit" {
		t.Fatalf("unexpected module: %s", mcpkit.Module)
	}
}
```

- [ ] **Step 7: 验证 go.work 依赖元数据可生成**

Run: `go work sync`
Expected: 生成/更新 `go.work.sum` 且命令成功退出

- [ ] **Step 8: 验证工作区解析与跨 module 引用（以测试为准）**

Run: `go env GOWORK`
Expected: `GOWORK=/workspace/go.work`

Run: `go test ./modules/mcpkit/... ./services/vcenter/... ./services/ces/...`
Expected: PASS（其中 `services/vcenter` 的 `TestWorkspace_RequireResolves` 必须通过）
