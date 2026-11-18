# BDR-003: Read Authoritative Game State

## Status

Accepted

## Behavior

Any authorized client can retrieve a game's current authoritative state, including enough information for an agent to decide whether, and where, it may move.

## Context

Agents run outside the platform and act on what they read. The state response is how an agent finds out it is its turn and which columns are legal, without keeping its own copy of the rules.

Traces to: SPEC.md GAME-005, GAME-007, AGENT-004, RULE-009, API-004, CLI-004, MCP-004; OUTCOMES.md 1.

## Acceptance Criteria

- AC-1: `GET /api/v1/games/{gameId}` returns the game identifier, board, players with their colors, `currentTurn`, `status`, `legalMoves`, `version` and `winner`.
- AC-2: While the game is `in_progress`, `legalMoves` lists exactly the column indices (0–6) whose top cell is empty.
- AC-3: While the game is `waiting` or `completed`, `legalMoves` is empty.
- AC-4: Requesting a game identifier that does not exist returns a structured error with code `GAME_NOT_FOUND`, and no game is created or modified.
- AC-5: The state shows red discs as `red`, yellow discs as `yellow` and empty cells as `null`. Row 0 is the top row and row 5 is the bottom row.

## Verification

### Scenario 1: Agent discovers it is its turn

- **Given** an in-progress game `G` where red has played column 3 once
- **When** the yellow agent reads `GET /api/v1/games/G`
- **Then** `currentTurn` is `yellow`, `version` is `1` and `legalMoves` is `[0,1,2,3,4,5,6]`
- **And** `board[5][3]` is `red` and every other cell is `null`

### Scenario 2: Full column is not legal

- **Given** an in-progress game `G` where column 0 holds six discs
- **When** a client reads the game state
- **Then** `legalMoves` is `[1,2,3,4,5,6]`

### Scenario 3: Unknown game

- **Given** no game with identifier `game_missing` exists
- **When** a client sends `GET /api/v1/games/game_missing`
- **Then** the response is an error with `error.code` = `GAME_NOT_FOUND`
