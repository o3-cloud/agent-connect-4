// Package oplog records unexpected server errors and the server-side
// duration of each REST move request as structured log records (SPEC.md
// NFR-006, BDR-031 AC-1). Game events are logged by the game service itself.
package oplog

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// EventServerError is the event name of an unexpected server error record.
const EventServerError = "server_error"

// EventMoveRequest is the event name of the record that times one REST move
// request, from the server receiving it to the response (BDR-031 AC-1).
const EventMoveRequest = "move_request"

const (
	gamesPath = "/api/v1/games/"
	movesPath = "/moves"
)

const internalBody = `{"error":{"code":"INTERNAL","message":"unexpected server error; no game state changed","retryable":true}}`

// Middleware logs one server_error record for each response with a 5xx status
// and for each panic. A panic before the response starts is answered with the
// SPEC.md §16 INTERNAL error instead of a dropped connection.
// http.ErrAbortHandler passes through unlogged, since it is a deliberate abort.
// It also logs one move_request record for each POST to a game's moves,
// whatever its outcome, with its status and durationMs.
// Records name the method, path and status only: never headers, so no
// credential reaches the log (SEC-005).
func Middleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recorder{ResponseWriter: w}
		if gameID, ok := moveGameID(r); ok {
			received := time.Now()
			defer func() {
				logger.Info("move request", "event", EventMoveRequest, "gameId", gameID,
					"status", rec.served(), "durationMs", float64(time.Since(received).Microseconds())/1000)
			}()
		}
		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler { //nolint:errorlint // sentinel compared as net/http does
					panic(p)
				}
				if !rec.wrote {
					rec.Header().Set("Content-Type", "application/json")
					rec.WriteHeader(http.StatusInternalServerError)
					_, _ = rec.Write([]byte(internalBody))
				}
				serverError(logger, r, http.StatusInternalServerError, fmt.Sprint(p))
				return
			}
			if rec.status >= http.StatusInternalServerError {
				serverError(logger, r, rec.status, http.StatusText(rec.status))
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// moveGameID returns the game of a REST move request: POST
// /api/v1/games/{gameId}/moves.
func moveGameID(r *http.Request) (string, bool) {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, gamesPath) || !strings.HasSuffix(r.URL.Path, movesPath) {
		return "", false
	}
	gameID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, gamesPath), movesPath)
	return gameID, gameID != "" && !strings.Contains(gameID, "/")
}

func serverError(logger *slog.Logger, r *http.Request, status int, cause string) {
	logger.Error("unexpected server error",
		"event", EventServerError,
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"error", cause,
	)
}

// recorder captures the response status. It forwards Flush and Unwrap so
// streaming responses (MCP over SSE) keep working.
type recorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

// served returns the status the client got: net/http sends 200 for a
// handler that writes nothing.
func (r *recorder) served() int {
	if !r.wrote {
		return http.StatusOK
	}
	return r.status
}

func (r *recorder) WriteHeader(status int) {
	if !r.wrote {
		r.status, r.wrote = status, true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *recorder) Write(p []byte) (int, error) {
	if !r.wrote {
		r.status, r.wrote = http.StatusOK, true
	}
	return r.ResponseWriter.Write(p)
}

func (r *recorder) Flush() {
	if !r.wrote {
		r.status, r.wrote = http.StatusOK, true
	}
	_ = http.NewResponseController(r.ResponseWriter).Flush()
}

func (r *recorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
