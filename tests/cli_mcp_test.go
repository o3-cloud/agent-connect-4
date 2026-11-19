package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"connect4arena/internal/cli"
)

// stdioMCP runs `connect4 mcp` against server with key, connected to an MCP
// client over in-memory pipes as a subprocess's stdin and stdout would be.
// The returned wait closes the client and reports the command's exit code and
// stderr.
func stdioMCP(t *testing.T, server *httptest.Server, key string) (*mcp.ClientSession, func() (int, string)) {
	t.Helper()
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	var stderr bytes.Buffer
	done := make(chan int, 1)
	env := map[string]string{"CONNECT4_SERVER": server.URL, "CONNECT4_API_KEY": key}
	go func() {
		code := cli.ExecuteWithInput([]string{"mcp"}, stdinR, stdoutW, &stderr, func(k string) string { return env[k] })
		_ = stdoutW.Close()
		done <- code
	}()
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.IOTransport{Reader: stdoutR, Writer: stdinW}, nil)
	if err != nil {
		t.Fatalf("connect to connect4 mcp: %v (stderr: %s)", err, stderr.String())
	}
	exited, code := false, 0
	wait := func() (int, string) {
		if !exited {
			_ = session.Close()
			select {
			case code = <-done:
				exited = true
			case <-time.After(5 * time.Second):
				t.Fatal("connect4 mcp did not exit after its stdin closed")
			}
		}
		return code, stderr.String()
	}
	t.Cleanup(func() { wait() })
	return session, wait
}

func structured(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// mcp-stdio-transport, ADR-009 item 3, MCP-001, MCP-009: `connect4 mcp`
// serves the arena's MCP tools on stdin and stdout for agents that launch MCP
// servers as subprocesses. It proxies to the arena named by CONNECT4_SERVER as
// CONNECT4_API_KEY, so the arena stays the single authority: the same tools,
// the same results and the same error objects as /mcp.
func TestCLIServesMCPOverStdio(t *testing.T) {
	a := newArena(t)
	stdio, wait := stdioMCP(t, a.server, a.red.token)
	viaHTTP := mustConnectMCP(t, a.yellow)

	// The same six tools with the same descriptions and input schemas.
	remote, err := viaHTTP.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	local, err := stdio.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(local.Tools) != 6 || len(local.Tools) != len(remote.Tools) {
		t.Fatalf("stdio lists %d tools, /mcp lists %d, want 6", len(local.Tools), len(remote.Tools))
	}
	for _, want := range remote.Tools {
		i := slices.IndexFunc(local.Tools, func(tool *mcp.Tool) bool { return tool.Name == want.Name })
		if i < 0 {
			t.Fatalf("stdio does not offer %s", want.Name)
		}
		got, _ := json.Marshal(local.Tools[i])
		expected, _ := json.Marshal(want)
		if string(got) != string(expected) {
			t.Fatalf("tool %s differs:\nstdio %s\n/mcp  %s", want.Name, got, expected)
		}
	}

	// Red plays over stdio, yellow over HTTP MCP, in one game on the arena.
	call := func(session *mcp.ClientSession, tool string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			t.Fatalf("%s: protocol error %v", tool, err)
		}
		return result
	}
	created := call(stdio, "create_game", map[string]any{"mode": "agent_vs_agent"})
	var game map[string]any
	_ = json.Unmarshal([]byte(structured(t, created)), &game)
	gameID, _ := game["gameId"].(string)
	if created.IsError || !strings.HasPrefix(gameID, "game_") {
		t.Fatalf("create_game over stdio = %s", structured(t, created))
	}
	joined := call(stdio, "join_game", map[string]any{"game_id": gameID, "disc": "red", "display_name": "Stdio"})
	if !strings.Contains(structured(t, joined), `"displayName":"Stdio"`) {
		t.Fatalf("join_game over stdio = %s", structured(t, joined))
	}
	call(viaHTTP, "join_game", map[string]any{"game_id": gameID, "disc": "yellow", "display_name": "HTTP"})
	moved := call(stdio, "make_move", map[string]any{"game_id": gameID, "column": 3, "expected_version": 0, "idempotency_key": "s-1"})
	if moved.IsError {
		t.Fatalf("make_move over stdio failed: %s", structured(t, moved))
	}

	// The REST API sees the stdio player's move: one authority.
	state := a.anon.do(http.MethodGet, "/api/v1/games/"+gameID, nil).body
	red := state["players"].(map[string]any)["red"].(map[string]any)
	if cell(state, 5, 3) != "red" || red["id"] == nil || !strings.HasPrefix(red["id"].(string), "agt_") {
		t.Fatalf("arena state after the stdio move = %v", state)
	}

	// The same results and error objects as /mcp.
	readStdio := structured(t, call(stdio, "get_game_state", map[string]any{"game_id": gameID}))
	readHTTP := structured(t, call(viaHTTP, "get_game_state", map[string]any{"game_id": gameID}))
	if readStdio != readHTTP {
		t.Fatalf("get_game_state differs:\nstdio %s\n/mcp  %s", readStdio, readHTTP)
	}
	again := map[string]any{"game_id": gameID, "column": 3, "expected_version": 1, "idempotency_key": "s-2"}
	stdioErr := call(stdio, "make_move", again)
	sameMoveOverHTTP := call(mustConnectMCP(t, a.red), "make_move", again)
	if !stdioErr.IsError || structured(t, stdioErr) != structured(t, sameMoveOverHTTP) ||
		!strings.Contains(structured(t, stdioErr), `"NOT_YOUR_TURN"`) {
		t.Fatalf("NOT_YOUR_TURN differs:\nstdio %s\n/mcp  %s", structured(t, stdioErr), structured(t, sameMoveOverHTTP))
	}

	// Closing stdin ends the server cleanly.
	if code, stderr := wait(); code != 0 {
		t.Fatalf("connect4 mcp exited %d after stdin closed: %s", code, stderr)
	}
}

// Without a key, or with an unreachable arena, `connect4 mcp` fails at once
// with the CLI's error codes rather than serving tools that cannot work.
func TestCLIMCPNeedsKeyAndServer(t *testing.T) {
	server := newServer(t)
	run := func(env map[string]string) (int, string) {
		var stdout, stderr bytes.Buffer
		code := cli.ExecuteWithInput([]string{"mcp"}, strings.NewReader(""), &stdout, &stderr, func(k string) string { return env[k] })
		return code, stderr.String()
	}
	if code, stderr := run(map[string]string{"CONNECT4_SERVER": server.URL}); code != 4 || !strings.Contains(stderr, "UNAUTHENTICATED") {
		t.Fatalf("no key: exit %d, stderr %q; want 4 UNAUTHENTICATED", code, stderr)
	}
	key := cliKey(t, server, "Stdio")
	if code, stderr := run(map[string]string{"CONNECT4_SERVER": server.URL, "CONNECT4_API_KEY": "c4_wrong"}); code != 4 ||
		!strings.Contains(stderr, "UNAUTHENTICATED") {
		t.Fatalf("unknown key: exit %d, stderr %q; want 4 UNAUTHENTICATED", code, stderr)
	}
	if code, stderr := run(map[string]string{"CONNECT4_SERVER": "http://127.0.0.1:1", "CONNECT4_API_KEY": key}); code != 1 ||
		!strings.Contains(stderr, "cannot reach the arena server") {
		t.Fatalf("unreachable server: exit %d, stderr %q; want 1 with a reachability message", code, stderr)
	}
}
