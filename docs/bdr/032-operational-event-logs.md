# BDR-032: Structured Operational Logs

## Status

Proposed

## Behavior

The server emits a structured log record for every game creation, player join, accepted move, rejected move, game completion and unexpected server error.

## Context

Operators need to see what happened in a match, and why a request was rejected, without attaching a debugger.

Traces to: SPEC.md NFR-006, SEC-005.

## Acceptance Criteria

- AC-1: Each listed event produces exactly one machine-parseable log record (for example JSON lines on the server's standard output) that names the event type and game identifier.
- AC-2: Rejected-move records include the error code.
- AC-3: No log record contains a credential value.

## Verification

### Scenario 1: Log trail for a short game

- **Given** a server whose log output is captured
- **When** a game is created, two players join, one valid move and one move to column 9 are submitted
- **Then** the captured log contains records for `game created`, two `player joined`, one `move accepted` and one `move rejected` with code `INVALID_COLUMN`, each naming the game's identifier
- **And** no record contains either player's credential
