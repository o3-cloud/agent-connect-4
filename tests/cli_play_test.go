package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"connect4arena/internal/cli"
)

// cliRun runs the connect4 CLI in-process against server with an optional API
// key, and returns its exit code, stdout and stderr.
func cliRun(t *testing.T, server *httptest.Server, key string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	env := map[string]string{"CONNECT4_SERVER": server.URL, "CONNECT4_API_KEY": key}
	code := cli.Execute(args, &stdout, &stderr, func(k string) string { return env[k] })
	return code, stdout.String(), stderr.String()
}

// cliJSON runs a --json command and decodes its stdout, failing unless stdout
// is exactly one JSON document (BDR-019 AC-2).
func cliJSON(t *testing.T, server *httptest.Server, key string, args ...string) (int, map[string]any) {
	t.Helper()
	code, stdout, _ := cliRun(t, server, key, append(args, "--json")...)
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var doc map[string]any
	if err := decoder.Decode(&doc); err != nil {
		t.Fatalf("connect4 %v: stdout is not JSON: %v\n%s", args, err, stdout)
	}
	if decoder.More() {
		t.Fatalf("connect4 %v: stdout has more than one JSON document:\n%s", args, stdout)
	}
	return code, doc
}

func cliKey(t *testing.T, server *httptest.Server, name string) string {
	t.Helper()
	code, doc := cliJSON(t, server, "", "register", "--name", name)
	key, _ := doc["apiKey"].(string)
	if code != 0 || !strings.HasPrefix(key, "c4_") {
		t.Fatalf("register: exit %d, %v", code, doc)
	}
	return key
}

// BDR-019 scenario 1 and AC-1..3: create, join, state and move through the CLI,
// with one JSON document on stdout and exit 0.
func TestCLIPlaysAGameWithJSONOutput(t *testing.T) {
	server := newServer(t)
	red, yellow := cliKey(t, server, "Red"), cliKey(t, server, "Yellow")

	code, created := cliJSON(t, server, red, "create", "--mode", "agent_vs_agent")
	game, _ := created["gameId"].(string)
	if code != 0 || game == "" || created["status"] != "waiting" {
		t.Fatalf("create: exit %d, %v", code, created)
	}
	if code, joined := cliJSON(t, server, red, "join", game, "--disc", "red", "--name", "Red"); code != 0 || joined["status"] != "waiting" {
		t.Fatalf("join red: exit %d, %v", code, joined)
	}
	if code, joined := cliJSON(t, server, yellow, "join", game, "--disc", "yellow", "--name", "Yellow"); code != 0 || joined["status"] != "in_progress" {
		t.Fatalf("join yellow: exit %d, %v", code, joined)
	}

	moves := []struct {
		key    string
		column int
	}{{red, 0}, {yellow, 0}, {red, 1}, {yellow, 1}, {red, 2}, {yellow, 2}, {red, 3}}
	var last map[string]any
	for _, m := range moves {
		code, last = cliJSON(t, server, m.key, "move", game, "--column", strconv.Itoa(m.column))
		if code != 0 || last["accepted"] != true {
			t.Fatalf("move %d: exit %d, %v", m.column, code, last)
		}
	}
	if last["status"] != "completed" || last["winner"] != "red" {
		t.Fatalf("final move = %v", last)
	}

	code, state := cliJSON(t, server, "", "state", game)
	if code != 0 || state["status"] != "completed" || number(state["version"]) != 7 {
		t.Fatalf("state: exit %d, %v", code, state)
	}
	code, history := cliJSON(t, server, "", "history", game)
	if code != 0 || len(history["moves"].([]any)) != 7 {
		t.Fatalf("history: exit %d, %v", code, history)
	}
	for _, out := range []map[string]any{created, state, history, last} {
		raw, _ := json.Marshal(out)
		if strings.Contains(string(raw), red) || strings.Contains(string(raw), yellow) {
			t.Fatalf("an API key appears in CLI output: %s", raw) // BDR-013 AC-4
		}
	}
}

// BDR-019 scenario 2 and AC-3..4: failures exit with the SPEC.md §16 code and,
// with --json, print the server's error object on stdout.
func TestCLIFailuresUseSpecExitCodes(t *testing.T) {
	server := newServer(t)
	red, yellow := cliKey(t, server, "Red"), cliKey(t, server, "Yellow")
	_, created := cliJSON(t, server, red, "create")
	game := created["gameId"].(string)
	cliJSON(t, server, red, "join", game, "--disc", "red", "--name", "Red")
	cliJSON(t, server, yellow, "join", game, "--disc", "yellow", "--name", "Yellow")

	cases := []struct {
		name string
		key  string
		args []string
		exit int
		code string
	}{
		{"not your turn", yellow, []string{"move", game, "--column", "3"}, 6, "NOT_YOUR_TURN"},
		{"bad column", red, []string{"move", game, "--column", "9"}, 3, "INVALID_COLUMN"},
		{"no key", "", []string{"move", game, "--column", "3"}, 4, "UNAUTHENTICATED"},
		{"made-up key", "token-alpha", []string{"create"}, 4, "UNAUTHENTICATED"},
		{"unknown game", "", []string{"state", "game_missing"}, 5, "GAME_NOT_FOUND"},
		{"missing argument", red, []string{"move", game}, 2, "INVALID_ARGUMENT"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, doc := cliJSON(t, server, c.key, c.args...)
			errObject, _ := doc["error"].(map[string]any)
			if code != c.exit || errObject["code"] != c.code {
				t.Fatalf("exit %d error %v, want exit %d code %s", code, doc, c.exit, c.code)
			}
		})
	}

	// Without --json the error goes to stderr and stdout stays empty.
	code, stdout, stderr := cliRun(t, server, yellow, "move", game, "--column", "3")
	if code != 6 || stdout != "" || !strings.Contains(stderr, "NOT_YOUR_TURN") {
		t.Fatalf("text-mode failure: exit %d stdout %q stderr %q", code, stdout, stderr)
	}

	// A server that cannot be reached is INTERNAL, exit 1.
	var out, errOut bytes.Buffer
	env := map[string]string{"CONNECT4_SERVER": "http://127.0.0.1:1"}
	if code := cli.Execute([]string{"state", game, "--json"}, &out, &errOut, func(k string) string { return env[k] }); code != 1 ||
		!strings.Contains(out.String(), `"INTERNAL"`) {
		t.Fatalf("unreachable server: exit %d stdout %s", code, out.String())
	}
}

// Text output is for humans: create prints the game id and state draws the board.
func TestCLITextOutput(t *testing.T) {
	server := newServer(t)
	red := cliKey(t, server, "Red")
	code, stdout, stderr := cliRun(t, server, red, "create")
	game := strings.TrimSpace(stdout)
	if code != 0 || !strings.HasPrefix(game, "game_") || stderr != "" {
		t.Fatalf("create: exit %d stdout %q stderr %q", code, stdout, stderr)
	}
	code, stdout, _ = cliRun(t, server, "", "state", game)
	if code != 0 || !strings.Contains(stdout, "waiting") || strings.Count(stdout, ". . . . . . .") != 6 {
		t.Fatalf("state: exit %d\n%s", code, stdout)
	}
	code, stdout, _ = cliRun(t, server, "", "register", "--name", "Blue")
	if code != 0 || !strings.Contains(stdout, "export CONNECT4_API_KEY=c4_") {
		t.Fatalf("register: exit %d\n%s", code, stdout)
	}
}
