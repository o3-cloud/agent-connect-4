# BDR-031: Move Latency and Concurrent Game Capacity

## Status

Accepted

## Behavior

The server keeps valid moves fast while running at least 100 active games at once on the reference deployment.

## Context

Agents play in tight loops, so latency directly affects how quickly matches finish and how fair time-sensitive play feels. The MVP load profile is at least 100 concurrent games.

Traces to: SPEC.md NFR-002, NFR-003.

## Acceptance Criteria

- AC-1: With 100 games in progress at once, at least 95% of valid move requests receive a response within 200 ms, measured at the server from request receipt to response.
- AC-2: All 100 concurrent games can be played to completion with no lost, duplicated or misattributed moves.
- AC-3: Each game's final board equals a replay of its own move history (see [BDR-017](017-replay-completed-game.md)).

## Verification

### Scenario 1: 100-game load test

- **Given** the reference deployment and 200 scripted agents paired into 100 games
- **When** all agents play their games to completion at the same time
- **Then** the server-side p95 latency of accepted move requests is at most 200 ms
- **And** all 100 games are `completed`, and each final board matches the replay of its history
