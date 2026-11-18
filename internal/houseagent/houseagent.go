// Package houseagent is Arena Bot, the house reference agent (ADR-014,
// AGENT-007). It is an ordinary API client: it registers its own API key and
// plays only through the public REST API, so it follows exactly the rules and
// checks every external agent does. It never imports the game service.
package houseagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// DisplayName is the name Arena Bot shows to other players.
const DisplayName = "Arena Bot"

const (
	pollInterval = 25 * time.Millisecond
	joinWait     = 10 * time.Minute // give up if no human joins
	gameLimit    = 6 * time.Hour    // stop watching an abandoned game
)

// Agent plays house games against the API served by handler.
type Agent struct {
	client *http.Client
	base   string

	once   sync.Once
	key    string
	regErr error
}

// New returns an agent that reaches the API through handler in-process: each
// request still passes through routing, authentication and validation, with
// no network hop.
func New(handler http.Handler) *Agent {
	return &Agent{client: &http.Client{Transport: inProcess{handler}}, base: "http://arena/api/v1"}
}

type inProcess struct{ handler http.Handler }

func (t inProcess) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, r)
	return recorder.Result(), nil
}

// Start plays gameID in the background. A failure ends only that game's
// goroutine (AGENT-005).
func (a *Agent) Start(gameID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("house agent stopped", "gameId", gameID, "panic", r)
			}
		}()
		if err := a.play(gameID); err != nil {
			slog.Warn("house agent stopped", "gameId", gameID, "error", err)
		}
	}()
}

type gameState struct {
	Status      string     `json:"status"`
	Board       [][]string `json:"board"`
	OpenSlots   []string   `json:"openSlots"`
	CurrentTurn string     `json:"currentTurn"`
	LegalMoves  []int      `json:"legalMoves"`
	Version     int        `json:"version"`
}

func (a *Agent) play(gameID string) error {
	a.once.Do(func() {
		var out struct {
			APIKey string `json:"apiKey"`
		}
		a.regErr = a.call(http.MethodPost, "/agents", map[string]string{"displayName": DisplayName, "type": "agent"}, &out)
		a.key = out.APIKey
	})
	if a.regErr != nil {
		return fmt.Errorf("register: %w", a.regErr)
	}

	// Wait for the human to take a slot, then take the other one.
	var state gameState
	deadline := time.Now().Add(joinWait)
	for {
		if err := a.call(http.MethodGet, "/games/"+gameID, nil, &state); err != nil {
			return err
		}
		if state.Status != "waiting" {
			return nil // someone else filled the game
		}
		if len(state.OpenSlots) == 1 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no human joined within %s", joinWait)
		}
		time.Sleep(pollInterval)
	}
	me := state.OpenSlots[0]
	join := map[string]string{"disc": me, "displayName": DisplayName, "type": "agent"}
	if err := a.call(http.MethodPost, "/games/"+gameID+"/join", join, nil); err != nil {
		return fmt.Errorf("join: %w", err)
	}

	stop := time.Now().Add(gameLimit)
	for time.Now().Before(stop) {
		if err := a.call(http.MethodGet, "/games/"+gameID, nil, &state); err != nil {
			return err
		}
		switch {
		case state.Status == "completed":
			return nil
		case state.CurrentTurn == me && len(state.LegalMoves) > 0:
			move := map[string]int{"column": Choose(state.Board, state.LegalMoves, me), "expectedVersion": state.Version}
			// A stale or rejected move is retried from fresh state on the next pass.
			_ = a.call(http.MethodPost, "/games/"+gameID+"/moves", move, nil)
		default:
			time.Sleep(pollInterval)
		}
	}
	return fmt.Errorf("game still running after %s", gameLimit)
}

func (a *Agent) call(method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	request, err := http.NewRequest(method, a.base+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if a.key != "" {
		request.Header.Set("Authorization", "Bearer "+a.key)
	}
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= 400 {
		return fmt.Errorf("%s %s: HTTP %d", method, path, response.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(out)
}

// centreOrder ranks columns by distance from the centre.
var centreOrder = []int{3, 2, 4, 1, 5, 0, 6}

// Choose picks a column for me ("red" or "yellow") from the legal moves: a
// winning move if there is one, else a move that blocks the opponent's
// immediate win, else the legal column nearest the centre. Row 0 of board is
// the top; empty cells are "". It is deterministic for a given position.
func Choose(board [][]string, legal []int, me string) int {
	opponent := "red"
	if me == "red" {
		opponent = "yellow"
	}
	for _, disc := range []string{me, opponent} {
		for _, column := range legal {
			if wins(board, column, disc) {
				return column
			}
		}
	}
	allowed := map[int]bool{}
	for _, column := range legal {
		allowed[column] = true
	}
	for _, column := range centreOrder {
		if allowed[column] {
			return column
		}
	}
	return legal[0]
}

// wins reports whether dropping disc into column makes four in a row.
func wins(board [][]string, column int, disc string) bool {
	row := -1
	for r := len(board) - 1; r >= 0; r-- {
		if board[r][column] == "" {
			row = r
			break
		}
	}
	if row < 0 {
		return false
	}
	at := func(r, c int) bool {
		if r == row && c == column {
			return true
		}
		return r >= 0 && r < len(board) && c >= 0 && c < len(board[r]) && board[r][c] == disc
	}
	for _, d := range [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} {
		count := 1
		for _, sign := range []int{1, -1} {
			for step := 1; step < 4 && at(row+sign*step*d[0], column+sign*step*d[1]); step++ {
				count++
			}
		}
		if count >= 4 {
			return true
		}
	}
	return false
}
