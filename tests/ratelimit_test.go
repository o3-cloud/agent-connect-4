package tests

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"connect4arena/internal/app"
)

func limitedServer(t *testing.T, limit int, window time.Duration) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(app.New(app.Config{RateLimit: limit, RateWindow: window}))
	t.Cleanup(server.Close)
	return server
}

// wantRateLimited asserts SPEC.md §16 RATE_LIMITED with retryAfterMs and Retry-After.
func wantRateLimited(t *testing.T, r response, window time.Duration) {
	t.Helper()
	errObject := r.wantError(t, http.StatusTooManyRequests, "RATE_LIMITED")
	if errObject["retryable"] != true {
		t.Fatalf("RATE_LIMITED retryable = %v", errObject["retryable"])
	}
	ms := number(errObject["retryAfterMs"])
	if ms <= 0 || ms > int(window.Milliseconds()) {
		t.Fatalf("retryAfterMs = %v, want 1..%d", errObject["retryAfterMs"], window.Milliseconds())
	}
	seconds, err := strconv.Atoi(r.header.Get("Retry-After"))
	if err != nil || seconds < 1 || seconds > int(window.Seconds())+1 {
		t.Fatalf("Retry-After = %q", r.header.Get("Retry-After"))
	}
}

// BDR-014 scenario 1, AC-2..3: one noisy client is limited; another plays on.
func TestRateLimitIsolatesNoisyClient(t *testing.T) {
	const limit, window = 5, 10 * time.Second
	server := limitedServer(t, limit, window)
	a, aOpp := register(t, server, "A"), register(t, server, "A2")
	b, bOpp := register(t, server, "B"), register(t, server, "B2")

	start := func(red, yellow client) string {
		created := red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "agent_vs_agent"}).wantStatus(t, http.StatusCreated)
		id := created.body["gameId"].(string)
		red.do(http.MethodPost, "/api/v1/games/"+id+"/join", map[string]any{"disc": "red", "displayName": "R", "type": "agent"}).wantStatus(t, http.StatusOK)
		yellow.do(http.MethodPost, "/api/v1/games/"+id+"/join", map[string]any{"disc": "yellow", "displayName": "Y", "type": "agent"}).wantStatus(t, http.StatusOK)
		return id
	}
	gameA, gameB := start(a, aOpp), start(b, bOpp) // a and b have each used 2 requests

	limited := 0
	for range 10 {
		r := a.do(http.MethodGet, "/api/v1/games/"+gameA, nil)
		switch r.status {
		case http.StatusOK:
		case http.StatusTooManyRequests:
			wantRateLimited(t, r, window)
			limited++
		default:
			t.Fatalf("state request: %d %s", r.status, r.raw)
		}
	}
	if limited < 5 {
		t.Fatalf("%d of 10 requests were rate limited, want at least 5", limited)
	}

	// AC-2: a rate-limited move changes nothing.
	wantRateLimited(t, a.do(http.MethodPost, "/api/v1/games/"+gameA+"/moves", map[string]any{"column": 3, "expectedVersion": 0}), window)
	if state := aOpp.do(http.MethodGet, "/api/v1/games/"+gameA, nil).wantStatus(t, http.StatusOK).body; number(state["version"]) != 0 {
		t.Fatalf("rate-limited move changed game A: version %v", state["version"])
	}

	// AC-3: agent B is unaffected.
	moved := b.do(http.MethodPost, "/api/v1/games/"+gameB+"/moves", map[string]any{"column": 3, "expectedVersion": 0}).wantStatus(t, http.StatusOK)
	if moved.body["accepted"] != true {
		t.Fatalf("B's move = %v", moved.body)
	}
}

// AC-1: the first N requests in a window succeed; the window then resets.
func TestRateLimitAllowsNPerWindow(t *testing.T) {
	const limit, window = 4, 300 * time.Millisecond
	server := limitedServer(t, limit, window)
	agent := register(t, server, "A")
	for i := range limit {
		if r := agent.do(http.MethodGet, "/api/v1/games", nil); r.status != http.StatusOK {
			t.Fatalf("request %d of %d: %d %s", i+1, limit, r.status, r.raw)
		}
	}
	wantRateLimited(t, agent.do(http.MethodGet, "/api/v1/games", nil), window)

	time.Sleep(window + 50*time.Millisecond)
	agent.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK)
}

// Anonymous and forged-key requests share their IP's budget, so rotating
// made-up keys does not escape the limit. Health checks are never limited.
func TestRateLimitByIPWithoutValidKey(t *testing.T) {
	const limit, window = 3, 10 * time.Second
	server := limitedServer(t, limit, window)
	for i := range limit {
		forged := client{t: t, server: server, token: "c4_forged_" + strconv.Itoa(i)}
		if r := forged.do(http.MethodGet, "/api/v1/games", nil); r.status != http.StatusOK {
			t.Fatalf("request %d: %d", i+1, r.status)
		}
	}
	anon := client{t: t, server: server}
	wantRateLimited(t, anon.do(http.MethodGet, "/api/v1/games", nil), window)
	wantRateLimited(t, client{t: t, server: server, token: "c4_forged_new"}.do(http.MethodGet, "/api/v1/games", nil), window)

	if r := anon.do(http.MethodGet, "/healthz", nil); r.status != http.StatusNoContent {
		t.Fatalf("/healthz = %d while rate limited, want 204", r.status)
	}
}

// The MCP endpoint shares the agent's budget with REST.
func TestRateLimitCoversMCP(t *testing.T) {
	const limit, window = 2, 10 * time.Second
	server := limitedServer(t, limit, window)
	agent := register(t, server, "A") // registration counts against the IP, not the agent
	for range limit {
		agent.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK)
	}
	r := agent.do(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	wantRateLimited(t, r, window)
}

func TestRateLimitConfigFromEnv(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

	cfg, err := app.ConfigFromEnv(env(nil))
	if err != nil || cfg != app.DefaultConfig() || cfg.RateLimit <= 0 || cfg.RateWindow <= 0 {
		t.Fatalf("defaults = %+v, %v", cfg, err)
	}
	cfg, err = app.ConfigFromEnv(env(map[string]string{"ARENA_RATE_LIMIT": "5", "ARENA_RATE_WINDOW": "10s"}))
	if err != nil || cfg.RateLimit != 5 || cfg.RateWindow != 10*time.Second {
		t.Fatalf("configured = %+v, %v", cfg, err)
	}
	for _, bad := range []map[string]string{
		{"ARENA_RATE_LIMIT": "zero"}, {"ARENA_RATE_LIMIT": "0"}, {"ARENA_RATE_LIMIT": "-3"},
		{"ARENA_RATE_WINDOW": "soon"}, {"ARENA_RATE_WINDOW": "0s"},
	} {
		if _, err := app.ConfigFromEnv(env(bad)); err == nil {
			t.Fatalf("ConfigFromEnv(%v) succeeded, want an error", bad)
		}
	}
}
