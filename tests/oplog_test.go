package tests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connect4arena/internal/app"
	"connect4arena/internal/oplog"
)

// logSink is a concurrency-safe buffer of JSON log lines.
type logSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *logSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *logSink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// records parses every line as JSON (AC-1: machine-parseable) and returns
// those whose gameId is gameID, or all records when gameID is empty.
func (s *logSink) records(t *testing.T, gameID string) []map[string]any {
	t.Helper()
	var out []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(s.String()))
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("log line is not JSON: %v\n%s", err, scanner.Text())
		}
		if gameID == "" || record["gameId"] == gameID {
			out = append(out, record)
		}
	}
	return out
}

func (s *logSink) events(t *testing.T, gameID, event string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, record := range s.records(t, gameID) {
		if record["event"] == event {
			out = append(out, record)
		}
	}
	return out
}

func (s *logSink) wantEvents(t *testing.T, gameID, event string, n int) []map[string]any {
	t.Helper()
	got := s.events(t, gameID, event)
	if len(got) != n {
		t.Fatalf("%d %q records for game %s, want %d; log:\n%s", len(got), event, gameID, n, s.String())
	}
	return got
}

func loggedArena(t *testing.T) (*arena, *logSink) {
	t.Helper()
	sink := &logSink{}
	cfg := app.DefaultConfig()
	cfg.Logger = slog.New(slog.NewJSONHandler(sink, nil))
	server := httptest.NewServer(app.New(cfg))
	t.Cleanup(server.Close)
	return &arena{
		server: server,
		red:    register(t, server, "Alpha"),
		yellow: register(t, server, "Beta"),
		other:  register(t, server, "Gamma"),
		anon:   client{t: t, server: server},
	}, sink
}

// BDR-032 scenario 1, AC-1..3: the log trail for a short game.
func TestLogTrailForShortGame(t *testing.T) {
	a, sink := loggedArena(t)
	a.start(t)
	a.play(t, 3)
	a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 9, "expectedVersion": 1}).wantError(t, http.StatusUnprocessableEntity, "INVALID_COLUMN")

	created := sink.wantEvents(t, a.gameID, "game_created", 1)[0]
	if created["mode"] != "agent_vs_agent" {
		t.Fatalf("game_created = %v", created)
	}
	joined := sink.wantEvents(t, a.gameID, "player_joined", 2)
	if joined[0]["disc"] != "red" || joined[1]["disc"] != "yellow" {
		t.Fatalf("player_joined = %v", joined)
	}
	accepted := sink.wantEvents(t, a.gameID, "move_accepted", 1)[0]
	if number(accepted["column"]) != 3 || accepted["disc"] != "red" || number(accepted["version"]) != 1 {
		t.Fatalf("move_accepted = %v", accepted)
	}
	rejected := sink.wantEvents(t, a.gameID, "move_rejected", 1)[0]
	if rejected["code"] != "INVALID_COLUMN" || number(rejected["column"]) != 9 {
		t.Fatalf("move_rejected = %v", rejected)
	}
	sink.wantEvents(t, a.gameID, "game_completed", 0)

	for _, record := range sink.records(t, a.gameID) {
		if record["level"] == nil || record["time"] == nil || record["msg"] == nil {
			t.Fatalf("record lacks level, time or msg: %v", record)
		}
	}
	// AC-3: no credential in any record, including registration.
	for _, key := range []string{a.red.token, a.yellow.token, a.other.token} {
		if strings.Contains(sink.String(), key) {
			t.Fatalf("log contains an API key:\n%s", sink.String())
		}
	}
}

func TestLogRecordsWinAsCompletion(t *testing.T) {
	a, sink := loggedArena(t)
	a.start(t)
	a.play(t, 0, 1, 0, 1, 0, 1, 0)

	completed := sink.wantEvents(t, a.gameID, "game_completed", 1)[0]
	if completed["winner"] != "red" || completed["resultReason"] != "connect_four" || number(completed["moves"]) != 7 {
		t.Fatalf("game_completed = %v", completed)
	}
	sink.wantEvents(t, a.gameID, "move_accepted", 7)
}

func TestLogRecordsResignationAsCompletion(t *testing.T) {
	a, sink := loggedArena(t)
	a.start(t)
	a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantStatus(t, http.StatusOK)

	completed := sink.wantEvents(t, a.gameID, "game_completed", 1)[0]
	if completed["winner"] != "red" || completed["resultReason"] != "resignation" {
		t.Fatalf("game_completed = %v", completed)
	}
}

func TestLogRecordsEveryRejectedMoveCode(t *testing.T) {
	a, sink := loggedArena(t)
	a.start(t)
	move := func(c client, column, version int) response {
		return c.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", map[string]any{"column": column, "expectedVersion": version})
	}
	move(a.yellow, 3, 0).wantError(t, http.StatusConflict, "NOT_YOUR_TURN")
	move(a.red, 3, 5).wantError(t, http.StatusConflict, "STALE_STATE")
	move(a.other, 3, 0).wantError(t, http.StatusForbidden, "FORBIDDEN")

	rejected := sink.wantEvents(t, a.gameID, "move_rejected", 3)
	for i, code := range []string{"NOT_YOUR_TURN", "STALE_STATE", "FORBIDDEN"} {
		if rejected[i]["code"] != code {
			t.Fatalf("move_rejected[%d] = %v, want code %s", i, rejected[i], code)
		}
	}
}

// An idempotent retry returns the recorded result; it is not a second move.
func TestLogDoesNotRecordReplayAsSecondMove(t *testing.T) {
	a, sink := loggedArena(t)
	a.start(t)
	body := map[string]any{"column": 3, "expectedVersion": 0, "idempotencyKey": "k-1"}
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", body).wantStatus(t, http.StatusOK)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", body).wantStatus(t, http.StatusOK)

	sink.wantEvents(t, a.gameID, "move_accepted", 1)
	sink.wantEvents(t, a.gameID, "move_rejected", 0)
}

// Moves made over MCP are logged by the same service.
func TestLogRecordsMCPMoves(t *testing.T) {
	a, sink := loggedArena(t)
	a.start(t)
	session := mustConnectMCP(t, a.red)
	callOK(t, session, "make_move", map[string]any{"game_id": a.gameID, "column": 2, "expected_version": 0, "idempotency_key": "m-1"})

	sink.wantEvents(t, a.gameID, "move_accepted", 1)
	if strings.Contains(sink.String(), a.red.token) {
		t.Fatalf("log contains an API key:\n%s", sink.String())
	}
}

func serveLogged(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *logSink) {
	t.Helper()
	sink := &logSink{}
	server := httptest.NewServer(oplog.Middleware(handler, slog.New(slog.NewJSONHandler(sink, nil))))
	t.Cleanup(server.Close)
	return server, sink
}

// AC-1: an unexpected server error (a panic) gives one server_error record and
// a SPEC.md §16 INTERNAL response instead of a dropped connection.
func TestLogRecordsPanicAsServerError(t *testing.T) {
	server, sink := serveLogged(t, func(http.ResponseWriter, *http.Request) {
		panic(errors.New("boom"))
	})
	c := client{t: t, server: server}
	errObject := c.do(http.MethodGet, "/api/v1/games/g_1", nil).wantError(t, http.StatusInternalServerError, "INTERNAL")
	if errObject["retryable"] != true {
		t.Fatalf("INTERNAL retryable = %v", errObject["retryable"])
	}

	records := sink.events(t, "", "server_error")
	if len(records) != 1 {
		t.Fatalf("%d server_error records, want 1; log:\n%s", len(records), sink.String())
	}
	r := records[0]
	if r["level"] != "ERROR" || r["method"] != "GET" || r["path"] != "/api/v1/games/g_1" || number(r["status"]) != 500 || !strings.Contains(r["error"].(string), "boom") {
		t.Fatalf("server_error = %v", r)
	}
}

func TestLogRecordsServerErrorStatus(t *testing.T) {
	server, sink := serveLogged(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fail":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			_, _ = io.WriteString(w, "ok")
		}
	})
	c := client{t: t, server: server}
	c.do(http.MethodGet, "/ok", nil).wantStatus(t, http.StatusOK)
	c.do(http.MethodGet, "/missing", nil).wantStatus(t, http.StatusNotFound)
	c.do(http.MethodGet, "/fail", nil).wantStatus(t, http.StatusServiceUnavailable)

	records := sink.records(t, "")
	if len(records) != 1 || records[0]["event"] != "server_error" || number(records[0]["status"]) != 503 || records[0]["path"] != "/fail" {
		t.Fatalf("records = %v, want one server_error for /fail", records)
	}
}

// http.ErrAbortHandler is the standard way to abort a response; it is not an
// unexpected error and must still abort the connection.
func TestLogLetsAbortHandlerThrough(t *testing.T) {
	server, sink := serveLogged(t, func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	})
	if resp, err := server.Client().Get(server.URL + "/x"); err == nil {
		_ = resp.Body.Close()
		t.Fatalf("aborted request returned %d, want a dropped connection", resp.StatusCode)
	}
	if got := sink.String(); got != "" {
		t.Fatalf("abort was logged:\n%s", got)
	}
}
