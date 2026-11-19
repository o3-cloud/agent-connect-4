// Package mcpserver exposes the game service as MCP tools over Streamable
// HTTP (ADR-009, SPEC.md MCP-001..009, §17). Tools call the same game service
// as the REST API and return the same views and §16 error objects.
package mcpserver

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
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

	addTool(server, &mcp.Tool{
		Name:        "create_game",
		Description: "Create a Connect Four game. It starts in status waiting; join it with join_game.",
	}, nil, t.createGame)
	addTool(server, &mcp.Tool{
		Name:        "join_game",
		Description: "Take the red or yellow slot in a waiting game as the authenticated agent. Red moves first; the game starts when both slots are filled.",
	}, nil, t.joinGame)
	addTool(server, &mcp.Tool{
		Name:        "get_game_state",
		Description: "Get a game's authoritative state: board (row 0 is the top), current turn, legal columns, players, version and result.",
	}, nil, t.getGameState)
	addTool(server, &mcp.Tool{
		Name: "make_move",
		Description: "Drop your disc into a column (0 to 6) on your turn. expected_version must equal the game's current version. " +
			"Resending the same idempotency_key returns the recorded result without playing again.",
	}, t.checkColumn, t.makeMove)
	addTool(server, &mcp.Tool{
		Name:        "list_games",
		Description: "List games, optionally only those in one status (waiting, in_progress or completed).",
	}, nil, t.listGames)
	addTool(server, &mcp.Tool{
		Name:        "get_game_history",
		Description: "Get a game's accepted moves in order and, once it is over, the winner and result reason.",
	}, nil, t.getGameHistory)
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

// toolHandler is a tool's logic: it returns the structured result, or an
// error that is reported as a §16 error object.
type toolHandler[In any] func(ctx context.Context, req *mcp.CallToolRequest, in In) (any, error)

// addTool registers a tool whose input schema is derived from In (AC-1). It
// validates the arguments itself rather than through mcp.AddTool, so every
// failure, schema-level or game-rule, is a tool-level error carrying the
// same {"error": {...}} object REST returns (MCP-008, BDR-015 AC-3). check,
// if set, runs before schema validation to give specific codes.
func addTool[In any](server *mcp.Server, tool *mcp.Tool, check func(map[string]any) error, h toolHandler[In]) {
	schema, err := jsonschema.For[In](nil)
	if err != nil {
		panic("mcpserver: input schema for " + tool.Name + ": " + err.Error())
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		panic("mcpserver: resolve schema for " + tool.Name + ": " + err.Error())
	}
	tool.InputSchema = schema
	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw := req.Params.Arguments
		if len(raw) == 0 || string(raw) == "null" {
			raw = json.RawMessage("{}")
		}
		args := map[string]any{}
		if err := json.Unmarshal(raw, &args); err != nil {
			return failed(invalidArgument("arguments must be a JSON object"))
		}
		if check != nil {
			if err := check(args); err != nil {
				return failed(err)
			}
		}
		if err := resolved.Validate(args); err != nil {
			return failed(invalidArgument(err.Error()))
		}
		var in In
		if err := json.Unmarshal(raw, &in); err != nil {
			return failed(invalidArgument(err.Error()))
		}
		out, err := h(ctx, req, in)
		if err != nil {
			return failed(err)
		}
		return succeeded(out)
	})
}

func succeeded(out any) (*mcp.CallToolResult, error) {
	body, err := json.Marshal(out)
	if err != nil {
		return failed(err)
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(body)}},
		StructuredContent: json.RawMessage(body),
	}, nil
}

// failed reports err as a tool-level error carrying the §16 error object
// (MCP-008), never as a protocol error.
func failed(err error) (*mcp.CallToolResult, error) {
	body, _ := json.Marshal(api.ErrorFor(err))
	return &mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: string(body)}},
		StructuredContent: json.RawMessage(body),
	}, nil
}

func invalidArgument(message string) error {
	return &game.Error{Code: game.CodeInvalidArgument, Message: message}
}

// checkColumn makes a non-integer column INVALID_COLUMN with the
// game's id and version, as REST does (RULE-003, 05bc64f). An unknown game is
// GAME_NOT_FOUND. A non-string game_id is left to schema validation.
func (t tools) checkColumn(args map[string]any) error {
	gameID, ok := args["game_id"].(string)
	if !ok {
		return nil
	}
	value, present := args["column"]
	if !present {
		return nil // a missing argument is INVALID_ARGUMENT, as on REST
	}
	if column, ok := value.(float64); ok && column == math.Trunc(column) {
		return nil // the range is the service's to check
	}
	g, err := t.service.Get(gameID)
	if err != nil {
		return err
	}
	version := g.Version
	return &game.Error{Code: game.CodeInvalidColumn, Message: "column must be an integer from 0 to 6", GameID: g.ID, CurrentVersion: &version}
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

func (t tools) createGame(_ context.Context, _ *mcp.CallToolRequest, in createGameInput) (any, error) {
	g, err := t.service.Create(game.Mode(in.Mode))
	if err != nil {
		return nil, err
	}
	return api.GameViewOf(g), nil
}

func (t tools) joinGame(_ context.Context, req *mcp.CallToolRequest, in joinGameInput) (any, error) {
	id, playerType := caller(req)
	g, err := t.service.Join(in.GameID, id, engine.Disc(in.Disc), in.DisplayName, playerType)
	if err != nil {
		return nil, err
	}
	return api.GameViewOf(g), nil
}

func (t tools) getGameState(_ context.Context, _ *mcp.CallToolRequest, in gameIDInput) (any, error) {
	g, err := t.service.Get(in.GameID)
	if err != nil {
		return nil, err
	}
	return api.GameViewOf(g), nil
}

func (t tools) makeMove(_ context.Context, req *mcp.CallToolRequest, in makeMoveInput) (any, error) {
	id, _ := caller(req)
	result, err := t.service.MoveWithKey(in.GameID, id, in.Column, in.ExpectedVersion, in.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	return api.MoveResponseOf(result), nil
}

func (t tools) listGames(_ context.Context, _ *mcp.CallToolRequest, in listGamesInput) (any, error) {
	games, err := t.service.ListByStatus(game.Status(in.Status))
	if err != nil {
		return nil, err
	}
	out := gamesOutput{Games: []api.GameView{}}
	for _, g := range games {
		out.Games = append(out.Games, api.GameViewOf(g))
	}
	return out, nil
}

func (t tools) getGameHistory(_ context.Context, _ *mcp.CallToolRequest, in gameIDInput) (any, error) {
	g, err := t.service.Get(in.GameID)
	if err != nil {
		return nil, err
	}
	return api.HistoryViewOf(g), nil
}
