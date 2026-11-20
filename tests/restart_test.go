package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connect4arena/internal/app"
)

// journalServer serves an arena that keeps its state in the journal at path
// (ADR-015). stop shuts it down as a server restart would; calling it again,
// or letting the test end, is harmless.
func journalServer(t *testing.T, path string) (server *httptest.Server, stop func()) {
	t.Helper()
	cfg := app.DefaultConfig()
	cfg.JournalPath = path
	handler, closeJournal, err := app.Open(cfg)
	if err != nil {
		t.Fatalf("app.Open: %v", err)
	}
	server = httptest.NewServer(handler)
	var once sync.Once
	stop = func() {
		once.Do(func() {
			server.Close()
			if err := closeJournal(); err != nil {
				t.Errorf("close journal: %v", err)
			}
		})
	}
	t.Cleanup(stop)
	return server, stop
}

// rebind points a's clients at server, keeping their API keys.
func (a *arena) rebind(server *httptest.Server) {
	a.server = server
	for _, c := range []*client{&a.red, &a.yellow, &a.other, &a.anon} {
		c.server = server
	}
}

func newJournalArena(t *testing.T, path string) (*arena, func()) {
	server, stop := journalServer(t, path)
	return &arena{
		server: server,
		red:    register(t, server, "Alpha"),
		yellow: register(t, server, "Beta"),
		other:  register(t, server, "Gamma"),
		anon:   client{t: t, server: server},
	}, stop
}

// snapshot reads everything a client can see about a's game and the lobby.
func (a *arena) snapshot(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{
		"game":    a.anon.do(http.MethodGet, "/api/v1/games/"+a.gameID, nil).wantStatus(t, http.StatusOK).body,
		"history": a.anon.do(http.MethodGet, "/api/v1/games/"+a.gameID+"/moves", nil).wantStatus(t, http.StatusOK).body,
		"lobby":   a.anon.do(http.MethodGet, "/api/v1/games", nil).wantStatus(t, http.StatusOK).body,
	}
}

func (a *arena) move(t *testing.T, mover client, column, expectedVersion int, key string) response {
	t.Helper()
	body := map[string]any{"column": column, "expectedVersion": expectedVersion}
	if key != "" {
		body["idempotencyKey"] = key
	}
	return mover.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/moves", body)
}

// BDR-018 scenario 1, AC-1..3, AT-010, API-008: an in-progress game at
// version 5 survives a restart with the same board, players, turn, version
// and history; the player to move moves with the version from before the
// restart, using the API key issued before it; and an idempotent retry of a
// move sent before the restart returns the recorded result instead of
// playing again.
func TestGamesSurviveARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	a.start(t)
	for i, column := range []int{3, 3, 4, 4, 2} {
		mover := a.red
		if i%2 == 1 {
			mover = a.yellow
		}
		a.move(t, mover, column, i, fmt.Sprintf("m%d", i+1)).wantStatus(t, http.StatusOK)
	}
	before := a.snapshot(t)
	if number(before["game"].(map[string]any)["version"]) != 5 {
		t.Fatalf("setup: version = %v, want 5", before["game"].(map[string]any)["version"])
	}
	retried := a.move(t, a.red, 2, 4, "m5").wantStatus(t, http.StatusOK).body

	stop()
	server, _ := journalServer(t, path)
	a.rebind(server)

	if after := a.snapshot(t); !reflect.DeepEqual(before, after) {
		t.Fatalf("state changed across the restart:\nbefore %v\nafter  %v", before, after)
	}
	if replay := a.move(t, a.red, 2, 4, "m5").wantStatus(t, http.StatusOK).body; !reflect.DeepEqual(replay, retried) {
		t.Fatalf("retry after restart = %v, want the recorded result %v", replay, retried)
	}
	a.move(t, a.yellow, 0, 5, "").wantStatus(t, http.StatusOK)
	a.move(t, a.yellow, 1, 5, "").wantError(t, http.StatusConflict, "NOT_YOUR_TURN")
	if v := number(a.state(t)["version"]); v != 6 {
		t.Fatalf("version after the first post-restart move = %d, want 6", v)
	}
}

// BDR-018 AC-4: completed games keep their winner and result reason, both
// for a connect-four win and a resignation.
func TestCompletedGamesSurviveARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	a.start(t)
	won := a.gameID
	a.play(t, 0, 1, 0, 1, 0, 1, 0)
	a.movesPlayed = 0
	a.start(t)
	resigned := a.gameID
	a.yellow.do(http.MethodPost, "/api/v1/games/"+resigned+"/resign", nil).wantStatus(t, http.StatusOK)

	before := map[string]map[string]any{}
	for _, id := range []string{won, resigned} {
		a.gameID = id
		before[id] = a.snapshot(t)
	}
	stop()
	server, _ := journalServer(t, path)
	a.rebind(server)

	for id, want := range map[string]string{won: "connect_four", resigned: "resignation"} {
		a.gameID = id
		after := a.snapshot(t)
		if !reflect.DeepEqual(before[id], after) {
			t.Fatalf("game %s changed across the restart:\nbefore %v\nafter  %v", id, before[id], after)
		}
		game := after["game"].(map[string]any)
		if game["status"] != "completed" || game["winner"] != "red" || game["resultReason"] != want {
			t.Fatalf("game %s after restart: status %v winner %v reason %v", id, game["status"], game["winner"], game["resultReason"])
		}
	}
}

// ADR-015 verification: of many concurrent moves for one turn, exactly one is
// journaled, and the restored game matches the live one.
func TestJournaledConcurrentMovesCommitOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	a.start(t)
	var wg sync.WaitGroup
	statuses := make([]int, 7)
	for column := range statuses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[column] = a.move(t, a.red, column, 0, fmt.Sprintf("k%d", column)).status
		}()
	}
	wg.Wait()
	accepted := 0
	for _, status := range statuses {
		if status == http.StatusOK {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("%d concurrent moves accepted, want 1 (statuses %v)", accepted, statuses)
	}
	before := a.snapshot(t)
	stop()
	raw, _ := os.ReadFile(path)
	if n := strings.Count(string(raw), `"type":"move_accepted"`); n != 1 {
		t.Fatalf("journal holds %d move_accepted events, want 1", n)
	}
	server, _ := journalServer(t, path)
	a.rebind(server)
	if after := a.snapshot(t); !reflect.DeepEqual(before, after) {
		t.Fatalf("restored game differs:\nbefore %v\nafter  %v", before, after)
	}
}

// ADR-015 verification: a journal cut off mid-event restores every complete
// event and accepts the next move.
func TestTornJournalRestoresAndAcceptsTheNextMove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	a.start(t)
	a.play(t, 3, 4)
	before := a.snapshot(t)
	stop()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"seq":99,"at":"2026-11-20T00:00:00Z","type":"move_acc`)
	_ = f.Close()

	server, _ := journalServer(t, path)
	a.rebind(server)
	if after := a.snapshot(t); !reflect.DeepEqual(before, after) {
		t.Fatalf("restored game differs:\nbefore %v\nafter  %v", before, after)
	}
	a.play(t, 2)
}

// ADR-015 step 4: an event the service cannot apply stops start-up rather
// than being skipped.
func TestJournalThatDoesNotReplayRefusesToStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	a.start(t)
	a.play(t, 3)
	stop()
	raw, _ := os.ReadFile(path)
	// Make the journaled move a different player's, so replay rejects it.
	broken := strings.Replace(string(raw), `"type":"move_accepted","data":{"gameId":"`+a.gameID+`","playerId":"agt_`,
		`"type":"move_accepted","data":{"gameId":"`+a.gameID+`","playerId":"agt_x`, 1)
	if broken == string(raw) {
		t.Fatalf("test setup: no move_accepted event to break in %s", raw)
	}
	if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := app.DefaultConfig()
	cfg.JournalPath = path
	if _, closeJournal, err := app.Open(cfg); err == nil {
		_ = closeJournal()
		t.Fatal("started from a journal whose move does not replay")
	} else if !strings.Contains(err.Error(), "seq") {
		t.Fatalf("error %q does not say which event failed", err)
	}
}

// ADR-014, ADR-015 step 6: Arena Bot keeps one identity across restarts and
// carries on with its in-progress house games.
func TestHouseAgentResumesAfterARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	human := a.red
	created := human.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
		wantStatus(t, http.StatusCreated)
	a.gameID = created.body["gameId"].(string)
	human.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Human", "type": "human"}).wantStatus(t, http.StatusOK)
	a.move(t, human, 0, 0, "").wantStatus(t, http.StatusOK)
	state := waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) == 2 })
	botID := state["players"].(map[string]any)["yellow"].(map[string]any)["id"]

	stop()
	server, _ := journalServer(t, path)
	a.rebind(server)
	human = a.red

	a.move(t, human, 0, 2, "").wantStatus(t, http.StatusOK)
	waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) == 4 })

	created = human.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
		wantStatus(t, http.StatusCreated)
	a.gameID = created.body["gameId"].(string)
	state = human.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Human", "type": "human"}).wantStatus(t, http.StatusOK).body
	if id := state["players"].(map[string]any)["yellow"].(map[string]any)["id"]; id != botID {
		t.Fatalf("Arena Bot is %v after the restart, was %v", id, botID)
	}
	stop()
	raw, _ := os.ReadFile(path)
	registrations := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, `"type":"agent_registered"`) && strings.Contains(line, `"displayName":"Arena Bot"`) {
			registrations++
		}
	}
	if registrations != 1 {
		t.Fatalf("Arena Bot registered %d times, want once", registrations)
	}
	if strings.Contains(string(raw), "c4_") {
		t.Fatal("the journal contains an API key")
	}
}
