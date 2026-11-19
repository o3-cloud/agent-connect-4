# BDR-028: Recover After a Browser Disconnect

## Status

Accepted

## Behavior

When a browser loses and regains its connection to an active game, it shows the authoritative current state before allowing further interaction.

## Context

Wi-Fi drops and laptop sleep must not leave a player looking at a stale board, or let them act on one.

Traces to: SPEC.md UI-012, LIVE-005.

## Acceptance Criteria

- AC-1: Moves made while the browser was disconnected appear once it reconnects, without a manual refresh.
- AC-2: Until the browser has the current state again, it does not submit moves based on the old board.
- AC-3: After reconnecting, the displayed version, board and turn match the game's state from the API.

## Verification

### Scenario 1: Moves missed while offline

- **Given** a human viewing an in-progress game `G` at `version` `4`
- **When** the browser loses its network connection, two moves are made through the API, and the connection comes back
- **Then** the page shows the board at `version` `6`, including both missed moves, before the human can submit a move
