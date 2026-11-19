package tests

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var boardVersion = regexp.MustCompile(`data-version="(\d+)"`)

// click presses column on the board the browser is showing. Like the page's
// column control, it sends the version of that board, so a click on a board
// that has gone stale can be told apart from a click on the current one.
func (b *browser) click(board, gameID string, column int) string {
	b.t.Helper()
	m := boardVersion.FindStringSubmatch(board)
	if m == nil {
		b.t.Fatalf("the board shows no data-version:\n%s", board)
	}
	_, _, fragment := b.post(fmt.Sprintf("/games/%s/moves/%d", gameID, column), url.Values{"version": {m[1]}})
	return fragment
}

// humanBrowserGame starts a human_vs_agent game between a human, who plays
// red in a browser and also holds its API key, and an external agent playing
// yellow over REST.
func humanBrowserGame(t *testing.T) (b *browser, human, agent client, gameID string) {
	t.Helper()
	a := newArena(t)
	anon := client{t: t, server: a.server}
	issued := anon.do(http.MethodPost, "/api/v1/agents", map[string]any{"displayName": "Ada", "type": "human"}).
		wantStatus(t, http.StatusCreated)
	human = client{t: t, server: a.server, token: issued.body["apiKey"].(string)}
	agent = a.yellow
	created := human.do(http.MethodPost, "/api/v1/games", map[string]any{"mode": "human_vs_agent"}).wantStatus(t, http.StatusCreated)
	gameID = created.body["gameId"].(string)
	human.do(http.MethodPost, "/api/v1/games/"+gameID+"/join",
		map[string]any{"disc": "red", "displayName": "Ada", "type": "human"}).wantStatus(t, http.StatusOK)
	agent.do(http.MethodPost, "/api/v1/games/"+gameID+"/join",
		map[string]any{"disc": "yellow", "displayName": "Beta", "type": "agent"}).wantStatus(t, http.StatusOK)

	jar, _ := cookiejar.New(nil)
	base, _ := url.Parse(a.server.URL)
	jar.SetCookies(base, []*http.Cookie{{Name: "c4_session", Value: human.token, Path: "/"}})
	return &browser{t: t, base: a.server.URL, client: &http.Client{Jar: jar}}, human, agent, gameID
}

// BDR-028 Scenario 1 and AC-1..3: the browser shows the board at version 0 and
// loses its connection. Meanwhile the human moves from another device and the
// agent replies. A click on the stale board must not move (AC-2). The page's
// next refresh shows the current board, version and turn (AC-1, AC-3), and a
// click on that board moves.
func TestBrowserDoesNotMoveFromAStaleBoard(t *testing.T) {
	b, human, agent, gameID := humanBrowserGame(t)
	_, page := b.get("/games/" + gameID)
	if !strings.Contains(page, `data-version="0"`) || !strings.Contains(page, "Your turn") {
		t.Fatalf("game page is not at version 0 on the human's turn:\n%s", page)
	}
	// The column controls carry the version of the board they are drawn on.
	if !strings.Contains(page, `hx-vals='{"version": 0}'`) {
		t.Fatalf("column controls do not send the board's version:\n%s", page)
	}

	// Offline: two moves happen through the API.
	human.do(http.MethodPost, "/api/v1/games/"+gameID+"/moves", map[string]any{"column": 0, "expectedVersion": 0}).
		wantStatus(t, http.StatusOK)
	agent.do(http.MethodPost, "/api/v1/games/"+gameID+"/moves", map[string]any{"column": 6, "expectedVersion": 1}).
		wantStatus(t, http.StatusOK)

	// Back online, the human clicks the stale board before it refreshes.
	fragment := b.click(page, gameID, 3)
	state := human.do(http.MethodGet, "/api/v1/games/"+gameID, nil).body
	if number(state["version"]) != 2 || cell(state, 5, 3) != nil {
		t.Fatalf("a click on the version 0 board moved: %v", state)
	}
	for _, want := range []string{`data-version="2"`, `data-cell="5-0" data-disc="red"`, `data-cell="5-6" data-disc="yellow"`,
		"The board changed", "Your turn"} {
		if !strings.Contains(fragment, want) {
			t.Fatalf("the response to a stale click lacks %q:\n%s", want, fragment)
		}
	}

	// AC-1, AC-3: the refreshed board matches the API.
	_, refreshed := b.get("/games/" + gameID + "/board")
	if m := boardVersion.FindStringSubmatch(refreshed); m == nil || m[1] != strconv.Itoa(number(state["version"])) {
		t.Fatalf("refreshed board version does not match the API's %v:\n%s", state["version"], refreshed)
	}
	if !strings.Contains(refreshed, `data-move="2" data-disc="yellow" data-column="6"`) {
		t.Fatalf("refreshed board misses a move made while offline:\n%s", refreshed)
	}

	// A click on the current board moves.
	b.click(refreshed, gameID, 3)
	after := human.do(http.MethodGet, "/api/v1/games/"+gameID, nil).body
	if number(after["version"]) != 3 || cell(after, 5, 3) != "red" {
		t.Fatalf("a click on the current board did not move: %v", after)
	}
}

// A move request that names no version, or a malformed one, moves nothing.
func TestBrowserMoveWithoutVersionDoesNotMove(t *testing.T) {
	b, human, _, gameID := humanBrowserGame(t)
	for _, form := range []url.Values{nil, {"version": {"latest"}}} {
		_, _, fragment := b.post("/games/"+gameID+"/moves/3", form)
		if !strings.Contains(fragment, `data-version="0"`) {
			t.Fatalf("move with version %v did not return the current board:\n%s", form, fragment)
		}
	}
	if state := human.do(http.MethodGet, "/api/v1/games/"+gameID, nil).body; number(state["version"]) != 0 {
		t.Fatalf("a move without a version changed the game: %v", state)
	}
}

// While the page cannot reach the server it says so and offers no moves
// (AC-2); the first successful refresh replaces it with the current board.
func TestGamePageMarksItselfOfflineWhenRefreshFails(t *testing.T) {
	b, _, _, gameID := humanBrowserGame(t)
	_, page := b.get("/games/" + gameID)
	for _, want := range []string{"htmx:sendError", "htmx:responseError", "Reconnecting"} {
		if !strings.Contains(page, want) {
			t.Fatalf("game page has no offline handling (%q):\n%s", want, page)
		}
	}
}
