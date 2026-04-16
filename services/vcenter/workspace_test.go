package vcenter_test

import (
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit"
)

func TestWorkspace_RequireResolves(t *testing.T) {
	if mcpkit.Module != "mcpkit" {
		t.Fatalf("unexpected module: %s", mcpkit.Module)
	}
}
