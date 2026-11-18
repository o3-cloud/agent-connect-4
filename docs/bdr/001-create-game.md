# BDR-001: Create a Game

## Status

Accepted

## Behavior

An authenticated client can create a new Connect Four game in either supported mode and receive its identifier and initial state.

## Context

Every match starts with game creation, through any interface. Agents and humans need a stable identifier to share, and an empty, predictable starting state.

Traces to: SPEC.md GAME-001, GAME-002, API-002, §16 (Create Game); OUTCOMES.md 1, 3.

## Acceptance Criteria

- AC-1: `POST /api/v1/games` with `{"mode": "agent_vs_agent"}` or `{"mode": "human_vs_agent"}` returns a `gameId` that no other game uses.
- AC-2: The response reports `status` `waiting`, `version` `0`, and a `board` of exactly 6 rows × 7 columns in which every cell is `null`.
- AC-3: The response includes information on how to join, such as the open player slots.
- AC-4: A request with a `mode` outside the two supported values is rejected with a structured error, and no game appears in the game list.
- AC-5: Creating two games returns two different `gameId` values.

## Verification

### Scenario 1: Create an agent-vs-agent game

- **Given** an authenticated agent credential
- **When** the client sends `POST /api/v1/games` with body `{"mode": "agent_vs_agent"}`
- **Then** the response contains a non-empty `gameId`, `status` `waiting` and `version` `0`
- **And** `board` has 6 rows, each with 7 `null` cells

### Scenario 2: Unsupported mode is rejected

- **Given** an authenticated agent credential
- **When** the client sends `POST /api/v1/games` with body `{"mode": "human_vs_human"}`
- **Then** the response has a non-2xx status and a body of the form `{"error": {"code": ..., "message": ...}}`
- **And** `GET /api/v1/games` lists no new game

### Scenario 3: Identifiers are unique

- **Given** an authenticated agent credential
- **When** the client creates two games one after the other
- **Then** the two `gameId` values differ
