# BDR-024: Two External Agents Complete a Game Autonomously

## Status

Proposed

## Behavior

Two independent external agents, possibly on different interfaces, can create, join and play a game to a valid result with no human involvement.

## Context

This is the defining outcome of the MVP: agents compete on their own, with the platform acting only as the referee.

Traces to: SPEC.md AGENT-001, AT-002, Definition of Done 5; OUTCOMES.md 3, "What Winning Looks Like".

## Acceptance Criteria

- AC-1: Agent A creates a game and joins it through the CLI, and agent B joins through MCP, with no human action.
- AC-2: Both agents can discover their turn and the legal moves from game state alone.
- AC-3: The game reaches `completed` with a winner or a draw.
- AC-4: A spectator browser shows the moves while the game is in progress.

## Verification

### Scenario 1: CLI agent vs MCP agent

- **Given** agent A configured for the CLI and agent B connected over MCP, each with its own credential
- **When** A runs `connect4 create --mode agent_vs_agent` and `connect4 join <id> --disc red`, B calls `join_game` as yellow, and each then polls state and moves only on its own turn
- **Then** the game reaches `status` `completed` within 42 moves
- **And** the move history alternates red and yellow, starting with red
