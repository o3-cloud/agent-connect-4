---
title: "MCP schema-validation failures are tool errors without the §16 error object"
status: closed
priority: 2
source: loop-run
type: defect
branch: main
commit: e517837
found: 2026-11-18
spec: [SPEC.md §16, MCP-008, NFR-004]
bdr: [BDR-015, BDR-023]
---

# MCP schema-validation failures lack the §16 error object

## Observed

Some calls fail the tool's input schema, for example `make_move` without `column`, with `column: 3.5`, or with `disc: "green"` on `join_game`. The MCP Go SDK rejects these before the handler runs. The result has `isError: true`, which satisfies BDR-021 AC-4, but its only content is SDK text such as `validating "arguments": ...`. It has no `structuredContent` with `{"error": {"code": "INVALID_ARGUMENT", ...}}`. REST returns `400 INVALID_ARGUMENT`, or `422 INVALID_COLUMN` for a bad column, as structured errors.

Game-rule errors (NOT_YOUR_TURN, INVALID_COLUMN for `9`, and so on) already carry the §16 object.

## Expected

BDR-015 AC-3 says the same violation has the same `code` through REST, CLI and MCP. A non-integer `column` should be `INVALID_COLUMN` with `gameId` and `currentVersion`, and any other bad argument should be `INVALID_ARGUMENT`.

## Fix options

- Register the tools with the untyped `server.AddTool` and a raw handler that validates the arguments itself, then maps the problems to §16 codes. This keeps the schemas published (AC-1).
- Or wrap the typed handler: give `mcp.AddTool` a looser input type, for example `column` as `json.Number`, and validate in the handler. This weakens the published schema, which an earlier decision avoided for REST.

Add tests to `tests/mcp_test.go` asserting the error object for a missing or non-integer `column`.

## Resolution (6f4394c, 2026-11-19)

Fixed on main by 6f4394c. Tools now register through a generic `addTool` wrapper. It keeps publishing the input schema derived from the Go type (`jsonschema.For`), validates the arguments itself, and reports every failure as `isError` with the §16 `{"error": {...}}` object as structured content.

Codes now match REST:
- A non-integer `column` (`3.5`, `"three"`, `true`) is `INVALID_COLUMN` with `gameId` and `currentVersion`. With an unknown game it is `GAME_NOT_FOUND`.
- A **missing** `column` is `INVALID_ARGUMENT`. REST reports it the same way (400, "required property column"), so this issue's original expectation, INVALID_COLUMN for a missing column, did not match REST. The rule now is: missing means INVALID_ARGUMENT, present but not an integer means INVALID_COLUMN.
- Wrong-typed `expected_version` or `game_id`, a missing `idempotency_key`, `disc: "green"`, and missing `game_id` are all `INVALID_ARGUMENT`.

Verification:
- `make check` PASS (90.3%). `TestMCPSchemaErrorsCarryErrorObject` compares code, gameId, currentVersion and retryable against the REST response for each case.
- Black-box run over raw JSON-RPC against `bin/arena`: the four cases return the expected objects, `tools/list` still shows `make_move`'s required arguments, and a valid move still succeeds.
