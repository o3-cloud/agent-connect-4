# BDR-013: Authenticate Agent Operations and Protect Credentials

## Status

Accepted

## Behavior

Creating, joining and moving require valid credentials, and credentials never appear in any response, history, event or log that the system exposes.

## Context

The platform is open to any agent, so identity has to come from credentials the server checks. Those credentials must never leak to spectators or opponents.

Traces to: SPEC.md SEC-001, SEC-003, SEC-005.

## Acceptance Criteria

- AC-1: Creating a game without credentials is rejected, and no game is created.
- AC-2: Joining a game without credentials is rejected, and the slot stays open.
- AC-3: Submitting a move without credentials, or with invalid ones, is rejected, and the game is unchanged.
- AC-4: The credential value used in a test never appears in game state, move history, game lists, live updates sent to spectators, or CLI and MCP output.

## Verification

### Scenario 1: Unauthenticated create

- **Given** no credentials
- **When** a client sends `POST /api/v1/games` with `{"mode": "agent_vs_agent"}`
- **Then** the response is an error and the game list does not grow

### Scenario 2: Credential never leaks

- **Given** agents A and B, using known credential strings, play a full game
- **When** a test collects the state, move history, game list and spectator updates for that game
- **Then** none of the collected output contains either credential string
