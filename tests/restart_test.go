package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
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

// BDR-018, ADR-014: a house game that nobody has joined yet keeps its Arena
// Bot seat across a restart, so the human who joins afterwards still gets the
// bot as the opponent and the bot plays.
func TestHouseSeatSurvivesARestartBeforeAnyoneJoins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	created := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
		wantStatus(t, http.StatusCreated)
	a.gameID = created.body["gameId"].(string)

	stop()
	server, _ := journalServer(t, path)
	a.rebind(server)

	state := a.red.do(http.MethodPost, "/api/v1/games/"+a.gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Human", "type": "human"}).wantStatus(t, http.StatusOK).body
	if state["status"] != "in_progress" {
		t.Fatalf("after the restart the house game is %v once the human joins, want in_progress", state["status"])
	}
	if name := state["players"].(map[string]any)["yellow"].(map[string]any)["displayName"]; name != "Arena Bot" {
		t.Fatalf("yellow is %v, want Arena Bot", name)
	}
	a.move(t, a.red, 3, 0, "").wantStatus(t, http.StatusOK)
	waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) == 2 })
}

// writeJournal writes events as journal lines (ADR-015), numbered from 1.
func writeJournal(t *testing.T, path string, at time.Time, events ...string) {
	t.Helper()
	var lines strings.Builder
	for i, e := range events {
		fmt.Fprintf(&lines, `{"seq":%d,"at":%q,"type":%s}`+"\n", i+1, at.UTC().Format(time.RFC3339Nano), e)
	}
	if err := os.WriteFile(path, []byte(lines.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A crash after a human takes the first seat of a house game but before
// Arena Bot takes the second leaves a waiting game with one player; on
// restart the bot takes the open seat.
func TestHouseBotTakesTheSeatItMissedInACrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	writeJournal(t, path, time.Now(),
		`"game_created","data":{"gameId":"g_crash","mode":"human_vs_agent","house":true}`,
		`"player_joined","data":{"gameId":"g_crash","playerId":"a_human","disc":"red","displayName":"Human","playerType":"human"}`)
	server, _ := journalServer(t, path)
	a := &arena{server: server, anon: client{t: t, server: server}, gameID: "g_crash"}

	state := waitFor(t, a, time.Second, func(s map[string]any) bool { return s["status"] == "in_progress" })
	if name := state["players"].(map[string]any)["yellow"].(map[string]any)["displayName"]; name != "Arena Bot" {
		t.Fatalf("yellow is %v, want Arena Bot", name)
	}
}

// Arena Bot gives up a house game nobody joins within 10 minutes; a restart
// does not give the game a fresh 10 minutes.
func TestExpiredHouseSeatIsNotOfferedAfterARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	writeJournal(t, path, time.Now().Add(-11*time.Minute),
		`"game_created","data":{"gameId":"g_old","mode":"human_vs_agent","house":true}`)
	server, _ := journalServer(t, path)
	a := &arena{server: server, anon: client{t: t, server: server}, gameID: "g_old"}
	a.red = register(t, server, "Alpha")

	state := a.red.do(http.MethodPost, "/api/v1/games/g_old/join",
		map[string]any{"disc": "red", "displayName": "Human", "type": "human"}).wantStatus(t, http.StatusOK).body
	if state["status"] != "waiting" || state["players"].(map[string]any)["yellow"] != nil {
		t.Fatalf("an expired house game was seated: status %v, players %v", state["status"], state["players"])
	}
}

// Only human_vs_agent games have a house opponent; a journal that says
// otherwise does not replay, so the arena refuses to start (ADR-015).
func TestJournalWithAHouseAgentVsAgentGameRefusesToStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	writeJournal(t, path, time.Now(),
		`"game_created","data":{"gameId":"g_bad","mode":"agent_vs_agent","house":true}`)
	cfg := app.DefaultConfig()
	cfg.JournalPath = path
	if _, _, err := app.Open(cfg); err == nil || !strings.Contains(err.Error(), "event seq 1") {
		t.Fatalf("app.Open = %v, want an error naming event seq 1", err)
	}
}

// The house offer is journaled with the game it belongs to, and only there.
func TestHouseOfferIsJournaledWithTheGame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	a, stop := newJournalArena(t, path)
	house := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent", "opponent": "house"}).
		wantStatus(t, http.StatusCreated).body["gameId"].(string)
	plain := a.red.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent"}).
		wantStatus(t, http.StatusCreated).body["gameId"].(string)
	stop()
	raw, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, `"type":"game_created"`) {
			continue
		}
		switch {
		case strings.Contains(line, house) && !strings.Contains(line, `"house":true`):
			t.Fatalf("the house game's game_created line has no house offer: %s", line)
		case strings.Contains(line, plain) && strings.Contains(line, `"house"`):
			t.Fatalf("a plain game's game_created line mentions house: %s", line)
		}
	}
}

// Arena Bot plays a house game it takes during start-up with one loop only.
// Two loops would both send a move for the bot's turn; the service would
// refuse the later one (NOT_YOUR_TURN or STALE_STATE) and log move_rejected.
// A single loop moves only on fresh state, so it is never refused.
func TestHouseBotSeatedAtStartUpPlaysOnce(t *testing.T) {
	const key = "c4_restart-test-human-key"
	hash := sha256.Sum256([]byte(key))
	path := filepath.Join(t.TempDir(), "arena.jsonl")
	writeJournal(t, path, time.Now(),
		`"agent_registered","data":{"agentId":"agt_human","displayName":"Human","type":"human","keyHash":"`+hex.EncodeToString(hash[:])+`"}`,
		`"game_created","data":{"gameId":"g_once","mode":"human_vs_agent","house":true}`,
		`"player_joined","data":{"gameId":"g_once","playerId":"agt_human","disc":"red","displayName":"Human","playerType":"human"}`)
	sink := &logSink{}
	cfg := app.DefaultConfig()
	cfg.JournalPath = path
	cfg.Logger = slog.New(slog.NewJSONHandler(sink, nil))
	handler, closeJournal, err := app.Open(cfg)
	if err != nil {
		t.Fatalf("app.Open: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close(); _ = closeJournal() })
	a := &arena{server: server, anon: client{t: t, server: server}, gameID: "g_once",
		red: client{t: t, server: server, token: key}}

	state := waitFor(t, a, time.Second, func(s map[string]any) bool { return s["status"] == "in_progress" })
	botMoves := 0
	for botMoves < 6 && state["status"] == "in_progress" {
		version := number(state["version"])
		a.move(t, a.red, ints(state["legalMoves"])[0], version, "").wantStatus(t, http.StatusOK)
		state = waitFor(t, a, time.Second, func(s map[string]any) bool {
			return number(s["version"]) == version+2 || s["status"] == "completed"
		})
		if number(state["version"]) == version+2 {
			botMoves++
		}
	}
	if botMoves < 3 {
		t.Fatalf("Arena Bot moved %d times, want at least 3 to tell one loop from two", botMoves)
	}
	time.Sleep(100 * time.Millisecond) // let a second loop, if any, send its stale move
	sink.wantEvents(t, "g_once", "move_rejected", 0)
}
