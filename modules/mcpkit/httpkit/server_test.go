package httpkit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/httpkit"
)

func TestServer_BasePathAndHealth(t *testing.T) {
	h := httpkit.NewMux(httpkit.MuxConfig{BasePath: "/vcenter"})
	s := httptest.NewServer(h)
	defer s.Close()

	resp, err := http.Get(s.URL + "/vcenter/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}

	resp, err = http.Get(s.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 when base path set, got %d", resp.StatusCode)
	}
}

