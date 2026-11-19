---
title: "Walking skeleton: engine → game service → REST → two agents play a full game"
status: closed
source: backlog
type: delivery
priority: 2
blocked_by: [delivery-rename-module-and-cmd-layout]
branch: main
commit: 4218fba
found: 2026-11-18
outcomes: [1, 3, 5]
spec: [SPEC §20 Phase 1, RULE-001..010, GAME-001..008, TURN-001..006, API-001..007, NFR-007]
bdr: [BDR-001, BDR-002, BDR-003, BDR-005, BDR-006, BDR-007, BDR-008]
---

# Walking skeleton: engine → game service → REST → two agents play a full game

Promoted from `backlog/idea-walking-skeleton-first.md`.

## Why

All six outcomes are at 0 acceptance criteria met because no Connect Four code exists yet. Building the system layer by layer delays the first moment a whole game can be played. The thinnest end-to-end slice starts meeting criteria in several BDRs at once and gives every later capability something to attach to.

## Scope

Build on a short-lived `feat/walking-skeleton` branch, test first (`docs/specs/practices/tdd.md`):

1. `internal/engine`: a pure 6 × 7 board, drop into column, horizontal, vertical and diagonal win checks, draw check, legal moves. Port the cases from `prototype/arena.test.js` and the move sequences in BDR-008 and BDR-009.
2. `internal/game`: game service with an in-memory store, a mutex per game, versions, turn order, and join rules (two slots, red moves first).
3. `internal/api`: Huma routes under `/api/v1` (ADR-008) for create, join, get state and submit move. Errors use the `{"error": {code, message, ...}}` shape and the HTTP statuses in SPEC.md §16 "Error Codes" from the start.
4. A Go integration test in which two HTTP clients play a game to `completed`.

Out of scope for now, added afterwards one at a time behind the slice: persistence, authentication (use a fixed test identity per client for now), idempotency keys, rate limits, the CLI, MCP and the browser UI.

## Acceptance

- [x] `make check` passes on main with the skeleton merged, coverage ≥ 80%.
- [x] An end-to-end test drives two HTTP clients through create → join → join → alternate moves → `completed` with a winner (BDR-008, horizontal sequence 0,0,1,1,2,2,3).
- [x] `internal/engine` imports nothing from HTTP, storage or the game service (NFR-007).
- [x] The create, join, state and legal-move scenarios in BDR-001, BDR-002, BDR-003 and BDR-005 pass as HTTP tests.
- [x] `INVALID_COLUMN`, `COLUMN_FULL`, `NOT_YOUR_TURN`, `STALE_STATE` and `GAME_COMPLETED` come back with the SPEC.md §16 HTTP statuses (BDR-006, BDR-007, BDR-008).
- [x] Unknown routes still return 404.

## Status of earlier blockers

All resolved: the Go toolchain pin, the govulncheck failures and the catch-all route are fixed, the BDRs and SPEC.md are on main, and SPEC.md §16 pins the error codes.

## Resolution

Merged to main 2026-11-18 as 2d4aece (engine), 2bcfa94 (game service), eee9bc7 (REST API). Verified:

- `make check` PASS on main, coverage 95.7%, govulncheck clean; `make health` no issues.
- `TestAPITwoAgentsPlayAFullGame` drives two HTTP clients create -> join -> join -> 0,0,1,1,2,2,3 -> `completed`, winner red; also played black-box with curl against `bin/arena`.
- `TestEngineImportsOnlyPureStandardLibrary` enforces NFR-007 (engine imports only `errors`).
- HTTP tests cover BDR-001, 002, 003, 005, 006, 007, 008, 009 and every listed error code with its SPEC.md §16 status.
- Unknown `/api/v1/*` paths return 404 `ROUTE_NOT_FOUND`; other unknown paths return plain 404.

Deferred, each tracked by its BDR issue: persistence (BDR-018), real credentials (BDR-013; any bearer token is a test identity now), idempotency (BDR-012; `idempotencyKey` accepted but not enforced), rate limits (BDR-014), move-history endpoint (BDR-016), CLI (BDR-019), MCP (BDR-021), browser UI (BDR-026..030). New gap: `api-invalid-column-type-lacks-version.md`.

Note: Huma v2.41.0 needs Go >= 1.26.10, so `go.mod` now says `go 1.26.10`.
