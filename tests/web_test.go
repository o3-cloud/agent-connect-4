package tests

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// browser is an HTTP client with a cookie jar, standing in for a web browser.
type browser struct {
	t      *testing.T
	base   string
	client *http.Client
}

func newBrowser(t *testing.T) (*browser, *arena) {
	a := newArena(t)
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, base: a.server.URL, client: &http.Client{Jar: jar}}, a
}

func (b *browser) get(path string) (int, string) {
	b.t.Helper()
	resp, err := b.client.Get(b.base + path)
	if err != nil {
		b.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func (b *browser) post(path string, form url.Values) (int, string, string) {
	b.t.Helper()
	req, _ := http.NewRequest(http.MethodPost, b.base+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Request.URL.Path, string(body)
}

var moveControl = regexp.MustCompile(`hx-post="/games/[^/"]+/moves/(\d)"`)

// startHumanGame plays BDR-026 AC-1: choose Human vs Agent on the dashboard
// and create the game; Arena Bot (ADR-014) takes the other slot.
func startHumanGame(t *testing.T, b *browser, a *arena, disc string) string {
	t.Helper()
	status, page := b.get("/")
	if status != http.StatusOK || !strings.Contains(page, `action="/play"`) ||
		!strings.Contains(page, `name="disc"`) || !strings.Contains(page, `name="name"`) ||
		!strings.Contains(page, "/static/htmx.min.js") {
		t.Fatalf("dashboard has no Human vs Agent setup form (status %d):\n%s", status, page)
	}
	status, path, _ := b.post("/play", url.Values{"name": {"Ada"}, "disc": {disc}})
	if status != http.StatusOK || !strings.HasPrefix(path, "/games/game_") {
		t.Fatalf("POST /play ended at %s with %d", path, status)
	}
	a.gameID = strings.TrimPrefix(path, "/games/")
	waitFor(t, a, time.Second, func(s map[string]any) bool { return s["status"] == "in_progress" })
	return a.gameID
}

// BDR-026 scenario 1 and AC-1, AC-2, AC-4.
func TestBrowserHumanPlaysFirstMove(t *testing.T) {
	b, a := newBrowser(t)
	game := startHumanGame(t, b, a, "red")

	_, page := b.get("/games/" + game)
	for _, want := range []string{"Ada", "Arena Bot", "red", "yellow", "Your turn", `hx-get="/games/` + game + `/board"`} {
		if !strings.Contains(page, want) {
			t.Fatalf("game page lacks %q (AC-4):\n%s", want, page)
		}
	}
	if got := len(moveControl.FindAllString(page, -1)); got != 7 {
		t.Fatalf("game page has %d column controls on the human's turn, want 7", got)
	}

	fragment := b.click(page, game, 3)
	if !strings.Contains(fragment, `data-cell="5-3" data-disc="red"`) {
		t.Fatalf("board after clicking column 3 has no red disc at the bottom of column 3:\n%s", fragment)
	}
	// The response is rendered before Arena Bot replies, so the human sees it is not their turn (scenario 1, AC-3).
	if moveControl.MatchString(fragment) || !strings.Contains(fragment, "Arena Bot") {
		t.Fatalf("board after the human's move still offers moves:\n%s", fragment)
	}
	state := a.state(t)
	if cell(state, 5, 3) != "red" {
		t.Fatalf("API board[5][3] = %v, want red", cell(state, 5, 3))
	}
}

// BDR-026 scenario 2 and AC-3: a click while it is not the human's turn
// submits nothing.
func TestBrowserBoardLockedOffTurn(t *testing.T) {
	b, a := newBrowser(t)
	game := startHumanGame(t, b, a, "red")
	_, page := b.get("/games/" + game)
	b.click(page, game, 3)
	waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) == 2 })

	// Now it is red's turn again; make it yellow's by having the human move,
	// then click again on the board that move returned: it must not move.
	_, page = b.get("/games/" + game + "/board")
	afterMove := b.click(page, game, 0)
	afterClick := b.click(afterMove, game, 6)
	if moveControl.MatchString(afterMove) {
		t.Fatalf("board offers moves while Arena Bot is to move:\n%s", afterMove)
	}
	state := waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) >= 4 })
	if cell(state, 5, 6) == "red" {
		t.Fatalf("an off-turn click placed a red disc in column 6: %v", state["board"])
	}
	if !strings.Contains(afterClick, "data-cell") {
		t.Fatalf("off-turn click did not return the board:\n%s", afterClick)
	}
	// A spectator in another browser never gets move controls.
	spectator, _ := newBrowser(t)
	spectator.base = b.base
	if _, page := spectator.get("/games/" + game); moveControl.MatchString(page) || !strings.Contains(page, "Spectating") {
		t.Fatalf("spectator page offers moves or is not marked as spectating:\n%s", page)
	}
}

// BDR-026 scenario 3 and AC-5: the result is shown and the board is closed.
func TestBrowserShowsResult(t *testing.T) {
	b, a := newBrowser(t)
	game := startHumanGame(t, b, a, "red")
	for i := 0; i < 21; i++ {
		state := a.state(t)
		if state["status"] == "completed" {
			break
		}
		version := number(state["version"])
		_, board := b.get("/games/" + game + "/board")
		b.click(board, game, ints(state["legalMoves"])[0])
		waitFor(t, a, time.Second, func(s map[string]any) bool {
			return s["status"] == "completed" || number(s["version"]) == version+2
		})
	}
	state := a.state(t)
	if state["status"] != "completed" {
		t.Fatalf("game did not finish: %v", state)
	}
	_, page := b.get("/games/" + game + "/board")
	if moveControl.MatchString(page) || !regexp.MustCompile(`(Arena Bot wins|You win|Draw)`).MatchString(page) {
		t.Fatalf("finished board offers moves or shows no result:\n%s", page)
	}
	if strings.Contains(page, `hx-trigger="every`) {
		t.Fatalf("finished board keeps polling:\n%s", page)
	}
	b.click(page, game, 3)
	if after := a.state(t); number(after["version"]) != number(state["version"]) {
		t.Fatal("a click after the game ended changed the game")
	}
	if strings.Contains(page, "c4_") {
		t.Fatal("an API key appears in the page")
	}
}

func TestBrowserRejectsBadSetupAndServesAssets(t *testing.T) {
	b, _ := newBrowser(t)
	if status, _, _ := b.post("/play", url.Values{"name": {""}, "disc": {"red"}}); status != http.StatusBadRequest {
		t.Fatalf("empty name: status %d, want 400", status)
	}
	if status, _, _ := b.post("/play", url.Values{"name": {"Ada"}, "disc": {"green"}}); status != http.StatusBadRequest {
		t.Fatalf("bad disc: status %d, want 400", status)
	}
	if status, _ := b.get("/games/game_missing"); status != http.StatusNotFound {
		t.Fatalf("unknown game page: status %d, want 404", status)
	}
	if status, js := b.get("/static/htmx.min.js"); status != http.StatusOK || !strings.HasPrefix(js, "var htmx=") {
		t.Fatalf("htmx not served: status %d", status)
	}
}

// BDR-030 AC-1..3: from the start page a visitor needs only a name, Play, and
// a column; no key or account; and the game page is ready for the first move
// as soon as it loads, well within 2 seconds.
func TestBrowserReadyForFirstMoveOnArrival(t *testing.T) {
	b, _ := newBrowser(t)
	start := time.Now()
	status, path, page := b.post("/play", url.Values{"name": {"Ada"}, "disc": {"red"}})
	ready := time.Since(start)
	if status != http.StatusOK || !strings.HasPrefix(path, "/games/") {
		t.Fatalf("POST /play ended at %s with %d", path, status)
	}
	if !strings.Contains(page, "Your turn") || len(moveControl.FindAllString(page, -1)) != 7 {
		t.Fatalf("the first game page is not ready for a move:\n%s", page)
	}
	if ready > 2*time.Second {
		t.Fatalf("game page took %s to be ready, want under 2s", ready)
	}
	if strings.Contains(page, "c4_") {
		t.Fatal("an API key appears in the page")
	}
}

// BDR-029 AC-3 and BDR-027 AC-4: the page shows the move history, in the
// order the moves were played.
func TestBrowserShowsMoveHistoryInOrder(t *testing.T) {
	b, a := newBrowser(t)
	game := startHumanGame(t, b, a, "red")
	_, page := b.get("/games/" + game + "/board")
	b.click(page, game, 3)
	waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) == 2 })
	_, page = b.get("/games/" + game + "/board")
	b.click(page, game, 2)
	waitFor(t, a, time.Second, func(s map[string]any) bool { return number(s["version"]) == 4 })

	_, board := b.get("/games/" + game + "/board")
	history := a.anon.do(http.MethodGet, "/api/v1/games/"+game+"/moves", nil).body["moves"].([]any)
	last := -1
	for _, m := range history {
		move := m.(map[string]any)
		entry := `data-move="` + strconv.Itoa(number(move["moveNumber"])) + `" data-disc="` + move["disc"].(string) +
			`" data-column="` + strconv.Itoa(number(move["column"])) + `"`
		at := strings.Index(board, entry)
		if at < 0 || at < last {
			t.Fatalf("history entry %s missing or out of order:\n%s", entry, board)
		}
		last = at
	}
}
