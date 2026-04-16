package tools

import (
	"context"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcp"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/vcenter"
)

type Options struct {
	EnableDangerousOps bool
	API                vcenter.API
}

func BuildRegistryWithOptions(opts Options) *mcp.Registry {
	r := mcp.NewRegistry()
	r.Register(mcp.Tool{
		Name:        "vcenter.ping",
		Description: "ping",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	})

	if opts.EnableDangerousOps && opts.API != nil {
		registerDangerousTools(r, opts.API)
	}

	return r
}

func registerDangerousTools(r *mcp.Registry, api vcenter.API) {
	_ = api
	r.Register(mcp.Tool{
		Name:        "vcenter.power",
		Description: "VM power operations",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return nil, nil
		},
	})

	r.Register(mcp.Tool{
		Name:        "vcenter.snapshot",
		Description: "VM snapshot operations",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, input map[string]any) (any, error) {
			return nil, nil
		},
	})
}

