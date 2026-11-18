package tests

import (
	"net/http"
	"reflect"
	"sync"
	"testing"

	"connect4arena/internal/game"
)

// BDR-012 AC-1..3: a retry with the same key replays the recorded result.
func TestMoveRetryWithSameKeyReplaysResult(t *testing.T) {
	service, started := startedGame(t)
	first, err := service.MoveWithKey(started.ID, alice, 3, 0, "k-1")
	if err != nil {
		t.Fatalf("first move: %v", err)
	}

	retry, err := service.MoveWithKey(started.ID, alice, 3, 0, "k-1")
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !reflect.DeepEqual(retry, first) {
		t.Fatalf("retry result = %+v, want %+v", retry, first)
	}
	g, _ := service.Get(started.ID)
	if g.Version != 1 || len(g.Moves) != 1 || g.CurrentTurn != "yellow" {
		t.Fatalf("after retry: version=%d moves=%d turn=%s, want 1, 1, yellow", g.Version, len(g.Moves), g.CurrentTurn)
	}

	// AC-3: the retry still replays after the game has moved on.
	if _, err := service.MoveWithKey(started.ID, bob, 3, 1, "k-1"); err != nil {
		t.Fatalf("yellow move: %v", err)
	}
	late, err := service.MoveWithKey(started.ID, alice, 3, 0, "k-1")
	if err != nil {
		t.Fatalf("late retry: %v", err)
	}
	if !reflect.DeepEqual(late, first) {
		t.Fatalf("late retry = %+v, want %+v", late, first)
	}
	if g, _ := service.Get(started.ID); g.Version != 2 || len(g.Moves) != 2 {
		t.Fatalf("late retry changed the game: version=%d moves=%d", g.Version, len(g.Moves))
	}
}

// BDR-012 AC-4: keys are scoped to one player in one game.
func TestIdempotencyKeyIsScopedToPlayerAndGame(t *testing.T) {
	service, started := startedGame(t)
	if _, err := service.MoveWithKey(started.ID, alice, 3, 0, "shared"); err != nil {
		t.Fatal(err)
	}
	// Same key, other player: a new move, not a replay.
	yellow, err := service.MoveWithKey(started.ID, bob, 4, 1, "shared")
	if err != nil {
		t.Fatalf("yellow with same key: %v", err)
	}
	if yellow.Move.MoveNumber != 2 || yellow.Move.Column != 4 {
		t.Fatalf("yellow move = %+v, want a new move 2 in column 4", yellow.Move)
	}

	// Same key, same player, other game: a new move.
	other, _ := service.Create(game.ModeAgentVsAgent)
	if _, err := service.Join(other.ID, alice, "red", "Alpha", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Join(other.ID, bob, "yellow", "Beta", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	second, err := service.MoveWithKey(other.ID, alice, 0, 0, "shared")
	if err != nil {
		t.Fatalf("same key in other game: %v", err)
	}
	if second.GameID != other.ID || second.Move.Column != 0 {
		t.Fatalf("other-game move = %+v", second)
	}
}

// Rejected moves are not recorded, an outsider cannot replay someone else's
// key, and an empty key means no idempotency.
func TestIdempotencyEdgeCases(t *testing.T) {
	service, started := startedGame(t)
	_, err := service.MoveWithKey(started.ID, alice, 9, 0, "k-bad")
	wantCode(t, err, game.CodeInvalidColumn)
	fixed, err := service.MoveWithKey(started.ID, alice, 2, 0, "k-bad")
	if err != nil {
		t.Fatalf("retry of a rejected move with the same key: %v", err)
	}
	if fixed.Move.Column != 2 {
		t.Fatalf("move = %+v, want column 2", fixed.Move)
	}

	_, err = service.MoveWithKey(started.ID, carol, 2, 0, "k-bad")
	wantCode(t, err, game.CodeForbidden)

	if _, err := service.MoveWithKey(started.ID, bob, 2, 1, ""); err != nil {
		t.Fatal(err)
	}
	_, err = service.MoveWithKey(started.ID, bob, 2, 1, "")
	wantCode(t, err, game.CodeNotYourTurn)
}

// Concurrent retries with one key place exactly one disc.
func TestConcurrentRetriesPlaceOneDisc(t *testing.T) {
	service, started := startedGame(t)
	const n = 16
	results := make([]game.MoveResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = service.MoveWithKey(started.ID, alice, 3, 0, "burst")
		}()
	}
	wg.Wait()
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("request %d: %v", i, errs[i])
		}
		if !reflect.DeepEqual(results[i], results[0]) {
			t.Fatalf("request %d = %+v, want %+v", i, results[i], results[0])
		}
	}
	if g, _ := service.Get(started.ID); g.Version != 1 || len(g.Moves) != 1 {
		t.Fatalf("version=%d moves=%d, want 1 and 1", g.Version, len(g.Moves))
	}
}

// BDR-012 scenario 1 over REST.
func TestAPIRetryAfterLostResponse(t *testing.T) {
	a := newArena(t)
	a.start(t)
	body := map[string]any{"column": 3, "expectedVersion": 0, "idempotencyKey": "k-1"}
	first := a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", body).wantStatus(t, http.StatusOK).body
	retry := a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", body).wantStatus(t, http.StatusOK).body
	if !reflect.DeepEqual(first, retry) {
		t.Fatalf("retry response differs:\n%v\n%v", first, retry)
	}
	move := retry["move"].(map[string]any)
	if number(move["moveNumber"]) != 1 || number(move["column"]) != 3 || number(move["row"]) != 5 {
		t.Fatalf("move = %v, want number 1, column 3, row 5", move)
	}
	state := a.state(t)
	if number(state["version"]) != 1 || len(state["moves"].([]any)) != 1 {
		t.Fatalf("version=%v moves=%d, want 1 and 1", state["version"], len(state["moves"].([]any)))
	}

	// AC-4 over REST: yellow reusing the key makes its own move.
	yellow := a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 4, "expectedVersion": 1, "idempotencyKey": "k-1"}).wantStatus(t, http.StatusOK).body
	if number(yellow["move"].(map[string]any)["moveNumber"]) != 2 {
		t.Fatalf("yellow move = %v, want move 2", yellow["move"])
	}
}
