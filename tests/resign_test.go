package tests

import (
	"net/http"
	"testing"

	"connect4arena/internal/engine"
	"connect4arena/internal/game"
)

// BDR-010 AC-1, AC-2: either player may resign at any point in an in-progress game.
func TestResignCompletesGameForOpponent(t *testing.T) {
	cases := []struct {
		name       string
		resigner   string
		winner     engine.Disc
		movesFirst []int
	}{
		{"red resigns on yellow's turn", alice, engine.Yellow, []int{3}},
		{"red resigns on own turn", alice, engine.Yellow, nil},
		{"yellow resigns on red's turn", bob, engine.Red, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, started := startedGame(t)
			playMoves(t, service, started.ID, tc.movesFirst...)
			before, _ := service.Get(started.ID)

			resigned, err := service.Resign(started.ID, tc.resigner)
			if err != nil {
				t.Fatalf("Resign: %v", err)
			}
			if resigned.Status != game.StatusCompleted || resigned.Winner != tc.winner ||
				resigned.ResultReason != game.ReasonResignation {
				t.Fatalf("after resign: status=%s winner=%s reason=%s", resigned.Status, resigned.Winner, resigned.ResultReason)
			}
			if resigned.CurrentTurn != engine.Empty || len(resigned.LegalMoves) != 0 {
				t.Fatalf("completed game still has turn %q and legal moves %v", resigned.CurrentTurn, resigned.LegalMoves)
			}
			if resigned.Version != before.Version+1 {
				t.Fatalf("version = %d, want %d", resigned.Version, before.Version+1)
			}
			stored, _ := service.Get(started.ID)
			if stored.Status != game.StatusCompleted || stored.ResultReason != game.ReasonResignation {
				t.Fatalf("stored game not completed by resignation: %+v", stored)
			}
		})
	}
}

// BDR-010 AC-3: after resignation, moves and further resignations are rejected.
func TestMovesAfterResignationAreRejected(t *testing.T) {
	service, started := startedGame(t)
	resigned, err := service.Resign(started.ID, bob)
	if err != nil {
		t.Fatalf("Resign: %v", err)
	}
	gameErr := wantCode(t, moveErr(service, started.ID, alice, 0, resigned.Version), game.CodeGameCompleted)
	if gameErr.CurrentVersion == nil || *gameErr.CurrentVersion != resigned.Version {
		t.Fatalf("GAME_COMPLETED currentVersion = %v, want %d", gameErr.CurrentVersion, resigned.Version)
	}
	_, err = service.Resign(started.ID, alice)
	wantCode(t, err, game.CodeGameCompleted)
}

func moveErr(service *game.Service, id, player string, column, version int) error {
	_, err := service.Move(id, player, column, version)
	return err
}

// BDR-010 AC-4 and SPEC.md §16: outsiders, unknown games and waiting games cannot be resigned.
func TestResignRejections(t *testing.T) {
	service, started := startedGame(t)
	_, err := service.Resign(started.ID, carol)
	wantCode(t, err, game.CodeForbidden)
	if g, _ := service.Get(started.ID); g.Status != game.StatusInProgress || g.Version != started.Version {
		t.Fatalf("outsider resignation changed the game: status=%s version=%d", g.Status, g.Version)
	}

	_, err = service.Resign("game_missing", alice)
	wantCode(t, err, game.CodeGameNotFound)

	waiting, _ := service.Create(game.ModeAgentVsAgent)
	if _, err := service.Join(waiting.ID, alice, engine.Red, "Alpha", game.PlayerAgent); err != nil {
		t.Fatal(err)
	}
	_, err = service.Resign(waiting.ID, alice)
	wantCode(t, err, game.CodeGameNotStarted)
}

// BDR-010 scenario 1 over REST.
func TestAPIRedResigns(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 3) // yellow to move

	resigned := a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantStatus(t, http.StatusOK).body
	if resigned["status"] != "completed" || resigned["winner"] != "yellow" || resigned["resultReason"] != "resignation" {
		t.Fatalf("resign response: %v", resigned)
	}

	state := a.state(t)
	if state["status"] != "completed" || state["winner"] != "yellow" || state["resultReason"] != "resignation" {
		t.Fatalf("state after resign: status=%v winner=%v reason=%v", state["status"], state["winner"], state["resultReason"])
	}
	if state["currentTurn"] != nil {
		t.Fatalf("currentTurn = %v, want null", state["currentTurn"])
	}

	version := number(state["version"])
	errObject := a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
		map[string]any{"column": 0, "expectedVersion": version}).wantError(t, http.StatusConflict, "GAME_COMPLETED")
	if number(errObject["currentVersion"]) != version {
		t.Fatalf("currentVersion = %v, want %d", errObject["currentVersion"], version)
	}
}

// BDR-010 scenario 2 over REST, plus authentication and lifecycle errors.
func TestAPIResignRejections(t *testing.T) {
	a := newArena(t)
	a.create(t)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Alpha", "type": "agent"}).wantStatus(t, http.StatusOK)
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantError(t, http.StatusConflict, "GAME_NOT_STARTED")

	a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "yellow", "displayName": "Beta", "type": "agent"}).wantStatus(t, http.StatusOK)

	a.other.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantError(t, http.StatusForbidden, "FORBIDDEN")
	a.anon.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantError(t, http.StatusUnauthorized, "UNAUTHENTICATED")
	a.red.do(http.MethodPost, "/api/v1/games/game_missing/resign", nil).wantError(t, http.StatusNotFound, "GAME_NOT_FOUND")

	if state := a.state(t); state["status"] != "in_progress" {
		t.Fatalf("status = %v, want in_progress", state["status"])
	}
}
