package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/auth"
)

func TestBearerAuth(t *testing.T) {
	h := auth.Bearer("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	s := httptest.NewServer(h)
	defer s.Close()

	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, s.URL, nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, s.URL, nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

