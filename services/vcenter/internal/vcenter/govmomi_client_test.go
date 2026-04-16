package vcenter_test

import (
	"context"
	"testing"

	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/vcenter"
)

var _ vcenter.API = (*vcenter.GovmomiClient)(nil)

func TestNewGovmomiClient_InvalidURL(t *testing.T) {
	_, err := vcenter.NewGovmomiClient(context.Background(), ":", "u", "p", true, "")
	if err == nil {
		t.Fatal("expected error")
	}
}
