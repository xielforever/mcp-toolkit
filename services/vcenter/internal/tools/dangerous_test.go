package tools_test

import (
	"context"
	"testing"

	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/tools"
)

func TestDangerousOps_DefaultDisabled(t *testing.T) {
	reg := tools.BuildRegistryWithOptions(tools.Options{EnableDangerousOps: false})
	_, err := reg.Call(context.Background(), "vcenter.power", map[string]any{"vmMoRef": "vm-1", "action": "on"})
	if err == nil {
		t.Fatalf("expected error when dangerous ops disabled")
	}
}

