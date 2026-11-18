package tests

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"connect4arena/internal/engine"
	"connect4arena/internal/game"
)

const (
	alice = "player_alice"
	bob   = "player_bob"
	carol = "player_carol"
)

func wantCode(t *testing.T, err error, code game.Code) game.Error {
	t.Helper()
	var gameErr *game.Error
	if !errors.As(err, &gameErr) {
		t.Fatalf("error = %v, want *game.Error with code %s", err, code)
	}
	if gameErr.Code != code {
		t.Fatalf("error code = %s (%s), want %s", gameErr.Code, gameErr.Message, code)
	}
	return *gameErr
}

// startedGame returns a service and an in-progress game with alice as red and bob as yellow.
func startedGame(t *testing.T) (*game.Service, game.Game) {
	t.Helper()
	service := game.NewService()
	created, err := service.Create(game.ModeAgentVsAgent)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := service.Join(created.ID, alice, engine.Red, "Alpha", game.PlayerAgent); err != nil {
		t.Fatalf("join red: %v", err)
	}
	started, err := service.Join(created.ID, bob, engine.Yellow, "Beta", game.PlayerAgent)
	if err != nil {
		t.Fatalf("join yellow: %v", err)
	}
	return service, started
}

func playMoves(t *testing.T, service *game.Service, id string, columns ...int) game.MoveResult {
	t.Helper()
	var result game.MoveResult
	for i, column := range columns {
		player := alice
		if i%2 == 1 {
			player = bob
		}
		var err error
		result, err = service.Move(id, player, column, i)
		if err != nil {
			t.Fatalf("move %d (column %d): %v", i+1, column, err)
		}
	}
	return result
}

func TestCreateGameStartsWaitingWithEmptyBoard(t *testing.T) {
	service := game.NewService()
	for _, mode := range []game.Mode{game.ModeAgentVsAgent, game.ModeHumanVsAgent} {
		created, err := service.Create(mode)
		if err != nil {
			t.Fatalf("Create(%s): %v", mode, err)
		}
		if created.ID == "" || created.Mode != mode || created.Status != game.StatusWaiting || created.Version != 0 {
			t.Fatalf("created = %+v, want waiting %s game at version 0", created, mode)
		}
		if created.Board != [engine.Rows][engine.Columns]engine.Disc{} {
			t.Fatal("new board is not empty")
		}
		if created.CurrentTurn != engine.Empty || len(created.LegalMoves) != 0 {
			t.Fatalf("currentTurn=%q legalMoves=%v, want none while waiting", created.CurrentTurn, created.LegalMoves)
		}
		if got := created.OpenSlots(); len(got) != 2 || got[0] != engine.Red || got[1] != engine.Yellow {
			t.Fatalf("OpenSlots() = %v, want [red yellow]", got)
		}
	}
}

func TestCreateGameRejectsUnknownMode(t *testing.T) {
	service := game.NewService()
	_, err := service.Create("human_vs_human")
	wantCode(t, err, game.CodeInvalidArgument)
	if len(service.List()) != 0 {
		t.Fatal("rejected create added a game")
	}
}

func TestCreatedGameIDsAreUnique(t *testing.T) {
	service := game.NewService()
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		created, err := service.Create(game.ModeAgentVsAgent)
		if err != nil {
			t.Fatal(err)
		}
		if seen[created.ID] {
			t.Fatalf("duplicate game id %s", created.ID)
		}
		seen[created.ID] = true
	}
	if got := service.List(); len(got) != 100 {
		t.Fatalf("List() has %d games, want 100", len(got))
	}
}

func TestJoinFillsSlotsAndStartsGame(t *testing.T) {
	service := game.NewService()
	created, _ := service.Create(game.ModeAgentVsAgent)

	afterRed, err := service.Join(created.ID, alice, engine.Red, "Alpha", game.PlayerAgent)
	if err != nil {
		t.Fatal(err)
	}
	if afterRed.Status != game.StatusWaiting || afterRed.CurrentTurn != engine.Empty {
		t.Fatalf("after one join: status=%s turn=%q, want waiting and no turn", afterRed.Status, afterRed.CurrentTurn)
	}
	if afterRed.Red == nil || afterRed.Red.DisplayName != "Alpha" || afterRed.Red.ID != alice || afterRed.Red.Disc != engine.Red {
		t.Fatalf("red player = %+v", afterRed.Red)
	}

	started, err := service.Join(created.ID, bob, engine.Yellow, "Beta", game.PlayerAgent)
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != game.StatusInProgress || started.CurrentTurn != engine.Red || started.Version != 0 {
		t.Fatalf("after both joins: status=%s turn=%q version=%d", started.Status, started.CurrentTurn, started.Version)
	}
	if len(started.LegalMoves) != 7 || len(started.OpenSlots()) != 0 {
		t.Fatalf("legalMoves=%v openSlots=%v", started.LegalMoves, started.OpenSlots())
	}
}

func TestJoinRejections(t *testing.T) {
	service := game.NewService()
	created, _ := service.Create(game.ModeHumanVsAgent)
	if _, err := service.Join(created.ID, alice, engine.Red, "Alpha", game.PlayerHuman); err != nil {
		t.Fatal(err)
	}

	_, err := service.Join(created.ID, carol, engine.Red, "Gamma", game.PlayerAgent)
	wantCode(t, err, game.CodeSlotTaken)
	_, err = service.Join(created.ID, alice, engine.Yellow, "Alpha again", game.PlayerAgent)
	wantCode(t, err, game.CodeAlreadyJoined)
	_, err = service.Join(created.ID, carol, engine.Empty, "Gamma", game.PlayerAgent)
	wantCode(t, err, game.CodeInvalidArgument)
	_, err = service.Join(created.ID, carol, engine.Yellow, "", game.PlayerAgent)
	wantCode(t, err, game.CodeInvalidArgument)
	_, err = service.Join(created.ID, carol, engine.Yellow, "Gamma", "robot")
	wantCode(t, err, game.CodeInvalidArgument)
	_, err = service.Join("game_missing", carol, engine.Yellow, "Gamma", game.PlayerAgent)
	wantCode(t, err, game.CodeGameNotFound)

	state, _ := service.Get(created.ID)
	if state.Red.DisplayName != "Alpha" || state.Yellow != nil {
		t.Fatalf("slots changed after rejected joins: red=%+v yellow=%+v", state.Red, state.Yellow)
	}

	if _, err := service.Join(created.ID, bob, engine.Yellow, "Beta", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	_, err = service.Join(created.ID, carol, engine.Yellow, "Gamma", game.PlayerAgent)
	wantCode(t, err, game.CodeGameNotJoinable)
}

func TestGetUnknownGame(t *testing.T) {
	service := game.NewService()
	_, err := service.Get("game_missing")
	gameErr := wantCode(t, err, game.CodeGameNotFound)
	if gameErr.GameID != "game_missing" || gameErr.Error() == "" {
		t.Fatalf("error = %+v, want gameId context and a message", gameErr)
	}
	if len(service.List()) != 0 {
		t.Fatal("Get created a game")
	}
}

func TestLegalMoveStacksAndAlternatesTurns(t *testing.T) {
	service, started := startedGame(t)

	first, err := service.Move(started.ID, alice, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Accepted || first.Move.MoveNumber != 1 || first.Move.Row != 5 || first.Move.Column != 3 ||
		first.Move.Disc != engine.Red || first.Move.PlayerID != alice || first.NextTurn != engine.Yellow ||
		first.Version != 1 || first.Status != game.StatusInProgress {
		t.Fatalf("first move result = %+v", first)
	}

	second, err := service.Move(started.ID, bob, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Move.Row != 4 || second.NextTurn != engine.Red || second.Version != 2 {
		t.Fatalf("second move result = %+v", second)
	}

	state, _ := service.Get(started.ID)
	if state.Board[5][3] != engine.Red || state.Board[4][3] != engine.Yellow || state.CurrentTurn != engine.Red ||
		len(state.Moves) != 2 || state.Moves[1].MoveNumber != 2 || state.MoveNumber() != 2 {
		t.Fatalf("state after two moves = %+v", state)
	}
}

func TestRejectedMovesLeaveStateUnchanged(t *testing.T) {
	service, started := startedGame(t)
	playMoves(t, service, started.ID, 0, 0, 0, 0, 0, 0)
	before, _ := service.Get(started.ID)

	tests := []struct {
		name    string
		player  string
		column  int
		version int
		code    game.Code
	}{
		{name: "column too high", player: alice, column: 7, version: 6, code: game.CodeInvalidColumn},
		{name: "column negative", player: alice, column: -1, version: 6, code: game.CodeInvalidColumn},
		{name: "full column", player: alice, column: 0, version: 6, code: game.CodeColumnFull},
		{name: "out of turn", player: bob, column: 3, version: 6, code: game.CodeNotYourTurn},
		{name: "stale version", player: alice, column: 3, version: 5, code: game.CodeStaleState},
		{name: "non-participant", player: carol, column: 3, version: 6, code: game.CodeForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Move(started.ID, test.player, test.column, test.version)
			gameErr := wantCode(t, err, test.code)
			if test.code != game.CodeForbidden && (gameErr.CurrentVersion == nil || *gameErr.CurrentVersion != 6) {
				t.Fatalf("currentVersion = %v, want 6", gameErr.CurrentVersion)
			}
			after, _ := service.Get(started.ID)
			if after.Board != before.Board || after.CurrentTurn != before.CurrentTurn || after.Version != before.Version {
				t.Fatal("rejected move changed the game")
			}
		})
	}

	_, err := service.Move("game_missing", alice, 0, 0)
	wantCode(t, err, game.CodeGameNotFound)
}

func TestMoveBeforeGameStarts(t *testing.T) {
	service := game.NewService()
	created, _ := service.Create(game.ModeAgentVsAgent)
	if _, err := service.Join(created.ID, alice, engine.Red, "Alpha", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	_, err := service.Move(created.ID, alice, 3, 0)
	wantCode(t, err, game.CodeGameNotStarted)
}

func TestWinCompletesGameAndRejectsFurtherMoves(t *testing.T) {
	service, started := startedGame(t)
	result := playMoves(t, service, started.ID, 0, 0, 1, 1, 2, 2, 3)
	if result.Status != game.StatusCompleted || result.Winner != engine.Red || result.ResultReason != game.ReasonConnectFour ||
		result.NextTurn != engine.Empty || result.Version != 7 {
		t.Fatalf("winning move result = %+v", result)
	}

	state, _ := service.Get(started.ID)
	if state.Status != game.StatusCompleted || state.CurrentTurn != engine.Empty || len(state.LegalMoves) != 0 {
		t.Fatalf("state after win = status %s turn %q legal %v", state.Status, state.CurrentTurn, state.LegalMoves)
	}

	_, err := service.Move(started.ID, bob, 4, 7)
	wantCode(t, err, game.CodeGameCompleted)
	after, _ := service.Get(started.ID)
	if after.Version != 7 || after.Board != state.Board {
		t.Fatal("move after completion changed the game")
	}
}

func TestDrawCompletesGame(t *testing.T) {
	service, started := startedGame(t)
	result := playMoves(t, service, started.ID, 0, 1, 0, 1, 0, 1, 2, 3, 2, 3, 2, 3, 4, 5, 4, 5, 4, 5, 6, 0, 6, 0, 6, 0,
		1, 2, 1, 2, 1, 2, 3, 4, 3, 4, 3, 4, 5, 6, 5, 6, 5, 6)
	if result.Status != game.StatusCompleted || result.Winner != engine.Empty || result.ResultReason != game.ReasonDraw {
		t.Fatalf("final move result = %+v, want draw", result)
	}
	state, _ := service.Get(started.ID)
	if len(state.Moves) != 42 {
		t.Fatalf("history has %d moves, want 42", len(state.Moves))
	}
}

func TestGetReturnsSnapshot(t *testing.T) {
	service, started := startedGame(t)
	snapshot, _ := service.Get(started.ID)
	snapshot.Red.DisplayName = "tampered"
	snapshot.LegalMoves[0] = 99
	playMoves(t, service, started.ID, 0)
	moved, _ := service.Get(started.ID)
	moved.Moves[0].Column = 99

	state, _ := service.Get(started.ID)
	if state.Red.DisplayName != "Alpha" || state.LegalMoves[0] != 0 || state.Moves[0].Column != 0 {
		t.Fatal("mutating a returned game changed the stored game")
	}
}

// TURN-004: concurrent submissions for the same turn accept exactly one move.
func TestConcurrentMovesAcceptExactlyOne(t *testing.T) {
	service, started := startedGame(t)
	const attempts = 32
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(column int) {
			defer wg.Done()
			if _, err := service.Move(started.ID, alice, column, 0); err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(i % engine.Columns)
	}
	wg.Wait()
	state, _ := service.Get(started.ID)
	if accepted != 1 || state.Version != 1 || len(state.Moves) != 1 {
		t.Fatalf("accepted=%d version=%d moves=%d, want exactly one", accepted, state.Version, len(state.Moves))
	}
}

func TestListReturnsGamesInCreationOrder(t *testing.T) {
	service := game.NewService()
	var ids []string
	for i := 0; i < 3; i++ {
		created, _ := service.Create(game.ModeAgentVsAgent)
		ids = append(ids, created.ID)
	}
	list := service.List()
	for i, g := range list {
		if g.ID != ids[i] {
			t.Fatalf("List()[%d] = %s, want %s", i, g.ID, ids[i])
		}
	}
}

func TestErrorMessageIncludesCode(t *testing.T) {
	err := &game.Error{Code: game.CodeNotYourTurn, Message: "wait"}
	if got, want := err.Error(), fmt.Sprintf("%s: wait", game.CodeNotYourTurn); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

// BDR-004 AC-3: filtering by status returns only games in that status.
func TestListByStatusFiltersGames(t *testing.T) {
	service, started := startedGame(t)
	waiting, _ := service.Create(game.ModeHumanVsAgent)
	finished, _ := service.Create(game.ModeAgentVsAgent)
	if _, err := service.Join(finished.ID, alice, engine.Red, "Alpha", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Join(finished.ID, bob, engine.Yellow, "Beta", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	playMoves(t, service, finished.ID, 0, 0, 1, 1, 2, 2, 3)

	for status, want := range map[game.Status]string{
		game.StatusWaiting:    waiting.ID,
		game.StatusInProgress: started.ID,
		game.StatusCompleted:  finished.ID,
	} {
		got, err := service.ListByStatus(status)
		if err != nil {
			t.Fatalf("ListByStatus(%s): %v", status, err)
		}
		if len(got) != 1 || got[0].ID != want || got[0].Status != status {
			t.Fatalf("ListByStatus(%s) = %+v, want only %s", status, got, want)
		}
	}

	all, err := service.ListByStatus("")
	if err != nil || len(all) != 3 {
		t.Fatalf("ListByStatus(\"\") = %d games, %v; want all 3", len(all), err)
	}
	_, err = service.ListByStatus("paused")
	wantCode(t, err, game.CodeInvalidArgument)
}
