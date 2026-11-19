package tests

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// startGame creates an agent_vs_agent game, joins red and yellow, and returns
// its id.
func startGame(t *testing.T, red, yellow client) string {
	t.Helper()
	created := red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "agent_vs_agent"}).wantStatus(t, http.StatusCreated)
	gameID := created.body["gameId"].(string)
	red.do(http.MethodPost, "/api/v1/games/"+gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Red", "type": "agent"}).wantStatus(t, http.StatusOK)
	yellow.do(http.MethodPost, "/api/v1/games/"+gameID+"/join",
		map[string]any{"disc": "yellow", "displayName": "Yellow", "type": "agent"}).wantStatus(t, http.StatusOK)
	return gameID
}

// stallMove opens a connection and sends a move request with only half its
// body, then goes silent, as a hung agent would. Closing the returned
// connection is the agent's process dying.
func stallMove(t *testing.T, c client, gameID string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", strings.TrimPrefix(c.server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	body := `{"column":3,"expectedVersion":2}`
	_, _ = fmt.Fprintf(conn, "POST /api/v1/games/%s/moves HTTP/1.1\r\nHost: arena\r\nAuthorization: Bearer %s\r\n"+
		"Content-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", gameID, c.token, len(body), body[:len(body)/2])
	return conn
}

// BDR-025 Scenario 1, AGENT-005: agent A stops in the middle of game G1 on its
// own turn: its MCP session drops and a move it was sending is cut off. G1
// keeps its last valid state and stays readable (AC-1), agents C and D finish
// their own game G2 meanwhile (AC-3), and A comes back later with the same
// credential, over another interface, and moves from the stored state (AC-2).
func TestAgentFailureIsIsolatedToItsGame(t *testing.T) {
	server := newServer(t)
	agentA, agentB := register(t, server, "A"), register(t, server, "B")
	agentC, agentD := register(t, server, "C"), register(t, server, "D")
	anon := client{t: t, server: server}
	g1, g2 := startGame(t, agentA, agentB), startGame(t, agentC, agentD)

	// A plays over MCP, B over REST, until it is A's turn again.
	sessionA := mustConnectMCP(t, agentA)
	callOK(t, sessionA, "make_move", map[string]any{"game_id": g1, "column": 3, "expected_version": 0, "idempotency_key": "a-1"})
	agentB.do(http.MethodPost, "/api/v1/games/"+g1+"/moves", map[string]any{"column": 4, "expectedVersion": 1}).
		wantStatus(t, http.StatusOK)
	before := anon.do(http.MethodGet, "/api/v1/games/"+g1, nil).wantStatus(t, http.StatusOK)
	if before.body["currentTurn"] != "red" || number(before.body["version"]) != 2 {
		t.Fatalf("G1 before A fails = %v, want red to move at version 2", before.body)
	}

	// A hangs: its MCP session goes away and a half-sent move stalls.
	_ = sessionA.Close()
	stalled := stallMove(t, agentA, g1)

	// AC-3: C and D play G2 to the end while A is unresponsive.
	for i, column := range []int{0, 1, 0, 1, 0, 1, 0} {
		mover := agentC
		if i%2 == 1 {
			mover = agentD
		}
		mover.do(http.MethodPost, "/api/v1/games/"+g2+"/moves", map[string]any{"column": column, "expectedVersion": i}).
			wantStatus(t, http.StatusOK)
	}
	if g := anon.do(http.MethodGet, "/api/v1/games/"+g2, nil).body; g["status"] != "completed" || g["winner"] != "red" {
		t.Fatalf("G2 = %v, want red to win while A is unresponsive", g)
	}

	// AC-1: G1 is exactly as A left it, and readable, while A hangs and
	// after its process dies.
	if during := anon.do(http.MethodGet, "/api/v1/games/"+g1, nil).wantStatus(t, http.StatusOK); during.raw != before.raw {
		t.Fatalf("G1 changed while A hung:\nbefore %s\nduring %s", before.raw, during.raw)
	}
	_ = stalled.Close()
	time.Sleep(50 * time.Millisecond) // give the cut-off request time to do damage
	after := anon.do(http.MethodGet, "/api/v1/games/"+g1, nil).wantStatus(t, http.StatusOK)
	if after.raw != before.raw {
		t.Fatalf("G1 changed while A was gone:\nbefore %s\nafter  %s", before.raw, after.raw)
	}
	if history := anon.do(http.MethodGet, "/api/v1/games/"+g1+"/moves", nil).body; len(history["moves"].([]any)) != 2 {
		t.Fatalf("G1 history = %v, want the 2 moves made before A failed", history)
	}

	// B cannot take A's turn while A is away.
	agentB.do(http.MethodPost, "/api/v1/games/"+g1+"/moves", map[string]any{"column": 0, "expectedVersion": 2}).
		wantError(t, http.StatusConflict, "NOT_YOUR_TURN")

	// AC-2: A restarts with the same key, now over REST, and continues.
	moved := agentA.do(http.MethodPost, "/api/v1/games/"+g1+"/moves", map[string]any{"column": 3, "expectedVersion": 2}).
		wantStatus(t, http.StatusOK)
	if number(moved.body["version"]) != 3 || moved.body["nextTurn"] != "yellow" {
		t.Fatalf("A's move after restarting = %v, want version 3 with yellow next", moved.body)
	}
}
