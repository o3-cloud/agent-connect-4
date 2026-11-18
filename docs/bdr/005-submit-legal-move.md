# BDR-005: Submit a Legal Move

## Status

Accepted

## Behavior

The player whose turn it is can drop a disc into any non-full column, and the disc lands in the lowest empty cell, the version increments and the turn passes to the opponent.

## Context

This is the core gameplay loop. Every interface depends on the same predictable result for a valid move: gravity, alternating turns, and a version number that clients use to stay in sync.

Traces to: SPEC.md RULE-002, TURN-001, TURN-005, API-005, CLI-005, MCP-005, AT-001, AT-003, §16 (Submit Move).

## Acceptance Criteria

- AC-1: A move with a valid column places the mover's disc in the lowest empty row of that column.
- AC-2: The response reports `accepted` `true`, the move's `moveNumber`, `column`, `row` and `disc`, the `nextTurn`, the `status`, and the new `version`.
- AC-3: Each accepted move increases `version` by exactly 1.
- AC-4: After an accepted move that does not end the game, `currentTurn` is the opposing color.

## Verification

### Scenario 1: First move lands on the bottom row

- **Given** an in-progress game `G` at `version` `0` with red to move
- **When** the red player sends `POST /api/v1/games/G/moves` with `{"column": 4, "expectedVersion": 0, "idempotencyKey": "m1"}`
- **Then** the response has `accepted` = `true`, `move.row` = `5`, `move.disc` = `red`, `nextTurn` = `yellow` and `version` = `1`
- **And** `GET /api/v1/games/G` shows `board[5][4]` = `red`

### Scenario 2: Discs stack

- **Given** an in-progress game where `board[5][3]` is `red` and yellow is to move
- **When** yellow plays column 3 with the current version
- **Then** the move lands in `row` `4`, and red is to move
