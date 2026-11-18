// Package game is the authoritative game service: lifecycle, players, turns
// and versions (GAME-*, TURN-*). It is shared by every interface (MCP-009) and
// delegates the rules to the engine package.
package game

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"connect4arena/internal/engine"
)

// Mode is a supported game configuration (GAME-002).
type Mode string

// Game modes.
const (
	ModeHumanVsAgent Mode = "human_vs_agent"
	ModeAgentVsAgent Mode = "agent_vs_agent"
)

// Status is the lifecycle state of a game.
type Status string

// Game statuses.
const (
	StatusWaiting    Status = "waiting"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

// PlayerType says whether a participant is a human or an agent.
type PlayerType string

// Player types.
const (
	PlayerHuman PlayerType = "human"
	PlayerAgent PlayerType = "agent"
)

// ResultReason explains how a game ended.
type ResultReason string

// Result reasons.
const (
	ReasonNone        ResultReason = ""
	ReasonConnectFour ResultReason = "connect_four"
	ReasonDraw        ResultReason = "draw"
	ReasonResignation ResultReason = "resignation"
)

const maxDisplayName = 64

// Player occupies one slot of a game.
type Player struct {
	ID          string
	Type        PlayerType
	DisplayName string
	Disc        engine.Disc
}

// Move is one accepted move.
type Move struct {
	MoveNumber int
	PlayerID   string
	Column     int
	Row        int
	Disc       engine.Disc
	Timestamp  time.Time
}

// Game is a snapshot of a game's authoritative state (SPEC.md §15).
type Game struct {
	ID           string
	Mode         Mode
	Status       Status
	Board        [engine.Rows][engine.Columns]engine.Disc
	Red          *Player
	Yellow       *Player
	CurrentTurn  engine.Disc
	LegalMoves   []int
	Winner       engine.Disc
	ResultReason ResultReason
	Version      int
	Moves        []Move
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MoveNumber returns the number of accepted moves.
func (g Game) MoveNumber() int {
	return len(g.Moves)
}

// OpenSlots returns the disc colors that still have no player.
func (g Game) OpenSlots() []engine.Disc {
	slots := []engine.Disc{}
	if g.Red == nil {
		slots = append(slots, engine.Red)
	}
	if g.Yellow == nil {
		slots = append(slots, engine.Yellow)
	}
	return slots
}

// MoveResult is the outcome of an accepted move.
type MoveResult struct {
	GameID       string
	Accepted     bool
	Move         Move
	NextTurn     engine.Disc
	Status       Status
	Winner       engine.Disc
	ResultReason ResultReason
	Version      int
}

type record struct {
	mu    sync.Mutex
	game  Game
	board *engine.Board
	// replays holds accepted move results by player and idempotency key (API-008).
	replays map[replayKey]MoveResult
}

type replayKey struct {
	playerID, key string
}

// Service holds games in memory. It is safe for concurrent use; each game has
// its own lock, so moves are atomic per game (TURN-004).
type Service struct {
	mu    sync.RWMutex
	games map[string]*record
	order []string
	now   func() time.Time
}

// NewService returns an empty in-memory game service.
func NewService() *Service {
	return &Service{
		games: map[string]*record{},
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// Create starts a new game waiting for two players (GAME-001, GAME-002).
func (s *Service) Create(mode Mode) (Game, error) {
	if mode != ModeHumanVsAgent && mode != ModeAgentVsAgent {
		return Game{}, newError(CodeInvalidArgument, "", "mode must be human_vs_agent or agent_vs_agent")
	}
	now := s.now()
	rec := &record{
		board: engine.NewBoard(),
		game: Game{
			Mode:       mode,
			Status:     StatusWaiting,
			LegalMoves: []int{},
			Moves:      []Move{},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	id := newGameID()
	for s.games[id] != nil {
		id = newGameID()
	}
	rec.game.ID = id
	s.games[id] = rec
	s.order = append(s.order, id)
	return rec.game.snapshot(), nil
}

// Join assigns playerID to the requested slot (GAME-003) and starts the game
// once both slots are filled (GAME-004). playerID comes from the caller's
// credentials, never from the request body.
func (s *Service) Join(gameID, playerID string, disc engine.Disc, displayName string, playerType PlayerType) (Game, error) {
	if disc != engine.Red && disc != engine.Yellow {
		return Game{}, newError(CodeInvalidArgument, gameID, "disc must be red or yellow")
	}
	if playerType != PlayerHuman && playerType != PlayerAgent {
		return Game{}, newError(CodeInvalidArgument, gameID, "type must be human or agent")
	}
	if displayName == "" || len(displayName) > maxDisplayName {
		return Game{}, newError(CodeInvalidArgument, gameID, "displayName must be 1 to 64 characters")
	}
	rec, err := s.lookup(gameID)
	if err != nil {
		return Game{}, err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	g := &rec.game
	if g.Status != StatusWaiting {
		return Game{}, ruleError(CodeGameNotJoinable, g, "the game is no longer accepting players")
	}
	if g.slot(disc) != nil {
		return Game{}, ruleError(CodeSlotTaken, g, "the "+string(disc)+" slot already has a player")
	}
	if other := g.slot(disc.Opponent()); other != nil && other.ID == playerID {
		return Game{}, ruleError(CodeAlreadyJoined, g, "you already hold the "+string(other.Disc)+" slot in this game")
	}

	player := &Player{ID: playerID, Type: playerType, DisplayName: displayName, Disc: disc}
	if disc == engine.Red {
		g.Red = player
	} else {
		g.Yellow = player
	}
	if g.Red != nil && g.Yellow != nil {
		g.Status = StatusInProgress
		g.CurrentTurn = engine.Red
		g.LegalMoves = rec.board.LegalMoves()
	}
	g.UpdatedAt = s.now()
	return g.snapshot(), nil
}

// Get returns the current state of a game (GAME-005, GAME-007).
func (s *Service) Get(gameID string) (Game, error) {
	rec, err := s.lookup(gameID)
	if err != nil {
		return Game{}, err
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.game.snapshot(), nil
}

// List returns every game in creation order (GAME-008).
func (s *Service) List() []Game {
	games, _ := s.ListByStatus("")
	return games
}

// ListByStatus returns the games in status, in creation order; an empty
// status returns every game (GAME-008, MCP-006).
func (s *Service) ListByStatus(status Status) ([]Game, error) {
	switch status {
	case "", StatusWaiting, StatusInProgress, StatusCompleted:
	default:
		return nil, newError(CodeInvalidArgument, "", "status must be waiting, in_progress or completed")
	}

	s.mu.RLock()
	records := make([]*record, 0, len(s.order))
	for _, id := range s.order {
		records = append(records, s.games[id])
	}
	s.mu.RUnlock()

	games := make([]Game, 0, len(records))
	for _, rec := range records {
		rec.mu.Lock()
		if status == "" || rec.game.Status == status {
			games = append(games, rec.game.snapshot())
		}
		rec.mu.Unlock()
	}
	return games, nil
}

// Move submits playerID's disc to column, provided expectedVersion matches the
// current version (TURN-001..006, RULE-002..010).
func (s *Service) Move(gameID, playerID string, column, expectedVersion int) (MoveResult, error) {
	return s.MoveWithKey(gameID, playerID, column, expectedVersion, "")
}

// MoveWithKey is Move with an idempotency key (API-008). If playerID already
// had a move accepted in this game under the same non-empty key, it returns
// that recorded result without playing again, even if the game has moved on.
// Rejected moves are not recorded, so a retry of one is evaluated afresh.
func (s *Service) MoveWithKey(gameID, playerID string, column, expectedVersion int, key string) (MoveResult, error) {
	rec, err := s.lookup(gameID)
	if err != nil {
		return MoveResult{}, err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	g := &rec.game
	mover := g.playerByID(playerID)
	if mover == nil {
		return MoveResult{}, newError(CodeForbidden, gameID, "you are not a player in this game")
	}
	if key != "" {
		if prior, ok := rec.replays[replayKey{playerID, key}]; ok {
			return prior, nil
		}
	}
	switch {
	case g.Status == StatusCompleted:
		return MoveResult{}, ruleError(CodeGameCompleted, g, "the game is over")
	case g.Status == StatusWaiting:
		return MoveResult{}, ruleError(CodeGameNotStarted, g, "the game starts when both players have joined")
	case mover.Disc != g.CurrentTurn:
		return MoveResult{}, ruleError(CodeNotYourTurn, g, "the current turn belongs to the opposing player")
	case expectedVersion != g.Version:
		return MoveResult{}, ruleError(CodeStaleState, g, "expectedVersion does not match the current version; re-read the game state")
	}

	row, outcome, err := rec.board.Drop(column, mover.Disc)
	switch {
	case errors.Is(err, engine.ErrInvalidColumn):
		return MoveResult{}, ruleError(CodeInvalidColumn, g, "column must be an integer from 0 to 6")
	case errors.Is(err, engine.ErrColumnFull):
		return MoveResult{}, ruleError(CodeColumnFull, g, "the column already holds six discs")
	case err != nil:
		return MoveResult{}, newError(CodeInternal, gameID, "the move could not be applied")
	}

	now := s.now()
	move := Move{
		MoveNumber: len(g.Moves) + 1,
		PlayerID:   playerID,
		Column:     column,
		Row:        row,
		Disc:       mover.Disc,
		Timestamp:  now,
	}
	g.Board = rec.board.Cells()
	g.Moves = append(g.Moves, move)
	g.Version++
	g.UpdatedAt = now
	if outcome.Over {
		g.Status = StatusCompleted
		g.CurrentTurn = engine.Empty
		g.LegalMoves = []int{}
		g.Winner = outcome.Winner
		g.ResultReason = ResultReason(outcome.Reason)
	} else {
		g.CurrentTurn = mover.Disc.Opponent()
		g.LegalMoves = rec.board.LegalMoves()
	}

	result := MoveResult{
		GameID:       g.ID,
		Accepted:     true,
		Move:         move,
		NextTurn:     g.CurrentTurn,
		Status:       g.Status,
		Winner:       g.Winner,
		ResultReason: g.ResultReason,
		Version:      g.Version,
	}
	if key != "" {
		if rec.replays == nil {
			rec.replays = map[replayKey]MoveResult{}
		}
		rec.replays[replayKey{playerID, key}] = result
	}
	return result, nil
}

// Resign ends playerID's in-progress game with the opponent as winner
// (TURN-007, GAME-006). A player may resign whether or not it is their turn.
func (s *Service) Resign(gameID, playerID string) (Game, error) {
	rec, err := s.lookup(gameID)
	if err != nil {
		return Game{}, err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	g := &rec.game
	resigner := g.playerByID(playerID)
	switch {
	case resigner == nil:
		return Game{}, newError(CodeForbidden, gameID, "you are not a player in this game")
	case g.Status == StatusCompleted:
		return Game{}, ruleError(CodeGameCompleted, g, "the game is over")
	case g.Status == StatusWaiting:
		return Game{}, ruleError(CodeGameNotStarted, g, "the game starts when both players have joined")
	}

	g.Status = StatusCompleted
	g.CurrentTurn = engine.Empty
	g.LegalMoves = []int{}
	g.Winner = resigner.Disc.Opponent()
	g.ResultReason = ReasonResignation
	g.Version++
	g.UpdatedAt = s.now()
	return g.snapshot(), nil
}

func (s *Service) lookup(gameID string) (*record, error) {
	s.mu.RLock()
	rec := s.games[gameID]
	s.mu.RUnlock()
	if rec == nil {
		return nil, newError(CodeGameNotFound, gameID, "no game has this identifier")
	}
	return rec, nil
}

func (g *Game) slot(disc engine.Disc) *Player {
	if disc == engine.Red {
		return g.Red
	}
	return g.Yellow
}

func (g *Game) playerByID(id string) *Player {
	for _, p := range []*Player{g.Red, g.Yellow} {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

// snapshot returns a deep copy, so callers cannot mutate stored state.
func (g *Game) snapshot() Game {
	out := *g
	if g.Red != nil {
		red := *g.Red
		out.Red = &red
	}
	if g.Yellow != nil {
		yellow := *g.Yellow
		out.Yellow = &yellow
	}
	out.LegalMoves = append([]int{}, g.LegalMoves...)
	out.Moves = append([]Move{}, g.Moves...)
	return out
}

func newGameID() string {
	var b [8]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error on supported platforms.
	return "game_" + hex.EncodeToString(b[:])
}
