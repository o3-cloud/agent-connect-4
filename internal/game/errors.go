package game

// Code is a machine-readable error code shared by every interface (SPEC.md §16, NFR-004).
type Code string

// Error codes from SPEC.md §16 "Error Codes".
const (
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	CodeUnauthenticated Code = "UNAUTHENTICATED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeGameNotFound    Code = "GAME_NOT_FOUND"
	CodeRouteNotFound   Code = "ROUTE_NOT_FOUND"
	CodeNotYourTurn     Code = "NOT_YOUR_TURN"
	CodeStaleState      Code = "STALE_STATE"
	CodeGameNotStarted  Code = "GAME_NOT_STARTED"
	CodeGameNotJoinable Code = "GAME_NOT_JOINABLE"
	CodeSlotTaken       Code = "SLOT_TAKEN"
	CodeAlreadyJoined   Code = "ALREADY_JOINED"
	CodeGameCompleted   Code = "GAME_COMPLETED"
	CodeInvalidColumn   Code = "INVALID_COLUMN"
	CodeColumnFull      Code = "COLUMN_FULL"
	CodeRateLimited     Code = "RATE_LIMITED"
	CodeInternal        Code = "INTERNAL"
)

// Retryable reports whether a client may retry after the condition in SPEC.md §16 clears.
func (c Code) Retryable() bool {
	switch c {
	case CodeNotYourTurn, CodeStaleState, CodeGameNotStarted, CodeRateLimited, CodeInternal:
		return true
	default:
		return false
	}
}

// Error is a game-level failure. GameID and CurrentVersion carry request context (API-007).
type Error struct {
	Code           Code
	Message        string
	GameID         string
	CurrentVersion *int
}

func (e *Error) Error() string {
	return string(e.Code) + ": " + e.Message
}

func newError(code Code, gameID, message string) *Error {
	return &Error{Code: code, Message: message, GameID: gameID}
}

func ruleError(code Code, g *Game, message string) *Error {
	version := g.Version
	return &Error{Code: code, Message: message, GameID: g.ID, CurrentVersion: &version}
}
