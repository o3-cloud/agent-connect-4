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
	joinWait     = 10 * time.Minute // forget a house game nobody joins
	gameLimit    = 6 * time.Hour    // stop watching an abandoned game
)

// Agent plays house games against the API served by handler.
type Agent struct {
	client *http.Client
	base   string

	once   sync.Once
	key    string
	regErr error

	mu      sync.Mutex
	pending map[string]bool // house games whose bot has not joined yet
}

// New returns an agent that reaches the API through handler in-process: each
// request still passes through routing, authentication and validation, with
// no network hop.
func New(handler http.Handler) *Agent {
	return &Agent{
		client:  &http.Client{Transport: inProcess{handler}},
		base:    "http://arena/api/v1",
		pending: map[string]bool{},
	}
}

type inProcess struct{ handler http.Handler }

func (t inProcess) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, r)
	return recorder.Result(), nil
}

// Start makes Arena Bot the opponent in gameID, a new house game. The bot
// takes the other slot when a player joins (see Seat), or gives up the game if
// nobody joins within joinWait.
func (a *Agent) Start(gameID string) {
	a.mu.Lock()
	a.pending[gameID] = true
	a.mu.Unlock()
	time.AfterFunc(joinWait, func() {
		a.mu.Lock()
		delete(a.pending, gameID)
		a.mu.Unlock()
	})
}

// Seat takes the open slot of gameID if it is a house game waiting for its
// bot, then plays the game in the background. The service calls it after a
// player takes the first slot and before that player's join returns
// (game.WithAwaitingOpponent), so the join response already shows the game
// in progress. For any other game it does nothing. A failure ends only that
// game (AGENT-005).
func (a *Agent) Seat(gameID string) {
	a.mu.Lock()
	house := a.pending[gameID]
	delete(a.pending, gameID)
	a.mu.Unlock()
	if !house {
		return
	}
	me, err := a.join(gameID)
	if err != nil {
		slog.Warn("house agent did not join", "gameId", gameID, "error", err)
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("house agent stopped", "gameId", gameID, "panic", r)
			}
		}()
		if err := a.play(gameID, me); err != nil {
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

// join registers Arena Bot once and takes the game's one open slot.
func (a *Agent) join(gameID string) (string, error) {
	a.once.Do(func() {
		var out struct {
			APIKey string `json:"apiKey"`
		}
		a.regErr = a.call(http.MethodPost, "/agents", map[string]string{"displayName": DisplayName, "type": "agent"}, &out)
		a.key = out.APIKey
	})
	if a.regErr != nil {
		return "", fmt.Errorf("register: %w", a.regErr)
	}
	var state gameState
	if err := a.call(http.MethodGet, "/games/"+gameID, nil, &state); err != nil {
		return "", err
	}
	if state.Status != "waiting" || len(state.OpenSlots) != 1 {
		return "", fmt.Errorf("game is %s with %d open slots", state.Status, len(state.OpenSlots))
	}
	me := state.OpenSlots[0]
	join := map[string]string{"disc": me, "displayName": DisplayName, "type": "agent"}
	if err := a.call(http.MethodPost, "/games/"+gameID+"/join", join, nil); err != nil {
		return "", fmt.Errorf("join: %w", err)
	}
	return me, nil
}

// play moves for me until the game ends.
func (a *Agent) play(gameID, me string) error {
	var state gameState
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
