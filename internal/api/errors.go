package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"connect4arena/internal/game"
)

// httpStatus maps each SPEC.md §16 error code to its HTTP status.
var httpStatus = map[game.Code]int{
	game.CodeInvalidArgument: http.StatusBadRequest,
	game.CodeUnauthenticated: http.StatusUnauthorized,
	game.CodeForbidden:       http.StatusForbidden,
	game.CodeGameNotFound:    http.StatusNotFound,
	game.CodeRouteNotFound:   http.StatusNotFound,
	game.CodeNotYourTurn:     http.StatusConflict,
	game.CodeStaleState:      http.StatusConflict,
	game.CodeGameNotStarted:  http.StatusConflict,
	game.CodeGameNotJoinable: http.StatusConflict,
	game.CodeSlotTaken:       http.StatusConflict,
	game.CodeAlreadyJoined:   http.StatusConflict,
	game.CodeGameCompleted:   http.StatusConflict,
	game.CodeInvalidColumn:   http.StatusUnprocessableEntity,
	game.CodeColumnFull:      http.StatusUnprocessableEntity,
	game.CodeRateLimited:     http.StatusTooManyRequests,
	game.CodeInternal:        http.StatusInternalServerError,
}

// ErrorBody is the error object of SPEC.md §16 (API-007).
type ErrorBody struct {
	Code           string              `json:"code" enum:"INVALID_ARGUMENT,UNAUTHENTICATED,FORBIDDEN,GAME_NOT_FOUND,ROUTE_NOT_FOUND,NOT_YOUR_TURN,STALE_STATE,GAME_NOT_STARTED,GAME_NOT_JOINABLE,SLOT_TAKEN,ALREADY_JOINED,GAME_COMPLETED,INVALID_COLUMN,COLUMN_FULL,RATE_LIMITED,INTERNAL" doc:"Machine-readable error code"`
	Message        string              `json:"message" doc:"Human-readable explanation"`
	Retryable      bool                `json:"retryable" doc:"Whether the request may succeed if retried after the condition clears"`
	GameID         string              `json:"gameId,omitempty" doc:"The game the error is about"`
	CurrentVersion *int                `json:"currentVersion,omitempty" doc:"The game's current state version, for game-rule errors"`
	Details        []*huma.ErrorDetail `json:"details,omitempty" doc:"Field-level validation problems"`
}

// ErrorResponse is the body of every API error: {"error": {...}}.
type ErrorResponse struct {
	status int
	Err    ErrorBody `json:"error"`
}

// GetStatus implements huma.StatusError.
func (e *ErrorResponse) GetStatus() int { return e.status }

func (e *ErrorResponse) Error() string { return e.Err.Code + ": " + e.Err.Message }

// ContentType implements huma.ContentTypeFilter so errors are plain JSON.
func (e *ErrorResponse) ContentType(string) string { return "application/json" }

func newErrorResponse(code game.Code, message string) *ErrorResponse {
	return &ErrorResponse{
		status: httpStatus[code],
		Err:    ErrorBody{Code: string(code), Message: message, Retryable: code.Retryable()},
	}
}

// fromError converts a service error into an API error.
func fromError(err error) error {
	var gameErr *game.Error
	if !errors.As(err, &gameErr) {
		return newErrorResponse(game.CodeInternal, "unexpected server error; no game state changed")
	}
	resp := newErrorResponse(gameErr.Code, gameErr.Message)
	resp.Err.GameID = gameErr.GameID
	resp.Err.CurrentVersion = gameErr.CurrentVersion
	return resp
}

// newHumaError replaces huma.NewError so framework-generated errors (request
// validation, malformed JSON) use the same shape and codes as game errors.
func newHumaError(status int, message string, errs ...error) huma.StatusError {
	details := make([]*huma.ErrorDetail, 0, len(errs))
	for _, err := range errs {
		var detailer huma.ErrorDetailer
		switch {
		case err == nil:
			continue
		case errors.As(err, &detailer):
			details = append(details, detailer.ErrorDetail())
		default:
			details = append(details, &huma.ErrorDetail{Message: err.Error()})
		}
	}

	code := game.CodeInternal
	switch {
	case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		code = game.CodeInvalidArgument
		for _, detail := range details {
			if detail.Location == "body.column" {
				code = game.CodeInvalidColumn
				message = "column must be an integer from 0 to 6"
			}
		}
	case status == http.StatusUnauthorized:
		code = game.CodeUnauthenticated
	case status == http.StatusForbidden:
		code = game.CodeForbidden
	case status == http.StatusNotFound:
		code = game.CodeRouteNotFound
	case status == http.StatusTooManyRequests:
		code = game.CodeRateLimited
	case status >= 400 && status < 500:
		code = game.CodeInvalidArgument
	}

	resp := newErrorResponse(code, message)
	if code == game.CodeInvalidArgument && status != http.StatusUnprocessableEntity {
		resp.status = status
	}
	if message == "" {
		resp.Err.Message = http.StatusText(resp.status)
	}
	resp.Err.Details = details
	return resp
}

// serviceKey is the request-context key under which Register stores the game
// service, so framework-generated errors can add game context.
type serviceKey struct{}

// newHumaErrorWithContext replaces huma.NewErrorWithContext. A column that
// fails schema validation (3.5, "three") never reaches the handler, so this
// adds the gameId and currentVersion that SPEC.md §16 requires on game-rule
// errors, or reports GAME_NOT_FOUND when the game does not exist.
func newHumaErrorWithContext(ctx huma.Context, status int, message string, errs ...error) huma.StatusError {
	se := newHumaError(status, message, errs...)
	resp, ok := se.(*ErrorResponse)
	if !ok || ctx == nil || resp.Err.Code != string(game.CodeInvalidColumn) {
		return se
	}
	service, _ := ctx.Context().Value(serviceKey{}).(*game.Service)
	gameID := ctx.Param("gameId")
	if service == nil || gameID == "" {
		return se
	}
	g, err := service.Get(gameID)
	if err != nil {
		var notFound *ErrorResponse
		if errors.As(fromError(err), &notFound) {
			return notFound
		}
		return se
	}
	version := g.Version
	resp.Err.GameID = g.ID
	resp.Err.CurrentVersion = &version
	return resp
}

// writeError writes an API error outside Huma, e.g. for unknown routes.
func writeError(w http.ResponseWriter, resp *ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	_ = json.NewEncoder(w).Encode(resp)
}

// playerID derives a stable, non-secret player identifier from a bearer token.
// For the walking skeleton any non-empty token is a valid test identity;
// real credential checks arrive with BDR-013.
func playerID(authorization string) (string, error) {
	const prefix = "Bearer "
	token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	if !strings.HasPrefix(authorization, prefix) || token == "" {
		return "", newErrorResponse(game.CodeUnauthenticated, "send credentials as an Authorization: Bearer <token> header")
	}
	return identityFor(token), nil
}
