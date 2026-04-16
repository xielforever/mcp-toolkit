package mcphttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

var (
	store    = newSessionStore(5 * time.Minute)
	reapOnce sync.Once
)

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func serveSSE(cfg Config, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reapOnce.Do(func() { go store.reap(context.Background()) })

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := newSessionID()
	sess := &session{id: id, created: time.Now(), send: make(chan []byte, 128)}
	store.put(sess)

	_, _ = w.Write([]byte("event: init\n"))
	_, _ = w.Write([]byte("data: {\"session_id\":\"" + id + "\"}\n\n"))
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			store.delete(id)
			return
		case msg := <-sess.send:
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(msg)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

