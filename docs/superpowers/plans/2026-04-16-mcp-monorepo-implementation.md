# MCP Monorepo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 Go Monorepo（go.work + 多 go.mod）并交付可运行的 vCenter MCP HTTP/SSE 服务（含 Bearer 鉴权与高危操作默认关闭），同时提供 CES 模块占位与 mcpmux docker-compose 接入示例。

**Architecture:** 仓库采用 `go.work` 聚合工作区；共享基础设施下沉到 `modules/mcpkit`（HTTP/SSE、鉴权、配置、错误映射、健康检查、tool 分发）；各服务在 `services/*` 以独立 module 方式依赖 `mcpkit`。

**Tech Stack:** Go 1.22、net/http、govmomi（vCenter）、Mermaid（文档图表）

---

## 依据文档（Specs）

- `docs/superpowers/specs/2026-04-16-01-monorepo-workspace-design.md`
- `docs/superpowers/specs/2026-04-16-02-mcpkit-http-sse-design.md`
- `docs/superpowers/specs/2026-04-16-03-authn-authz-design.md`
- `docs/superpowers/specs/2026-04-16-04-vcenter-service-design.md`
- `docs/superpowers/specs/2026-04-16-05-mcpmux-compose-integration-design.md`
- `docs/superpowers/specs/2026-04-16-06-ces-service-skeleton-design.md`

## 目标目录结构（实现后应满足）

```
.
├── go.work
├── modules/
│   └── mcpkit/
│       ├── go.mod
│       ├── httpkit/
│       ├── mcp/
│       └── auth/
├── services/
│   ├── vcenter/
│   │   ├── go.mod
│   │   ├── cmd/vcenter-mcp-server/main.go
│   │   └── internal/...
│   └── ces/
│       ├── go.mod
│       └── cmd/ces-mcp-server/main.go
└── deploy/
    └── mcpmux/
        ├── docker-compose.yaml
        ├── .env.example
        └── README.md
```

---

### Task 1: 初始化 Monorepo 工作区（go.work + 多 module）

**Files:**
- Create: `go.work`
- Create: `modules/mcpkit/go.mod`
- Create: `services/vcenter/go.mod`
- Create: `services/ces/go.mod`

- [ ] **Step 1: 创建 go.work**

```txt
go 1.22

use (
	./modules/mcpkit
	./services/vcenter
	./services/ces
)
```

- [ ] **Step 2: 创建 modules/mcpkit/go.mod**

```go
module github.com/<org>/<repo>/modules/mcpkit

go 1.22
```

- [ ] **Step 3: 创建 services/vcenter/go.mod**

```go
module github.com/<org>/<repo>/services/vcenter

go 1.22

require github.com/<org>/<repo>/modules/mcpkit v0.0.0
```

- [ ] **Step 4: 创建 services/ces/go.mod**

```go
module github.com/<org>/<repo>/services/ces

go 1.22

require github.com/<org>/<repo>/modules/mcpkit v0.0.0
```

- [ ] **Step 5: 验证工作区可解析**

Run: `go env GOWORK && go list ./...`
Expected: `GOWORK=/workspace/go.work` 且 `go list` 不报错（可能暂时无包或只有空包）

---

### Task 2: 在 mcpkit 中实现 HTTP 基础（server、base path、health、优雅退出）

**Files:**
- Create: `modules/mcpkit/httpkit/server.go`
- Create: `modules/mcpkit/httpkit/health.go`
- Create: `modules/mcpkit/httpkit/config.go`
- Test: `modules/mcpkit/httpkit/server_test.go`

- [ ] **Step 1: 写一个失败的测试，验证 base path 与健康检查路由**

```go
package httpkit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/<org>/<repo>/modules/mcpkit/httpkit"
)

func TestServer_BasePathAndHealth(t *testing.T) {
	mux := httpkit.NewMux(httpkit.MuxConfig{BasePath: "/vcenter"})
	s := httptest.NewServer(mux)
	defer s.Close()

	resp, err := http.Get(s.URL + "/vcenter/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}

	resp, err = http.Get(s.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 when base path set, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./modules/mcpkit/httpkit -run TestServer_BasePathAndHealth -v`
Expected: FAIL（缺少 `httpkit.NewMux` 等）

- [ ] **Step 3: 实现 httpkit 配置与路由装配**

`modules/mcpkit/httpkit/config.go`

```go
package httpkit

type MuxConfig struct {
	BasePath string
}
```

`modules/mcpkit/httpkit/health.go`

```go
package httpkit

import "net/http"

func RegisterHealth(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
```

`modules/mcpkit/httpkit/server.go`

```go
package httpkit

import (
	"net/http"
	"strings"
)

func NewMux(cfg MuxConfig) http.Handler {
	root := http.NewServeMux()
	RegisterHealth(root)

	if cfg.BasePath == "" || cfg.BasePath == "/" {
		return root
	}

	base := cfg.BasePath
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if strings.HasSuffix(base, "/") {
		base = strings.TrimSuffix(base, "/")
	}

	return http.StripPrefix(base, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/") {
			r.URL.Path = "/" + r.URL.Path
		}
		root.ServeHTTP(w, r)
	}))
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./modules/mcpkit/httpkit -run TestServer_BasePathAndHealth -v`
Expected: PASS

---

### Task 3: 在 mcpkit 中实现 Bearer 鉴权中间件（AuthN）

**Files:**
- Create: `modules/mcpkit/auth/bearer.go`
- Test: `modules/mcpkit/auth/bearer_test.go`

- [ ] **Step 1: 写一个失败的测试，覆盖 401/403/200**

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

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./modules/mcpkit/auth -run TestBearerAuth -v`
Expected: FAIL

- [ ] **Step 3: 实现 Bearer 中间件**

`modules/mcpkit/auth/bearer.go`

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

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./modules/mcpkit/auth -run TestBearerAuth -v`
Expected: PASS

---

### Task 4: 在 mcpkit 中实现 MCP 协议最小骨架（tools/list、tools/call）

**Files:**
- Create: `modules/mcpkit/mcp/types.go`
- Create: `modules/mcpkit/mcp/registry.go`
- Create: `modules/mcpkit/mcp/handler.go`
- Test: `modules/mcpkit/mcp/registry_test.go`

实现目标（最小可用）：

- tool registry：注册 tool name / input schema / handler
- `tools/list` 返回 tools 元信息
- `tools/call` 解析参数并调用 handler，返回结构化结果或错误

- [ ] **Step 1: 写一个失败的测试，验证 registry 可注册与调用**

```go
package mcp_test

import (
	"context"
	"testing"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
)

func TestRegistry_Call(t *testing.T) {
	r := mcp.NewRegistry()
	r.Register(mcp.Tool{
		Name:        "demo.echo",
		Description: "echo",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return input, nil
		},
	})

	out, err := r.Call(context.Background(), "demo.echo", map[string]any{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := out.(map[string]any)
	if !ok || m["k"] != "v" {
		t.Fatalf("unexpected output: %#v", out)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./modules/mcpkit/mcp -run TestRegistry_Call -v`
Expected: FAIL

- [ ] **Step 3: 实现 types 与 registry**

`modules/mcpkit/mcp/types.go`

```go
package mcp

import "context"

type ToolHandler func(ctx context.Context, input map[string]any) (any, error)

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     ToolHandler
}
```

`modules/mcpkit/mcp/registry.go`

```go
package mcp

import (
	"context"
	"errors"
	"sync"
)

var ErrToolNotFound = errors.New("tool not found")

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name] = t
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, Tool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
	}
	return out
}

func (r *Registry) Call(ctx context.Context, name string, input map[string]any) (any, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return nil, ErrToolNotFound
	}
	return t.Handler(ctx, input)
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./modules/mcpkit/mcp -run TestRegistry_Call -v`
Expected: PASS

---

### Task 5: 在 mcpkit 中实现 HTTP/SSE Transport（/sse + /messages）

**Files:**
- Create: `modules/mcpkit/mcphttp/config.go`
- Create: `modules/mcpkit/mcphttp/sse.go`
- Create: `modules/mcpkit/mcphttp/session.go`
- Create: `modules/mcpkit/mcphttp/handler.go`
- Test: `modules/mcpkit/mcphttp/transport_test.go`

约束：

- `/sse` 建立 SSE；`/messages` 接收客户端消息
- `/sse` 与 `/messages` 强制 Bearer 鉴权（token 来自服务侧 env，挂载在服务 main 中）
- 单 session 串行处理消息（session 内消息队列）

- [ ] **Step 1: 写失败测试，验证需要鉴权且 base path 生效**

```go
package mcphttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/<org>/<repo>/modules/mcpkit/auth"
	"github.com/<org>/<repo>/modules/mcpkit/httpkit"
	"github.com/<org>/<repo>/modules/mcpkit/mcp"
	"github.com/<org>/<repo>/modules/mcpkit/mcphttp"
)

func TestTransport_AuthAndBasePath(t *testing.T) {
	reg := mcp.NewRegistry()
	h := mcphttp.NewHandler(mcphttp.Config{Registry: reg})

	mux := http.NewServeMux()
	mux.Handle("/sse", auth.Bearer("t")(h))
	mux.Handle("/messages", auth.Bearer("t")(h))
	httpkit.RegisterHealth(mux)

	s := httptest.NewServer(httpkit.NewMux(httpkit.MuxConfig{BasePath: "/vcenter"}))
	defer s.Close()

	req, _ := http.NewRequest(http.MethodGet, s.URL+"/vcenter/sse", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: 实现 mcphttp 的最小 handler（先只把路由与鉴权打通）**

`modules/mcpkit/mcphttp/config.go`

```go
package mcphttp

import "github.com/<org>/<repo>/modules/mcpkit/mcp"

type Config struct {
	Registry *mcp.Registry
}
```

`modules/mcpkit/mcphttp/handler.go`

```go
package mcphttp

import "net/http"

type Handler struct {
	cfg Config
}

func NewHandler(cfg Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/sse":
		serveSSE(h.cfg, w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/messages":
		serveMessages(h.cfg, w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
```

- [ ] **Step 3: 完成 session 管理、SSE 写入与 messages 处理**

`modules/mcpkit/mcphttp/session.go`

```go
package mcphttp

import (
	"context"
	"sync"
	"time"
)

type session struct {
	id      string
	created time.Time
	send    chan []byte
	mu      sync.Mutex
}

type sessionStore struct {
	mu      sync.RWMutex
	ttl     time.Duration
	sessions map[string]*session
}

func newSessionStore(ttl time.Duration) *sessionStore {
	return &sessionStore{ttl: ttl, sessions: make(map[string]*session)}
}

func (s *sessionStore) get(id string) (*session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.sessions[id]
	return v, ok
}

func (s *sessionStore) put(sess *session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.id] = sess
}

func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *sessionStore) reap(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()
			s.mu.Lock()
			for id, sess := range s.sessions {
				if now.Sub(sess.created) > s.ttl {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}
```

`modules/mcpkit/mcphttp/sse.go`

```go
package mcphttp

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

var store = newSessionStore(5 * time.Minute)

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func serveSSE(cfg Config, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := newSessionID()
	sess := &session{id: id, created: time.Now(), send: make(chan []byte, 128)}
	store.put(sess)

	_, _ = w.Write([]byte("event: init\n"))
	_, _ = w.Write([]byte("data: {\"session_id\":\"" + id + "\"}\n\n"))
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			store.delete(id)
			return
		case msg := <-sess.send:
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(msg)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}
```

`modules/mcpkit/mcphttp/handler.go` 追加 `serveMessages` 依赖：

```go
package mcphttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
)

type messageEnvelope struct {
	SessionID string         `json:"session_id"`
	Type      string         `json:"type"`
	ID        string         `json:"id"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params"`
}

func serveMessages(cfg Config, w http.ResponseWriter, r *http.Request) {
	var env messageEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sess, ok := store.get(env.SessionID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch env.Method {
	case "tools/list":
		out := map[string]any{"tools": cfg.Registry.List()}
		b, _ := json.Marshal(out)
		sess.send <- b
		w.WriteHeader(http.StatusAccepted)
	case "tools/call":
		name, _ := env.Params["name"].(string)
		args, _ := env.Params["arguments"].(map[string]any)
		res, err := cfg.Registry.Call(ctx, name, args)
		if err == mcp.ErrToolNotFound {
			b, _ := json.Marshal(map[string]any{"error": "tool_not_found"})
			sess.send <- b
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if err != nil {
			b, _ := json.Marshal(map[string]any{"error": "tool_error", "message": err.Error()})
			sess.send <- b
			w.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(map[string]any{"result": res})
		sess.send <- b
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
```

- [ ] **Step 4: 运行 mcpkit 测试集**

Run: `go test ./modules/mcpkit/... -v`
Expected: PASS

---

### Task 6: vCenter 服务脚手架（main、env 配置、挂载 mcpkit）

**Files:**
- Create: `services/vcenter/cmd/vcenter-mcp-server/main.go`
- Create: `services/vcenter/internal/config/config.go`
- Create: `services/vcenter/internal/tools/registry.go`
- Create: `services/vcenter/internal/vcenter/client.go`

- [ ] **Step 1: 写配置加载（env）与校验**

`services/vcenter/internal/config/config.go`

```go
package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	ListenAddr          string
	BasePath            string
	ServiceToken        string
	VCenterURL          string
	VCenterUsername     string
	VCenterPassword     string
	VCenterInsecure     bool
	VCenterCAFile       string
	EnableDangerousOps  bool
}

func Load() (Config, error) {
	c := Config{
		ListenAddr: os.Getenv("HTTP_LISTEN_ADDR"),
		BasePath:   os.Getenv("MCP_HTTP_BASE_PATH"),
		ServiceToken: os.Getenv("MCP_SERVICE_TOKEN"),
		VCenterURL: os.Getenv("VCENTER_URL"),
		VCenterUsername: os.Getenv("VCENTER_USERNAME"),
		VCenterPassword: os.Getenv("VCENTER_PASSWORD"),
		VCenterCAFile: os.Getenv("VCENTER_CA_FILE"),
	}
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:8080"
	}
	c.VCenterInsecure = strings.EqualFold(os.Getenv("VCENTER_INSECURE"), "true")
	c.EnableDangerousOps = strings.EqualFold(os.Getenv("VCENTER_ENABLE_DANGEROUS_OPS"), "true")

	if c.ServiceToken == "" {
		return Config{}, errors.New("MCP_SERVICE_TOKEN is required")
	}
	if c.VCenterURL == "" || c.VCenterUsername == "" || c.VCenterPassword == "" {
		return Config{}, errors.New("VCENTER_URL/VCENTER_USERNAME/VCENTER_PASSWORD are required")
	}
	return c, nil
}
```

- [ ] **Step 2: 实现 vCenter client 连接封装（先只占位，后续任务填充 govmomi 细节）**

`services/vcenter/internal/vcenter/client.go`

```go
package vcenter

type Client struct{}
```

- [ ] **Step 3: 实现 tool registry 装配（先注册最小 demo tool，后续替换为真实工具）**

`services/vcenter/internal/tools/registry.go`

```go
package tools

import (
	"context"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
)

func BuildRegistry() *mcp.Registry {
	r := mcp.NewRegistry()
	r.Register(mcp.Tool{
		Name:        "vcenter.ping",
		Description: "ping",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	})
	return r
}
```

- [ ] **Step 4: 实现 main，挂载鉴权、health、mcp transport**

`services/vcenter/cmd/vcenter-mcp-server/main.go`

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
	h := mcphttp.NewHandler(mcphttp.Config{Registry: reg})

	mux := http.NewServeMux()
	mux.Handle("/sse", auth.Bearer(cfg.ServiceToken)(h))
	mux.Handle("/messages", auth.Bearer(cfg.ServiceToken)(h))
	httpkit.RegisterHealth(mux)

	root := httpkit.NewMux(httpkit.MuxConfig{BasePath: cfg.BasePath})
	server := &http.Server{Addr: cfg.ListenAddr, Handler: root}
	log.Fatal(server.ListenAndServe())
}
```

- [ ] **Step 5: 编译与冒烟**

Run: `go test ./...`
Expected: PASS

Run: `go build ./services/vcenter/cmd/vcenter-mcp-server`
Expected: build success

---

### Task 7: 引入 govmomi 并实现 vCenter 资产清单工具（vcenter.list_inventory、vcenter.get_vm）

**Files:**
- Modify: `services/vcenter/go.mod`
- Modify: `services/vcenter/internal/vcenter/client.go`
- Create: `services/vcenter/internal/vcenter/inventory.go`
- Modify: `services/vcenter/internal/tools/registry.go`
- Test: `services/vcenter/internal/tools/inventory_test.go`

- [ ] **Step 1: 在 vcenter module 引入 govmomi 依赖**

Run: `go get github.com/vmware/govmomi@latest`
Expected: `services/vcenter/go.mod` 增加 govmomi 依赖

- [ ] **Step 2: 定义 vCenter client 接口，便于 fake 测试**

`services/vcenter/internal/vcenter/client.go`

```go
package vcenter

import "context"

type InventoryItem struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	MoRef string `json:"moRef"`
	Path  string `json:"path"`
}

type VMInfo struct {
	Name       string `json:"name"`
	MoRef      string `json:"moRef"`
	PowerState string `json:"powerState"`
	CPU        int32  `json:"cpu"`
	MemoryMB   int64  `json:"memoryMB"`
}

type API interface {
	ListInventory(ctx context.Context, types []string, nameContains string, limit int, cursor string) ([]InventoryItem, string, error)
	GetVM(ctx context.Context, moRef string, path string) (VMInfo, error)
}
```

- [ ] **Step 3: 先用 fake 实现写通过工具层测试（TDD）**

`services/vcenter/internal/tools/inventory_test.go`

```go
package tools_test

import (
	"context"
	"testing"

	"github.com/<org>/<repo>/services/vcenter/internal/tools"
	"github.com/<org>/<repo>/services/vcenter/internal/vcenter"
)

type fakeAPI struct{}

func (f fakeAPI) ListInventory(ctx context.Context, types []string, nameContains string, limit int, cursor string) ([]vcenter.InventoryItem, string, error) {
	return []vcenter.InventoryItem{{Type: "vm", Name: "vm-1", MoRef: "vm-1", Path: "/dc/vm/vm-1"}}, "", nil
}

func (f fakeAPI) GetVM(ctx context.Context, moRef string, path string) (vcenter.VMInfo, error) {
	return vcenter.VMInfo{Name: "vm-1", MoRef: "vm-1"}, nil
}

func TestTools_Inventory(t *testing.T) {
	reg := tools.BuildRegistryWithAPI(fakeAPI{})
	_, err := reg.Call(context.Background(), "vcenter.list_inventory", map[string]any{"types": []any{"vm"}})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 4: 实现 tool registry 注入 API，并补齐两个工具注册**

`services/vcenter/internal/tools/registry.go`

```go
package tools

import (
	"context"
	"errors"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
	"github.com/<org>/<repo>/services/vcenter/internal/vcenter"
)

func BuildRegistry() *mcp.Registry {
	return BuildRegistryWithAPI(nil)
}

func BuildRegistryWithAPI(api vcenter.API) *mcp.Registry {
	r := mcp.NewRegistry()

	r.Register(mcp.Tool{
		Name:        "vcenter.list_inventory",
		Description: "list inventory",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			if api == nil {
				return nil, errors.New("vcenter api not configured")
			}
			rawTypes, _ := input["types"].([]any)
			types := make([]string, 0, len(rawTypes))
			for _, v := range rawTypes {
				s, _ := v.(string)
				if s != "" {
					types = append(types, s)
				}
			}
			nameContains, _ := input["nameContains"].(string)
			limit := 0
			if v, ok := input["limit"].(float64); ok {
				limit = int(v)
			}
			cursor, _ := input["cursor"].(string)

			items, next, err := api.ListInventory(ctx, types, nameContains, limit, cursor)
			if err != nil {
				return nil, err
			}
			return map[string]any{"items": items, "nextCursor": next}, nil
		},
	})

	r.Register(mcp.Tool{
		Name:        "vcenter.get_vm",
		Description: "get vm",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			if api == nil {
				return nil, errors.New("vcenter api not configured")
			}
			moRef, _ := input["moRef"].(string)
			path, _ := input["path"].(string)
			return api.GetVM(ctx, moRef, path)
		},
	})

	return r
}
```

- [ ] **Step 5: 在 main 中创建真实 govmomi API 并注入 registry**

将 `tools.BuildRegistry()` 替换为 `tools.BuildRegistryWithAPI(api)`，其中 `api` 在后续 Step 6 实现。

- [ ] **Step 6: 实现 govmomi inventory 查询（最小可用）**

`services/vcenter/internal/vcenter/inventory.go`：用 `view.Manager` + `property.Collector` 批量拉取对象属性；输出映射为 `InventoryItem`。

验证命令：

Run: `go test ./services/vcenter/... -v`
Expected: PASS

---

### Task 8: 实现 vCenter 性能与事件/告警工具（vcenter.query_perf、vcenter.list_events、vcenter.list_alarms）

**Files:**
- Create: `services/vcenter/internal/vcenter/perf.go`
- Create: `services/vcenter/internal/vcenter/events.go`
- Create: `services/vcenter/internal/vcenter/alarms.go`
- Modify: `services/vcenter/internal/tools/registry.go`
- Test: `services/vcenter/internal/tools/perf_test.go`
- Test: `services/vcenter/internal/tools/events_test.go`

- [ ] **Step 1: 先用 fake API 写三类工具的 handler 测试（TDD）**

目标：确保输入解析（时间、limit、对象类型）与输出结构稳定。

- [ ] **Step 2: 实现 govmomi performance.Manager 查询与 counter 缓存**

- [ ] **Step 3: 实现 event.Manager 查询，限制默认返回数量与分页 cursor 策略**

- [ ] **Step 4: 实现告警定义与触发态读取（先只读）**

- [ ] **Step 5: 跑服务模块测试**

Run: `go test ./services/vcenter/... -v`
Expected: PASS

---

### Task 9: 实现高危运维工具并默认关闭（vcenter.power、vcenter.snapshot）

**Files:**
- Modify: `services/vcenter/internal/tools/registry.go`
- Create: `services/vcenter/internal/tools/dangerous.go`
- Test: `services/vcenter/internal/tools/dangerous_test.go`

- [ ] **Step 1: 写失败测试，确保默认不注册或调用受控失败**

```go
package tools_test

import (
	"context"
	"testing"

	"github.com/<org>/<repo>/services/vcenter/internal/tools"
)

func TestDangerousOps_DefaultDisabled(t *testing.T) {
	reg := tools.BuildRegistryWithOptions(tools.Options{EnableDangerousOps: false, API: nil})
	_, err := reg.Call(context.Background(), "vcenter.power", map[string]any{"vmMoRef": "vm-1", "action": "on"})
	if err == nil {
		t.Fatalf("expected error when dangerous ops disabled")
	}
}
```

- [ ] **Step 2: 实现 Options，基于开关注册危险 tools**

`services/vcenter/internal/tools/dangerous.go`

```go
package tools

import "github.com/<org>/<repo>/services/vcenter/internal/vcenter"

type Options struct {
	EnableDangerousOps bool
	API                vcenter.API
}
```

将 `BuildRegistryWithAPI` 演进为 `BuildRegistryWithOptions(opts Options)`，在 `EnableDangerousOps=true` 时才注册 `vcenter.power` / `vcenter.snapshot`。

- [ ] **Step 3: main 从 env 读取开关并传入 Options**

`cfg.EnableDangerousOps` 传入 `tools.BuildRegistryWithOptions(...)`。

- [ ] **Step 4: 跑服务模块测试**

Run: `go test ./services/vcenter/... -v`
Expected: PASS

---

### Task 10: CES 模块占位服务（可运行但不提供实际工具）

**Files:**
- Create: `services/ces/cmd/ces-mcp-server/main.go`
- Create: `services/ces/internal/config/config.go`
- Create: `services/ces/internal/tools/registry.go`

- [ ] **Step 1: 复用 vcenter 的做法实现 env 配置（仅校验 MCP_SERVICE_TOKEN 与监听）**
- [ ] **Step 2: 注册 `ces.ping`，其余工具保持不实现**
- [ ] **Step 3: 编译**

Run: `go build ./services/ces/cmd/ces-mcp-server`
Expected: build success

---

### Task 11: 生成 mcpmux docker-compose 接入示例（deploy/mcpmux）

**Files:**
- Create: `deploy/mcpmux/docker-compose.yaml`
- Create: `deploy/mcpmux/.env.example`
- Create: `deploy/mcpmux/README.md`

- [ ] **Step 1: 写 .env.example**

```dotenv
MCP_SERVICE_TOKEN=change-me
VCENTER_URL=https://vcenter.example/sdk
VCENTER_USERNAME=admin
VCENTER_PASSWORD=change-me
VCENTER_INSECURE=false
VCENTER_ENABLE_DANGEROUS_OPS=false
```

- [ ] **Step 2: 写 docker-compose.yaml（mcpmux 镜像与配置用占位，但透传 header 必须明确）**

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

- [ ] **Step 3: 写 README.md（明确 mcpmux 后端配置要点）**

README 必须包含：

- 后端名 `vcenter`
- 后端 URL `http://vcenter:8080`
- transport HTTP/SSE
- headers：`Authorization: Bearer ${MCP_SERVICE_TOKEN}`
- 验证步骤（healthz/readyz、tools/list、tools/call）

---

## 计划自检（对照 Specs 覆盖）

- Workspace/多模块：Task 1
- mcpkit HTTP/SSE：Task 2 + Task 4 + Task 5
- Bearer 鉴权：Task 3 + Task 5 + vcenter main
- vCenter MVP：Task 6-9
- CES 占位：Task 10
- mcpmux compose：Task 11

---

## 执行方式

计划已完成并保存到 `docs/superpowers/plans/2026-04-16-mcp-monorepo-implementation.md`。两个执行选项：

1. **Subagent-Driven（recommended）**：每个 Task 分发独立子代理实现，逐步 review
2. **Inline Execution**：在当前会话按 Task 顺序执行并在关键节点停下来复核

你选哪一种？
