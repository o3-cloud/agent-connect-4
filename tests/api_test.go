package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"connect4arena/internal/app"
)

// client is an HTTP client with a fixed test identity (bearer token).
type client struct {
	t      *testing.T
	server *httptest.Server
	token  string
}

type response struct {
	status int
	header http.Header
	body   map[string]any
	raw    string
}

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(app.Handler())
	t.Cleanup(server.Close)
	return server
}

func (c client) do(method, path string, body any) response {
	c.t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	default:
		encoded, err := json.Marshal(b)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	if reader != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.server.Client().Do(request)
	if err != nil {
		c.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := response{status: resp.StatusCode, header: resp.Header, raw: string(raw)}
	if strings.Contains(resp.Header.Get("Content-Type"), "json") {
		if err := json.Unmarshal(raw, &out.body); err != nil {
			c.t.Fatalf("decode %s %s: %v\n%s", method, path, err, raw)
		}
	}
	return out
}

func (r response) wantStatus(t *testing.T, status int) response {
	t.Helper()
	if r.status != status {
		t.Fatalf("status = %d, want %d; body: %s", r.status, status, r.raw)
	}
	return r
}

// wantError asserts the SPEC.md §16 error shape and returns the error object.
func (r response) wantError(t *testing.T, status int, code string) map[string]any {
	t.Helper()
	r.wantStatus(t, status)
	if !strings.HasPrefix(r.header.Get("Content-Type"), "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", r.header.Get("Content-Type"))
	}
	errObject, ok := r.body["error"].(map[string]any)
	if !ok {
		t.Fatalf("body has no error object: %s", r.raw)
	}
	if errObject["code"] != code {
		t.Fatalf("error.code = %v, want %s; body: %s", errObject["code"], code, r.raw)
	}
	if message, _ := errObject["message"].(string); message == "" {
		t.Fatalf("error.message is empty: %s", r.raw)
	}
	if _, ok := errObject["retryable"].(bool); !ok {
		t.Fatalf("error.retryable missing: %s", r.raw)
	}
	return errObject
}

type arena struct {
	server      *httptest.Server
	red, yellow client
	other, anon client
	gameID      string
	movesPlayed int
}

func newArena(t *testing.T) *arena {
	server := newServer(t)
	return &arena{
		server: server,
		red:    register(t, server, "Alpha"),
		yellow: register(t, server, "Beta"),
		other:  register(t, server, "Gamma"),
		anon:   client{t: t, server: server},
	}
}

// register issues an API key through POST /api/v1/agents (ADR-013) and
// returns a client that authenticates with it.
func register(t *testing.T, server *httptest.Server, name string) client {
	t.Helper()
	anon := client{t: t, server: server}
	issued := anon.do(http.MethodPost, "/api/v1/agents", map[string]any{"displayName": name, "type": "agent"}).
		wantStatus(t, http.StatusCreated)
	key, _ := issued.body["apiKey"].(string)
	if !strings.HasPrefix(key, "c4_") {
		t.Fatalf("apiKey = %q, want a c4_ key; body: %s", key, issued.raw)
	}
	return client{t: t, server: server, token: key}
}

func (a *arena) create(t *testing.T) string {
	t.Helper()
	created := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "agent_vs_agent"}).wantStatus(t, http.StatusCreated)
	a.gameID = created.body["gameId"].(string)
	return a.gameID
}

func (a *arena) start(t *testing.T) {
	t.Helper()
	a.create(t)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Alpha", "type": "agent"}).wantStatus(t, http.StatusOK)
	a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "yellow", "displayName": "Beta", "type": "agent"}).wantStatus(t, http.StatusOK)
}

func (a *arena) play(t *testing.T, columns ...int) response {
	t.Helper()
	var last response
	for _, column := range columns {
		mover := a.red
		if a.movesPlayed%2 == 1 {
			mover = a.yellow
		}
		last = mover.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
			map[string]any{"column": column, "expectedVersion": a.movesPlayed}).wantStatus(t, http.StatusOK)
		a.movesPlayed++
	}
	return last
}

func (a *arena) state(t *testing.T) map[string]any {
	t.Helper()
	return a.anon.do(http.MethodGet, "/api/v1/games/"+a.gameID, nil).wantStatus(t, http.StatusOK).body
}

func cell(state map[string]any, row, column int) any {
	return state["board"].([]any)[row].([]any)[column]
}

func number(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func ints(v any) []int {
	out := []int{}
	for _, x := range v.([]any) {
		out = append(out, number(x))
	}
	return out
}

// BDR-001 scenario 1, AC-1..3.
func TestAPICreateGame(t *testing.T) {
	a := newArena(t)
	for _, mode := range []string{"agent_vs_agent", "human_vs_agent"} {
		created := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": mode}).wantStatus(t, http.StatusCreated)
		body := created.body
		if id, _ := body["gameId"].(string); id == "" {
			t.Fatalf("gameId missing: %s", created.raw)
		}
		if body["mode"] != mode || body["status"] != "waiting" || number(body["version"]) != 0 {
			t.Fatalf("created = %s", created.raw)
		}
		board := body["board"].([]any)
		if len(board) != 6 {
			t.Fatalf("board has %d rows, want 6", len(board))
		}
		for r, row := range board {
			cells := row.([]any)
			if len(cells) != 7 {
				t.Fatalf("row %d has %d cells, want 7", r, len(cells))
			}
			for c, value := range cells {
				if value != nil {
					t.Fatalf("board[%d][%d] = %v, want null", r, c, value)
				}
			}
		}
		if got := body["openSlots"]; !reflect.DeepEqual(got, []any{"red", "yellow"}) {
			t.Fatalf("openSlots = %v, want [red yellow]", got)
		}
		if body["currentTurn"] != nil || body["winner"] != nil || len(body["legalMoves"].([]any)) != 0 {
			t.Fatalf("new game has turn/winner/legal moves: %s", created.raw)
		}
	}
}

// BDR-001 scenario 2 (AC-4) and scenario 3 (AC-5).
func TestAPICreateGameRejectsUnknownModeAndIDsAreUnique(t *testing.T) {
	a := newArena(t)
	a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_human"}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
	list := a.anon.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK)
	if games := list.body["games"].([]any); len(games) != 0 {
		t.Fatalf("rejected create listed a game: %s", list.raw)
	}

	first, second := a.create(t), a.create(t)
	if first == second {
		t.Fatalf("two games share id %s", first)
	}
	list = a.anon.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK)
	games := list.body["games"].([]any)
	if len(games) != 2 || games[0].(map[string]any)["gameId"] != first || games[1].(map[string]any)["mode"] != "agent_vs_agent" {
		t.Fatalf("list = %s", list.raw)
	}
}

func TestAPIMutationsRequireCredentials(t *testing.T) {
	a := newArena(t)
	a.anon.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "agent_vs_agent"}).
		wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")

	request, _ := http.NewRequest(http.MethodPost, a.server.URL+"/api/v1/games", strings.NewReader(`{"mode":"agent_vs_agent"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	resp, err := a.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Basic auth status = %d, want 401", resp.StatusCode)
	}
}

// BDR-002 scenarios 1-3, AC-1..5.
func TestAPIJoinAndStartGame(t *testing.T) {
	a := newArena(t)
	a.create(t)
	path := "/api/v1/games/" + a.gameID + "/join"

	joined := a.red.do(http.MethodPost, path, map[string]any{"disc": "red", "displayName": "Alpha", "type": "agent"}).
		wantStatus(t, http.StatusOK)
	if joined.body["status"] != "waiting" || joined.body["currentTurn"] != nil {
		t.Fatalf("after first join: %s", joined.raw)
	}
	state := a.state(t)
	red := state["players"].(map[string]any)["red"].(map[string]any)
	if red["displayName"] != "Alpha" || red["disc"] != "red" || red["type"] != "agent" {
		t.Fatalf("red player = %v", red)
	}
	if id, _ := red["id"].(string); !strings.HasPrefix(id, "agt_") || strings.Contains(id, a.red.token) {
		t.Fatalf("red player id = %q; want the registered agent id, which does not reveal the credential", id)
	}
	if state["status"] != "waiting" || state["currentTurn"] != nil {
		t.Fatalf("waiting game state: status %v turn %v", state["status"], state["currentTurn"])
	}

	a.other.do(http.MethodPost, path, map[string]any{"disc": "red", "displayName": "Gamma", "type": "agent"}).
		wantError(t, http.StatusConflict, "SLOT_TAKEN")
	a.red.do(http.MethodPost, path, map[string]any{"disc": "yellow", "displayName": "Alpha", "type": "agent"}).
		wantError(t, http.StatusConflict, "ALREADY_JOINED")
	a.other.do(http.MethodPost, path, map[string]any{"disc": "yellow", "displayName": "Gamma", "type": "agent", "playerId": red["id"]}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
	a.other.do(http.MethodPost, path, map[string]any{"disc": "green", "displayName": "Gamma", "type": "agent"}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")

	started := a.yellow.do(http.MethodPost, path, map[string]any{"disc": "yellow", "displayName": "Beta", "type": "agent"}).
		wantStatus(t, http.StatusOK)
	if started.body["status"] != "in_progress" || started.body["currentTurn"] != "red" {
		t.Fatalf("after second join: %s", started.raw)
	}
	state = a.state(t)
	players := state["players"].(map[string]any)
	if players["red"].(map[string]any)["displayName"] != "Alpha" || players["yellow"].(map[string]any)["displayName"] != "Beta" {
		t.Fatalf("players = %v", players)
	}
	if !reflect.DeepEqual(ints(state["legalMoves"]), []int{0, 1, 2, 3, 4, 5, 6}) || len(state["openSlots"].([]any)) != 0 {
		t.Fatalf("started state: legal %v open %v", state["legalMoves"], state["openSlots"])
	}

	a.other.do(http.MethodPost, path, map[string]any{"disc": "yellow", "displayName": "Gamma", "type": "agent"}).
		wantError(t, http.StatusConflict, "GAME_NOT_JOINABLE")
	a.other.do(http.MethodPost, "/api/v1/games/game_missing/join", map[string]any{"disc": "red", "displayName": "Gamma", "type": "agent"}).
		wantError(t, http.StatusNotFound, "GAME_NOT_FOUND")
}

// BDR-003 scenarios 1-3, AC-1..5.
func TestAPIReadGameState(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 3)

	state := a.yellow.do(http.MethodGet, "/api/v1/games/"+a.gameID, nil).wantStatus(t, http.StatusOK).body
	for _, key := range []string{"gameId", "board", "players", "currentTurn", "status", "legalMoves", "version", "winner", "moveNumber"} {
		if _, ok := state[key]; !ok {
			t.Fatalf("state is missing %q", key)
		}
	}
	if state["gameId"] != a.gameID || state["currentTurn"] != "yellow" || number(state["version"]) != 1 || number(state["moveNumber"]) != 1 {
		t.Fatalf("state = %v", state)
	}
	if !reflect.DeepEqual(ints(state["legalMoves"]), []int{0, 1, 2, 3, 4, 5, 6}) {
		t.Fatalf("legalMoves = %v", state["legalMoves"])
	}
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			want := any(nil)
			if r == 5 && c == 3 {
				want = "red"
			}
			if got := cell(state, r, c); got != want {
				t.Fatalf("board[%d][%d] = %v, want %v", r, c, got, want)
			}
		}
	}

	missing := a.anon.do(http.MethodGet, "/api/v1/games/game_missing", nil).wantError(t, http.StatusNotFound, "GAME_NOT_FOUND")
	if missing["gameId"] != "game_missing" || missing["retryable"] != false {
		t.Fatalf("error = %v", missing)
	}
}

func TestAPIFullColumnIsNotLegal(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 0, 0, 0, 0, 0, 0)
	if got := ints(a.state(t)["legalMoves"]); !reflect.DeepEqual(got, []int{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("legalMoves = %v, want [1..6]", got)
	}
}

// BDR-005 scenarios 1-2, AC-1..4.
func TestAPISubmitLegalMove(t *testing.T) {
	a := newArena(t)
	a.start(t)
	first := a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 4, "expectedVersion": 0, "idempotencyKey": "m1"}).wantStatus(t, http.StatusOK).body
	move := first["move"].(map[string]any)
	if first["accepted"] != true || first["gameId"] != a.gameID || number(move["moveNumber"]) != 1 || number(move["column"]) != 4 ||
		number(move["row"]) != 5 || move["disc"] != "red" || first["nextTurn"] != "yellow" || first["status"] != "in_progress" ||
		number(first["version"]) != 1 {
		t.Fatalf("move response = %v", first)
	}
	if cell(a.state(t), 5, 4) != "red" {
		t.Fatal("board[5][4] is not red")
	}

	second := a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 4, "expectedVersion": 1}).wantStatus(t, http.StatusOK).body
	if number(second["move"].(map[string]any)["row"]) != 4 || second["nextTurn"] != "red" || number(second["version"]) != 2 {
		t.Fatalf("second move response = %v", second)
	}
}

// BDR-006 scenarios 1-2, AC-1..4; TURN-006 stale state.
func TestAPIRejectInvalidMoves(t *testing.T) {
	a := newArena(t)
	a.start(t)
	path := "/api/v1/games/" + a.gameID + "/moves"
	before := a.state(t)

	for _, body := range []string{
		`{"column": 7, "expectedVersion": 0}`,
		`{"column": -1, "expectedVersion": 0}`,
		`{"column": 3.5, "expectedVersion": 0}`,
		`{"column": "three", "expectedVersion": 0}`,
	} {
		// SPEC.md §16: game-rule errors carry gameId and currentVersion, whether the
		// column fails schema validation (3.5, "three") or the range check (7, -1).
		ruleErr := a.red.do(http.MethodPost, path, body).wantError(t, http.StatusUnprocessableEntity, "INVALID_COLUMN")
		if _, ok := ruleErr["currentVersion"]; !ok || number(ruleErr["currentVersion"]) != 0 || ruleErr["gameId"] != a.gameID {
			t.Fatalf("INVALID_COLUMN context for %s = %v", body, ruleErr)
		}
	}
	a.red.do(http.MethodPost, "/api/v1/games/game_missing/moves", `{"column": 3.5, "expectedVersion": 0}`).
		wantError(t, http.StatusNotFound, "GAME_NOT_FOUND")

	stale := a.red.do(http.MethodPost, path, `{"column": 3, "expectedVersion": 4}`).wantError(t, http.StatusConflict, "STALE_STATE")
	if number(stale["currentVersion"]) != 0 || stale["retryable"] != true {
		t.Fatalf("STALE_STATE = %v", stale)
	}
	a.red.do(http.MethodPost, path, `{"column": 3}`).wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
	a.red.do(http.MethodPost, path, `{"column": 3, "expectedVersion": 0.5}`).wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
	a.red.do(http.MethodPost, path, `not json`).wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")

	after := a.state(t)
	if !reflect.DeepEqual(after["board"], before["board"]) || after["currentTurn"] != "red" || number(after["version"]) != 0 {
		t.Fatalf("rejected moves changed the game: %v", after)
	}

	a.play(t, 0, 0, 0, 0, 0, 0)
	full := a.red.do(http.MethodPost, path, map[string]any{"column": 0, "expectedVersion": 6}).wantError(t, http.StatusUnprocessableEntity, "COLUMN_FULL")
	if number(full["currentVersion"]) != 6 || full["retryable"] != false {
		t.Fatalf("COLUMN_FULL = %v", full)
	}
	state := a.state(t)
	if state["currentTurn"] != "red" || number(state["version"]) != 6 {
		t.Fatalf("state after full-column rejection = %v", state)
	}
}

// BDR-007 scenarios 1-3, AC-1..5.
func TestAPIEnforceTurnAndMembership(t *testing.T) {
	a := newArena(t)
	a.start(t)
	path := "/api/v1/games/" + a.gameID + "/moves"
	redID := a.state(t)["players"].(map[string]any)["red"].(map[string]any)["id"]

	outOfTurn := a.yellow.do(http.MethodPost, path, map[string]any{"column": 3, "expectedVersion": 0}).
		wantError(t, http.StatusConflict, "NOT_YOUR_TURN")
	if number(outOfTurn["currentVersion"]) != 0 || outOfTurn["retryable"] != true {
		t.Fatalf("NOT_YOUR_TURN = %v", outOfTurn)
	}
	a.other.do(http.MethodPost, path, map[string]any{"column": 3, "expectedVersion": 0}).wantError(t, http.StatusForbidden, "FORBIDDEN")
	a.anon.do(http.MethodPost, path, map[string]any{"column": 3, "expectedVersion": 0}).wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")
	a.yellow.do(http.MethodPost, path, map[string]any{"column": 3, "expectedVersion": 0, "playerId": redID}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")

	state := a.state(t)
	if cell(state, 5, 3) != nil || state["currentTurn"] != "red" || number(state["version"]) != 0 {
		t.Fatalf("rejected moves changed the game: %v", state)
	}
}

func TestAPIMoveBeforeGameStarts(t *testing.T) {
	a := newArena(t)
	a.create(t)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Alpha", "type": "agent"}).wantStatus(t, http.StatusOK)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", map[string]any{"column": 3, "expectedVersion": 0}).
		wantError(t, http.StatusConflict, "GAME_NOT_STARTED")
	a.red.do(http.MethodPost, "/api/v1/games/game_missing/moves", map[string]any{"column": 3, "expectedVersion": 0}).
		wantError(t, http.StatusNotFound, "GAME_NOT_FOUND")
}

// Walking skeleton end to end: two HTTP clients play a full game (BDR-008 scenarios 1 and 4).
func TestAPITwoAgentsPlayAFullGame(t *testing.T) {
	a := newArena(t)
	a.start(t)
	final := a.play(t, 0, 0, 1, 1, 2, 2, 3).body
	if final["status"] != "completed" || final["winner"] != "red" || final["resultReason"] != "connect_four" ||
		final["nextTurn"] != nil || number(final["version"]) != 7 {
		t.Fatalf("winning move response = %v", final)
	}

	state := a.state(t)
	if state["status"] != "completed" || state["winner"] != "red" || state["resultReason"] != "connect_four" ||
		state["currentTurn"] != nil || len(state["legalMoves"].([]any)) != 0 || len(state["moves"].([]any)) != 7 {
		t.Fatalf("completed state = %v", state)
	}

	completed := a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", map[string]any{"column": 4, "expectedVersion": 7}).
		wantError(t, http.StatusConflict, "GAME_COMPLETED")
	if number(completed["currentVersion"]) != 7 {
		t.Fatalf("GAME_COMPLETED = %v", completed)
	}
	if number(a.state(t)["version"]) != 7 {
		t.Fatal("move after completion changed the version")
	}
}

func TestAPIUnknownRoutesReturnStructuredNotFound(t *testing.T) {
	a := newArena(t)
	for _, path := range []string{"/api/v1/nope", "/api/v1/games/" + "x/unknown", "/api/v2/games"} {
		got := a.anon.do(http.MethodGet, path, nil)
		if strings.HasPrefix(path, "/api/v1/") {
			got.wantError(t, http.StatusNotFound, "ROUTE_NOT_FOUND")
		} else {
			got.wantStatus(t, http.StatusNotFound)
		}
	}
	a.anon.do(http.MethodDelete, "/api/v1/games", nil).wantError(t, http.StatusNotFound, "ROUTE_NOT_FOUND")
}

// NFR-005: the OpenAPI document is generated and lists the error codes.
func TestAPIServesOpenAPIDocument(t *testing.T) {
	a := newArena(t)
	doc := a.anon.do(http.MethodGet, "/api/v1/openapi.json", nil).wantStatus(t, http.StatusOK)
	paths, _ := doc.body["paths"].(map[string]any)
	for _, path := range []string{"/api/v1/games", "/api/v1/games/{gameId}", "/api/v1/games/{gameId}/join", "/api/v1/games/{gameId}/moves"} {
		if _, ok := paths[path]; !ok {
			t.Errorf("OpenAPI document is missing %s", path)
		}
	}
	for _, code := range []string{"NOT_YOUR_TURN", "STALE_STATE", "INVALID_COLUMN", "GAME_COMPLETED", "ROUTE_NOT_FOUND"} {
		if !strings.Contains(doc.raw, `"`+code+`"`) {
			t.Errorf("OpenAPI document does not list error code %s", code)
		}
	}
}

// BDR-009 scenario 1, AC-1..3.
func TestAPIFullBoardIsADraw(t *testing.T) {
	a := newArena(t)
	a.start(t)
	final := a.play(t, 0, 1, 0, 1, 0, 1, 2, 3, 2, 3, 2, 3, 4, 5, 4, 5, 4, 5, 6, 0, 6, 0, 6, 0,
		1, 2, 1, 2, 1, 2, 3, 4, 3, 4, 3, 4, 5, 6, 5, 6, 5, 6).body
	if final["status"] != "completed" || final["winner"] != nil || final["resultReason"] != "draw" {
		t.Fatalf("final move response = %v", final)
	}
	state := a.state(t)
	if len(state["legalMoves"].([]any)) != 0 || len(state["moves"].([]any)) != 42 || number(state["moveNumber"]) != 42 {
		t.Fatalf("drawn state: legal %v, %d moves", state["legalMoves"], len(state["moves"].([]any)))
	}
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", map[string]any{"column": 0, "expectedVersion": 42}).
		wantError(t, http.StatusConflict, "GAME_COMPLETED")
}

// BDR-004 scenarios 1-2, AC-1..4.
func TestAPIListGamesFiltersByStatus(t *testing.T) {
	a := newArena(t)
	waitingID := a.create(t)
	a.start(t)
	inProgressID := a.gameID
	a.start(t)
	a.movesPlayed = 0
	a.play(t, 0, 0, 1, 1, 2, 2, 3)
	completedID := a.gameID

	for status, want := range map[string]string{"waiting": waitingID, "in_progress": inProgressID, "completed": completedID} {
		list := a.anon.do(http.MethodGet, "/api/v1/games?status="+status, nil).wantStatus(t, http.StatusOK)
		games := list.body["games"].([]any)
		if len(games) != 1 {
			t.Fatalf("status=%s returned %d games: %s", status, len(games), list.raw)
		}
		entry := games[0].(map[string]any)
		if entry["gameId"] != want || entry["status"] != status || entry["mode"] != "agent_vs_agent" {
			t.Fatalf("status=%s entry = %v, want %s", status, entry, want)
		}
		if _, ok := entry["players"].(map[string]any); !ok {
			t.Fatalf("status=%s entry has no players: %v", status, entry)
		}
	}

	all := a.anon.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK)
	if games := all.body["games"].([]any); len(games) != 3 {
		t.Fatalf("unfiltered list has %d games, want 3", len(games))
	}
	for _, token := range []string{a.red.token, a.yellow.token} {
		if strings.Contains(all.raw, token) {
			t.Fatalf("game list leaks credential %q", token)
		}
	}

	a.anon.do(http.MethodGet, "/api/v1/games?status=paused", nil).wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
}
