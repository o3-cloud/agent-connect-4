// Package oplog records unexpected server errors as structured log records
// (SPEC.md NFR-006). Game events are logged by the game service itself.
package oplog

import (
	"fmt"
	"log/slog"
	"net/http"
)

// EventServerError is the event name of an unexpected server error record.
const EventServerError = "server_error"

const internalBody = `{"error":{"code":"INTERNAL","message":"unexpected server error; no game state changed","retryable":true}}`

// Middleware logs one server_error record for each response with a 5xx status
// and for each panic. A panic before the response starts is answered with the
// SPEC.md §16 INTERNAL error instead of a dropped connection.
// http.ErrAbortHandler passes through unlogged, since it is a deliberate abort.
// Records name the method, path and status only: never headers, so no
// credential reaches the log (SEC-005).
func Middleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recorder{ResponseWriter: w}
		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler { //nolint:errorlint // sentinel compared as net/http does
					panic(p)
				}
				if !rec.wrote {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(internalBody))
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
