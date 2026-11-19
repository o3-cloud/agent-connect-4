// Package app implements the Connect Four Arena HTTP service.
package app

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connect4arena/internal/api"
	"connect4arena/internal/game"
	"connect4arena/internal/houseagent"
	"connect4arena/internal/identity"
	"connect4arena/internal/mcpserver"
	"connect4arena/internal/oplog"
	"connect4arena/internal/ratelimit"
	"connect4arena/internal/web"
)

// Config holds the server's tunable settings.
type Config struct {
	// RateLimit is the number of API and MCP requests each client may make
	// per RateWindow (SEC-007).
	RateLimit  int
	RateWindow time.Duration
	// Logger receives the structured operational log (NFR-006). Nil logs
	// nothing.
	Logger *slog.Logger
}

// DefaultConfig allows 600 requests per minute per client: enough for an
// agent polling every second while it plays, far below abusive rates.
func DefaultConfig() Config {
	return Config{RateLimit: 600, RateWindow: time.Minute}
}

// ConfigFromEnv reads ARENA_RATE_LIMIT (requests per window) and
// ARENA_RATE_WINDOW (a Go duration such as 10s), defaulting each to
// DefaultConfig.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := DefaultConfig()
	if v := getenv("ARENA_RATE_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("ARENA_RATE_LIMIT must be a positive integer, got %q", v)
		}
		cfg.RateLimit = n
	}
	if v := getenv("ARENA_RATE_WINDOW"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("ARENA_RATE_WINDOW must be a positive duration such as 10s, got %q", v)
		}
		cfg.RateWindow = d
	}
	return cfg, nil
}

// Handler returns the service's HTTP handler with DefaultConfig.
func Handler() http.Handler {
	return New(DefaultConfig())
}

// New returns the service's HTTP handler: REST under /api/v1 and MCP at /mcp,
// both backed by one new in-memory game service and agent registry, and the
// browser UI. Arena Bot (ADR-014) plays house games through the same REST API.
// API and MCP requests are rate limited per client (SEC-007). Game events and
// unexpected server errors go to cfg.Logger (NFR-006).
func New(cfg Config) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	mux := http.NewServeMux()
	service, registry := game.NewService(game.WithLogger(logger)), identity.NewRegistry()
	bot := houseagent.New(mux)
	api.Register(mux, service, registry, api.WithHouseOpponent(bot.Start))
	web.Register(mux, service, registry, bot.Start)
	mux.Handle(mcpserver.Path, mcpserver.Handler(service, registry))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return oplog.Middleware(rateLimit(mux, ratelimit.New(cfg.RateLimit, cfg.RateWindow), registry), logger)
}

// rateLimit applies the limiter to API and MCP requests. A request with a
// valid API key spends its agent's budget, so agents behind one address do not
// share a limit. Any other request, including one with a made-up key, spends
// its IP address's budget, so rotating fake keys does not evade the limit.
// Arena Bot calls the mux directly and is not limited.
func rateLimit(next http.Handler, limiter *ratelimit.Limiter, registry *identity.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, api.Prefix+"/") && r.URL.Path != mcpserver.Path {
			next.ServeHTTP(w, r)
			return
		}
		if ok, retryAfter := limiter.Allow(clientOf(r, registry)); !ok {
			api.WriteRateLimited(w, retryAfter)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientOf(r *http.Request, registry *identity.Registry) string {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		if agent, ok := registry.Authenticate(token); ok {
			return "agent:" + agent.ID
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}
