package main

import (
	"log"
	"net/http"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/auth"
	"github.com/xielforever/mcp-toolkit/modules/mcpkit/httpkit"
	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcphttp"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/config"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	reg := tools.BuildRegistryWithOptions(tools.Options{EnableDangerousOps: cfg.EnableDangerousOps})
	mcpHandler := mcphttp.NewHandler(mcphttp.Config{Registry: reg})

	mux := http.NewServeMux()
	mux.Handle("/sse", auth.Bearer(cfg.ServiceToken)(mcpHandler))
	mux.Handle("/messages", auth.Bearer(cfg.ServiceToken)(mcpHandler))
	httpkit.RegisterHealth(mux)

	h := httpkit.WrapBasePath(httpkit.MuxConfig{BasePath: cfg.BasePath}, mux)
	server := &http.Server{Addr: cfg.ListenAddr, Handler: h}
	log.Fatal(server.ListenAndServe())
}

