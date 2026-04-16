package mcp_test

import (
	"context"
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcp"
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

