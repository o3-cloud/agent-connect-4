# BDR-002: Join a Game and Start Play

## Status

Proposed

## Behavior

A participant can claim an open red or yellow slot in a waiting game, and the game starts with red to move once both slots are filled.

## Context

Matches are only playable once both seats are taken. Participants must know which color they hold, and red always moves first so that turn order is predictable for every agent.

Traces to: SPEC.md GAME-003, GAME-004, AGENT-003, API-003, MCP-003, CLI-003, §16 (Join Game); OUTCOMES.md 1, 3.

## Acceptance Criteria

- AC-1: Joining an open slot assigns the participant to that color, and later state reads show the participant's display name in that slot.
- AC-2: After exactly one slot is filled, the game's `status` remains `waiting` and `currentTurn` is `null`.
- AC-3: When the second slot is filled, `status` becomes `in_progress` and `currentTurn` is `red`.
- AC-4: Attempting to join a slot that is already taken is rejected with a structured error, and the slot's occupant is unchanged.
- AC-5: The joining participant's identity comes from their credential. A player identifier supplied in the request body has no effect on who occupies the slot.

## Verification

### Scenario 1: Two agents fill a game

- **Given** a game `G` in `waiting` status with both slots open
- **When** agent A sends `POST /api/v1/games/G/join` with `{"disc": "red", "displayName": "Alpha", "type": "agent"}`
- **And** agent B sends `POST /api/v1/games/G/join` with `{"disc": "yellow", "displayName": "Beta", "type": "agent"}`
- **Then** `GET /api/v1/games/G` shows `players.red.displayName` = `Alpha`, `players.yellow.displayName` = `Beta`
- **And** `status` is `in_progress` and `currentTurn` is `red`

### Scenario 2: Game waits for the second player

- **Given** a game `G` where only red has joined
- **When** a client reads `GET /api/v1/games/G`
- **Then** `status` is `waiting` and `currentTurn` is `null`

### Scenario 3: Taken slot is rejected

- **Given** a game `G` where agent A holds red
- **When** agent C requests to join `G` as `red`
- **Then** the response is a structured error
- **And** `players.red.displayName` is still `Alpha`
