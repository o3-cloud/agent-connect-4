package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bearer adds an Authorization header to every request.
type bearer struct {
	token string
	next  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.next.RoundTrip(r)
}

// connectMCP opens an MCP session to the server's /mcp endpoint as the
// client's agent.
func connectMCP(t *testing.T, server *httptest.Server, token string) (*mcp.ClientSession, error) {
	t.Helper()
	// server.Client() is shared with the REST helpers, so wrap a copy.
	httpClient := &http.Client{Transport: bearer{token: token, next: server.Client().Transport}}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             server.URL + "/mcp",
		HTTPClient:           httpClient,
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "connect4-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err == nil {
		t.Cleanup(func() { _ = session.Close() })
	}
	return session, err
}

func mustConnectMCP(t *testing.T, c client) *mcp.ClientSession {
	t.Helper()
	session, err := connectMCP(t, c.server, c.token)
	if err != nil {
		t.Fatalf("connect MCP: %v", err)
	}
	return session
}

// call invokes a tool and decodes its structured result.
func call(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any) (map[string]any, bool) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: protocol error %v (tool failures must be tool-level errors)", tool, err)
	}
	var out map[string]any
	if result.StructuredContent != nil {
		raw, _ := json.Marshal(result.StructuredContent)
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode structured content: %v", tool, err)
		}
	}
	return out, result.IsError
}

func callOK(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any) map[string]any {
	t.Helper()
	out, isError := call(t, session, tool, args)
	if isError {
		t.Fatalf("%s(%v) failed: %v", tool, args, out)
	}
	return out
}

// wantToolError asserts a tool-level error carrying the SPEC.md §16 error object.
func wantToolError(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any, code string) map[string]any {
	t.Helper()
	out, isError := call(t, session, tool, args)
	if !isError {
		t.Fatalf("%s(%v) succeeded, want %s: %v", tool, args, code, out)
	}
	errObject, _ := out["error"].(map[string]any)
	if errObject["code"] != code {
		t.Fatalf("%s error = %v, want code %s", tool, out, code)
	}
	if _, ok := errObject["retryable"].(bool); !ok || errObject["message"] == "" {
		t.Fatalf("%s error lacks message or retryable: %v", tool, errObject)
	}
	return errObject
}

// BDR-021 scenario 1, AC-1: the six game tools, each with an input schema.
func TestMCPListsGameTools(t *testing.T) {
	a := newArena(t)
	session := mustConnectMCP(t, a.red)
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	required := map[string][]string{
		"create_game":      {"mode"},
		"join_game":        {"game_id", "disc", "display_name"},
		"get_game_state":   {"game_id"},
		"make_move":        {"game_id", "column", "expected_version", "idempotency_key"},
		"list_games":       {},
		"get_game_history": {"game_id"},
	}
	seen := map[string]bool{}
	for _, tool := range listed.Tools {
		want, ok := required[tool.Name]
		if !ok {
			continue
		}
		seen[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("%s has no description", tool.Name)
		}
		schema, _ := json.Marshal(tool.InputSchema)
		var parsed struct {
			Type       string         `json:"type"`
			Required   []string       `json:"required"`
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(schema, &parsed); err != nil || parsed.Type != "object" {
			t.Fatalf("%s input schema = %s", tool.Name, schema)
		}
		for _, arg := range want {
			if !slices.Contains(parsed.Required, arg) {
				t.Errorf("%s schema does not require %s: %s", tool.Name, arg, schema)
			}
		}
		// AC-3: identity never comes from tool arguments.
		for name := range parsed.Properties {
			if strings.Contains(name, "player") || strings.Contains(name, "key") && name != "idempotency_key" {
				t.Errorf("%s takes identity-like argument %q", tool.Name, name)
			}
		}
	}
	for name := range required {
		if !seen[name] {
			t.Errorf("tool %s not listed", name)
		}
	}
}

// BDR-021 scenario 2, AC-2..3: play over MCP; the REST view agrees.
func TestMCPMakeMovePlacesDisc(t *testing.T) {
	a := newArena(t)
	session := mustConnectMCP(t, a.red)

	created := callOK(t, session, "create_game", map[string]any{"mode": "agent_vs_agent"})
	a.gameID, _ = created["gameId"].(string)
	if !strings.HasPrefix(a.gameID, "game_") || created["status"] != "waiting" {
		t.Fatalf("create_game = %v", created)
	}
	joined := callOK(t, session, "join_game", map[string]any{"game_id": a.gameID, "disc": "red", "display_name": "Alpha"})
	red := joined["players"].(map[string]any)["red"].(map[string]any)
	if !strings.HasPrefix(red["id"].(string), "agt_") || red["type"] != "agent" {
		t.Fatalf("red player = %v", red)
	}
	a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "yellow", "displayName": "Beta", "type": "agent"}).wantStatus(t, http.StatusOK)

	moved := callOK(t, session, "make_move", map[string]any{
		"game_id": a.gameID, "column": 3, "expected_version": 0, "idempotency_key": "mcp-1",
	})
	move := moved["move"].(map[string]any)
	if number(move["row"]) != 5 || number(move["column"]) != 3 || move["disc"] != "red" || move["playerId"] != red["id"] {
		t.Fatalf("make_move = %v", moved)
	}

	state := callOK(t, session, "get_game_state", map[string]any{"game_id": a.gameID})
	if cell(state, 5, 3) != "red" || number(state["version"]) != 1 || state["currentTurn"] != "yellow" {
		t.Fatalf("get_game_state: cell=%v version=%v turn=%v", cell(state, 5, 3), state["version"], state["currentTurn"])
	}
	rest := a.state(t)
	for _, field := range []string{"board", "version", "moves", "currentTurn", "players"} {
		if !jsonEqual(state[field], rest[field]) {
			t.Fatalf("MCP and REST disagree on %s:\nMCP  %v\nREST %v", field, state[field], rest[field])
		}
	}

	// Retrying with the same idempotency key replays the result (API-008).
	again := callOK(t, session, "make_move", map[string]any{
		"game_id": a.gameID, "column": 3, "expected_version": 0, "idempotency_key": "mcp-1",
	})
	if !jsonEqual(again, moved) {
		t.Fatalf("retry = %v, want %v", again, moved)
	}
}

func jsonEqual(x, y any) bool {
	a, _ := json.Marshal(x)
	b, _ := json.Marshal(y)
	return string(a) == string(b)
}

// BDR-021 scenario 3, AC-4: bad arguments are tool-level errors and change nothing.
func TestMCPInvalidArgumentsAreToolErrors(t *testing.T) {
	a := newArena(t)
	a.start(t)
	session := mustConnectMCP(t, a.red)

	if out, isError := call(t, session, "make_move", map[string]any{
		"game_id": a.gameID, "expected_version": 0, "idempotency_key": "no-column",
	}); !isError {
		t.Fatalf("make_move without column succeeded: %v", out)
	}
	wantToolError(t, session, "make_move", map[string]any{
		"game_id": a.gameID, "column": 9, "expected_version": 0, "idempotency_key": "k9",
	}, "INVALID_COLUMN")
	wantToolError(t, session, "create_game", map[string]any{"mode": "solo"}, "INVALID_ARGUMENT")
	wantToolError(t, session, "list_games", map[string]any{"status": "paused"}, "INVALID_ARGUMENT")
	if out, isError := call(t, session, "join_game", map[string]any{"game_id": a.gameID, "disc": "green", "display_name": "X"}); !isError {
		t.Fatalf("join_game with disc green succeeded: %v", out)
	}

	if state := a.state(t); number(state["version"]) != 0 || len(state["moves"].([]any)) != 0 {
		t.Fatalf("invalid calls changed the game: version=%v moves=%v", state["version"], state["moves"])
	}
}

// MCP-008, BDR-015 AC-2..4: game-rule errors carry the same §16 object as REST.
func TestMCPRuleErrorsMatchREST(t *testing.T) {
	a := newArena(t)
	a.start(t)
	yellow := mustConnectMCP(t, a.yellow)
	outsider := mustConnectMCP(t, a.other)

	errObject := wantToolError(t, yellow, "make_move", map[string]any{
		"game_id": a.gameID, "column": 0, "expected_version": 0, "idempotency_key": "early",
	}, "NOT_YOUR_TURN")
	if errObject["gameId"] != a.gameID || number(errObject["currentVersion"]) != 0 || errObject["retryable"] != true {
		t.Fatalf("NOT_YOUR_TURN = %v", errObject)
	}
	restErr := a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 0, "expectedVersion": 0}).wantError(t, http.StatusConflict, "NOT_YOUR_TURN")
	if !jsonEqual(errObject, restErr) {
		t.Fatalf("MCP and REST errors differ:\nMCP  %v\nREST %v", errObject, restErr)
	}

	wantToolError(t, outsider, "make_move", map[string]any{
		"game_id": a.gameID, "column": 0, "expected_version": 0, "idempotency_key": "x",
	}, "FORBIDDEN")
	wantToolError(t, outsider, "get_game_state", map[string]any{"game_id": "game_missing"}, "GAME_NOT_FOUND")
	wantToolError(t, outsider, "join_game", map[string]any{"game_id": a.gameID, "disc": "red", "display_name": "Gamma"}, "GAME_NOT_JOINABLE")
}

// AC-3, SEC-001: the MCP endpoint requires an issued API key.
func TestMCPRequiresIssuedKey(t *testing.T) {
	a := newArena(t)
	for _, token := range []string{"", "c4_forged_key_that_was_never_issued_by_server"} {
		if _, err := connectMCP(t, a.server, token); err == nil {
			t.Fatalf("connected with token %q, want rejection", token)
		}
	}
	response := a.anon.do(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if response.status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated POST /mcp = %d, want 401", response.status)
	}
}

// MCP-006, MCP-007: list and history tools.
func TestMCPListAndHistory(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 0, 1, 0, 1, 0, 1, 0) // red wins
	waiting := a.create(t)
	session := mustConnectMCP(t, a.other)

	all := callOK(t, session, "list_games", map[string]any{})
	if games := all["games"].([]any); len(games) != 2 {
		t.Fatalf("list_games returned %d games, want 2", len(games))
	}
	filtered := callOK(t, session, "list_games", map[string]any{"status": "waiting"})
	games := filtered["games"].([]any)
	if len(games) != 1 || games[0].(map[string]any)["gameId"] != waiting {
		t.Fatalf("list_games(waiting) = %v", games)
	}

	var completed string
	for _, g := range all["games"].([]any) {
		if g.(map[string]any)["status"] == "completed" {
			completed = g.(map[string]any)["gameId"].(string)
		}
	}
	history := callOK(t, session, "get_game_history", map[string]any{"game_id": completed})
	if history["winner"] != "red" || history["resultReason"] != "connect_four" || len(history["moves"].([]any)) != 7 {
		t.Fatalf("get_game_history = winner %v reason %v moves %d", history["winner"], history["resultReason"], len(history["moves"].([]any)))
	}
	a.gameID = completed
	if rest := a.history(t); !jsonEqual(rest, history) {
		t.Fatalf("MCP and REST history differ:\nMCP  %v\nREST %v", history, rest)
	}

	// SEC-005: no API key appears in any MCP result.
	raw, _ := json.Marshal([]any{all, history})
	for _, c := range []client{a.red, a.yellow, a.other} {
		if strings.Contains(string(raw), c.token) {
			t.Fatal("an API key appears in MCP output")
		}
	}
}
