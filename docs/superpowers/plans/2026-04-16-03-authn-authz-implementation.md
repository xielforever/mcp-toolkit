# AuthN/AuthZ Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现在 MCP HTTP/SSE 服务中的统一鉴权机制：通过 `Authorization: Bearer <token>` 保护 `/sse` 与 `/messages`，并将高危运维能力以环境变量开关控制。

**Architecture:** 鉴权中间件在 `modules/mcpkit/auth`，由服务在装配路由时使用；token 通过 `MCP_SERVICE_TOKEN` 环境变量注入；高危操作开关在服务内通过 env 控制是否注册对应 tools。

**Tech Stack:** Go 1.22、net/http、testing、httptest

---

## Spec

- `docs/superpowers/specs/2026-04-16-03-authn-authz-design.md`

## Dependencies

- 依赖 mcpkit HTTP 基础与 transport：`docs/superpowers/plans/2026-04-16-02-mcpkit-http-sse-implementation.md`

---

### Task 1: 实现 Bearer 鉴权中间件（auth 包）

**Files:**
- Create: `modules/mcpkit/auth/bearer.go`
- Test: `modules/mcpkit/auth/bearer_test.go`

- [ ] **Step 1: 写失败测试覆盖 401 / 403 / 200**

Create `modules/mcpkit/auth/bearer_test.go`:

```go
package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/<org>/<repo>/modules/mcpkit/auth"
)

func TestBearerAuth(t *testing.T) {
	h := auth.Bearer("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	s := httptest.NewServer(h)
	defer s.Close()

	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, s.URL, nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./modules/mcpkit/auth -run TestBearerAuth -v`
Expected: FAIL

- [ ] **Step 3: 实现 Bearer 中间件**

Create `modules/mcpkit/auth/bearer.go`:

```go
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func Bearer(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			got := strings.TrimPrefix(h, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./modules/mcpkit/auth -run TestBearerAuth -v`
Expected: PASS

---

### Task 2: 在 vcenter main 中挂载鉴权并校验配置（AuthN 部分）

**Files:**
- Modify: `services/vcenter/internal/config/config.go`
- Modify: `services/vcenter/cmd/vcenter-mcp-server/main.go`

- [ ] **Step 1: 在配置中强制要求 MCP_SERVICE_TOKEN**

Ensure `services/vcenter/internal/config/config.go` 中 `Load()` 函数：

```go
	if c.ServiceToken == "" {
		return Config{}, errors.New("MCP_SERVICE_TOKEN is required")
	}
```

- [ ] **Step 2: 在 main 中使用 Bearer 中间件保护 /sse 与 /messages**

Update `services/vcenter/cmd/vcenter-mcp-server/main.go`（重点是 mux 装配方式）：

```go
package main

import (
	"log"
	"net/http"

	"github.com/<org>/<repo>/modules/mcpkit/auth"
	"github.com/<org>/<repo>/modules/mcpkit/httpkit"
	"github.com/<org>/<repo>/modules/mcpkit/mcphttp"
	"github.com/<org>/<repo>/services/vcenter/internal/config"
	"github.com/<org>/<repo>/services/vcenter/internal/tools"
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

- [ ] **Step 3: 提供冒烟测试方式**

Run:

```bash
MCP_SERVICE_TOKEN=secret \
VCENTER_URL=https://example/sdk \
VCENTER_USERNAME=u \
VCENTER_PASSWORD=p \
go run ./services/vcenter/cmd/vcenter-mcp-server
```

手动验证（用 curl）：

```bash
curl -i http://127.0.0.1:8080/sse
# 预期：401

curl -i -H "Authorization: Bearer wrong" http://127.0.0.1:8080/sse
# 预期：403
```

---

### Task 3: 高危运维操作开关（AuthZ 部分）

**Files:**
- Create: `services/vcenter/internal/tools/dangerous.go`
- Modify: `services/vcenter/internal/tools/registry.go`
- Test: `services/vcenter/internal/tools/dangerous_test.go`
- Modify: `services/vcenter/internal/config/config.go`

- [ ] **Step 1: 添加 Options 与 EnableDangerousOps**

Create `services/vcenter/internal/tools/dangerous.go`:

```go
package tools

import "github.com/<org>/<repo>/services/vcenter/internal/vcenter"

type Options struct {
	EnableDangerousOps bool
	API                vcenter.API
}
```

修改 `services/vcenter/internal/tools/registry.go`，将 `BuildRegistry` 升级为：

```go
func BuildRegistry() *mcp.Registry {
	return BuildRegistryWithOptions(Options{})
}

func BuildRegistryWithOptions(opts Options) *mcp.Registry {
	api := opts.API
	r := mcp.NewRegistry()
	// 注册非危险工具（list_inventory、get_vm 等）
	// ...
	if opts.EnableDangerousOps && api != nil {
		registerDangerousTools(r, api)
	}
	return r
}
```

- [ ] **Step 2: 实现 registerDangerousTools，并基于 govmomi API 调用**

在 `dangerous.go` 追加：

```go
package tools

import (
	"context"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
	"github.com/<org>/<repo>/services/vcenter/internal/vcenter"
)

func registerDangerousTools(r *mcp.Registry, api vcenter.API) {
	r.Register(mcp.Tool{
		Name:        "vcenter.power",
		Description: "VM power operations",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			// 具体实现见 vcenter API（Spec 04 的实现计划）
			return nil, nil
		},
	})

	r.Register(mcp.Tool{
		Name:        "vcenter.snapshot",
		Description: "VM snapshot operations",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			// 同上
			return nil, nil
		},
	})
}
```

（危险工具的业务细节在 vCenter Service 计划中实现，此处仅处理“是否注册”的授权层逻辑。）

- [ ] **Step 3: 为危险操作开关写测试，确认默认关闭**

Create `services/vcenter/internal/tools/dangerous_test.go`:

```go
package tools_test

import (
	"context"
	"testing"

	"github.com/<org>/<repo>/services/vcenter/internal/tools"
)

func TestDangerousOps_DefaultDisabled(t *testing.T) {
	reg := tools.BuildRegistryWithOptions(tools.Options{EnableDangerousOps: false})

	_, err := reg.Call(context.Background(), "vcenter.power", map[string]any{})
	if err == nil {
		t.Fatalf("expected error when dangerous ops disabled")
	}
}
```

- [ ] **Step 4: 从配置读取 VCENTER_ENABLE_DANGEROUS_OPS 并传入 Options**

确保 `services/vcenter/internal/config/config.go` 中已有：

```go
c.EnableDangerousOps = strings.EqualFold(os.Getenv("VCENTER_ENABLE_DANGEROUS_OPS"), "true")
```

在 main 中构造 registry 时改为：

```go
opts := tools.Options{
	EnableDangerousOps: cfg.EnableDangerousOps,
	API:                api, // vcenter API，在 Spec 04 实现计划中具体填充
}
reg := tools.BuildRegistryWithOptions(opts)
```

- [ ] **Step 5: 运行 vcenter 工具层测试**

Run: `go test ./services/vcenter/internal/tools -run TestDangerousOps_DefaultDisabled -v`
Expected: PASS

---

## Notes

- 401 与 403 的错误语义仅在 HTTP 层体现；在 MCP 协议层的错误包装可在 mcpkit 做统一映射（如需要）。
- 危险操作仍需在 Spec 04 的实现计划中补充具体 govmomi 逻辑；本实现计划只负责开关与是否注册。
