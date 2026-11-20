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
	"connect4arena/internal/journal"
	"connect4arena/internal/mcpserver"
	"connect4arena/internal/oplog"
	"connect4arena/internal/ratelimit"
	"connect4arena/internal/web"
)

// Config holds the server's tunable settings.
type Config struct {
	// Port is the TCP port the server listens on (PORT, default 8080).
	Port int
	// JournalPath is the event journal that keeps games and agents across
	// restarts (ARENA_JOURNAL_PATH, ADR-015). Empty keeps them in memory only.
	JournalPath string
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
	return Config{Port: 8080, RateLimit: 600, RateWindow: time.Minute}
}

// ConfigFromEnv reads PORT (1 to 65535), ARENA_RATE_LIMIT (requests per
// window), ARENA_RATE_WINDOW (a Go duration such as 10s) and
// ARENA_JOURNAL_PATH (a file path), defaulting each to DefaultConfig. Its errors name the variable, so the server can refuse to
// start before it logs anything.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := DefaultConfig()
	cfg.JournalPath = getenv("ARENA_JOURNAL_PATH")
	if v := getenv("PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 65535 {
			return Config{}, fmt.Errorf("PORT must be a port number from 1 to 65535, got %q", v)
		}
		cfg.Port = n
	}
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

// New returns an in-memory arena's HTTP handler; it ignores cfg.JournalPath.
// See Open.
func New(cfg Config) http.Handler {
	cfg.JournalPath = ""
	handler, _, _ := Open(cfg) // cannot fail without a journal
	return handler
}

// Open returns the service's HTTP handler: REST under /api/v1 and MCP at
// /mcp, both backed by one game service and agent registry, and the browser
// UI. Arena Bot (ADR-014) plays house games through the same REST API. API
// and MCP requests are rate limited per client (SEC-007). Game events and
// unexpected server errors go to cfg.Logger (NFR-006).
//
// With cfg.JournalPath set, every change is journaled there before it takes
// effect, and Open first restores the arena by replaying the journal through
// the same code as live play (ADR-015); Arena Bot carries on its in-progress
// games. Open fails if the journal cannot be read or does not replay.
// closeJournal closes the journal.
func Open(cfg Config) (handler http.Handler, closeJournal func() error, err error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	mux := http.NewServeMux()
	registry := identity.NewRegistry()
	bot := houseagent.New(mux, houseagent.WithCredentials(func() (string, error) {
		_, key, err := registry.HouseKey(houseagent.DisplayName)
		return key, err
	}))
	// Arena Bot sits down before a human's join returns (ADR-014).
	opts := []game.Option{game.WithLogger(logger), game.WithAwaitingOpponent(bot.Seat)}
	closeJournal = func() error { return nil }
	var events []journal.Event
	if cfg.JournalPath != "" {
		var j *journal.Journal
		if j, events, err = journal.Open(cfg.JournalPath, logger); err != nil {
			return nil, nil, err
		}
		closeJournal = j.Close
		opts = append(opts, game.WithStore(j))
		registry.WithStore(j)
	}
	service := game.NewService(opts...)
	if err := restore(service, registry, events); err != nil {
		_ = closeJournal()
		return nil, nil, fmt.Errorf("restore %s: %w", cfg.JournalPath, err)
	}

	api.Register(mux, service, registry, api.WithHouseOpponent(bot.Start))
	web.Register(mux, service, registry, bot.Start)
	mux.Handle(mcpserver.Path, mcpserver.Handler(service, registry))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if cfg.JournalPath != "" {
		resumed := resumeHouseGames(service, registry, bot)
		logger.Info("journal restored", "event", "journal_restored", "path", cfg.JournalPath,
			"events", len(events), "games", len(service.List()), "houseGamesResumed", resumed)
	}
	return oplog.Middleware(rateLimit(mux, ratelimit.New(cfg.RateLimit, cfg.RateWindow), registry, service), logger), closeJournal, nil
}

// restore replays the journal's events, in order, into the service and the
// registry. An event neither recognises, or one that does not replay, stops
// the restore: nothing is skipped silently.
func restore(service *game.Service, registry *identity.Registry, events []journal.Event) error {
	for _, e := range events {
		handled, err := service.Restore(e.Type, e.At, e.Data)
		if !handled && err == nil {
			handled, err = registry.Restore(e.Type, e.Data)
		}
		switch {
		case err != nil:
			return fmt.Errorf("event seq %d: %w", e.Seq, err)
		case !handled:
			return fmt.Errorf("event seq %d: unknown event type %q", e.Seq, e.Type)
		}
	}
	return nil
}

// resumeHouseGames restarts Arena Bot in every in-progress game it plays and
// returns how many there were.
func resumeHouseGames(service *game.Service, registry *identity.Registry, bot *houseagent.Agent) int {
	house, ok := registry.House()
	if !ok {
		return 0
	}
	games, _ := service.ListByStatus(game.StatusInProgress)
	resumed := 0
	for _, g := range games {
		for _, p := range []*game.Player{g.Red, g.Yellow} {
			if p != nil && p.ID == house.ID {
				bot.Resume(g.ID, string(p.Disc))
				resumed++
			}
		}
	}
	return resumed
}

// rateLimit applies the limiter to API and MCP requests. A request with a
// valid API key spends its agent's budget, so agents behind one address do not
// share a limit. Any other request, including one with a made-up key, spends
// its IP address's budget, so rotating fake keys does not evade the limit.
// Arena Bot calls the mux directly and is not limited.
func rateLimit(next http.Handler, limiter *ratelimit.Limiter, registry *identity.Registry, service *game.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, api.Prefix+"/") && r.URL.Path != mcpserver.Path {
			next.ServeHTTP(w, r)
			return
		}
		if ok, retryAfter := limiter.Allow(clientOf(r, registry)); !ok {
			if gameID, isMove := moveGameID(r); isMove {
				service.RecordRejectedMove(gameID, playerOf(r, registry), game.CodeRateLimited)
			}
			api.WriteRateLimited(w, retryAfter)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// moveGameID reports whether r submits a move, over REST or MCP, and to which
// game. A rate-limited move never reaches the game service, so rateLimit logs
// it (BDR-032 AC-1).
func moveGameID(r *http.Request) (string, bool) {
	if r.Method == http.MethodPost {
		if rest, ok := strings.CutPrefix(r.URL.Path, api.Prefix+"/games/"); ok {
			if gameID, ok := strings.CutSuffix(rest, "/moves"); ok && gameID != "" && !strings.Contains(gameID, "/") {
				return gameID, true
			}
		}
	}
	return mcpserver.MoveGameID(r)
}

// playerOf is the agent behind r's API key, or "" if it has no valid key.
func playerOf(r *http.Request, registry *identity.Registry) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	agent, _ := registry.Authenticate(token)
	return agent.ID
}

func clientOf(r *http.Request, registry *identity.Registry) string {
	if id := playerOf(r, registry); id != "" {
		return "agent:" + id
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}
