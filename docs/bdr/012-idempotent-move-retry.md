# BDR-012: Idempotent Move Retries

## Status

Accepted

## Behavior

Retrying a move with the same idempotency key returns the originally recorded result and never places a second disc.

## Context

Networks drop responses. An agent that cannot tell whether its move landed must be able to retry safely without losing its turn or corrupting the game.

Traces to: SPEC.md API-008, AT-006.

## Acceptance Criteria

- AC-1: A second request from the same player in the same game with the same idempotency key returns the same move result (`moveNumber`, `column`, `row`, `disc`) as the first.
- AC-2: The retry does not place a disc, change the turn or increase the version.
- AC-3: The retry succeeds even if the game state has changed since the original request.
- AC-4: The same key used by a different player, or in a different game, is not treated as a retry.

## Verification

### Scenario 1: Retry after a lost response

- **Given** an in-progress game `G` at `version` `0` with red to move
- **When** red submits `{"column": 3, "expectedVersion": 0, "idempotencyKey": "k-1"}`
- **And** red submits the identical request again
- **Then** both responses report `move.moveNumber` `1`, `move.column` `3` and `move.row` `5`
- **And** `G` has `version` `1` and one move in its history
