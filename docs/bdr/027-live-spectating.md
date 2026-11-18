# BDR-027: Watch Games Live Without Refreshing

## Status

Proposed

## Behavior

Anyone authorized to view a game can watch its board and move history update live as moves arrive through any interface, without being able to move.

## Context

Every game should be observable as it happens. That makes agent play transparent and lets humans follow along without reloading.

Traces to: SPEC.md UI-008, UI-011, LIVE-001..004, AT-008; OUTCOMES.md 4.

## Acceptance Criteria

- AC-1: When a move is accepted through any interface, an open game page shows the new disc and history entry without a manual refresh.
- AC-2: When a game completes, open game pages show the final result without a manual refresh.
- AC-3: A spectator of an agent-vs-agent game has no control that submits a move.
- AC-4: In the open page, the order of updates matches the order of moves in the history.

## Verification

### Scenario 1: Spectator sees an MCP move

- **Given** a spectator viewing an in-progress agent-vs-agent game `G` in a browser
- **When** an agent submits a valid move to `G` through MCP
- **Then** within 2 seconds, without a page reload, the spectator's board shows the new disc and the move history lists the move

### Scenario 2: Spectator sees the result

- **Given** a spectator viewing a game that is one move from a win
- **When** the winning move is submitted through the CLI
- **Then** the page shows the winner without a reload
