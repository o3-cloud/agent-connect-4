package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// exitCodes maps SPEC.md §16 error codes to CLI exit statuses (CLI-007).
var exitCodes = map[string]int{
	"INVALID_ARGUMENT":  2,
	"INVALID_COLUMN":    3,
	"COLUMN_FULL":       3,
	"UNAUTHENTICATED":   4,
	"FORBIDDEN":         4,
	"GAME_NOT_FOUND":    5,
	"ROUTE_NOT_FOUND":   5,
	"NOT_YOUR_TURN":     6,
	"STALE_STATE":       6,
	"GAME_NOT_STARTED":  6,
	"GAME_NOT_JOINABLE": 6,
	"SLOT_TAKEN":        6,
	"ALREADY_JOINED":    6,
	"GAME_COMPLETED":    7,
	"RATE_LIMITED":      8,
	"INTERNAL":          1,
}

// apiError is a failed operation, either reported by the server or raised by
// the CLI itself (bad arguments, unreachable server).
type apiError struct {
	Code      string
	Message   string
	Retryable bool
	raw       json.RawMessage // the server's full {"error": {...}} body, when there is one
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

func (e *apiError) exitCode() int {
	if code, ok := exitCodes[e.Code]; ok {
		return code
	}
	return 1
}

// document is the JSON to print for the error: the server's body verbatim, or
// one in the same shape.
func (e *apiError) document() string {
	if len(e.raw) > 0 {
		return string(e.raw)
	}
	body, _ := json.Marshal(map[string]any{"error": map[string]any{
		"code": e.Code, "message": e.Message, "retryable": e.Retryable,
	}})
	return string(body)
}

// document is a successful response: the raw JSON and its decoded form.
type document struct {
	raw json.RawMessage
	m   map[string]any
}

func (d document) str(key string) string {
	s, _ := d.m[key].(string)
	return s
}

func (d document) num(key string) int {
	f, _ := d.m[key].(float64)
	return int(f)
}

type client struct {
	base string
	key  string
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func (c *client) do(method, path string, body any) (document, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return document{}, &apiError{Code: "INVALID_ARGUMENT", Message: err.Error()}
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return document{}, &apiError{Code: "INVALID_ARGUMENT", Message: err.Error()}
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.key != "" {
		request.Header.Set("Authorization", "Bearer "+c.key)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return document{}, &apiError{Code: "INTERNAL", Retryable: true,
			Message: fmt.Sprintf("cannot reach the arena server at %s: %v", c.base, err)}
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return document{}, &apiError{Code: "INTERNAL", Retryable: true, Message: "reading the server response: " + err.Error()}
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return document{}, &apiError{Code: "INTERNAL", Message: fmt.Sprintf("the server sent %d with a non-JSON body", response.StatusCode)}
	}
	if response.StatusCode >= 400 {
		e := &apiError{Code: "INTERNAL", Message: http.StatusText(response.StatusCode), raw: json.RawMessage(bytes.TrimSpace(raw))}
		if obj, ok := decoded["error"].(map[string]any); ok {
			e.Code, _ = obj["code"].(string)
			e.Message, _ = obj["message"].(string)
			e.Retryable, _ = obj["retryable"].(bool)
		}
		return document{}, e
	}
	return document{raw: json.RawMessage(bytes.TrimSpace(raw)), m: decoded}, nil
}
