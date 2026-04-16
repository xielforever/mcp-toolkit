package tools

import (
	"context"
	"errors"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcp"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/vcenter"
)

type Options struct {
	EnableDangerousOps bool
	API                vcenter.API
}

func BuildRegistryWithOptions(opts Options) *mcp.Registry {
	api := opts.API
	r := mcp.NewRegistry()
	r.Register(mcp.Tool{
		Name:        "vcenter.ping",
		Description: "ping",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	})

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
