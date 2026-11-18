// Package app implements the Connect Four Arena HTTP service.
package app

import (
	"fmt"
	"net/http"

	"connect4arena/internal/api"
	"connect4arena/internal/game"
	"connect4arena/internal/identity"
	"connect4arena/internal/mcpserver"
)

// Handler returns the service's HTTP handler: REST under /api/v1 and MCP at
// /mcp, both backed by one new in-memory game service and agent registry.
func Handler() http.Handler {
	mux := http.NewServeMux()
	service, registry := game.NewService(), identity.NewRegistry()
	api.Register(mux, service, registry)
	mux.Handle(mcpserver.Path, mcpserver.Handler(service, registry))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "Connect Four Arena")
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}
