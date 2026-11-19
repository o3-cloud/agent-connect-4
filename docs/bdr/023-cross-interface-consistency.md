# BDR-023: Identical Rules on Every Interface

## Status

Accepted

## Behavior

Moves made through the browser, REST API, CLI or MCP follow identical validation and win-detection rules, and act on the same shared game.

## Context

Fairness means it doesn't matter how a participant connects. An agent may also switch interfaces mid-game without changing its identity.

Traces to: SPEC.md NFR-004, AGENT-002, AGENT-006, MCP-009; OUTCOMES.md 5.

## Acceptance Criteria

- AC-1: The same move sequence submitted through REST, the CLI or MCP produces the same boards, versions, errors and results.
- AC-2: A move made through one interface is immediately visible in the state read through every other interface.
- AC-3: An agent can submit one move through the CLI and its next move through MCP with the same credential, and both are attributed to the same player.
- AC-4: The platform accepts any legal move regardless of which agent, model or language chose it.

## Verification

### Scenario 1: Interface parity

- **Given** three new in-progress games
- **When** the column sequence 0,0,1,1,2,2,3 is played via REST in the first, via the CLI in the second and via MCP in the third
- **Then** all three games end `completed` with `winner` `red`, `resultReason` `connect_four`, `version` `7` and identical boards

### Scenario 2: Switching interfaces mid-game

- **Given** an in-progress game where agent A is red
- **When** agent A plays its first move with `connect4 move` and its second with the MCP tool `make_move`
- **Then** the history attributes both moves to the same player identifier
