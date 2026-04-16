# MCPKit HTTP/SSE Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `modules/mcpkit` 实现 MCP HTTP/SSE 基础设施（health、base path、tool registry、/sse 与 /messages transport），供各服务复用。

**Architecture:** `httpkit` 提供基础 HTTP 组装与 base path；`mcp` 提供 tools 注册与调用；`mcphttp` 提供 SSE 会话与消息收发，路由层可由服务自行决定是否加鉴权（鉴权在 Spec 03 实现）。

**Tech Stack:** Go 1.22、net/http、testing、httptest

---

## Spec

- `docs/superpowers/specs/2026-04-16-02-mcpkit-http-sse-design.md`

## Dependencies

- 需要 workspace/module 已就绪：`docs/superpowers/plans/2026-04-16-01-monorepo-workspace-implementation.md`

---

### Task 1: 实现 httpkit（base path + health）

**Files:**
- Create: `modules/mcpkit/httpkit/config.go`
- Create: `modules/mcpkit/httpkit/health.go`
- Create: `modules/mcpkit/httpkit/server.go`
- Test: `modules/mcpkit/httpkit/server_test.go`

- [ ] **Step 1: 写失败测试：base path 生效且 health 端点可用**

Create `modules/mcpkit/httpkit/server_test.go`:

```go
package httpkit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/<org>/<repo>/modules/mcpkit/httpkit"
)

func TestServer_BasePathAndHealth(t *testing.T) {
	h := httpkit.NewMux(httpkit.MuxConfig{BasePath: "/vcenter"})
	s := httptest.NewServer(h)
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

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./modules/mcpkit/httpkit -run TestServer_BasePathAndHealth -v`
Expected: FAIL（缺少实现）

- [ ] **Step 3: 实现 MuxConfig**

Create `modules/mcpkit/httpkit/config.go`:

```go
package httpkit

type MuxConfig struct {
	BasePath string
}
```

- [ ] **Step 4: 实现 health handlers**

Create `modules/mcpkit/httpkit/health.go`:

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

- [ ] **Step 5: 实现 base path mux**

Create `modules/mcpkit/httpkit/server.go`:

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
	base = strings.TrimSuffix(base, "/")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, base) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, base)
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}
		root.ServeHTTP(w, r2)
	})
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./modules/mcpkit/httpkit -run TestServer_BasePathAndHealth -v`
Expected: PASS

---

### Task 2: 实现 MCP tool registry（mcp 包）

**Files:**
- Create: `modules/mcpkit/mcp/types.go`
- Create: `modules/mcpkit/mcp/registry.go`
- Test: `modules/mcpkit/mcp/registry_test.go`

- [ ] **Step 1: 写失败测试：注册与调用**

Create `modules/mcpkit/mcp/registry_test.go`:

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

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./modules/mcpkit/mcp -run TestRegistry_Call -v`
Expected: FAIL

- [ ] **Step 3: 实现 types**

Create `modules/mcpkit/mcp/types.go`:

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

- [ ] **Step 4: 实现 registry**

Create `modules/mcpkit/mcp/registry.go`:

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

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./modules/mcpkit/mcp -run TestRegistry_Call -v`
Expected: PASS

---

### Task 3: 实现 MCP HTTP/SSE transport（mcphttp 包）

**Files:**
- Create: `modules/mcpkit/mcphttp/config.go`
- Create: `modules/mcpkit/mcphttp/session.go`
- Create: `modules/mcpkit/mcphttp/sse.go`
- Create: `modules/mcpkit/mcphttp/handler.go`
- Test: `modules/mcpkit/mcphttp/transport_test.go`

- [ ] **Step 1: 写失败测试：路由存在且未鉴权时仍可返回 404/202（鉴权在 Spec03 处理）**

Create `modules/mcpkit/mcphttp/transport_test.go`:

```go
package mcphttp_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
	"github.com/<org>/<repo>/modules/mcpkit/mcphttp"
)

func TestTransport_RoutesExist(t *testing.T) {
	reg := mcp.NewRegistry()
	h := mcphttp.NewHandler(mcphttp.Config{Registry: reg})
	s := httptest.NewServer(h)
	defer s.Close()

	resp, err := http.Get(s.URL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}

	resp, err = http.Post(s.URL+"/messages", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 and non-404 got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./modules/mcpkit/mcphttp -run TestTransport_RoutesExist -v`
Expected: FAIL

- [ ] **Step 3: 实现配置与 HTTP handler**

Create `modules/mcpkit/mcphttp/config.go`:

```go
package mcphttp

import "github.com/<org>/<repo>/modules/mcpkit/mcp"

type Config struct {
	Registry *mcp.Registry
}
```

Create `modules/mcpkit/mcphttp/handler.go`:

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

- [ ] **Step 4: 实现 session store（TTL + 回收）**

Create `modules/mcpkit/mcphttp/session.go`:

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
	mu       sync.RWMutex
	ttl      time.Duration
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

- [ ] **Step 5: 实现 SSE 连接与 init 事件**

Create `modules/mcpkit/mcphttp/sse.go`:

```go
package mcphttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

var store = newSessionStore(5 * time.Minute)
var reapOnce = make(chan struct{}, 1)

func ensureReaper(ctx context.Context) {
	select {
	case reapOnce <- struct{}{}:
		go store.reap(ctx)
	default:
	}
}

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

	ensureReaper(r.Context())

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

- [ ] **Step 6: 实现 messages 处理（tools/list、tools/call）**

Update `modules/mcpkit/mcphttp/handler.go` by appending:

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

- [ ] **Step 7: 运行 mcpkit 全量测试**

Run: `go test ./modules/mcpkit/... -v`
Expected: PASS

---

## Notes

- 鉴权强制（/sse 与 /messages）在 Spec 03 的实现计划中完成，通过在服务侧对路由挂载 Bearer middleware 达成。

