// Package mcpserver exposes the game service as MCP tools over Streamable
// HTTP (ADR-009, SPEC.md MCP-001..009, §17). Tools call the same game service
// as the REST API and return the same views and §16 error objects.
package mcpserver

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"connect4arena/internal/api"
	"connect4arena/internal/engine"
	"connect4arena/internal/game"
	"connect4arena/internal/identity"
)

// Path is where the MCP endpoint is mounted.
const Path = "/mcp"

const agentTypeKey = "type"

// Handler returns the MCP endpoint. Every request must carry an API key issued
// by registry (SEC-001, ADR-013); the acting player is that key's agent, never
// a tool argument (SEC-004, BDR-021 AC-3).
func Handler(service *game.Service, registry *identity.Registry) http.Handler {
	server := NewServer(service)
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	verify := func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		agent, ok := registry.Authenticate(token)
		if !ok {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{UserID: agent.ID, Extra: map[string]any{agentTypeKey: agent.Type}}, nil
	}
	return auth.RequireBearerToken(verify, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})(streamable)
}

// NewServer returns an MCP server with the six SPEC.md §17 game tools.
func NewServer(service *game.Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "connect4-arena", Version: "1.0.0"}, nil)
	t := tools{service: service}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_game",
		Description: "Create a Connect Four game. It starts in status waiting; join it with join_game.",
	}, t.createGame)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "join_game",
		Description: "Take the red or yellow slot in a waiting game as the authenticated agent. Red moves first; the game starts when both slots are filled.",
	}, t.joinGame)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_game_state",
		Description: "Get a game's authoritative state: board (row 0 is the top), current turn, legal columns, players, version and result.",
	}, t.getGameState)
	mcp.AddTool(server, &mcp.Tool{
		Name: "make_move",
		Description: "Drop your disc into a column (0 to 6) on your turn. expected_version must equal the game's current version. " +
			"Resending the same idempotency_key returns the recorded result without playing again.",
	}, t.makeMove)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_games",
		Description: "List games, optionally only those in one status (waiting, in_progress or completed).",
	}, t.listGames)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_game_history",
		Description: "Get a game's accepted moves in order and, once it is over, the winner and result reason.",
	}, t.getGameHistory)
	return server
}

type tools struct {
	service *game.Service
}

type createGameInput struct {
	Mode string `json:"mode" jsonschema:"human_vs_agent or agent_vs_agent"`
}

type joinGameInput struct {
	GameID      string `json:"game_id" jsonschema:"game identifier"`
	Disc        string `json:"disc" jsonschema:"red or yellow"`
	DisplayName string `json:"display_name" jsonschema:"name shown to other players, 1 to 64 characters"`
}

type gameIDInput struct {
	GameID string `json:"game_id" jsonschema:"game identifier"`
}

type makeMoveInput struct {
	GameID          string `json:"game_id" jsonschema:"game identifier"`
	Column          int    `json:"column" jsonschema:"column index from 0 to 6"`
	ExpectedVersion int    `json:"expected_version" jsonschema:"the game version your move is based on"`
	IdempotencyKey  string `json:"idempotency_key" jsonschema:"unique key for this move; reuse it only to retry the same move"`
}

type listGamesInput struct {
	Status string `json:"status,omitempty" jsonschema:"only games in this status: waiting, in_progress or completed"`
}

type gamesOutput struct {
	Games []api.GameView `json:"games"`
}

// failed reports err as a tool-level error carrying the §16 error object
// (MCP-008), never as a protocol error.
func failed(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{IsError: true}, api.ErrorFor(err), nil
}

// caller returns the authenticated agent's id and type.
func caller(req *mcp.CallToolRequest) (string, game.PlayerType) {
	info := req.Extra.TokenInfo
	if info == nil {
		return "", ""
	}
	agentType, _ := info.Extra[agentTypeKey].(string)
	return info.UserID, game.PlayerType(agentType)
}

func (t tools) createGame(_ context.Context, _ *mcp.CallToolRequest, in createGameInput) (*mcp.CallToolResult, any, error) {
	g, err := t.service.Create(game.Mode(in.Mode))
	if err != nil {
		return failed(err)
	}
	return nil, api.GameViewOf(g), nil
}

func (t tools) joinGame(_ context.Context, req *mcp.CallToolRequest, in joinGameInput) (*mcp.CallToolResult, any, error) {
	id, playerType := caller(req)
	g, err := t.service.Join(in.GameID, id, engine.Disc(in.Disc), in.DisplayName, playerType)
	if err != nil {
		return failed(err)
	}
	return nil, api.GameViewOf(g), nil
}

func (t tools) getGameState(_ context.Context, _ *mcp.CallToolRequest, in gameIDInput) (*mcp.CallToolResult, any, error) {
	g, err := t.service.Get(in.GameID)
	if err != nil {
		return failed(err)
	}
	return nil, api.GameViewOf(g), nil
}

func (t tools) makeMove(_ context.Context, req *mcp.CallToolRequest, in makeMoveInput) (*mcp.CallToolResult, any, error) {
	id, _ := caller(req)
	result, err := t.service.MoveWithKey(in.GameID, id, in.Column, in.ExpectedVersion, in.IdempotencyKey)
	if err != nil {
		return failed(err)
	}
	return nil, api.MoveResponseOf(result), nil
}

func (t tools) listGames(_ context.Context, _ *mcp.CallToolRequest, in listGamesInput) (*mcp.CallToolResult, any, error) {
	games, err := t.service.ListByStatus(game.Status(in.Status))
	if err != nil {
		return failed(err)
	}
	out := gamesOutput{Games: []api.GameView{}}
	for _, g := range games {
		out.Games = append(out.Games, api.GameViewOf(g))
	}
	return nil, out, nil
}

func (t tools) getGameHistory(_ context.Context, _ *mcp.CallToolRequest, in gameIDInput) (*mcp.CallToolResult, any, error) {
	g, err := t.service.Get(in.GameID)
	if err != nil {
		return failed(err)
	}
	return nil, api.HistoryViewOf(g), nil
}
