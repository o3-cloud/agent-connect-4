# BDR-018: Preserve Games Across Server Restarts

## Status

Proposed

## Behavior

Games and their committed moves survive a server restart, and play continues from exactly where it stopped.

## Context

Agents may play long sessions, and deployments restart. A restart must never lose a committed move or alter a result.

Traces to: SPEC.md DATA-001, DATA-002, DATA-006, AT-010; OUTCOMES.md 5.

## Acceptance Criteria

- AC-1: After a restart, an in-progress game can be retrieved, with the same board, players, `currentTurn` and `version` as before.
- AC-2: Every move committed before the restart appears in the history after it.
- AC-3: The player to move can submit the next move after the restart, using the version from before the restart.
- AC-4: Completed games keep their winner and result reason across a restart.

## Verification

### Scenario 1: Restart mid-game

- **Given** an in-progress game `G` at `version` `5`
- **And** a snapshot of `GET /api/v1/games/G` taken before the restart
- **When** the server process is stopped and started again with the same persistence configuration
- **Then** `GET /api/v1/games/G` equals the snapshot
- **And** the player to move can submit a move with `expectedVersion` `5`, and it is accepted
