package mcphttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcp"
)

type Handler struct {
	cfg Config
}

func NewHandler(cfg Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/sse":
		serveSSE(h.cfg, w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/messages":
		serveMessages(h.cfg, w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type messageEnvelope struct {
	SessionID string         `json:"session_id"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params"`
}

func serveMessages(cfg Config, w http.ResponseWriter, r *http.Request) {
	var env messageEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if env.SessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sess, ok := store.get(env.SessionID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch env.Method {
	case "tools/list":
		out := map[string]any{"tools": cfg.Registry.List()}
		b, _ := json.Marshal(out)
		sess.send <- b
		w.WriteHeader(http.StatusAccepted)
	case "tools/call":
		name, _ := env.Params["name"].(string)
		args, _ := env.Params["arguments"].(map[string]any)
		res, err := cfg.Registry.Call(ctx, name, args)
		if err == mcp.ErrToolNotFound {
			b, _ := json.Marshal(map[string]any{"error": "tool_not_found"})
			sess.send <- b
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if err != nil {
			b, _ := json.Marshal(map[string]any{"error": "tool_error", "message": err.Error()})
			sess.send <- b
			w.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(map[string]any{"result": res})
		sess.send <- b
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
