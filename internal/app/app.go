// Package app implements the Connect Four Arena HTTP service.
package app

import (
	"net/http"

	"connect4arena/internal/api"
	"connect4arena/internal/game"
	"connect4arena/internal/houseagent"
	"connect4arena/internal/identity"
	"connect4arena/internal/mcpserver"
	"connect4arena/internal/web"
)

// Handler returns the service's HTTP handler: REST under /api/v1 and MCP at
// /mcp, both backed by one new in-memory game service and agent registry. Arena Bot
// (ADR-014) plays house games through the same REST API.
func Handler() http.Handler {
	mux := http.NewServeMux()
	service, registry := game.NewService(), identity.NewRegistry()
	bot := houseagent.New(mux)
	api.Register(mux, service, registry, api.WithHouseOpponent(bot.Start))
	web.Register(mux, service, registry, bot.Start)
	mux.Handle(mcpserver.Path, mcpserver.Handler(service, registry))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}
