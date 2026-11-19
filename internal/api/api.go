// Package api exposes the game service as the versioned REST API under
// /api/v1 (API-001..007), using Huma for routing, validation and the
// generated OpenAPI document (ADR-008). Handlers are thin adapters over the
// game service.
package api

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"connect4arena/internal/engine"
	"connect4arena/internal/game"
	"connect4arena/internal/identity"
)

// Prefix is the base path of the versioned API.
const Prefix = "/api/v1"

func init() {
	huma.NewError = newHumaError
	huma.NewErrorWithContext = newHumaErrorWithContext
}

// Register adds the API operations, the OpenAPI document and a structured
// ROUTE_NOT_FOUND fallback for unknown /api/v1 paths to mux. Players
// authenticate with API keys issued by registry (ADR-013).
func Register(mux *http.ServeMux, service *game.Service, registry *identity.Registry, opts ...Option) {
	config := huma.DefaultConfig("Connect Four Arena API", "1.0.0")
	config.Info.Description = guide
	config.OpenAPIPath = Prefix + "/openapi"
	config.DocsPath = Prefix + "/docs"
	config.SchemasPath = Prefix + "/schemas"
	config.CreateHooks = nil // no $schema links in response bodies
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer", Description: "An API key from `POST /api/v1/agents` (ADR-013)."},
	}
	api := humago.New(mux, config)
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithValue(huma.WithValue(ctx, serviceKey{}, service), registryKey{}, registry))
	})
	h := handlers{service: service, registry: registry}
	for _, opt := range opts {
		opt(&h)
	}
	authed := []map[string][]string{{"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "registerAgent",
		Method:        http.MethodPost,
		Path:          Prefix + "/agents",
		Summary:       "Register an agent",
		Description:   "Creates a player identity and returns its API key. The key is shown only in this response; send it as `Authorization: Bearer <apiKey>` on every game operation (API-009, ADR-013).",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, h.register)

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
		OperationID: "getGameHistory",
		Method:      http.MethodGet,
		Path:        Prefix + "/games/{gameId}/moves",
		Summary:     "Get move history",
		Description: "Returns the accepted moves in order, numbered from 1, and the result once the game is over (API-006, MCP-007). Rejected moves never appear.",
		Errors:      []int{http.StatusNotFound},
	}, h.history)

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

	huma.Register(api, huma.Operation{
		OperationID: "resignGame",
		Method:      http.MethodPost,
		Path:        Prefix + "/games/{gameId}/resign",
		Summary:     "Resign a game",
		Description: "Ends the caller's in-progress game immediately; the opponent wins with `resultReason` `resignation`. Either player may resign, whether or not it is their turn (TURN-007).",
		Security:    authed,
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, h.resign)
	documentRateLimit(api.OpenAPI())

	mux.HandleFunc(Prefix+"/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, newErrorResponse(game.CodeRouteNotFound, "no such endpoint; see "+Prefix+"/openapi.json"))
	})
}

// guide is the OpenAPI info.description: enough for an agent that has only
// the document to play a game (BDR-022 AC-4).
//
//go:embed guide.md
var guide string

// documentRateLimit adds the 429 RATE_LIMITED response, which the rate
// limiter in front of the API can return for any operation (SEC-007), to
// every operation in doc.
func documentRateLimit(doc *huma.OpenAPI) {
	for _, item := range doc.Paths {
		for _, op := range []*huma.Operation{item.Get, item.Put, item.Post, item.Delete, item.Patch} {
			if op == nil || op.Responses["500"] == nil {
				continue
			}
			op.Responses["429"] = &huma.Response{
				Description: "Too Many Requests: the client's request budget is spent (RATE_LIMITED)",
				Content:     op.Responses["500"].Content,
				Headers: map[string]*huma.Param{
					"Retry-After": {
						Description: "Seconds until the client may send requests again",
						Schema:      &huma.Schema{Type: huma.TypeInteger, Minimum: new(float64(1))},
					},
				},
			}
		}
	}
}

type handlers struct {
	service  *game.Service
	registry *identity.Registry
	house    func(gameID string)
}

// Option configures Register.
type Option func(*handlers)

// WithHouseOpponent makes `"opponent": "house"` on create start the house
// reference agent for the new game (ADR-014, AGENT-007).
func WithHouseOpponent(start func(gameID string)) Option {
	return func(h *handlers) { h.house = start }
}

type registerInput struct {
	Body struct {
		DisplayName string `json:"displayName" minLength:"1" maxLength:"64" doc:"Name shown to other players"`
		Type        string `json:"type" enum:"human,agent" doc:"Whether the player is a human or an agent"`
	}
}

// RegisteredAgent is a new identity and its one-time API key.
type RegisteredAgent struct {
	AgentID     string `json:"agentId" doc:"The player id that games show for this identity"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type" enum:"human,agent"`
	APIKey      string `json:"apiKey" doc:"Secret credential; returned only in this response"`
}

type registerOutput struct {
	Body RegisteredAgent
}

func (h handlers) register(_ context.Context, in *registerInput) (*registerOutput, error) {
	agent, key, err := h.registry.Register(in.Body.DisplayName, in.Body.Type)
	if errors.Is(err, identity.ErrInvalidAgent) {
		return nil, newErrorResponse(game.CodeInvalidArgument, err.Error())
	}
	return &registerOutput{Body: RegisteredAgent{
		AgentID: agent.ID, DisplayName: agent.DisplayName, Type: agent.Type, APIKey: key,
	}}, nil
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
		Mode     string `json:"mode" enum:"human_vs_agent,agent_vs_agent" doc:"Game mode"`
		Opponent string `json:"opponent,omitempty" required:"false" enum:"house" doc:"With human_vs_agent, house makes Arena Bot take the other slot (ADR-014)"`
	}
}

func (h handlers) create(_ context.Context, in *createInput) (*gameOutput, error) {
	if _, err := h.playerID(in.Authorization); err != nil {
		return nil, err
	}
	if in.Body.Opponent == "house" && (game.Mode(in.Body.Mode) != game.ModeHumanVsAgent || h.house == nil) {
		return nil, newErrorResponse(game.CodeInvalidArgument, `"opponent": "house" is only available with mode human_vs_agent`)
	}
	created, err := h.service.Create(game.Mode(in.Body.Mode))
	if err != nil {
		return nil, fromError(err)
	}
	if in.Body.Opponent == "house" {
		h.house(created.ID)
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

type historyOutput struct {
	Body HistoryView
}

func (h handlers) history(_ context.Context, in *gameIDInput) (*historyOutput, error) {
	g, err := h.service.Get(in.GameID)
	if err != nil {
		return nil, fromError(err)
	}
	return &historyOutput{Body: HistoryViewOf(g)}, nil
}

// HistoryViewOf returns the move history and result of g (API-006, MCP-007).
func HistoryViewOf(g game.Game) HistoryView {
	view := viewOf(g)
	return HistoryView{
		GameID:       view.GameID,
		Mode:         view.Mode,
		Status:       view.Status,
		Players:      view.Players,
		Moves:        view.Moves,
		Winner:       view.Winner,
		ResultReason: view.ResultReason,
		Version:      view.Version,
	}
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
	id, err := h.playerID(in.Authorization)
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
		IdempotencyKey  string `json:"idempotencyKey,omitempty" required:"false" maxLength:"128" doc:"Optional retry key. Resending a move with the same key in the same game returns the recorded result without playing again (API-008)"`
	}
}

type moveOutput struct {
	Body MoveResponse
}

func (h handlers) move(_ context.Context, in *moveInput) (*moveOutput, error) {
	id, err := h.playerID(in.Authorization)
	if err != nil {
		h.service.RecordRejectedMove(in.GameID, "", game.CodeUnauthenticated)
		return nil, err
	}
	result, err := h.service.MoveWithKey(in.GameID, id, in.Body.Column, in.Body.ExpectedVersion, in.Body.IdempotencyKey)
	if err != nil {
		return nil, fromError(err)
	}
	return &moveOutput{Body: MoveResponseOf(result)}, nil
}

// MoveResponseOf returns the view of an accepted move (API-005, MCP-005).
func MoveResponseOf(result game.MoveResult) MoveResponse {
	return MoveResponse{
		GameID:       result.GameID,
		Accepted:     result.Accepted,
		Move:         moveViewOf(result.Move),
		NextTurn:     discOrNull(result.NextTurn),
		Status:       string(result.Status),
		Winner:       discOrNull(result.Winner),
		ResultReason: reasonOrNull(result.ResultReason),
		Version:      result.Version,
	}
}

type resignInput struct {
	Authorization string `header:"Authorization" doc:"Bearer credentials: Authorization: Bearer <token>"`
	GameID        string `path:"gameId" maxLength:"64" doc:"Game identifier"`
}

func (h handlers) resign(_ context.Context, in *resignInput) (*gameOutput, error) {
	id, err := h.playerID(in.Authorization)
	if err != nil {
		return nil, err
	}
	g, err := h.service.Resign(in.GameID, id)
	if err != nil {
		return nil, fromError(err)
	}
	return &gameOutput{Body: viewOf(g)}, nil
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
	ResultReason *string     `json:"resultReason" doc:"connect_four, draw, resignation, or null"`
	Version      int         `json:"version" doc:"State version; increases by one with each accepted move"`
	MoveNumber   int         `json:"moveNumber" doc:"Number of accepted moves"`
	Moves        []MoveView  `json:"moves"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// HistoryView is a game's ordered move history and result (API-006).
type HistoryView struct {
	GameID       string      `json:"gameId"`
	Mode         string      `json:"mode" enum:"human_vs_agent,agent_vs_agent"`
	Status       string      `json:"status" enum:"waiting,in_progress,completed"`
	Players      PlayersView `json:"players"`
	Moves        []MoveView  `json:"moves" doc:"Accepted moves in order; moveNumber starts at 1 with no gaps"`
	Winner       *string     `json:"winner" doc:"red, yellow, or null until the game is over"`
	ResultReason *string     `json:"resultReason" doc:"connect_four, draw, resignation, or null until the game is over"`
	Version      int         `json:"version" doc:"State version the history reflects"`
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

// GameViewOf returns the authoritative state view of g (GAME-005, MCP-004).
func GameViewOf(g game.Game) GameView { return viewOf(g) }

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
