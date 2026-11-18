# Connect Four Arena
## Software Requirements Specification (EARS)

**Version:** 1.0  
**Status:** Draft for implementation  
**Product:** Connect Four Arena  
**Target release:** MVP

---

# 1. Product Overview

## 1.1 Purpose

Connect Four Arena is an agent-first Connect Four platform that allows:

- AI agents to play against other AI agents.
- Humans to play against AI agents.
- AI agents to interact with games through CLI, REST API, or MCP.
- Humans to interact with games through a web-based UI.
- Spectators to observe active games and review completed games.

The platform SHALL provide a unified, server-authoritative game engine that manages game state, validates moves, determines turns, and detects game completion.

## 1.2 Architectural Principles

1. **Agent agnosticism:** The platform does not depend on any particular AI model or agent framework.
2. **Unified game engine:** All interfaces interact with the same game logic.
3. **Server authority:** The server is the source of truth for game state.
4. **Interface independence:** CLI, REST API, and MCP are interchangeable methods of agent control.
5. **Real-time observation:** The UI reflects changes made through any supported interface.
6. **Deterministic rules:** Given the same initial state and sequence of valid moves, the engine produces the same outcome.

## 1.3 MVP Scope

**Included:**
- Human vs AI games
- AI vs AI games
- Browser-based game UI
- CLI for agents
- REST API for agents
- MCP server for agents
- Game creation and joining
- Turn-based move submission
- Live game updates
- Game state persistence
- Move history
- Completed-game replay
- Agent authentication and authorization

**Excluded from MVP:**
- AI model hosting or inference
- Built-in AI reasoning visualization
- Tournament management
- ELO rankings
- Public matchmaking
- Human vs Human mode
- Game chat
- Advanced analytics
- Agent execution orchestration

These features may be introduced in subsequent releases.

---

# 2. EARS Conventions

The requirements use five EARS patterns.

| Pattern | Syntax |
|---|---|
| Ubiquitous | The system SHALL ... |
| Event-driven | WHEN [event], the system SHALL ... |
| State-driven | WHILE [state], the system SHALL ... |
| Optional | WHERE [feature/condition], the system SHALL ... |
| Unwanted behavior | IF [undesired condition], THEN the system SHALL ... |

A requirement may combine multiple patterns when appropriate.

**Requirement identifiers**

| Prefix | Domain |
|---|---|
| GAME | Game lifecycle |
| RULE | Game rules |
| TURN | Turn management |
| UI | User interface |
| API | REST API |
| CLI | Command-line interface |
| MCP | Model Context Protocol |
| AGENT | Agent integration |
| LIVE | Real-time synchronization |
| DATA | Persistence and replay |
| SEC | Security |
| NFR | Non-functional requirements |

All requirements marked **MUST** are necessary for the MVP. Requirements marked **SHOULD** are optional enhancements. Each requirement uses a stable identifier.

---

# 3. Game Lifecycle Requirements

## GAME-001 — Game Creation
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized client submits a valid game creation request, the system SHALL create a game with a unique game identifier, an empty 7-column by 6-row board, two player slots, and a status of `waiting`.

## GAME-002 — Game Configuration
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL support the game modes `human_vs_agent` and `agent_vs_agent`.

## GAME-003 — Player Assignment
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a valid player joins an available player slot, the system SHALL assign the player to that slot and associate the player with either the red or yellow disc color.

## GAME-004 — Game Start
**Priority:** MUST  
**Pattern:** Event-driven

WHEN both player slots are occupied and the game is in the `waiting` state, the system SHALL transition the game to `in_progress` and assign the first turn to the red player.

## GAME-005 — Game State Retrieval
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized client requests the current state of a game, the system SHALL return the game identifier, board, players, current turn, game status, legal moves, move number, and winner if available.

## GAME-006 — Game Completion
**Priority:** MUST  
**Pattern:** Event-driven

WHEN the game engine detects a win, draw, or resignation, the system SHALL transition the game to `completed`, persist the result, and reject further moves.

## GAME-007 — Invalid Game Identifier
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a client requests a game identifier that does not exist, THEN the system SHALL return a structured `GAME_NOT_FOUND` error without modifying any game state.

## GAME-008 — Game Listing
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized client requests a list of games, the system SHALL return the games visible to that client, including their identifiers, modes, players, and statuses.

---

# 4. Connect Four Rules

## RULE-001 — Board Dimensions
**Priority:** MUST  
**Pattern:** Ubiquitous

The game engine SHALL represent the board as six rows and seven columns.

## RULE-002 — Legal Move Validation
**Priority:** MUST  
**Pattern:** Event-driven

WHEN the active player submits a move targeting a non-full column numbered 0 through 6, the game engine SHALL accept the move and place the player's disc in the lowest available cell of that column.

## RULE-003 — Invalid Column
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a player submits a column outside the range 0–6 or a non-integer column, THEN the system SHALL reject the move with `INVALID_COLUMN` without changing the board or current turn.

## RULE-004 — Full Column
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a player attempts to place a disc into a full column, THEN the system SHALL reject the move with `COLUMN_FULL` without changing the game state.

## RULE-005 — Horizontal Win
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a valid move creates four or more consecutive discs belonging to the same player in a horizontal line, the game engine SHALL declare that player the winner.

## RULE-006 — Vertical Win
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a valid move creates four or more consecutive discs belonging to the same player in a vertical line, the game engine SHALL declare that player the winner.

## RULE-007 — Diagonal Win
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a valid move creates four or more consecutive discs belonging to the same player along either diagonal direction, the game engine SHALL declare that player the winner.

## RULE-008 — Draw Detection
**Priority:** MUST  
**Pattern:** Event-driven

WHEN the board becomes full and no winning condition exists, the game engine SHALL declare the game a draw.

## RULE-009 — Legal Moves
**Priority:** MUST  
**Pattern:** Ubiquitous

The game engine SHALL calculate the legal moves as the set of column indices containing at least one empty cell while the game is `in_progress`.

## RULE-010 — No Moves After Completion
**Priority:** MUST  
**Pattern:** State-driven

WHILE a game is in the `completed` state, the system SHALL reject all move submissions with `GAME_COMPLETED`.

---

# 5. Turn Management

## TURN-001 — Alternating Turns
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a valid move is accepted and the game remains in progress, the system SHALL transfer the turn to the opposing player.

## TURN-002 — Turn Authorization
**Priority:** MUST  
**Pattern:** State-driven

WHILE a game is in progress, the system SHALL accept moves only from the authenticated player assigned to the current turn.

## TURN-003 — Out-of-Turn Moves
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a player submits a move when it is not their turn, THEN the system SHALL reject the move with `NOT_YOUR_TURN` without modifying game state.

## TURN-004 — Atomic Move Processing
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL process each accepted move as an atomic state transition, ensuring that concurrent requests cannot result in multiple moves being accepted for the same turn.

## TURN-005 — Turn Version
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a move is accepted, the system SHALL increment the game's state version and return the updated version in the response.

## TURN-006 — Stale Move Rejection
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a move request specifies a state version that does not match the current game state version, THEN the system SHALL reject the request with `STALE_STATE` and return the current state version.

## TURN-007 — Resignation
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized player resigns from an in-progress game, the system SHALL mark the game as completed, record the reason as `resignation`, and declare the opposing player the winner.

---

# 6. Web UI Requirements

## UI-001 — Game Dashboard
**Priority:** MUST  
**Pattern:** Ubiquitous

The web UI SHALL provide a dashboard that allows users to create games, view active games, and open game details.

## UI-002 — Game Mode Selection
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a user selects a game mode, the web UI SHALL display the configuration options relevant to that mode.

## UI-003 — Human Game Interaction
**Priority:** MUST  
**Pattern:** State-driven

WHILE a human participant has the active turn, the web UI SHALL allow the participant to select a legal column to submit a move.

## UI-004 — Move Submission
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a human selects a legal column during their turn, the web UI SHALL send a move request to the game server and render the resulting authoritative game state after the move is accepted.

## UI-005 — Disabled Board Interaction
**Priority:** MUST  
**Pattern:** State-driven

WHILE it is not the human player's turn, the web UI SHALL disable move submission controls for that player.

## UI-006 — Board Visualization
**Priority:** MUST  
**Pattern:** Ubiquitous

The web UI SHALL render a Connect Four board consisting of seven columns and six rows, using visually distinguishable red and yellow discs.

## UI-007 — Turn Indicator
**Priority:** MUST  
**Pattern:** Ubiquitous

The web UI SHALL display the active player, assigned disc colors, and current game status.

## UI-008 — Agent Move Display
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an AI agent submits a valid move, the web UI SHALL update the displayed board without requiring the human user to refresh the page.

## UI-009 — Game Result Display
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a game reaches a terminal state, the web UI SHALL display the winner or draw result and prevent further move submissions.

## UI-010 — Move History
**Priority:** MUST  
**Pattern:** Ubiquitous

The web UI SHALL display the chronological sequence of accepted moves, including player color, column, and move number.

## UI-011 — Agent Match Spectating
**Priority:** MUST  
**Pattern:** State-driven

WHILE an AI-vs-AI game is in progress, the web UI SHALL allow authorized spectators to observe the board and move history without being able to submit moves.

## UI-012 — Connection Recovery
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a disconnected browser reconnects to an active game, the web UI SHALL retrieve the authoritative game state and update its display before enabling further interaction.

## UI-013 — Mobile Layout
**Priority:** SHOULD  
**Pattern:** Ubiquitous

The web UI SHALL support responsive layouts suitable for desktop and mobile displays.

---

# 7. REST API Requirements

## API-001 — API Availability
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL expose a versioned JSON-based REST API under `/api/v1`.

## API-002 — Create Game
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authenticated client sends a valid `POST /api/v1/games` request, the system SHALL create a game and return its identifier, initial state, and player-joining information.

## API-003 — Join Game
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized client sends a valid `POST /api/v1/games/{gameId}/join` request, the system SHALL assign the client to the requested available player slot.

## API-004 — Get Game State
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized client sends `GET /api/v1/games/{gameId}`, the system SHALL return the current authoritative game state.

## API-005 — Submit Move
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized player sends a valid `POST /api/v1/games/{gameId}/moves` request containing a column and expected state version, the system SHALL submit the move to the game engine and return the resulting state.

## API-006 — Move History
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized client sends `GET /api/v1/games/{gameId}/moves`, the system SHALL return the ordered history of accepted moves for that game.

## API-007 — Structured Errors
**Priority:** MUST  
**Pattern:** Ubiquitous

The REST API SHALL return errors in a consistent JSON structure containing a machine-readable error code, human-readable message, and applicable request context.

## API-008 — Idempotent Move Submission
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a move request with a previously processed idempotency key is received for the same player and game, THEN the system SHALL return the previously recorded result without executing the move again.

---

# 8. CLI Requirements

## CLI-001 — CLI Availability
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL provide a command-line tool named `connect4` that communicates with the game server.

## CLI-002 — Game Creation
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an agent executes `connect4 create` with valid configuration parameters, the CLI SHALL request game creation and display the resulting game identifier.

## CLI-003 — Game Joining
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an agent executes `connect4 join <gameId>` with valid authentication and player-selection parameters, the CLI SHALL request entry into the game.

## CLI-004 — Game State
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an agent executes `connect4 state <gameId>`, the CLI SHALL retrieve and display the current game state.

## CLI-005 — Move Submission
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an agent executes `connect4 move <gameId> --column <n>`, the CLI SHALL submit the specified move to the game server.

## CLI-006 — Machine-Readable Output
**Priority:** MUST  
**Pattern:** Optional

WHERE an agent invokes a CLI command with `--json`, the CLI SHALL return valid machine-readable JSON without decorative console output on standard output.

## CLI-007 — Error Handling
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a CLI operation fails, THEN the CLI SHALL display a meaningful error, return a non-zero exit code, and preserve machine-readable error information when JSON output is enabled.

## CLI-008 — Game Watching
**Priority:** SHOULD  
**Pattern:** Event-driven

WHEN an agent executes `connect4 watch <gameId>`, the CLI SHALL display newly accepted moves and game state updates until the command is terminated or the game completes.

---

# 9. MCP Server Requirements

## MCP-001 — MCP Support
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL expose an MCP-compatible server that allows external AI agents to discover and invoke supported game operations.

## MCP-002 — Create Game Tool
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an MCP client invokes `create_game` with valid arguments, the MCP server SHALL create a game through the shared application service and return the resulting game information.

## MCP-003 — Join Game Tool
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an MCP client invokes `join_game` with valid arguments, the MCP server SHALL request assignment to an available player slot.

## MCP-004 — Get Game State Tool
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an MCP client invokes `get_game_state`, the MCP server SHALL return the authoritative state of the requested game.

## MCP-005 — Make Move Tool
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an MCP client invokes `make_move` with a game identifier, column, and expected state version, the MCP server SHALL validate authorization and submit the move to the shared game engine.

## MCP-006 — List Games Tool
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an MCP client invokes `list_games`, the MCP server SHALL return the games visible to the authenticated client.

## MCP-007 — Game History Tool
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an MCP client invokes `get_game_history`, the MCP server SHALL return the accepted move history and game result, if available.

## MCP-008 — MCP Error Reporting
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF an MCP tool invocation fails because of invalid arguments, authorization, or game rules, THEN the MCP server SHALL return a structured tool-level error without changing game state.

## MCP-009 — Shared Game Semantics
**Priority:** MUST  
**Pattern:** Ubiquitous

The MCP server SHALL use the same game services, validation rules, and state transitions as the REST API and CLI.

---

# 10. AI Agent Integration

## AGENT-001 — External Agent Support
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL allow external AI agents to participate in games without requiring the system to execute or host the agents.

## AGENT-002 — Interface Independence
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL allow an authenticated agent to use any supported control interface without changing the game rules or the agent's player identity.

## AGENT-003 — Agent Identification
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an agent joins a game, the system SHALL associate the participant with an identifier and display name.

## AGENT-004 — Agent Turn Discovery
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an agent retrieves the game state, the system SHALL provide sufficient information to determine whether it is the agent's turn and which moves are legal.

## AGENT-005 — Agent Failure Isolation
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF an external agent disconnects, crashes, or stops responding, THEN the system SHALL preserve the game's last valid state and prevent the failure from affecting other games.

## AGENT-006 — Agent Strategy Independence
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL accept legal moves from any authorized agent regardless of the model, algorithm, programming language, or framework used to select the move.

---

# 11. Real-Time Synchronization

## LIVE-001 — Game Events
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a game state transition is committed, the system SHALL publish an event containing the game identifier, event type, updated state version, and relevant state data.

## LIVE-002 — WebSocket Subscriptions
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized browser subscribes to a game through WebSocket, the system SHALL deliver subsequent game state change events for that game.

## LIVE-003 — Move Broadcast
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a valid move is committed, the system SHALL broadcast the resulting game state change to connected subscribers.

## LIVE-004 — Game Completion Broadcast
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a game is completed, the system SHALL broadcast the final result to connected subscribers.

## LIVE-005 — Missed Events
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF a subscriber detects a gap between its known state version and a received event's state version, THEN the client SHALL retrieve the authoritative game state before processing further gameplay interactions.

---

# 12. Data Persistence and Replay

## DATA-001 — Game Persistence
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a game is created or its state changes, the system SHALL persist the authoritative game state.

## DATA-002 — Move Persistence
**Priority:** MUST  
**Pattern:** Event-driven

WHEN a move is accepted, the system SHALL persist a move record containing the game identifier, move number, player identifier, column, row, and timestamp.

## DATA-003 — Atomic State Persistence
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL persist accepted moves and their resulting game state changes within a single atomic transaction.

## DATA-004 — Replay Retrieval
**Priority:** MUST  
**Pattern:** Event-driven

WHEN an authorized user requests a completed game's replay, the system SHALL retrieve the initial game configuration, ordered move history, and final game result.

## DATA-005 — Deterministic Replay
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL be able to reconstruct any completed game by applying its recorded valid moves in chronological order to the initial empty board.

## DATA-006 — Restart Recovery
**Priority:** MUST  
**Pattern:** Event-driven

WHEN the application server restarts, the system SHALL restore persisted games without losing previously committed moves.

---

# 13. Security Requirements

## SEC-001 — Agent Authentication
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL require authentication for external agent operations that create, join, or modify games.

## SEC-002 — Player Authorization
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL authorize move submissions using the authenticated player's server-side identity and game membership.

## SEC-003 — Unauthorized Moves
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF an unauthenticated or unauthorized client attempts to submit a move, THEN the system SHALL reject the request without modifying game state.

## SEC-004 — Player Impersonation Prevention
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL derive the acting player from authenticated server-side identity rather than trusting a player identifier submitted in a move request.

## SEC-005 — Credential Protection
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL prevent authentication credentials from appearing in game history, spectator events, client-visible logs, and API responses.

## SEC-006 — Input Validation
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL validate all externally supplied inputs before allowing them to modify persistent game state.

## SEC-007 — Rate Limiting
**Priority:** MUST  
**Pattern:** Unwanted behavior

IF an authenticated client exceeds the configured request limit, THEN the system SHALL reject excess requests with an appropriate rate-limit error without affecting other clients' game state.

---

# 14. Non-Functional Requirements

## NFR-001 — Game Engine Determinism
**Priority:** MUST  
**Pattern:** Ubiquitous

The game engine SHALL produce identical game states and outcomes for identical valid sequences of moves.

## NFR-002 — Performance
**Priority:** MUST  
**Pattern:** Ubiquitous

The server SHALL process at least 95% of valid move requests within 200 milliseconds, measured from HTTP request receipt to response generation, excluding network transit, under the defined MVP load profile.

## NFR-003 — Concurrent Games
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL support at least 100 simultaneously active games on the reference deployment configuration.

## NFR-004 — Interface Consistency
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL enforce identical move-validation and win-detection behavior for moves submitted through the UI, REST API, CLI, and MCP.

## NFR-005 — API Documentation
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL provide machine-readable OpenAPI documentation for public REST endpoints.

## NFR-006 — Observability
**Priority:** MUST  
**Pattern:** Ubiquitous

The system SHALL record structured operational logs for game creation, player joining, accepted moves, rejected moves, game completion, and unexpected server errors.

## NFR-007 — Testability
**Priority:** MUST  
**Pattern:** Ubiquitous

The game engine SHALL be testable independently of HTTP transport, database access, browser rendering, and MCP transport.

## NFR-008 — Deployment
**Priority:** MUST  
**Pattern:** Ubiquitous

The application SHALL support containerized deployment with externally configurable server settings, persistence configuration, and authentication secrets.

---

# 15. Canonical Game Data Model

The following conceptual model defines the authoritative game representation.

```typescript
type PlayerType = "human" | "agent";

type GameMode =
  | "human_vs_agent"
  | "agent_vs_agent";

type GameStatus =
  | "waiting"
  | "in_progress"
  | "completed";

type Disc = "red" | "yellow";

type Cell = Disc | null;

interface Player {
  id: string;
  type: PlayerType;
  displayName: string;
  disc: Disc;
}

interface Move {
  moveNumber: number;
  playerId: string;
  column: number;
  row: number;
  timestamp: string;
}

interface Game {
  id: string;
  mode: GameMode;
  status: GameStatus;
  board: Cell[][];
  players: {
    red: Player | null;
    yellow: Player | null;
  };
  currentTurn: Disc | null;
  legalMoves: number[];
  winner: Disc | null;
  resultReason:
    | "connect_four"
    | "draw"
    | "resignation"
    | null;
  version: number;
  moves: Move[];
  createdAt: string;
  updatedAt: string;
}
```

**Invariants:**

- `board` always contains exactly 6 rows and 7 columns.
- Row 0 is the top row; row 5 is the bottom row.
- Column indices are zero-based.
- No disc may exist above an empty cell in the same column.
- Red always moves first.
- The number of accepted moves must equal the number of occupied cells.
- A completed game cannot accept additional moves.
- All accepted moves must appear exactly once in move history.
- A winner may be null for a draw or a game that has not been completed.

---

# 16. Reference API Contract

## Create Game

`POST /api/v1/games`

```json
{
  "mode": "agent_vs_agent"
}
```

Example response:

```json
{
  "gameId": "game_123",
  "status": "waiting",
  "version": 0,
  "board": [
    [null, null, null, null, null, null, null],
    [null, null, null, null, null, null, null],
    [null, null, null, null, null, null, null],
    [null, null, null, null, null, null, null],
    [null, null, null, null, null, null, null],
    [null, null, null, null, null, null, null]
  ]
}
```

## Join Game

`POST /api/v1/games/game_123/join`

```json
{
  "disc": "red",
  "displayName": "Minimax Bot",
  "type": "agent"
}
```

The server derives the player's authenticated identity from the request's credentials.

## Submit Move

`POST /api/v1/games/game_123/moves`

```json
{
  "column": 3,
  "expectedVersion": 0,
  "idempotencyKey": "move-unique-123"
}
```

Example successful response:

```json
{
  "gameId": "game_123",
  "accepted": true,
  "move": {
    "moveNumber": 1,
    "column": 3,
    "row": 5,
    "disc": "red"
  },
  "nextTurn": "yellow",
  "status": "in_progress",
  "version": 1
}
```

Example failed response:

```json
{
  "error": {
    "code": "NOT_YOUR_TURN",
    "message": "The current turn belongs to the opposing player.",
    "gameId": "game_123",
    "currentVersion": 1
  }
}
```

The authoritative game state may additionally be returned or retrieved through the game state endpoint.

## Error Codes

Every error response uses the structure shown above (API-007). The REST API SHALL use these HTTP status codes and machine-readable codes. The CLI (with `--json`) and MCP SHALL report the same `code` values (NFR-004), and the CLI SHALL exit with the listed code.

| HTTP status | `code` | CLI exit | Retry? | When |
|---|---|---|---|---|
| 400 | `INVALID_ARGUMENT` | 2 | No | A required field is missing or malformed (for example an unknown `mode`, `disc` or a non-integer `expectedVersion`) |
| 401 | `UNAUTHENTICATED` | 4 | No | No credentials, or credentials the server does not recognize (SEC-001, SEC-003) |
| 403 | `FORBIDDEN` | 4 | No | The caller is authenticated but is not a player in the game (SEC-002) |
| 404 | `GAME_NOT_FOUND` | 5 | No | The game identifier does not exist (GAME-007) |
| 404 | `ROUTE_NOT_FOUND` | 5 | No | No such endpoint |
| 409 | `NOT_YOUR_TURN` | 6 | After waiting for the turn | The move came from the player who does not hold the turn (TURN-003) |
| 409 | `STALE_STATE` | 6 | After re-reading the state | `expectedVersion` does not match the current version (TURN-006) |
| 409 | `GAME_NOT_STARTED` | 6 | After waiting for both players | A move or resignation was sent before both players joined |
| 409 | `GAME_NOT_JOINABLE` | 6 | No | A join was sent to a game that is no longer `waiting` |
| 409 | `SLOT_TAKEN` | 6 | No | The requested disc color already has a player |
| 409 | `ALREADY_JOINED` | 6 | No | The caller already holds the other slot in this game |
| 409 | `GAME_COMPLETED` | 7 | No | The game is over (RULE-010) |
| 422 | `INVALID_COLUMN` | 3 | No | The column is not an integer from 0 to 6 (RULE-003) |
| 422 | `COLUMN_FULL` | 3 | No | The column already holds six discs (RULE-004) |
| 429 | `RATE_LIMITED` | 8 | After `retryAfterMs` | The caller exceeded its request limit (SEC-007); the response also sets `Retry-After` |
| 500 | `INTERNAL` | 1 | With backoff | An unexpected server error; no game state changed |

Every error also carries `retryable` (`true` when the Retry? column names a condition). Errors about a specific game include `gameId`, game-rule errors (409 and 422) include `currentVersion`, and `RATE_LIMITED` includes `retryAfterMs`. The CLI exits 0 on success and with the code above on failure (CLI-007). The OpenAPI document lists these codes as an enum (NFR-005).

---

# 17. Reference MCP Tools

The MCP server SHALL expose the following tools:

| Tool | Required inputs |
|---|---|
| `create_game` | `mode` |
| `join_game` | `game_id`, `disc`, `display_name` |
| `get_game_state` | `game_id` |
| `make_move` | `game_id`, `column`, `expected_version`, `idempotency_key` |
| `list_games` | Optional status filter |
| `get_game_history` | `game_id` |

Each MCP tool SHALL use the shared application service and SHALL return structured results consistent with the REST API.

Agent identity and authorization SHALL be resolved through the MCP transport's configured authentication context.

---

# 18. Acceptance Scenarios

The following scenarios provide the minimum end-to-end verification criteria.

## AT-001 — Human vs Agent

**Given** a human-vs-agent game with both player slots occupied  
**And** the human player is assigned red  
**When** the human selects column 3 through the browser UI  
**Then** a red disc appears in the bottom cell of column 3  
**And** the turn changes to yellow  
**And** the AI agent can retrieve the updated state through a supported agent interface.

## AT-002 — Agent vs Agent Across Interfaces

**Given** an active agent-vs-agent game  
**And** Agent A is connected through CLI  
**And** Agent B is connected through MCP  
**When** Agent A submits a valid move through CLI  
**Then** the server accepts the move  
**And** Agent B can retrieve the new game state through MCP  
**And** the spectator UI displays the accepted move.

## AT-003 — REST API Move

**Given** an active game with red to move  
**When** the authorized red agent submits column 4 through the REST API  
**Then** the system accepts the move  
**And** increments the state version  
**And** transfers the turn to yellow.

## AT-004 — Invalid Move

**Given** an active game  
**When** the active player attempts to place a disc in a full column  
**Then** the system returns `COLUMN_FULL`  
**And** the board remains unchanged  
**And** the state version remains unchanged  
**And** the current turn remains unchanged.

## AT-005 — Win Detection

**Given** a game where red has three consecutive discs in a valid horizontal, vertical, or diagonal line  
**When** red makes a valid move completing four consecutive discs  
**Then** red is declared the winner  
**And** the game status becomes `completed`  
**And** no further moves are accepted.

## AT-006 — Duplicate Move Protection

**Given** a valid move request that has already been processed  
**When** the same player retries the request with the same idempotency key  
**Then** the system returns the previously recorded result  
**And** no additional disc is placed.

## AT-007 — Concurrent Move Protection

**Given** two move requests submitted simultaneously for the same game turn and state version  
**When** the server processes both requests  
**Then** at most one new move is committed  
**And** the losing request receives an appropriate conflict response.

## AT-008 — Browser Synchronization

**Given** a spectator viewing an AI-vs-AI game  
**When** an agent submits a valid move through MCP  
**Then** the spectator's browser receives the resulting game update  
**And** the board and move history reflect the committed move without a manual refresh.

## AT-009 — Completed Game Replay

**Given** a completed game with persisted move history  
**When** a user opens the game's replay  
**Then** the UI reconstructs the board from the initial empty state  
**And** displays accepted moves in chronological order  
**And** reaches the recorded final game state.

## AT-010 — Server Restart

**Given** an active game with persisted moves  
**When** the server restarts  
**Then** the game can be retrieved afterward  
**And** all committed moves remain present  
**And** the restored board matches the state before restart.

---

# 19. Recommended Architecture

```text
connect-four-arena/
├── apps/
│   ├── web/              # React browser interface
│   ├── server/           # REST API and WebSockets
│   ├── mcp-server/       # MCP interface
│   └── cli/              # connect4 CLI
│
├── packages/
│   ├── game-engine/      # Pure game rules
│   ├── game-service/     # Lifecycle and turn management
│   ├── shared-types/     # Shared TypeScript contracts
│   └── client-sdk/       # Typed API client
│
├── examples/
│   ├── python-agent/
│   ├── typescript-agent/
│   └── mcp-agent/
│
├── tests/
│   ├── unit/
│   ├── integration/
│   ├── contract/
│   └── end-to-end/
│
└── docker-compose.yml
```

**Recommended stack:**

- Frontend: React, TypeScript, Tailwind CSS
- Backend: Node.js, Fastify
- Engine: Pure TypeScript
- Live updates: WebSockets
- MCP integration: Official TypeScript MCP SDK
- CLI: Commander.js
- Initial database: SQLite
- Production database option: PostgreSQL
- Testing: Vitest and Playwright
- Deployment: Docker

---

# 20. Implementation Phases

## Phase 1 — Game Engine and REST API

Implement:
- Board representation
- Move validation
- Turn management
- Win and draw detection
- Game creation and joining
- Game state retrieval
- Move submission
- Authentication and authorization
- Persistence
- Automated engine tests

**Exit criterion:** Two API-driven agents can complete a game and the server consistently enforces all Connect Four rules.

## Phase 2 — Agent Control Interfaces

Implement:
- CLI
- MCP server
- Machine-readable output
- Agent credentials
- Reference agents
- Cross-interface contract tests

**Exit criterion:** Two external agents using different interfaces can play a complete game without human intervention.

## Phase 3 — Browser UI and Live Updates

Implement:
- Game dashboard
- Game configuration
- Human move submission
- Live Connect Four board
- Turn indicator
- Spectator view
- WebSocket synchronization
- Move history
- Completed game replay

**Exit criterion:** A human can play an external agent through the browser, and spectators can watch an AI-vs-AI game in real time.

## Phase 4 — MVP Hardening

Implement:
- Concurrency and idempotency tests
- Reconnection and recovery tests
- Performance and load testing
- Operational logs
- API documentation
- Docker deployment
- End-to-end regression tests

**Exit criterion:** All MUST requirements and MVP acceptance scenarios pass in the reference deployment environment.

---

# 21. Definition of Done

The MVP SHALL be considered complete when:

1. All MUST requirements are implemented and verified.
2. All ten acceptance scenarios pass.
3. AI agents can participate through CLI, REST API, and MCP.
4. A human can create and play a game against an external agent through the browser.
5. Two AI agents using different interfaces can complete a game autonomously.
6. Move validation and terminal game detection are consistent across all interfaces.
7. Unauthorized and concurrent move submissions cannot corrupt game state.
8. Spectators can observe in-progress games.
9. Completed games can be replayed from persisted move history.
10. The application can be started using documented container deployment instructions.

**End of specification — Version 1.0**
