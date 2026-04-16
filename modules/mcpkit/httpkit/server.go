package httpkit

import (
	"net/http"
	"strings"
)

func WrapBasePath(cfg MuxConfig, next http.Handler) http.Handler {
	if cfg.BasePath == "" || cfg.BasePath == "/" {
		return next
	}

	base := cfg.BasePath
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimSuffix(base, "/")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, base) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, base)
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}
		next.ServeHTTP(w, r2)
	})
}

func NewMux(cfg MuxConfig) http.Handler {
	mux := http.NewServeMux()
	RegisterHealth(mux)
	return WrapBasePath(cfg, mux)
}

