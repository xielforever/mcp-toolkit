package mcphttp_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcp"
	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcphttp"
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
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}

	resp, err = http.Post(s.URL+"/messages", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 and non-404 got %d", resp.StatusCode)
	}
}

