# CES Service Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 创建 `services/ces` 的可运行占位 MCP 服务模块（HTTP/SSE + Bearer 鉴权），不实现 CES 具体业务能力，仅提供最小 ping tool，为后续迭代留出结构边界。

**Architecture:** 复用 `modules/mcpkit` 的 httpkit/mcphttp/auth；CES 服务与 vcenter 服务保持同样的 `cmd`、`internal/config`、`internal/tools` 组织方式。

**Tech Stack:** Go 1.22、net/http

---

## Spec

- `docs/superpowers/specs/2026-04-16-06-ces-service-skeleton-design.md`

## Dependencies

- Workspace：`docs/superpowers/plans/2026-04-16-01-monorepo-workspace-implementation.md`
- MCPKit：`docs/superpowers/plans/2026-04-16-02-mcpkit-http-sse-implementation.md`
- Auth：`docs/superpowers/plans/2026-04-16-03-authn-authz-implementation.md`

---

### Task 1: 创建 CES 服务配置与 tool registry

**Files:**
- Create: `services/ces/internal/config/config.go`
- Create: `services/ces/internal/tools/registry.go`

- [ ] **Step 1: 实现 CES 配置加载（仅 token + listen/base path）**

Create `services/ces/internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
)

type Config struct {
	ListenAddr   string
	BasePath     string
	ServiceToken string
}

func Load() (Config, error) {
	c := Config{
		ListenAddr:   os.Getenv("HTTP_LISTEN_ADDR"),
		BasePath:     os.Getenv("MCP_HTTP_BASE_PATH"),
		ServiceToken: os.Getenv("MCP_SERVICE_TOKEN"),
	}
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:8080"
	}
	if c.ServiceToken == "" {
		return Config{}, errors.New("MCP_SERVICE_TOKEN is required")
	}
	return c, nil
}
```

- [ ] **Step 2: 注册 ces.ping**

Create `services/ces/internal/tools/registry.go`:

```go
package tools

import (
	"context"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
)

func BuildRegistry() *mcp.Registry {
	r := mcp.NewRegistry()
	r.Register(mcp.Tool{
		Name:        "ces.ping",
		Description: "ping",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	})
	return r
}
```

---

### Task 2: 实现 CES 服务 main 并挂载 MCP HTTP/SSE + 鉴权

**Files:**
- Create: `services/ces/cmd/ces-mcp-server/main.go`

- [ ] **Step 1: 实现 main**

Create `services/ces/cmd/ces-mcp-server/main.go`:

```go
package main

import (
	"log"
	"net/http"

	"github.com/<org>/<repo>/modules/mcpkit/auth"
	"github.com/<org>/<repo>/modules/mcpkit/httpkit"
	"github.com/<org>/<repo>/modules/mcpkit/mcphttp"
	"github.com/<org>/<repo>/services/ces/internal/config"
	"github.com/<org>/<repo>/services/ces/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	reg := tools.BuildRegistry()
	mcpHandler := mcphttp.NewHandler(mcphttp.Config{Registry: reg})

	mux := http.NewServeMux()
	mux.Handle("/sse", auth.Bearer(cfg.ServiceToken)(mcpHandler))
	mux.Handle("/messages", auth.Bearer(cfg.ServiceToken)(mcpHandler))
	httpkit.RegisterHealth(mux)

	root := httpkit.NewMux(httpkit.MuxConfig{BasePath: cfg.BasePath})
	server := &http.Server{Addr: cfg.ListenAddr, Handler: root}
	log.Fatal(server.ListenAndServe())
}
```

- [ ] **Step 2: 编译**

Run: `go build ./services/ces/cmd/ces-mcp-server`
Expected: build success

