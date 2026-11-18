# BDR-016: Ordered Move History

## Status

Proposed

## Behavior

Any authorized client can retrieve the complete, ordered history of accepted moves for a game, and the game result once it has one.

## Context

Observers, agents and reviewers need to see how a game unfolded, not just its final board. The history is also the input for replay.

Traces to: SPEC.md API-006, MCP-007, UI-010, DATA-002; OUTCOMES.md 4.

## Acceptance Criteria

- AC-1: `GET /api/v1/games/{gameId}/moves` returns the accepted moves in order, each with move number, player identifier, color, column, row and timestamp.
- AC-2: Move numbers start at 1 and increase by 1 with no gaps or duplicates.
- AC-3: Rejected moves never appear in the history.
- AC-4: For a completed game, the history response (and the MCP `get_game_history` result) includes the winner and result reason.

## Verification

### Scenario 1: History after three moves and a rejection

- **Given** an in-progress game where columns 3, 3 and 4 were accepted and one move to column 9 was rejected
- **When** a client requests the move history
- **Then** it lists exactly 3 moves, numbered 1, 2 and 3, with columns 3, 3 and 4 and colors red, yellow and red
- **And** the rows are 5, 4 and 5
