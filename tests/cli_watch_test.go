package tests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connect4arena/internal/cli"
)

// watcher is a running `connect4 watch` whose stdout is read line by line.
type watcher struct {
	t     *testing.T
	lines chan string
	done  chan int
}

func startWatch(t *testing.T, server *httptest.Server, args ...string) *watcher {
	t.Helper()
	reader, writer := io.Pipe()
	w := &watcher{t: t, lines: make(chan string, 64), done: make(chan int, 1)}
	env := map[string]string{"CONNECT4_SERVER": server.URL}
	go func() {
		var stderr bytes.Buffer
		code := cli.Execute(append([]string{"watch"}, args...), writer, &stderr, func(k string) string { return env[k] })
		_ = writer.Close()
		w.done <- code
	}()
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			w.lines <- scanner.Text()
		}
		close(w.lines)
	}()
	return w
}

func (w *watcher) next() string {
	w.t.Helper()
	select {
	case line, ok := <-w.lines:
		if !ok {
			w.t.Fatal("watch output ended early")
		}
		return line
	case <-time.After(5 * time.Second):
		w.t.Fatal("timed out waiting for watch output")
	}
	return ""
}

func (w *watcher) nextJSON() map[string]any {
	w.t.Helper()
	line := w.next()
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		w.t.Fatalf("watch --json line is not one JSON document: %q: %v", line, err)
	}
	return event
}

func (w *watcher) exitCode() int {
	w.t.Helper()
	select {
	case code := <-w.done:
		return code
	case <-time.After(5 * time.Second):
		w.t.Fatal("watch did not exit")
	}
	return -1
}

// BDR-020 scenario 1, AC-1..3: the winning move and the result, one JSON
// document per line, then exit 0.
func TestCLIWatchUntilRedWins(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 0, 1, 0, 1, 0, 1) // red wins with column 0

	w := startWatch(t, a.server, a.gameID, "--json", "--interval", "10ms")
	baseline := w.nextJSON()
	if baseline["event"] != "watching" || baseline["gameId"] != a.gameID || number(baseline["version"]) != 6 {
		t.Fatalf("first line = %v, want a watching event at version 6", baseline)
	}

	a.play(t, 0)

	move := w.nextJSON()
	m, _ := move["move"].(map[string]any)
	if move["event"] != "move" || move["gameId"] != a.gameID || number(move["version"]) != 7 ||
		number(m["moveNumber"]) != 7 || m["disc"] != "red" || number(m["column"]) != 0 || number(m["row"]) != 2 {
		t.Fatalf("move line = %v", move)
	}
	completed := w.nextJSON()
	if completed["event"] != "game_completed" || completed["winner"] != "red" || completed["resultReason"] != "connect_four" {
		t.Fatalf("result line = %v", completed)
	}
	if code := w.exitCode(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if extra, ok := <-w.lines; ok {
		t.Fatalf("output after the result: %q", extra)
	}
}

// AC-1: every move accepted after watch starts, in order, and none from before.
func TestCLIWatchPrintsMovesInOrder(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 3) // before watch starts: not printed

	w := startWatch(t, a.server, a.gameID, "--json", "--interval", "10ms")
	if baseline := w.nextJSON(); number(baseline["version"]) != 1 {
		t.Fatalf("baseline = %v", baseline)
	}
	a.play(t, 4, 5, 6)
	for want := 2; want <= 4; want++ {
		event := w.nextJSON()
		if event["event"] != "move" || number(event["move"].(map[string]any)["moveNumber"]) != want {
			t.Fatalf("event = %v, want move %d", event, want)
		}
	}

	// A resignation completes the game without a move.
	a.yellow.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantStatus(t, http.StatusOK)
	completed := w.nextJSON()
	if completed["event"] != "game_completed" || completed["winner"] != "red" || completed["resultReason"] != "resignation" {
		t.Fatalf("result = %v", completed)
	}
	if code := w.exitCode(); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
}

// Text output, and a game that is already over.
func TestCLIWatchTextAndCompletedGame(t *testing.T) {
	a := newArena(t)
	a.start(t)
	w := startWatch(t, a.server, a.gameID, "--interval", "10ms")
	if line := w.next(); !strings.Contains(line, "Watching "+a.gameID) {
		t.Fatalf("first line = %q", line)
	}
	a.play(t, 2)
	if line := w.next(); !strings.Contains(line, "1. red column 2") {
		t.Fatalf("move line = %q", line)
	}
	a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/resign", nil).wantStatus(t, http.StatusOK)
	if line := w.next(); !strings.Contains(line, "yellow") || !strings.Contains(line, "resignation") {
		t.Fatalf("result line = %q", line)
	}
	if code := w.exitCode(); code != 0 {
		t.Fatalf("exit code = %d", code)
	}

	// Watching a completed game prints the result and exits 0 at once.
	code, stdout, _ := cliRun(t, a.server, "", "watch", a.gameID, "--json")
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var last map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil || code != 0 || last["event"] != "game_completed" {
		t.Fatalf("watch completed game: exit %d, output %q", code, stdout)
	}
}

func TestCLIWatchErrors(t *testing.T) {
	server := newServer(t)
	code, doc := cliJSON(t, server, "", "watch", "game_missing")
	if code != 5 || doc["error"].(map[string]any)["code"] != "GAME_NOT_FOUND" {
		t.Fatalf("unknown game: exit %d, %v", code, doc)
	}
	if code, _, stderr := cliRun(t, server, "", "watch", "g", "--interval", "0s"); code != 2 || !strings.Contains(stderr, "INVALID_ARGUMENT") {
		t.Fatalf("zero interval: exit %d, stderr %q", code, stderr)
	}
	if code, _, _ := cliRun(t, server, "", "watch"); code != 2 {
		t.Fatalf("missing game id: exit %d, want 2", code)
	}
}
