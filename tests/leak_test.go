package tests

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

// BDR-013 AC-4 (SEC-005): the credential values used by both agents never
// appear in game state, move history, game lists, the updates a spectator
// receives (connect4 watch, the browser board), or CLI and MCP output.
func TestCredentialsNeverLeakToAnySurface(t *testing.T) {
	a := newArena(t)
	a.start(t)
	keys := []string{a.red.token, a.yellow.token}

	// A spectator follows the game live from the start.
	watch := startWatch(t, a.server, a.gameID, "--json")
	var outputs []string
	outputs = append(outputs, watch.next())

	// Play to a red win: red takes the bottom row.
	a.play(t, 0, 0, 1, 1, 2, 2, 3)
	for line := range watch.lines {
		outputs = append(outputs, line)
	}
	if code := watch.exitCode(); code != 0 {
		t.Fatalf("watch exited %d", code)
	}

	path := "/api/v1/games/" + a.gameID
	for _, r := range []response{
		a.anon.do(http.MethodGet, path, nil),
		a.anon.do(http.MethodGet, path+"/moves", nil),
		a.anon.do(http.MethodGet, "/api/v1/games", nil),
		a.red.do(http.MethodPost, path+"/moves", map[string]any{"column": 4, "expectedVersion": 7}), // GAME_COMPLETED
	} {
		outputs = append(outputs, r.raw)
	}

	// MCP output for both players.
	for _, player := range []client{a.red, a.yellow} {
		session := mustConnectMCP(t, player)
		for _, c := range []struct {
			tool string
			args map[string]any
		}{
			{"get_game_state", map[string]any{"gameId": a.gameID}},
			{"get_game_history", map[string]any{"gameId": a.gameID}},
			{"list_games", map[string]any{}},
			{"make_move", map[string]any{"gameId": a.gameID, "column": 4, "expectedVersion": 7}},
		} {
			result, _ := call(t, session, c.tool, c.args)
			raw, _ := json.Marshal(result)
			outputs = append(outputs, string(raw))
		}
	}

	// The browser board a spectator sees, and the CLI's own state and history.
	jar, _ := cookiejar.New(nil)
	spectator := &browser{t: t, base: a.server.URL, client: &http.Client{Jar: jar}}
	for _, p := range []string{"/games/" + a.gameID, "/games/" + a.gameID + "/board"} {
		_, page := spectator.get(p)
		outputs = append(outputs, page)
	}
	for _, args := range [][]string{{"state", a.gameID}, {"history", a.gameID}, {"state", a.gameID, "--json"}} {
		_, stdout, stderr := cliRun(t, a.server, a.red.token, args...)
		outputs = append(outputs, stdout, stderr)
	}

	if len(outputs) < 20 {
		t.Fatalf("collected only %d outputs", len(outputs))
	}
	for i, out := range outputs {
		for _, key := range keys {
			if strings.Contains(out, key) {
				t.Fatalf("output %d contains an API key: %s", i, out)
			}
		}
	}
}
