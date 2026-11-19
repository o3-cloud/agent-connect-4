---
title: "INVALID_COLUMN for a non-integer column omits currentVersion and gameId"
status: closed
priority: 1
source: loop-run
type: defect
branch: main
commit: eee9bc7
found: 2026-11-18
spec: [SPEC.md §16, RULE-003, API-007]
bdr: [BDR-006, BDR-015]
---

# INVALID_COLUMN for a non-integer column omits currentVersion and gameId

## Observed

`POST /api/v1/games/{id}/moves` with `{"column": 3.5, ...}` or `{"column": "three", ...}` returns 422 `INVALID_COLUMN`, but the error has no `gameId` or `currentVersion`. Huma rejects the body during schema validation, before the handler can look up the game. An out-of-range integer (`7`, `-1`) reaches the service and does include both fields.

## Expected

SPEC.md §16: "Errors about a specific game include `gameId`, game-rule errors (409 and 422) include `currentVersion`."

## Fix options

- Declare `column` loosely in the schema (for example a JSON number) and validate it as an integer in the handler, where the game is known. This keeps the code but weakens the OpenAPI type.
- Or, in the handler-side error path, look up the game for `path.gameId` when Huma reports a `body.column` validation error (for example via `huma.NewErrorWithContext` and the request path).

Add an assertion for `gameId` and `currentVersion` on the `3.5` and `"three"` cases in `TestAPIRejectInvalidMoves`.

## Decision (2026-11-18)

Use the second option: keep `column` typed as `integer` in the OpenAPI document, because agents read that schema to learn the rules (API-008, NFR-005), and add `gameId` and `currentVersion` in the error path. When Huma reports a validation error at `body.column` for a move request, look up the game from the `gameId` path parameter. If the game exists, return `422 INVALID_COLUMN` with both fields. If it doesn't, return `404 GAME_NOT_FOUND`. No weaker schema.

## Fixed (05bc64f, 2026-11-18)

The Huma error hook now looks up the game for schema-level `body.column` failures: `3.5` and `"three"` return `422 INVALID_COLUMN` with `gameId` and `currentVersion`, and an unknown game returns `404 GAME_NOT_FOUND`. `TestAPIRejectInvalidMoves` asserts both fields on all four bad columns. `make check` passes (coverage 95.5%).
