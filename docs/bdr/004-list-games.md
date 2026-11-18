# BDR-004: List Visible Games

## Status

Proposed

## Behavior

An authorized client can list the games visible to it, with each game's identifier, mode, players and status.

## Context

Agents and humans have to find games to join or watch, so a dashboard or agent needs an index of games without already knowing their identifiers.

Traces to: SPEC.md GAME-008, MCP-006, UI-001, §17 (`list_games` with optional status filter); OUTCOMES.md 4.

## Acceptance Criteria

- AC-1: The list includes each visible game's identifier, mode, players and status.
- AC-2: A newly created game appears in the list with status `waiting`.
- AC-3: Filtering by status returns only games in that status.
- AC-4: The list never includes credentials or secrets.

## Verification

### Scenario 1: New game appears in the list

- **Given** an authenticated client creates game `G` in mode `agent_vs_agent`
- **When** the client lists games
- **Then** the list contains an entry for `G` with mode `agent_vs_agent` and status `waiting`

### Scenario 2: Filter by status

- **Given** one `waiting`, one `in_progress` and one `completed` game
- **When** the client lists games with the status filter `in_progress` (for example the MCP tool `list_games` with `{"status": "in_progress"}`)
- **Then** every returned entry has status `in_progress`, and the in-progress game is included
