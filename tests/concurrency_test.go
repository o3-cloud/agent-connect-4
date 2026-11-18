package tests

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
)

func discCount(state map[string]any) int {
	n := 0
	for _, row := range state["board"].([]any) {
		for _, c := range row.([]any) {
			if c != nil {
				n++
			}
		}
	}
	return n
}

// BDR-011 scenario 1, AC-1..2: a stale move is rejected and changes nothing.
func TestAPIStaleMoveIsRejectedWithoutChanges(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 3, 3, 4) // version 3, yellow to move
	before := a.state(t)

	errObject := a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 5, "expectedVersion": 2}).wantError(t, http.StatusConflict, "STALE_STATE")
	if number(errObject["currentVersion"]) != 3 || errObject["gameId"] != a.gameID || errObject["retryable"] != true {
		t.Fatalf("STALE_STATE error = %v, want currentVersion 3, gameId and retryable", errObject)
	}

	after := a.state(t)
	for _, field := range []string{"board", "currentTurn", "version", "moves", "legalMoves"} {
		if !reflect.DeepEqual(before[field], after[field]) {
			t.Fatalf("%s changed by a stale move:\nbefore %v\nafter  %v", field, before[field], after[field])
		}
	}
}

// BDR-011 scenario 2, AC-3..4: simultaneous moves for one turn commit at most one.
func TestAPISimultaneousMovesCommitOne(t *testing.T) {
	for round := range 10 {
		t.Run(fmt.Sprintf("round %d", round), func(t *testing.T) {
			a := newArena(t)
			a.start(t)
			path := "/api/v1/games/" + a.gameID + "/moves"
			bodies := []map[string]any{
				{"column": 2, "expectedVersion": 0, "idempotencyKey": "k-2"},
				{"column": 4, "expectedVersion": 0, "idempotencyKey": "k-4"},
			}
			responses := make([]response, len(bodies))
			var wg sync.WaitGroup
			for i, body := range bodies {
				wg.Add(1)
				go func() {
					defer wg.Done()
					responses[i] = a.red.do(http.MethodPost, path, body)
				}()
			}
			wg.Wait()

			accepted := 0
			for _, r := range responses {
				switch r.status {
				case http.StatusOK:
					accepted++
				case http.StatusConflict:
					code := r.body["error"].(map[string]any)["code"]
					if code != "STALE_STATE" && code != "NOT_YOUR_TURN" {
						t.Fatalf("conflict code = %v, want STALE_STATE or NOT_YOUR_TURN", code)
					}
				default:
					t.Fatalf("unexpected response %d: %s", r.status, r.raw)
				}
			}
			if accepted != 1 {
				t.Fatalf("%d requests accepted, want exactly 1", accepted)
			}
			state := a.state(t)
			if number(state["version"]) != 1 || discCount(state) != 1 || len(state["moves"].([]any)) != 1 {
				t.Fatalf("version=%v discs=%d moves=%d, want 1, 1, 1", state["version"], discCount(state), len(state["moves"].([]any)))
			}
		})
	}
}

// BDR-011 AC-4: both players hammering the same game never desynchronise the
// board from the history.
func TestAPIRaceKeepsBoardAndHistoryInStep(t *testing.T) {
	a := newArena(t)
	a.start(t)
	path := "/api/v1/games/" + a.gameID + "/moves"

	var wg sync.WaitGroup
	for _, c := range []client{a.red, a.yellow} {
		for column := range 7 {
			for version := range 4 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					r := c.do(http.MethodPost, path, map[string]any{"column": column, "expectedVersion": version})
					if r.status != http.StatusOK && r.status != http.StatusConflict {
						t.Errorf("unexpected response %d: %s", r.status, r.raw)
					}
				}()
			}
		}
	}
	wg.Wait()

	state := a.state(t)
	moves := state["moves"].([]any)
	if discCount(state) != len(moves) || number(state["version"]) != len(moves) {
		t.Fatalf("discs=%d moves=%d version=%v, want all equal", discCount(state), len(moves), state["version"])
	}
	for i, m := range moves {
		if number(m.(map[string]any)["moveNumber"]) != i+1 {
			t.Fatalf("move %d has moveNumber %v", i, m.(map[string]any)["moveNumber"])
		}
		wantDisc := "red"
		if i%2 == 1 {
			wantDisc = "yellow"
		}
		if m.(map[string]any)["disc"] != wantDisc {
			t.Fatalf("move %d disc = %v, want %s (turns must alternate)", i+1, m.(map[string]any)["disc"], wantDisc)
		}
	}
}
