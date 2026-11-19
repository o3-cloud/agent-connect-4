package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"connect4arena/internal/cli"
)

// cliAgent runs connect4 commands as an external agent would, returning
// errors instead of failing the test so it can run in its own goroutine.
type cliAgent struct {
	server, key string
}

func (a cliAgent) run(args ...string) (map[string]any, error) {
	var stdout, stderr bytes.Buffer
	env := map[string]string{"CONNECT4_SERVER": a.server, "CONNECT4_API_KEY": a.key}
	code := cli.Execute(append(args, "--json"), &stdout, &stderr, func(k string) string { return env[k] })
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		return nil, fmt.Errorf("connect4 %v: exit %d, stdout not JSON: %v", args, code, err)
	}
	if code != 0 {
		return doc, fmt.Errorf("connect4 %v: exit %d: %v", args, code, doc)
	}
	return doc, nil
}

func mcpTool(session *mcp.ClientSession, tool string, args map[string]any) (map[string]any, error) {
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	raw, _ := json.Marshal(result.StructuredContent)
	_ = json.Unmarshal(raw, &out)
	if result.IsError {
		return out, fmt.Errorf("%s: %v", tool, out)
	}
	return out, nil
}

// playUntilDone is an agent's whole strategy: read the state, and on its own
// turn pick a column from legalMoves (AC-2). It returns when the game is over.
func playUntilDone(disc string, read func() (map[string]any, error), move func(state map[string]any) error) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		state, err := read()
		if err != nil {
			return err
		}
		switch {
		case state["status"] == "completed":
			return nil
		case state["status"] == "in_progress" && state["currentTurn"] == disc:
			if err := move(state); err != nil {
				return err
			}
		default:
			time.Sleep(20 * time.Millisecond) // poll politely: 600 requests per minute per agent
		}
	}
	return errors.New(disc + " agent: game did not finish within 20s")
}

var spectatorMove = regexp.MustCompile(`data-move="\d+"`)

// BDR-024 scenario 1, AC-1..4: a CLI agent and an MCP agent play a whole game
// on their own while a browser spectator watches.
func TestAutonomousCLIAgentVsMCPAgent(t *testing.T) {
	server := newServer(t)
	a := cliAgent{server: server.URL, key: cliKey(t, server, "Agent A")}
	b := register(t, server, "Agent B")
	session := mustConnectMCP(t, b)

	// AC-1: A creates and joins over the CLI; B joins over MCP.
	created, err := a.run("create", "--mode", "agent_vs_agent")
	if err != nil {
		t.Fatal(err)
	}
	game := created["gameId"].(string)
	if _, err := a.run("join", game, "--disc", "red", "--name", "Agent A"); err != nil {
		t.Fatal(err)
	}
	if _, err := mcpTool(session, "join_game", map[string]any{"game_id": game, "disc": "yellow", "display_name": "Agent B"}); err != nil {
		t.Fatal(err)
	}

	// B holds its first move until the spectator has looked, so AC-4 is
	// checked while the game is certainly in progress.
	spectated := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- playUntilDone("red",
			func() (map[string]any, error) { return a.run("state", game) },
			func(state map[string]any) error {
				legal := ints(state["legalMoves"])
				_, err := a.run("move", game, "--column", strconv.Itoa(legal[0]),
					"--expected-version", strconv.Itoa(number(state["version"])))
				return err
			})
	}()
	go func() {
		moved := 0
		done <- playUntilDone("yellow",
			func() (map[string]any, error) {
				return mcpTool(session, "get_game_state", map[string]any{"game_id": game})
			},
			func(state map[string]any) error {
				if moved == 0 {
					<-spectated
				}
				moved++
				legal := ints(state["legalMoves"])
				_, err := mcpTool(session, "make_move", map[string]any{
					"game_id": game, "column": legal[len(legal)-1], "expected_version": number(state["version"]),
					"idempotency_key": "b-" + strconv.Itoa(moved),
				})
				return err
			})
	}()

	// AC-4: an anonymous browser sees the game in progress with A's move.
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get(server.URL + "/games/" + game)
		if err != nil {
			t.Fatal(err)
		}
		page, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if bytes.Contains(page, []byte(`data-status="in_progress"`)) && len(spectatorMove.FindAll(page, -1)) == 1 {
			if !bytes.Contains(page, []byte("Agent A")) || !bytes.Contains(page, []byte("Agent B")) ||
				!bytes.Contains(page, []byte(`hx-trigger="every`)) {
				t.Fatalf("spectator page lacks players or live refresh:\n%s", page)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("spectator never saw the game in progress with one move:\n%s", page)
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(spectated)

	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}

	// AC-3: a valid result within 42 moves, red and yellow alternating.
	state, err := a.run("state", game)
	if err != nil {
		t.Fatal(err)
	}
	if state["status"] != "completed" || (state["winner"] == nil && state["resultReason"] != "draw") {
		t.Fatalf("final state = %v", state)
	}
	moves := state["moves"].([]any)
	if len(moves) < 7 || len(moves) > 42 {
		t.Fatalf("game ended after %d moves", len(moves))
	}
	for i, m := range moves {
		want := "red"
		if i%2 == 1 {
			want = "yellow"
		}
		if disc := m.(map[string]any)["disc"]; disc != want {
			t.Fatalf("move %d was %v, want %s", i+1, disc, want)
		}
	}
	t.Logf("%s won by %s in %d moves", state["winner"], state["resultReason"], len(moves))
}
