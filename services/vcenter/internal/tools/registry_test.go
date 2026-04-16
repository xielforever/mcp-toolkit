package tools_test

import (
	"context"
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcp"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/tools"
)

func TestRegistry_InventoryToolExists(t *testing.T) {
	reg := tools.BuildRegistryWithOptions(tools.Options{})
	for _, tc := range []struct {
		name  string
		input map[string]any
	}{
		{name: "vcenter.list_inventory", input: map[string]any{"types": []any{"vm"}}},
		{name: "vcenter.get_vm", input: map[string]any{"moRef": "vm-1"}},
		{name: "vcenter.query_perf", input: map[string]any{"objectType": "vm", "moRef": "vm-1", "metrics": []any{"cpu.usage"}}},
		{name: "vcenter.list_events", input: map[string]any{"startTime": "t1", "endTime": "t2"}},
		{name: "vcenter.list_alarms", input: map[string]any{}},
	} {
		_, err := reg.Call(context.Background(), tc.name, tc.input)
		if err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
		if err == mcp.ErrToolNotFound {
			t.Fatalf("expected tool to be registered (%s), got tool not found", tc.name)
		}
	}
}
