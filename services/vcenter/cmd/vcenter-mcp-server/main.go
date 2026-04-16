package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/xielforever/mcp-toolkit/modules/mcpkit/auth"
	"github.com/xielforever/mcp-toolkit/modules/mcpkit/httpkit"
	"github.com/xielforever/mcp-toolkit/modules/mcpkit/mcphttp"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/config"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/tools"
	"github.com/xielforever/mcp-toolkit/services/vcenter/internal/vcenter"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	api, err := vcenter.NewGovmomiClient(ctx, cfg.VCenterURL, cfg.VCenterUsername, cfg.VCenterPassword, cfg.VCenterInsecure, cfg.VCenterCAFile)
	if err != nil {
		log.Fatal(err)
	}

	reg := tools.BuildRegistryWithOptions(tools.Options{EnableDangerousOps: cfg.EnableDangerousOps, API: api})
	mcpHandler := mcphttp.NewHandler(mcphttp.Config{Registry: reg})

	mux := http.NewServeMux()
	mux.Handle("/sse", auth.Bearer(cfg.ServiceToken)(mcpHandler))
	mux.Handle("/messages", auth.Bearer(cfg.ServiceToken)(mcpHandler))
	httpkit.RegisterHealth(mux)

	h := httpkit.WrapBasePath(httpkit.MuxConfig{BasePath: cfg.BasePath}, mux)
	server := &http.Server{Addr: cfg.ListenAddr, Handler: h}
	log.Fatal(server.ListenAndServe())
}
