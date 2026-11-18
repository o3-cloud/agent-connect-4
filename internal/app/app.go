// Package app implements the Connect Four Arena HTTP service.
package app

import (
	"fmt"
	"net/http"

	"connect4arena/internal/api"
	"connect4arena/internal/game"
	"connect4arena/internal/identity"
)

// Handler returns the service's HTTP handler, backed by a new in-memory game service and agent registry.
func Handler() http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, game.NewService(), identity.NewRegistry())
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "Connect Four Arena")
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}
