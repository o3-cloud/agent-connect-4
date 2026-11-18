# BDR-011: Reject Stale and Concurrent Moves

## Status

Proposed

## Behavior

A move is accepted only if it names the game's current version, so at most one move is ever committed per turn, even when requests arrive at the same time.

## Context

Agents poll, retry, and sometimes race themselves across several processes. Version checks stop a move based on an outdated board from being applied, and stop simultaneous requests from placing two discs in one turn.

Traces to: SPEC.md TURN-004, TURN-005, TURN-006, AT-007; OUTCOMES.md 5.

## Acceptance Criteria

- AC-1: A move whose `expectedVersion` differs from the current version is rejected with error code `STALE_STATE`, and the error reports the current version.
- AC-2: A rejected stale move does not change the board, turn or version.
- AC-3: When two moves for the same turn and version are sent at the same time, at most one is accepted, and the other receives a conflict error (`STALE_STATE` or `NOT_YOUR_TURN`).
- AC-4: After any race, the number of discs on the board equals the number of moves in the history.

## Verification

### Scenario 1: Stale version

- **Given** an in-progress game `G` at `version` `3` with yellow to move
- **When** yellow submits a move with `expectedVersion` `2`
- **Then** the response is an error with `error.code` = `STALE_STATE` and `error.currentVersion` = `3`
- **And** `version` is still `3`

### Scenario 2: Simultaneous moves

- **Given** an in-progress game `G` at `version` `0` with red to move
- **When** red sends two move requests at once with `expectedVersion` `0`, one for column 2 and one for column 4, using different idempotency keys
- **Then** exactly one request succeeds and the other returns a conflict error
- **And** `G` has `version` `1` and exactly one disc on the board
