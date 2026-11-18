# BDR-006: Reject Invalid Moves Without Changing State

## Status

Accepted

## Behavior

The system rejects a move to a column that is out of range, not an integer, or full, and the board, turn and version stay unchanged.

## Context

External agents will send bad input, whether through bugs, hallucinations or hostile behaviour. Rejection must be explicit, machine-readable and side-effect free, so the agent can correct itself and the game stays valid.

Traces to: SPEC.md RULE-003, RULE-004, SEC-006, AT-004.

## Acceptance Criteria

- AC-1: A column below 0 or above 6 is rejected with error code `INVALID_COLUMN`.
- AC-2: A non-integer column (for example `3.5` or `"three"`) is rejected with error code `INVALID_COLUMN`.
- AC-3: A move into a column that already holds six discs is rejected with error code `COLUMN_FULL`.
- AC-4: After any rejected move, `board`, `currentTurn` and `version` are the same as before the request.

## Verification

### Scenario 1: Out-of-range column

- **Given** an in-progress game `G` at `version` `0` with red to move
- **When** red submits `{"column": 7, "expectedVersion": 0}`
- **Then** the response is an error with `error.code` = `INVALID_COLUMN`
- **And** `GET /api/v1/games/G` still shows `version` `0`, `currentTurn` `red` and an empty board

### Scenario 2: Full column

- **Given** an in-progress game where the moves 0,0,0,0,0,0 have been played and red is to move at `version` `6`
- **When** red submits `{"column": 0, "expectedVersion": 6}`
- **Then** the response is an error with `error.code` = `COLUMN_FULL`
- **And** the board, `currentTurn` (`red`) and `version` (`6`) are unchanged
