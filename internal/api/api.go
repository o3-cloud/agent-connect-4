// Package api exposes the game service as the versioned REST API under
// /api/v1 (API-001..007), using Huma for routing, validation and the
// generated OpenAPI document (ADR-008). Handlers are thin adapters over the
// game service.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"connect4arena/internal/engine"
	"connect4arena/internal/game"
)

// Prefix is the base path of the versioned API.
const Prefix = "/api/v1"

func init() {
	huma.NewError = newHumaError
}

// Register adds the API operations, the OpenAPI document and a structured
// ROUTE_NOT_FOUND fallback for unknown /api/v1 paths to mux.
func Register(mux *http.ServeMux, service *game.Service) {
	config := huma.DefaultConfig("Connect Four Arena API", "1.0.0")
	config.OpenAPIPath = Prefix + "/openapi"
	config.DocsPath = Prefix + "/docs"
	config.SchemasPath = Prefix + "/schemas"
	config.CreateHooks = nil // no $schema links in response bodies
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer", Description: "Any non-empty token is a test identity until BDR-013."},
	}
	api := humago.New(mux, config)
	h := handlers{service: service}
	authed := []map[string][]string{{"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "createGame",
		Method:        http.MethodPost,
		Path:          Prefix + "/games",
		Summary:       "Create a game",
		Description:   "Creates a game in `waiting` status with an empty 6 × 7 board and two open player slots (GAME-001, API-002).",
		DefaultStatus: http.StatusCreated,
		Security:      authed,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, h.create)

	huma.Register(api, huma.Operation{
		OperationID: "listGames",
		Method:      http.MethodGet,
		Path:        Prefix + "/games",
		Summary:     "List games",
		Description: "Lists games in creation order, optionally only those in one status (GAME-008).",
		Errors:      []int{http.StatusBadRequest},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "getGame",
		Method:      http.MethodGet,
		Path:        Prefix + "/games/{gameId}",
		Summary:     "Get game state",
		Description: "Returns the authoritative state of a game (GAME-005, API-004).",
		Errors:      []int{http.StatusNotFound},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "joinGame",
		Method:      http.MethodPost,
		Path:        Prefix + "/games/{gameId}/join",
		Summary:     "Join a game",
		Description: "Takes the requested player slot. The player's identity comes from the credentials, never the body (GAME-003, API-003).",
		Security:    authed,
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, h.join)

	huma.Register(api, huma.Operation{
		OperationID: "submitMove",
		Method:      http.MethodPost,
		Path:        Prefix + "/games/{gameId}/moves",
		Summary:     "Submit a move",
		Description: "Drops the caller's disc into a column. `expectedVersion` must equal the game's current version (TURN-005, TURN-006, API-005).",
		Security:    authed,
		Errors: []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusConflict, http.StatusUnprocessableEntity},
	}, h.move)

	mux.HandleFunc(Prefix+"/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, newErrorResponse(game.CodeRouteNotFound, "no such endpoint; see "+Prefix+"/openapi.json"))
	})
}

type handlers struct {
	service *game.Service
}

type gameIDInput struct {
	GameID string `path:"gameId" maxLength:"64" doc:"Game identifier"`
}

type gameOutput struct {
	Body GameView
}

type createInput struct {
	Authorization string `header:"Authorization" doc:"Bearer credentials: Authorization: Bearer <token>"`
	Body          struct {
		Mode string `json:"mode" enum:"human_vs_agent,agent_vs_agent" doc:"Game mode"`
	}
}

func (h handlers) create(_ context.Context, in *createInput) (*gameOutput, error) {
	if _, err := playerID(in.Authorization); err != nil {
		return nil, err
	}
	created, err := h.service.Create(game.Mode(in.Body.Mode))
	if err != nil {
		return nil, fromError(err)
	}
	return &gameOutput{Body: viewOf(created)}, nil
}

type listOutput struct {
	Body struct {
		Games []GameView `json:"games"`
	}
}

type listInput struct {
	Status string `query:"status" required:"false" enum:"waiting,in_progress,completed" doc:"Only return games in this status"`
}

func (h handlers) list(_ context.Context, in *listInput) (*listOutput, error) {
	games, err := h.service.ListByStatus(game.Status(in.Status))
	if err != nil {
		return nil, fromError(err)
	}
	out := &listOutput{}
	out.Body.Games = []GameView{}
	for _, g := range games {
		out.Body.Games = append(out.Body.Games, viewOf(g))
	}
	return out, nil
}

func (h handlers) get(_ context.Context, in *gameIDInput) (*gameOutput, error) {
	g, err := h.service.Get(in.GameID)
	if err != nil {
		return nil, fromError(err)
	}
	return &gameOutput{Body: viewOf(g)}, nil
}

type joinInput struct {
	Authorization string `header:"Authorization" doc:"Bearer credentials: Authorization: Bearer <token>"`
	GameID        string `path:"gameId" maxLength:"64" doc:"Game identifier"`
	Body          struct {
		Disc        string `json:"disc" enum:"red,yellow" doc:"The slot to take"`
		DisplayName string `json:"displayName" minLength:"1" maxLength:"64" doc:"Name shown to other players"`
		Type        string `json:"type" enum:"human,agent" doc:"Whether the player is a human or an agent"`
	}
}

func (h handlers) join(_ context.Context, in *joinInput) (*gameOutput, error) {
	id, err := playerID(in.Authorization)
	if err != nil {
		return nil, err
	}
	g, err := h.service.Join(in.GameID, id, engine.Disc(in.Body.Disc), in.Body.DisplayName, game.PlayerType(in.Body.Type))
	if err != nil {
		return nil, fromError(err)
	}
	return &gameOutput{Body: viewOf(g)}, nil
}

type moveInput struct {
	Authorization string `header:"Authorization" doc:"Bearer credentials: Authorization: Bearer <token>"`
	GameID        string `path:"gameId" maxLength:"64" doc:"Game identifier"`
	Body          struct {
		Column          int    `json:"column" doc:"Column index from 0 to 6"`
		ExpectedVersion int    `json:"expectedVersion" minimum:"0" doc:"The game version the move is based on"`
		IdempotencyKey  string `json:"idempotencyKey,omitempty" required:"false" maxLength:"128" doc:"Accepted but not yet enforced (API-008)"`
	}
}

type moveOutput struct {
	Body MoveResponse
}

func (h handlers) move(_ context.Context, in *moveInput) (*moveOutput, error) {
	id, err := playerID(in.Authorization)
	if err != nil {
		return nil, err
	}
	result, err := h.service.Move(in.GameID, id, in.Body.Column, in.Body.ExpectedVersion)
	if err != nil {
		return nil, fromError(err)
	}
	return &moveOutput{Body: MoveResponse{
		GameID:       result.GameID,
		Accepted:     result.Accepted,
		Move:         moveViewOf(result.Move),
		NextTurn:     discOrNull(result.NextTurn),
		Status:       string(result.Status),
		Winner:       discOrNull(result.Winner),
		ResultReason: reasonOrNull(result.ResultReason),
		Version:      result.Version,
	}}, nil
}

// PlayerView is a player in a game.
type PlayerView struct {
	ID          string `json:"id" doc:"Stable player identifier derived from the credentials"`
	Type        string `json:"type" enum:"human,agent"`
	DisplayName string `json:"displayName"`
	Disc        string `json:"disc" enum:"red,yellow"`
}

// PlayersView holds the two player slots; an open slot is null.
type PlayersView struct {
	Red    *PlayerView `json:"red"`
	Yellow *PlayerView `json:"yellow"`
}

// MoveView is one accepted move.
type MoveView struct {
	MoveNumber int       `json:"moveNumber"`
	PlayerID   string    `json:"playerId"`
	Column     int       `json:"column"`
	Row        int       `json:"row"`
	Disc       string    `json:"disc" enum:"red,yellow"`
	Timestamp  time.Time `json:"timestamp"`
}

// GameView is the authoritative game state (SPEC.md §15). Row 0 of board is
// the top row; empty cells are null.
type GameView struct {
	GameID       string      `json:"gameId"`
	Mode         string      `json:"mode" enum:"human_vs_agent,agent_vs_agent"`
	Status       string      `json:"status" enum:"waiting,in_progress,completed"`
	Board        [][]*string `json:"board" doc:"6 rows × 7 columns; row 0 is the top; cells are red, yellow or null"`
	Players      PlayersView `json:"players"`
	OpenSlots    []string    `json:"openSlots" doc:"Disc colors that can still be joined"`
	CurrentTurn  *string     `json:"currentTurn" doc:"red, yellow, or null when the game is not in progress"`
	LegalMoves   []int       `json:"legalMoves" doc:"Playable columns; empty unless the game is in progress"`
	Winner       *string     `json:"winner" doc:"red, yellow, or null"`
	ResultReason *string     `json:"resultReason" doc:"connect_four, draw, or null"`
	Version      int         `json:"version" doc:"State version; increases by one with each accepted move"`
	MoveNumber   int         `json:"moveNumber" doc:"Number of accepted moves"`
	Moves        []MoveView  `json:"moves"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// MoveResponse is the result of an accepted move (SPEC.md §16 "Submit Move").
type MoveResponse struct {
	GameID       string   `json:"gameId"`
	Accepted     bool     `json:"accepted"`
	Move         MoveView `json:"move"`
	NextTurn     *string  `json:"nextTurn" doc:"red, yellow, or null when the game is over"`
	Status       string   `json:"status" enum:"waiting,in_progress,completed"`
	Winner       *string  `json:"winner"`
	ResultReason *string  `json:"resultReason"`
	Version      int      `json:"version"`
}

func viewOf(g game.Game) GameView {
	board := make([][]*string, engine.Rows)
	for r := range board {
		board[r] = make([]*string, engine.Columns)
		for c := range board[r] {
			board[r][c] = discOrNull(g.Board[r][c])
		}
	}
	openSlots := []string{}
	for _, disc := range g.OpenSlots() {
		openSlots = append(openSlots, string(disc))
	}
	moves := make([]MoveView, 0, len(g.Moves))
	for _, m := range g.Moves {
		moves = append(moves, moveViewOf(m))
	}
	return GameView{
		GameID:       g.ID,
		Mode:         string(g.Mode),
		Status:       string(g.Status),
		Board:        board,
		Players:      PlayersView{Red: playerViewOf(g.Red), Yellow: playerViewOf(g.Yellow)},
		OpenSlots:    openSlots,
		CurrentTurn:  discOrNull(g.CurrentTurn),
		LegalMoves:   append([]int{}, g.LegalMoves...),
		Winner:       discOrNull(g.Winner),
		ResultReason: reasonOrNull(g.ResultReason),
		Version:      g.Version,
		MoveNumber:   g.MoveNumber(),
		Moves:        moves,
		CreatedAt:    g.CreatedAt,
		UpdatedAt:    g.UpdatedAt,
	}
}

func playerViewOf(p *game.Player) *PlayerView {
	if p == nil {
		return nil
	}
	return &PlayerView{ID: p.ID, Type: string(p.Type), DisplayName: p.DisplayName, Disc: string(p.Disc)}
}

func moveViewOf(m game.Move) MoveView {
	return MoveView{
		MoveNumber: m.MoveNumber,
		PlayerID:   m.PlayerID,
		Column:     m.Column,
		Row:        m.Row,
		Disc:       string(m.Disc),
		Timestamp:  m.Timestamp,
	}
}

func discOrNull(d engine.Disc) *string {
	if d == engine.Empty {
		return nil
	}
	s := string(d)
	return &s
}

func reasonOrNull(r game.ResultReason) *string {
	if r == game.ReasonNone {
		return nil
	}
	s := string(r)
	return &s
}

func identityFor(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "player_" + hex.EncodeToString(sum[:8])
}
