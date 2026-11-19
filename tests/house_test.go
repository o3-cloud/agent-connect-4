package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"connect4arena/internal/houseagent"
)

// waitFor polls the game until cond holds or the deadline passes.
func waitFor(t *testing.T, a *arena, within time.Duration, cond func(map[string]any) bool) map[string]any {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		state := a.state(t)
		if cond(state) {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %s; state: %v", within, state)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ADR-014, AGENT-007: Arena Bot joins a human_vs_agent game with
// "opponent": "house" and replies to every human move within 1 second,
// through to a finished game.
func TestHouseAgentPlaysAHuman(t *testing.T) {
	a := newArena(t)
	human := a.red
	created := human.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
		wantStatus(t, http.StatusCreated)
	a.gameID = created.body["gameId"].(string)
	human.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Human", "type": "human"}).wantStatus(t, http.StatusOK)

	state := waitFor(t, a, time.Second, func(s map[string]any) bool { return s["status"] == "in_progress" })
	bot := state["players"].(map[string]any)["yellow"].(map[string]any)
	if bot["displayName"] != "Arena Bot" || bot["type"] != "agent" {
		t.Fatalf("yellow player = %v, want Arena Bot", bot)
	}

	for turn := 0; turn < 21 && state["status"] == "in_progress"; turn++ {
		version := number(state["version"])
		column := ints(state["legalMoves"])[0]
		human.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves",
			map[string]any{"column": column, "expectedVersion": version}).wantStatus(t, http.StatusOK)
		state = waitFor(t, a, time.Second, func(s map[string]any) bool {
			return s["status"] == "completed" || (s["currentTurn"] == "red" && number(s["version"]) == version+2)
		})
	}
	if state["status"] != "completed" {
		t.Fatalf("game did not finish: %v", state)
	}
	if strings.Contains(a.anon.do(http.MethodGet, "/api/v1/games/"+a.gameID+"/moves", nil).raw, "c4_") {
		t.Fatal("an API key appears in the move history")
	}
}

func TestHouseOpponentOnlyForHumanVsAgent(t *testing.T) {
	a := newArena(t)
	a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "agent_vs_agent", "opponent": "house"}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
	a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "robot"}).
		wantError(t, http.StatusBadRequest, "INVALID_ARGUMENT")
}

// boardFrom builds a 6×7 board (row 0 at the top) from bottom-up column stacks.
func boardFrom(stacks map[int]string) [][]string {
	board := make([][]string, 6)
	for r := range board {
		board[r] = make([]string, 7)
	}
	for column, discs := range stacks {
		for i, d := range discs {
			if d == 'R' {
				board[5-i][column] = "red"
			} else {
				board[5-i][column] = "yellow"
			}
		}
	}
	return board
}

func legalFor(board [][]string) []int {
	var legal []int
	for c := 0; c < 7; c++ {
		if board[0][c] == "" {
			legal = append(legal, c)
		}
	}
	return legal
}

// ADR-014 strategy: win now, else block, else nearest the centre.
func TestHouseAgentStrategy(t *testing.T) {
	cases := []struct {
		name   string
		stacks map[int]string
		me     string
		want   int
	}{
		{"takes a vertical win", map[int]string{2: "YYY", 4: "RR"}, "yellow", 2},
		{"takes a horizontal win over a block", map[int]string{0: "Y", 1: "Y", 2: "Y", 6: "RRR"}, "yellow", 3},
		{"blocks a vertical threat", map[int]string{5: "RRR", 3: "Y"}, "yellow", 5},
		{"blocks a horizontal threat", map[int]string{1: "R", 2: "R", 3: "R", 6: "YY"}, "yellow", 0},
		{"opens in the centre", map[int]string{}, "red", 3},
		{"nearest the centre when 3 is full", map[int]string{3: "RYRYRY"}, "red", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			board := boardFrom(c.stacks)
			if got := houseagent.Choose(board, legalFor(board), c.me); got != c.want {
				t.Fatalf("Choose = %d, want %d", got, c.want)
			}
		})
	}
}

// house-agent-join-race-readme: Arena Bot takes the other slot before the
// human's join returns, so the join response is already in_progress and the
// human can move at once with no polling, whichever disc they choose.
func TestHouseAgentIsSeatedBeforeJoinReturns(t *testing.T) {
	a := newArena(t)
	for run := 0; run < 20; run++ {
		disc, botDisc := "red", "yellow"
		if run%2 == 1 {
			disc, botDisc = "yellow", "red"
		}
		created := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
			wantStatus(t, http.StatusCreated)
		gameID := created.body["gameId"].(string)
		joined := a.red.do(http.MethodPost, "/api/v1/games/"+gameID+"/join",
			map[string]any{"disc": disc, "displayName": "Human", "type": "human"}).wantStatus(t, http.StatusOK)
		if joined.body["status"] != "in_progress" {
			t.Fatalf("run %d: join returned status %v, want in_progress", run, joined.body["status"])
		}
		bot := joined.body["players"].(map[string]any)[botDisc].(map[string]any)
		if bot["displayName"] != houseagent.DisplayName {
			t.Fatalf("run %d: %s player = %v, want Arena Bot", run, botDisc, bot)
		}
		if disc == "red" {
			a.red.do(http.MethodPost, "/api/v1/games/"+gameID+"/moves",
				map[string]any{"column": 3, "expectedVersion": number(joined.body["version"])}).wantStatus(t, http.StatusOK)
		}
	}
}

// The README flow, through the CLI: create, join and move with no waiting.
func TestHouseAgentCLIJoinThenMove(t *testing.T) {
	server := newServer(t)
	key := cliKey(t, server, "You")
	for run := 0; run < 10; run++ {
		code, created := cliJSON(t, server, key, "create", "--mode", "human_vs_agent", "--opponent", "house")
		gameID, _ := created["gameId"].(string)
		if code != 0 || gameID == "" {
			t.Fatalf("create: exit %d, %v", code, created)
		}
		if code, joined := cliJSON(t, server, key, "join", gameID, "--disc", "red", "--name", "You", "--type", "human"); code != 0 || joined["status"] != "in_progress" {
			t.Fatalf("run %d: join: exit %d, %v", run, code, joined)
		}
		if code, moved := cliJSON(t, server, key, "move", gameID, "--column", "3"); code != 0 {
			t.Fatalf("run %d: move straight after join: exit %d, %v", run, code, moved)
		}
	}
}

// The same holds for a human who joins a house game through MCP.
func TestHouseAgentSeatedBeforeMCPJoinReturns(t *testing.T) {
	a := newArena(t)
	created := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
		wantStatus(t, http.StatusCreated)
	gameID := created.body["gameId"].(string)
	joined := callOK(t, mustConnectMCP(t, a.red), "join_game",
		map[string]any{"game_id": gameID, "disc": "yellow", "display_name": "Human"})
	if joined["status"] != "in_progress" {
		t.Fatalf("join_game returned %v, want in_progress", joined)
	}
}
