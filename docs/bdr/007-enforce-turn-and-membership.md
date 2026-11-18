# BDR-007: Enforce Turn Order and Player Membership

## Status

Proposed

## Behavior

Only the authenticated player who holds the current turn can make a move; moves from the opponent, from non-players or from unauthenticated callers are rejected without changing state.

## Context

Fair play needs the server to enforce whose move it is. The acting player comes from the server's view of who is calling, never from what the request claims, so one agent cannot move for another.

Traces to: SPEC.md TURN-002, TURN-003, SEC-002, SEC-003, SEC-004; OUTCOMES.md 5.

## Acceptance Criteria

- AC-1: A move from the player who does not hold the turn is rejected with error code `NOT_YOUR_TURN`.
- AC-2: A move from an authenticated client that is not a player in the game is rejected.
- AC-3: A move without valid credentials is rejected.
- AC-4: Including another player's identifier in the move body does not let the caller move as that player.
- AC-5: After any of these rejections, the board, turn and version are unchanged.

## Verification

### Scenario 1: Out-of-turn move

- **Given** an in-progress game `G` at `version` `0` with red to move
- **When** the yellow player submits a move to column 3 with `expectedVersion` `0`
- **Then** the response is an error with `error.code` = `NOT_YOUR_TURN` and `error.currentVersion` = `0`
- **And** the board is still empty

### Scenario 2: Non-participant

- **Given** an in-progress game `G` between agents A and B
- **When** agent C, authenticated but not in `G`, submits a move
- **Then** the response is an error and `G`'s state is unchanged

### Scenario 3: Impersonation attempt

- **Given** an in-progress game with red (agent A) to move
- **When** agent B submits a move whose body also contains `"playerId"` set to agent A's identifier
- **Then** the move is rejected and no disc is placed
