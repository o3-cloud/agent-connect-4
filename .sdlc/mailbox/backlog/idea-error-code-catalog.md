---
title: "Publish one error-code catalogue: codes, HTTP statuses, CLI exit codes, retry advice"
status: accepted
type: spec-change
outcomes: [1, 5]
spec: [API-007, CLI-007, MCP-008, SEC-001, SEC-003, SEC-007, GAME-007, RULE-003, RULE-004, RULE-010, TURN-003, TURN-006]
bdr: [BDR-015, BDR-013, BDR-014, BDR-023]
created: 2026-11-18
---

# Error-code catalogue

## Problem

SPEC.md names some error codes (`GAME_NOT_FOUND`, `INVALID_COLUMN`, `COLUMN_FULL`, `GAME_COMPLETED`, `NOT_YOUR_TURN`, `STALE_STATE`), but no HTTP statuses. It also has no codes at all for missing or invalid credentials, rate limits, a join into a taken slot, joining twice, a move before the game starts, or a malformed body. `hitl-notes.json` lists this as an open spec decision. Without one table, REST, CLI and MCP will each make up their own answers, which breaks NFR-004. Agents also can't tell which errors are worth retrying.

## Proposal

Add a single source of truth, `internal/errs/catalog.go`, and a generated doc, `docs/errors.md`:

| Code | HTTP | CLI exit | Retryable | Agent action |
|---|---|---|---|---|
| `VALIDATION_ERROR` | 400 | 2 | no | fix the request |
| `INVALID_COLUMN` | 422 | 3 | no | choose a column from `legalMoves` |
| `COLUMN_FULL` | 422 | 3 | no | choose a column from `legalMoves` |
| `UNAUTHENTICATED` | 401 | 4 | no | supply or refresh the key |
| `FORBIDDEN` / `NOT_A_PLAYER` | 403 | 4 | no | — |
| `GAME_NOT_FOUND` | 404 | 5 | no | — |
| `NOT_YOUR_TURN` | 409 | 6 | yes, after waiting | wait for your turn |
| `STALE_STATE` | 409 | 6 | yes, after refetching | refetch the state, then decide again |
| `GAME_NOT_STARTED` | 409 | 6 | yes, after waiting | wait |
| `SLOT_TAKEN` / `ALREADY_JOINED` | 409 | 6 | no | pick the other disc |
| `GAME_COMPLETED` | 409 | 7 | no | stop |
| `RATE_LIMITED` | 429 + `Retry-After` | 8 | yes, after `retryAfterMs` | back off |
| `INTERNAL` | 500 | 1 | yes, with backoff | — |

These numbers are a starting proposal for a human to confirm.

- REST bodies, CLI `--json` errors and MCP `isError` tool results all carry `{ code, message, retryable, gameId?, currentVersion?, retryAfterMs? }`.
- The OpenAPI spec lists the catalogue as an enum, so agents can discover it (BDR-022).

## Acceptance

- [ ] A contract test fails if any interface emits a code that isn't in the catalogue, or maps one to a different status or exit code.
- [x] SPEC §16 is amended with the table, and the `hitl-notes.json` item is closed.

## Decision (2026-11-18)

Accepted into SPEC.md §16 "Error Codes" (commits d6b9f25 and the follow-up on main), with two changes: the 400 code stays `INVALID_ARGUMENT` (already in SPEC.md), and `FORBIDDEN` is the single code for non-players (no `NOT_A_PLAYER` alias). The table also adds `GAME_NOT_JOINABLE` and `ROUTE_NOT_FOUND`. Still to do as implementation work: `internal/errs/catalog.go`, the OpenAPI enum and the cross-interface contract test.
