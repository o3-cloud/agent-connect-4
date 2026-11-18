package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackable-specs/stack-base-go/internal/app"
)

func TestHandlerReturnsExpectedResponse(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		statusCode int
		body       string
	}{
		{name: "returns service name", path: "/", statusCode: http.StatusOK, body: "stack-base-go\n"},
		{name: "returns healthy status", path: "/healthz", statusCode: http.StatusNoContent},
		{name: "rejects unknown path", path: "/unknown", statusCode: http.StatusNotFound, body: "404 page not found\n"},
		{name: "rejects unknown api path", path: "/api/v1/games", statusCode: http.StatusNotFound, body: "404 page not found\n"},
		{name: "rejects unknown mcp path", path: "/mcp", statusCode: http.StatusNotFound, body: "404 page not found\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			app.Handler().ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("status code = %d, want %d", response.Code, test.statusCode)
			}
			if response.Body.String() != test.body {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.body)
			}
		})
	}
}
