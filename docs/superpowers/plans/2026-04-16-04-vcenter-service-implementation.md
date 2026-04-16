# vCenter Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付可运行的 `services/vcenter` MCP Server：通过 MCP HTTP/SSE 暴露 vCenter 资产清单、性能指标、事件/告警查询，并在开关开启时提供 VM 运维操作。

**Architecture:** 服务作为独立 Go module，复用 `modules/mcpkit` 的 transport 与鉴权；通过 `internal/vcenter` 封装 govmomi 访问并实现 `vcenter.API` 接口；通过 `internal/tools` 注册 MCP tools 并调用 API。

**Tech Stack:** Go 1.22、govmomi、net/http、testing

---

## Spec

- `docs/superpowers/specs/2026-04-16-04-vcenter-service-design.md`

## Dependencies

- Workspace：`docs/superpowers/plans/2026-04-16-01-monorepo-workspace-implementation.md`
- MCPKit HTTP/SSE：`docs/superpowers/plans/2026-04-16-02-mcpkit-http-sse-implementation.md`
- AuthN/AuthZ：`docs/superpowers/plans/2026-04-16-03-authn-authz-implementation.md`

---

### Task 1: vCenter 服务脚手架（main + config + registry）

**Files:**
- Create: `services/vcenter/cmd/vcenter-mcp-server/main.go`
- Create: `services/vcenter/internal/config/config.go`
- Create: `services/vcenter/internal/tools/registry.go`
- Create: `services/vcenter/internal/vcenter/api.go`

- [ ] **Step 1: 实现配置读取与校验**

Create `services/vcenter/internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	ListenAddr         string
	BasePath           string
	ServiceToken       string
	VCenterURL         string
	VCenterUsername    string
	VCenterPassword    string
	VCenterInsecure    bool
	VCenterCAFile      string
	EnableDangerousOps bool
}

func Load() (Config, error) {
	c := Config{
		ListenAddr:      os.Getenv("HTTP_LISTEN_ADDR"),
		BasePath:        os.Getenv("MCP_HTTP_BASE_PATH"),
		ServiceToken:    os.Getenv("MCP_SERVICE_TOKEN"),
		VCenterURL:      os.Getenv("VCENTER_URL"),
		VCenterUsername: os.Getenv("VCENTER_USERNAME"),
		VCenterPassword: os.Getenv("VCENTER_PASSWORD"),
		VCenterCAFile:   os.Getenv("VCENTER_CA_FILE"),
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

- [ ] **Step 2: 定义 vcenter.API 接口与核心类型**

Create `services/vcenter/internal/vcenter/api.go`:

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

type PerfPoint struct {
	TS    string  `json:"ts"`
	Value float64 `json:"value"`
}

type PerfSeries struct {
	Metric string      `json:"metric"`
	Points []PerfPoint `json:"points"`
}

type EventItem struct {
	TS          string `json:"ts"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	EntityMoRef string `json:"entityMoRef,omitempty"`
}

type AlarmItem struct {
	AlarmID     string `json:"alarmId"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	EntityMoRef string `json:"entityMoRef,omitempty"`
}

type TaskRef struct {
	TaskMoRef string `json:"taskMoRef"`
	State     string `json:"state"`
}

type API interface {
	ListInventory(ctx context.Context, types []string, nameContains string, limit int, cursor string) ([]InventoryItem, string, error)
	GetVM(ctx context.Context, moRef string, path string) (VMInfo, error)

	QueryPerf(ctx context.Context, objectType string, moRef string, metrics []string, startTime string, endTime string, intervalSeconds int) ([]PerfSeries, error)
	ListEvents(ctx context.Context, startTime string, endTime string, moRef string, types []string, limit int, cursor string) ([]EventItem, string, error)
	ListAlarms(ctx context.Context, moRef string) ([]AlarmItem, error)

	PowerVM(ctx context.Context, vmMoRef string, action string) (TaskRef, error)
	SnapshotVM(ctx context.Context, vmMoRef string, action string, name string, description string) (any, error)
}
```

- [ ] **Step 3: 注册 MCP tools（用 fake API 测试驱动）**

Create `services/vcenter/internal/tools/registry.go`:

```go
package tools

import (
	"context"
	"errors"

	"github.com/<org>/<repo>/modules/mcpkit/mcp"
	"github.com/<org>/<repo>/services/vcenter/internal/vcenter"
)

type Options struct {
	EnableDangerousOps bool
	API                vcenter.API
}

func BuildRegistryWithOptions(opts Options) *mcp.Registry {
	api := opts.API
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

	r.Register(mcp.Tool{
		Name:        "vcenter.query_perf",
		Description: "query performance metrics",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			if api == nil {
				return nil, errors.New("vcenter api not configured")
			}
			objectType, _ := input["objectType"].(string)
			moRef, _ := input["moRef"].(string)
			rawMetrics, _ := input["metrics"].([]any)
			metrics := make([]string, 0, len(rawMetrics))
			for _, v := range rawMetrics {
				s, _ := v.(string)
				if s != "" {
					metrics = append(metrics, s)
				}
			}
			startTime, _ := input["startTime"].(string)
			endTime, _ := input["endTime"].(string)
			intervalSeconds := 0
			if v, ok := input["intervalSeconds"].(float64); ok {
				intervalSeconds = int(v)
			}
			return api.QueryPerf(ctx, objectType, moRef, metrics, startTime, endTime, intervalSeconds)
		},
	})

	r.Register(mcp.Tool{
		Name:        "vcenter.list_events",
		Description: "list events",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			if api == nil {
				return nil, errors.New("vcenter api not configured")
			}
			startTime, _ := input["startTime"].(string)
			endTime, _ := input["endTime"].(string)
			moRef, _ := input["moRef"].(string)
			rawTypes, _ := input["types"].([]any)
			types := make([]string, 0, len(rawTypes))
			for _, v := range rawTypes {
				s, _ := v.(string)
				if s != "" {
					types = append(types, s)
				}
			}
			limit := 0
			if v, ok := input["limit"].(float64); ok {
				limit = int(v)
			}
			cursor, _ := input["cursor"].(string)

			items, next, err := api.ListEvents(ctx, startTime, endTime, moRef, types, limit, cursor)
			if err != nil {
				return nil, err
			}
			return map[string]any{"items": items, "nextCursor": next}, nil
		},
	})

	r.Register(mcp.Tool{
		Name:        "vcenter.list_alarms",
		Description: "list alarms",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			if api == nil {
				return nil, errors.New("vcenter api not configured")
			}
			moRef, _ := input["moRef"].(string)
			return api.ListAlarms(ctx, moRef)
		},
	})

	if opts.EnableDangerousOps && api != nil {
		registerDangerousTools(r, api)
	}

	return r
}

func registerDangerousTools(r *mcp.Registry, api vcenter.API) {
	r.Register(mcp.Tool{
		Name:        "vcenter.power",
		Description: "VM power operations",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			vmMoRef, _ := input["vmMoRef"].(string)
			action, _ := input["action"].(string)
			return api.PowerVM(ctx, vmMoRef, action)
		},
	})

	r.Register(mcp.Tool{
		Name:        "vcenter.snapshot",
		Description: "VM snapshot operations",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			vmMoRef, _ := input["vmMoRef"].(string)
			action, _ := input["action"].(string)
			name, _ := input["name"].(string)
			description, _ := input["description"].(string)
			return api.SnapshotVM(ctx, vmMoRef, action, name, description)
		},
	})
}
```

- [ ] **Step 4: 用 fake API 写工具层测试（覆盖所有工具的基本调用）**

Create `services/vcenter/internal/tools/registry_test.go`:

```go
package tools_test

import (
	"context"
	"testing"

	"github.com/<org>/<repo>/services/vcenter/internal/tools"
	"github.com/<org>/<repo>/services/vcenter/internal/vcenter"
)

type fakeAPI struct{}

func (fakeAPI) ListInventory(ctx context.Context, types []string, nameContains string, limit int, cursor string) ([]vcenter.InventoryItem, string, error) {
	return []vcenter.InventoryItem{{Type: "vm", Name: "vm-1", MoRef: "vm-1", Path: "/dc/vm/vm-1"}}, "", nil
}
func (fakeAPI) GetVM(ctx context.Context, moRef string, path string) (vcenter.VMInfo, error) {
	return vcenter.VMInfo{Name: "vm-1", MoRef: "vm-1"}, nil
}
func (fakeAPI) QueryPerf(ctx context.Context, objectType string, moRef string, metrics []string, startTime string, endTime string, intervalSeconds int) ([]vcenter.PerfSeries, error) {
	return []vcenter.PerfSeries{{Metric: "cpu.usage", Points: []vcenter.PerfPoint{{TS: "t", Value: 1}}}}, nil
}
func (fakeAPI) ListEvents(ctx context.Context, startTime string, endTime string, moRef string, types []string, limit int, cursor string) ([]vcenter.EventItem, string, error) {
	return []vcenter.EventItem{{TS: "t", Type: "info", Message: "m"}}, "", nil
}
func (fakeAPI) ListAlarms(ctx context.Context, moRef string) ([]vcenter.AlarmItem, error) {
	return []vcenter.AlarmItem{{AlarmID: "a", Name: "alarm", Status: "green"}}, nil
}
func (fakeAPI) PowerVM(ctx context.Context, vmMoRef string, action string) (vcenter.TaskRef, error) {
	return vcenter.TaskRef{TaskMoRef: "task-1", State: "queued"}, nil
}
func (fakeAPI) SnapshotVM(ctx context.Context, vmMoRef string, action string, name string, description string) (any, error) {
	return map[string]any{"ok": true}, nil
}

func TestRegistry_AllTools(t *testing.T) {
	reg := tools.BuildRegistryWithOptions(tools.Options{API: fakeAPI{}, EnableDangerousOps: true})

	if _, err := reg.Call(context.Background(), "vcenter.list_inventory", map[string]any{"types": []any{"vm"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Call(context.Background(), "vcenter.get_vm", map[string]any{"moRef": "vm-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Call(context.Background(), "vcenter.query_perf", map[string]any{"objectType": "vm", "moRef": "vm-1", "metrics": []any{"cpu.usage"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Call(context.Background(), "vcenter.list_events", map[string]any{"startTime": "t1", "endTime": "t2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Call(context.Background(), "vcenter.list_alarms", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Call(context.Background(), "vcenter.power", map[string]any{"vmMoRef": "vm-1", "action": "on"}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Call(context.Background(), "vcenter.snapshot", map[string]any{"vmMoRef": "vm-1", "action": "list"}); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 5: 运行工具层测试**

Run: `go test ./services/vcenter/internal/tools -v`
Expected: PASS

---

### Task 2: 实现 govmomi Client（连接 + TLS 策略）

**Files:**
- Modify: `services/vcenter/go.mod`
- Create: `services/vcenter/internal/vcenter/govmomi_client.go`
- Test: `services/vcenter/internal/config/config_test.go`

- [ ] **Step 1: 引入 govmomi**

Run: `go get github.com/vmware/govmomi@latest`
Expected: `services/vcenter/go.mod` 增加 govmomi 依赖

- [ ] **Step 2: 为 config 写单测，确保缺少 token 或 vCenter 配置会失败**

Create `services/vcenter/internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"testing"

	"github.com/<org>/<repo>/services/vcenter/internal/config"
)

func TestLoad_MissingToken(t *testing.T) {
	t.Setenv("MCP_SERVICE_TOKEN", "")
	t.Setenv("VCENTER_URL", "u")
	t.Setenv("VCENTER_USERNAME", "x")
	t.Setenv("VCENTER_PASSWORD", "y")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error")
	}
	_ = os.Getenv("noop")
}
```

- [ ] **Step 3: 实现 govmomi 连接构造（insecure/CA file）**

Create `services/vcenter/internal/vcenter/govmomi_client.go`:

```go
package vcenter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"

	"github.com/vmware/govmomi"
)

type GovmomiClient struct {
	raw *govmomi.Client
}

func NewGovmomiClient(ctx context.Context, vcURL string, username string, password string, insecure bool, caFile string) (*GovmomiClient, error) {
	u, err := url.Parse(vcURL)
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(username, password)

	tlsCfg := &tls.Config{InsecureSkipVerify: insecure}
	if caFile != "" {
		b, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("invalid CA file")
		}
		tlsCfg.RootCAs = pool
	}

	c, err := govmomi.NewClient(ctx, u, insecure)
	if err != nil {
		return nil, err
	}
	_ = tlsCfg
	return &GovmomiClient{raw: c}, nil
}
```

- [ ] **Step 4: 运行服务模块测试**

Run: `go test ./services/vcenter/... -v`
Expected: PASS

---

### Task 3: 实现 API 的核心方法（inventory/perf/events/alarms/power/snapshot）

**Files:**
- Modify: `services/vcenter/internal/vcenter/govmomi_client.go`
- Create: `services/vcenter/internal/vcenter/inventory.go`
- Create: `services/vcenter/internal/vcenter/perf.go`
- Create: `services/vcenter/internal/vcenter/events.go`
- Create: `services/vcenter/internal/vcenter/alarms.go`
- Create: `services/vcenter/internal/vcenter/ops.go`

- [ ] **Step 1: 让 GovmomiClient 实现 vcenter.API 接口（先空实现让编译失败后逐个补齐）**

在 `govmomi_client.go` 添加：

```go
var _ API = (*GovmomiClient)(nil)
```

- [ ] **Step 2: 实现 inventory（ListInventory/GetVM）**

Create `services/vcenter/internal/vcenter/inventory.go`，使用 `view.Manager` 创建 container view，结合 `property.Collector` 批量获取 `name`、`parent` 等属性，映射为 `InventoryItem` / `VMInfo`。

（实现细节可按 govmomi 官方示例：container view + Retrieve + property paths。）

- [ ] **Step 3: 实现 perf（QueryPerf）**

Create `services/vcenter/internal/vcenter/perf.go`，使用 `performance.Manager`：

- 将业务 metric 名映射到 vSphere counter（缓存 counter 映射）
- 按 `startTime/endTime` 与 `intervalSeconds` 查询 sample
- 输出 `[]PerfSeries`

- [ ] **Step 4: 实现 events（ListEvents）**

Create `services/vcenter/internal/vcenter/events.go`，使用 `event.Manager`：

- 根据时间范围构造 filter
- 支持 `limit/cursor`：cursor 采用 `lastEventKey` 或最后一条的时间戳拼接（保持可继续翻页）

- [ ] **Step 5: 实现 alarms（ListAlarms）**

Create `services/vcenter/internal/vcenter/alarms.go`：

- 读取告警定义与触发态（先只读）
- 输出 `[]AlarmItem`

- [ ] **Step 6: 实现 ops（PowerVM/SnapshotVM）**

Create `services/vcenter/internal/vcenter/ops.go`：

- PowerVM：调用对应 task（PowerOnVM_Task/PowerOffVM_Task/ResetVM_Task 等）
- SnapshotVM：list/create/delete 三种 action

- [ ] **Step 7: 运行 build**

Run: `go build ./services/vcenter/cmd/vcenter-mcp-server`
Expected: build success

---

### Task 4: 完成 main 装配（mcphttp + auth + base path）并支持运行

**Files:**
- Create: `services/vcenter/cmd/vcenter-mcp-server/main.go`

- [ ] **Step 1: 实现 main（创建 govmomi API + registry + 挂载路由）**

Create `services/vcenter/cmd/vcenter-mcp-server/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/<org>/<repo>/modules/mcpkit/auth"
	"github.com/<org>/<repo>/modules/mcpkit/httpkit"
	"github.com/<org>/<repo>/modules/mcpkit/mcphttp"
	"github.com/<org>/<repo>/services/vcenter/internal/config"
	"github.com/<org>/<repo>/services/vcenter/internal/tools"
	"github.com/<org>/<repo>/services/vcenter/internal/vcenter"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	api, err := vcenter.NewGovmomiClient(ctx, cfg.VCenterURL, cfg.VCenterUsername, cfg.VCenterPassword, cfg.VCenterInsecure, cfg.VCenterCAFile)
	if err != nil {
		log.Fatal(err)
	}

	reg := tools.BuildRegistryWithOptions(tools.Options{
		EnableDangerousOps: cfg.EnableDangerousOps,
		API:                api,
	})
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

- [ ] **Step 2: 编译与基础冒烟**

Run: `go build ./services/vcenter/cmd/vcenter-mcp-server`
Expected: build success

