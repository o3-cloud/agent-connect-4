package tests

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func (a *arena) history(t *testing.T) map[string]any {
	t.Helper()
	return a.anon.do(http.MethodGet, "/api/v1/games/"+a.gameID+"/moves", nil).wantStatus(t, http.StatusOK).body
}

// BDR-016 scenario 1, AC-1..3: accepted moves only, in order, fully described.
func TestAPIMoveHistoryListsAcceptedMovesInOrder(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 3, 3)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 9, "expectedVersion": a.movesPlayed}).wantError(t, http.StatusUnprocessableEntity, "INVALID_COLUMN")
	a.play(t, 4)

	history := a.history(t)
	if history["gameId"] != a.gameID || history["status"] != "in_progress" {
		t.Fatalf("history header: gameId=%v status=%v", history["gameId"], history["status"])
	}
	if history["winner"] != nil || history["resultReason"] != nil {
		t.Fatalf("in-progress history has a result: winner=%v reason=%v", history["winner"], history["resultReason"])
	}
	if number(history["version"]) != 3 {
		t.Fatalf("version = %v, want 3", history["version"])
	}

	moves, _ := history["moves"].([]any)
	if len(moves) != 3 {
		t.Fatalf("got %d moves, want 3: %v", len(moves), history["moves"])
	}
	state := a.state(t)
	players := state["players"].(map[string]any)
	redID := players["red"].(map[string]any)["id"]
	yellowID := players["yellow"].(map[string]any)["id"]

	want := []struct {
		column, row int
		disc        string
		player      any
	}{{3, 5, "red", redID}, {3, 4, "yellow", yellowID}, {4, 5, "red", redID}}
	var previous time.Time
	for i, w := range want {
		m := moves[i].(map[string]any)
		if number(m["moveNumber"]) != i+1 || number(m["column"]) != w.column || number(m["row"]) != w.row ||
			m["disc"] != w.disc || m["playerId"] != w.player {
			t.Fatalf("move %d = %v, want number %d column %d row %d disc %s player %v", i, m, i+1, w.column, w.row, w.disc, w.player)
		}
		stamp, err := time.Parse(time.RFC3339Nano, m["timestamp"].(string))
		if err != nil {
			t.Fatalf("move %d timestamp: %v", i, err)
		}
		if stamp.Before(previous) {
			t.Fatalf("move %d timestamp %v is before the previous move's %v", i, stamp, previous)
		}
		previous = stamp
	}
	if !reflect.DeepEqual(moves, state["moves"]) {
		t.Fatalf("history moves differ from game state moves:\n%v\n%v", moves, state["moves"])
	}
}

// BDR-016 AC-4: a completed game's history carries the result.
func TestAPIMoveHistoryOfCompletedGameIncludesResult(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 0, 1, 0, 1, 0, 1, 0) // red wins vertically in column 0

	history := a.history(t)
	if history["status"] != "completed" || history["winner"] != "red" || history["resultReason"] != "connect_four" {
		t.Fatalf("completed history: status=%v winner=%v reason=%v", history["status"], history["winner"], history["resultReason"])
	}
	if moves := history["moves"].([]any); len(moves) != 7 {
		t.Fatalf("got %d moves, want 7", len(moves))
	}
}

func TestAPIMoveHistoryEdgeCases(t *testing.T) {
	a := newArena(t)
	a.create(t)
	history := a.history(t)
	if moves, ok := history["moves"].([]any); !ok || len(moves) != 0 {
		t.Fatalf("new game moves = %v, want empty array", history["moves"])
	}
	if history["status"] != "waiting" {
		t.Fatalf("status = %v, want waiting", history["status"])
	}

	a.anon.do(http.MethodGet, "/api/v1/games/game_missing/moves", nil).wantError(t, http.StatusNotFound, "GAME_NOT_FOUND")

	// SEC-005: credentials never appear in history.
	raw := a.anon.do(http.MethodGet, "/api/v1/games/"+a.gameID+"/moves", nil).raw
	for _, c := range []client{a.red, a.yellow} {
		if strings.Contains(raw, c.token) {
			t.Fatalf("history exposes an API key: %s", raw)
		}
	}
}
