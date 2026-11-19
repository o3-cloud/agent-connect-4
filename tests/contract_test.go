package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// iface plays and reads games through one public interface as one agent
// (BDR-023, SPEC.md AT-002, AT-008, NFR-004).
type iface interface {
	name() string
	// move submits a move; on failure it returns the §16 error object.
	move(gameID string, column, expectedVersion int, key string) (result, errObject map[string]any)
	state(gameID string) map[string]any
}

type restIface struct{ c client }

func (r restIface) name() string { return "REST" }

func (r restIface) move(gameID string, column, version int, key string) (map[string]any, map[string]any) {
	body := map[string]any{"column": column, "expectedVersion": version}
	if key != "" {
		body["idempotencyKey"] = key
	}
	resp := r.c.do(http.MethodPost, "/api/v1/games/"+gameID+"/moves", body)
	if resp.status != http.StatusOK {
		errObject, _ := resp.body["error"].(map[string]any)
		return nil, errObject
	}
	return resp.body, nil
}

func (r restIface) state(gameID string) map[string]any {
	return r.c.do(http.MethodGet, "/api/v1/games/"+gameID, nil).body
}

type cliIface struct {
	t      *testing.T
	server *httptest.Server
	key    string
}

func (c cliIface) name() string { return "CLI" }

func (c cliIface) move(gameID string, column, version int, key string) (map[string]any, map[string]any) {
	args := []string{"move", gameID, "--column", strconv.Itoa(column), "--expected-version", strconv.Itoa(version)}
	if key != "" {
		args = append(args, "--idempotency-key", key)
	}
	code, doc := cliJSON(c.t, c.server, c.key, args...)
	if errObject, ok := doc["error"].(map[string]any); ok {
		// CLI-007: the exit status is the §16 code's.
		if want := cliExitCodes[errObject["code"].(string)]; code != want {
			c.t.Fatalf("CLI exit %d for %v, want %d", code, errObject["code"], want)
		}
		return nil, errObject
	}
	if code != 0 {
		c.t.Fatalf("CLI move exit %d without an error object: %v", code, doc)
	}
	return doc, nil
}

func (c cliIface) state(gameID string) map[string]any {
	_, doc := cliJSON(c.t, c.server, c.key, "state", gameID)
	return doc
}

// cliExitCodes is the Exit column of SPEC.md §16.
var cliExitCodes = map[string]int{
	"INVALID_ARGUMENT": 2, "INVALID_COLUMN": 3, "COLUMN_FULL": 3, "UNAUTHENTICATED": 4, "FORBIDDEN": 4,
	"GAME_NOT_FOUND": 5, "NOT_YOUR_TURN": 6, "STALE_STATE": 6, "GAME_NOT_STARTED": 6, "GAME_COMPLETED": 7,
}

type mcpIface struct {
	t       *testing.T
	session *mcp.ClientSession
}

func (m mcpIface) name() string { return "MCP" }

func (m mcpIface) move(gameID string, column, version int, key string) (map[string]any, map[string]any) {
	if key == "" {
		key = fmt.Sprintf("auto-%s-%d-%d", gameID, column, version)
	}
	out, isError := call(m.t, m.session, "make_move", map[string]any{
		"game_id": gameID, "column": column, "expected_version": version, "idempotency_key": key,
	})
	if isError {
		errObject, _ := out["error"].(map[string]any)
		return nil, errObject
	}
	return out, nil
}

func (m mcpIface) state(gameID string) map[string]any {
	return callOK(m.t, m.session, "get_game_state", map[string]any{"game_id": gameID})
}

// interfacesFor returns one agent's three interfaces, all using its key.
func interfacesFor(t *testing.T, c client) []iface {
	return []iface{
		restIface{c: c},
		cliIface{t: t, server: c.server, key: c.token},
		mcpIface{t: t, session: mustConnectMCP(t, c)},
	}
}

// startedGameFor creates a new in-progress game on a's server with a.red as
// red and a.yellow as yellow, and returns its id.
func startedGameFor(t *testing.T, a *arena) string {
	t.Helper()
	saved := a.gameID
	a.start(t)
	id := a.gameID
	a.gameID = saved
	return id
}

// stable is the part of a game view that must match across interfaces; ids
// and timestamps differ between games by design.
func stable(state map[string]any) map[string]any {
	moves := []any{}
	for _, m := range state["moves"].([]any) {
		move := m.(map[string]any)
		moves = append(moves, []any{move["moveNumber"], move["disc"], move["column"], move["row"]})
	}
	return map[string]any{
		"board": state["board"], "status": state["status"], "winner": state["winner"],
		"resultReason": state["resultReason"], "version": state["version"],
		"currentTurn": state["currentTurn"], "legalMoves": state["legalMoves"], "moves": moves,
	}
}

// BDR-023 scenario 1, AC-1: the same sequence gives the same game everywhere.
func TestContractSameSequenceSameGame(t *testing.T) {
	a := newArena(t)
	red, yellow := interfacesFor(t, a.red), interfacesFor(t, a.yellow)
	sequence := []int{0, 0, 1, 1, 2, 2, 3}

	var reference map[string]any
	for i := range red {
		id := startedGameFor(t, a)
		for n, column := range sequence {
			mover := red[i]
			if n%2 == 1 {
				mover = yellow[i]
			}
			if _, errObject := mover.move(id, column, n, ""); errObject != nil {
				t.Fatalf("%s move %d: %v", mover.name(), n+1, errObject)
			}
		}
		final := stable(a.stateOf(t, id))
		if final["status"] != "completed" || final["winner"] != "red" || final["resultReason"] != "connect_four" || number(final["version"]) != 7 {
			t.Fatalf("%s game ended %v", red[i].name(), final)
		}
		if reference == nil {
			reference = final
		} else if !jsonEqual(final, reference) {
			t.Fatalf("%s game differs from REST:\n%v\n%v", red[i].name(), final, reference)
		}
	}
}

// BDR-023 AC-1, BDR-015 AC-3: each rule violation gives the same error
// object through every interface.
func TestContractSameViolationSameError(t *testing.T) {
	a := newArena(t)
	red, yellow, other := interfacesFor(t, a.red), interfacesFor(t, a.yellow), interfacesFor(t, a.other)

	violations := []struct {
		name   string
		setup  []int // accepted columns before the violation
		by     func(i int) iface
		column int
		stale  bool
		code   string
	}{
		{"not your turn", nil, func(i int) iface { return yellow[i] }, 0, false, "NOT_YOUR_TURN"},
		{"stale version", []int{3}, func(i int) iface { return yellow[i] }, 0, true, "STALE_STATE"},
		{"column out of range", nil, func(i int) iface { return red[i] }, 9, false, "INVALID_COLUMN"},
		{"full column", []int{0, 0, 0, 0, 0, 0}, func(i int) iface { return red[i] }, 0, false, "COLUMN_FULL"},
		{"game over", []int{0, 1, 0, 1, 0, 1, 0}, func(i int) iface { return yellow[i] }, 2, false, "GAME_COMPLETED"},
		{"outsider", nil, func(i int) iface { return other[i] }, 0, false, "FORBIDDEN"},
	}
	for _, v := range violations {
		t.Run(v.name, func(t *testing.T) {
			var reference map[string]any
			for i := range red {
				id := startedGameFor(t, a)
				for n, column := range v.setup {
					mover := red[0]
					if n%2 == 1 {
						mover = yellow[0]
					}
					if _, errObject := mover.move(id, column, n, ""); errObject != nil {
						t.Fatalf("setup move %d: %v", n+1, errObject)
					}
				}
				version := len(v.setup)
				if v.stale {
					version--
				}
				before := stable(a.stateOf(t, id))
				result, errObject := v.by(i).move(id, v.column, version, "violation")
				if errObject == nil {
					t.Fatalf("%s accepted the move: %v", red[i].name(), result)
				}
				if errObject["code"] != v.code || errObject["gameId"] != id {
					t.Fatalf("%s error = %v, want %s for %s", red[i].name(), errObject, v.code, id)
				}
				if after := stable(a.stateOf(t, id)); !jsonEqual(before, after) {
					t.Fatalf("%s: a rejected move changed the game", red[i].name())
				}
				comparable := map[string]any{"code": errObject["code"], "message": errObject["message"],
					"retryable": errObject["retryable"], "currentVersion": errObject["currentVersion"]}
				if reference == nil {
					reference = comparable
				} else if !jsonEqual(comparable, reference) {
					t.Fatalf("%s error differs from REST:\n%v\n%v", red[i].name(), comparable, reference)
				}
			}
		})
	}

	// GAME_NOT_FOUND has no game context beyond the id.
	for _, x := range red {
		if _, errObject := x.move("game_missing", 0, 0, ""); errObject == nil || errObject["code"] != "GAME_NOT_FOUND" {
			t.Fatalf("%s unknown game: %v", x.name(), errObject)
		}
	}
}

// BDR-023 AC-2: a move through one interface is visible at once through all.
func TestContractMovesVisibleEverywhere(t *testing.T) {
	a := newArena(t)
	red, yellow := interfacesFor(t, a.red), interfacesFor(t, a.yellow)
	readers := interfacesFor(t, a.other)
	id := startedGameFor(t, a)

	// Each move goes through a different interface, rotating REST, CLI, MCP.
	for n, column := range []int{3, 4, 3, 4, 2, 5} {
		mover := red[n%3]
		if n%2 == 1 {
			mover = yellow[n%3]
		}
		if _, errObject := mover.move(id, column, n, ""); errObject != nil {
			t.Fatalf("%s move %d: %v", mover.name(), n+1, errObject)
		}
		var reference map[string]any
		for _, r := range readers {
			view := stable(r.state(id))
			if number(view["version"]) != n+1 {
				t.Fatalf("after %s move %d, %s shows version %v", mover.name(), n+1, r.name(), view["version"])
			}
			if reference == nil {
				reference = view
			} else if !jsonEqual(view, reference) {
				t.Fatalf("%s view differs from REST after move %d", r.name(), n+1)
			}
		}
	}
}

// BDR-023 scenario 2, AC-3: one credential, CLI then MCP, one player.
func TestContractSwitchingInterfacesKeepsIdentity(t *testing.T) {
	a := newArena(t)
	red := interfacesFor(t, a.red)
	id := startedGameFor(t, a)

	if _, errObject := red[1].move(id, 3, 0, ""); errObject != nil { // CLI
		t.Fatalf("CLI move: %v", errObject)
	}
	if _, errObject := (restIface{c: a.yellow}).move(id, 4, 1, ""); errObject != nil {
		t.Fatalf("yellow move: %v", errObject)
	}
	if _, errObject := red[2].move(id, 3, 2, "mcp-second"); errObject != nil { // MCP
		t.Fatalf("MCP move: %v", errObject)
	}

	history := a.anon.do(http.MethodGet, "/api/v1/games/"+id+"/moves", nil).wantStatus(t, http.StatusOK).body
	moves := history["moves"].([]any)
	first, third := moves[0].(map[string]any), moves[2].(map[string]any)
	redID := history["players"].(map[string]any)["red"].(map[string]any)["id"]
	if first["playerId"] != redID || third["playerId"] != redID {
		t.Fatalf("moves attributed to %v and %v, want both %v", first["playerId"], third["playerId"], redID)
	}
}

// state reads a game through REST without credentials.
func (a *arena) stateOf(t *testing.T, id string) map[string]any {
	t.Helper()
	return a.anon.do(http.MethodGet, "/api/v1/games/"+id, nil).wantStatus(t, http.StatusOK).body
}
