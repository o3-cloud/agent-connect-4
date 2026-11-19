package tests

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"connect4arena/internal/engine"
)

// BDR-017 scenario 1, AC-1..2: the move history is a replay that reproduces
// the final board and result from an empty board.
func TestReplayHistoryReproducesFinalBoard(t *testing.T) {
	a := newArena(t)
	a.start(t)
	a.play(t, 0, 0, 1, 1, 2, 2, 3)
	final := a.state(t)
	history := a.history(t)

	players, _ := history["players"].(map[string]any)
	red, _ := players["red"].(map[string]any)
	yellow, _ := players["yellow"].(map[string]any)
	if history["mode"] != "agent_vs_agent" || red["displayName"] != "Alpha" || yellow["displayName"] != "Beta" {
		t.Fatalf("replay lacks initial configuration: mode=%v players=%v", history["mode"], history["players"])
	}
	if history["winner"] != "red" || history["resultReason"] != "connect_four" || history["status"] != "completed" {
		t.Fatalf("replay result: winner=%v reason=%v status=%v", history["winner"], history["resultReason"], history["status"])
	}

	board := engine.NewBoard()
	var result engine.Result
	for i, m := range history["moves"].([]any) {
		move := m.(map[string]any)
		if number(move["moveNumber"]) != i+1 {
			t.Fatalf("move %d has moveNumber %v", i+1, move["moveNumber"])
		}
		row, r, err := board.Drop(number(move["column"]), engine.Disc(move["disc"].(string)))
		if err != nil || row != number(move["row"]) {
			t.Fatalf("replaying move %d: row %d err %v, recorded row %v", i+1, row, err, move["row"])
		}
		result = r
	}
	cells := board.Cells()
	for row := range engine.Rows {
		for col := range engine.Columns {
			want, _ := cell(final, row, col).(string) // empty cells are null
			if got := string(cells[row][col]); got != want {
				t.Fatalf("replayed cell %d-%d = %q, final board has %q", row, col, got, want)
			}
		}
	}
	if !result.Over || string(result.Winner) != "red" || string(result.Reason) != "connect_four" {
		t.Fatalf("replayed result = %+v, want red by connect_four", result)
	}
}

// BDR-017 scenario 2, AC-3: the same columns give the same game.
func TestReplayIsDeterministicAcrossGames(t *testing.T) {
	g1, g2 := newArena(t), newArena(t)
	for _, a := range []*arena{g1, g2} {
		a.start(t)
		a.play(t, 3, 3, 4, 4, 5, 2, 6)
	}
	s1, s2 := stable(g1.state(t)), stable(g2.state(t))
	for _, field := range []string{"board", "winner", "resultReason", "status", "version"} {
		if !jsonEqual(s1[field], s2[field]) {
			t.Fatalf("%s differs: %v vs %v", field, s1[field], s2[field])
		}
	}
	if s1["winner"] != "red" || s1["resultReason"] != "connect_four" {
		t.Fatalf("result = %v by %v, want red by connect_four", s1["winner"], s1["resultReason"])
	}
}

var replayCell = regexp.MustCompile(`data-cell="(\d)-(\d)" data-disc="(\w*)"`)

// discsOf returns the replay page's board as row-col -> disc for filled cells.
func discsOf(t *testing.T, page string) map[string]string {
	t.Helper()
	cells := replayCell.FindAllStringSubmatch(page, -1)
	if len(cells) != engine.Rows*engine.Columns {
		t.Fatalf("replay board has %d cells, want 42:\n%s", len(cells), page)
	}
	out := map[string]string{}
	for _, c := range cells {
		if c[3] != "" {
			out[c[1]+"-"+c[2]] = c[3]
		}
	}
	return out
}

// BDR-017 scenario 3, AC-4: step through a completed game in the browser.
func TestBrowserReplayStepsThroughGame(t *testing.T) {
	b, a := newBrowser(t)
	a.start(t)
	a.play(t, 0, 0, 1, 1, 2, 2, 3)
	final := a.state(t)
	replay := "/games/" + a.gameID + "/replay"

	status, page := b.get(replay)
	if status != http.StatusOK || len(discsOf(t, page)) != 0 || !strings.Contains(page, "Move 0 of 7") {
		t.Fatalf("replay start (status %d) is not the empty board at move 0 of 7:\n%s", status, page)
	}
	if !strings.Contains(page, `href="`+replay+`?step=1"`) || strings.Contains(page, `hx-trigger="every`) {
		t.Fatalf("replay start lacks a next-step link or polls:\n%s", page)
	}

	for step := 1; step <= 7; step++ {
		_, page = b.get(replay + "?step=" + string(rune('0'+step)))
		discs := discsOf(t, page)
		if len(discs) != step {
			t.Fatalf("after step %d the board shows %d discs", step, len(discs))
		}
		if !strings.Contains(page, "Move "+string(rune('0'+step))+" of 7") {
			t.Fatalf("step %d page lacks its position:\n%s", step, page)
		}
		if step == 1 && discs["5-0"] != "red" {
			t.Fatalf("after step 1 the board is %v, want red at 5-0", discs)
		}
	}
	discs := discsOf(t, page)
	for row := range engine.Rows {
		for col := range engine.Columns {
			key := string(rune('0'+row)) + "-" + string(rune('0'+col))
			if want, _ := cell(final, row, col).(string); discs[key] != want {
				t.Fatalf("final replay cell %s = %q, game board has %q", key, discs[key], want)
			}
		}
	}
	if !strings.Contains(page, "Alpha wins") || strings.Contains(page, `href="`+replay+`?step=8"`) ||
		!strings.Contains(page, `href="`+replay+`?step=6"`) {
		t.Fatalf("final replay step lacks the result or has wrong links:\n%s", page)
	}
	if moveControl.MatchString(page) {
		t.Fatal("replay offers move controls")
	}
}

func TestBrowserReplayClampsStepsAndLinksFromGame(t *testing.T) {
	b, a := newBrowser(t)
	a.start(t)
	a.play(t, 0, 0, 1, 1, 2, 2, 3)
	replay := "/games/" + a.gameID + "/replay"

	for query, want := range map[string]string{"?step=99": "Move 7 of 7", "?step=-3": "Move 0 of 7", "?step=x": "Move 0 of 7"} {
		if _, page := b.get(replay + query); !strings.Contains(page, want) {
			t.Fatalf("%s: page lacks %q", query, want)
		}
	}
	if status, _ := b.get("/games/game_missing/replay"); status != http.StatusNotFound {
		t.Fatalf("unknown game replay status = %d, want 404", status)
	}
	if _, page := b.get("/games/" + a.gameID); !strings.Contains(page, `href="`+replay+`"`) {
		t.Fatalf("completed game page has no replay link:\n%s", page)
	}

	// An in-progress game has no replay link yet.
	a2 := newArena(t)
	b2 := &browser{t: t, base: a2.server.URL, client: b.client}
	a2.start(t)
	if _, page := b2.get("/games/" + a2.gameID); strings.Contains(page, "/replay") {
		t.Fatalf("in-progress game page links to a replay:\n%s", page)
	}
}
