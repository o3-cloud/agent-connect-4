---
title: "BDR-015 not satisfied on main: Structured, Machine-Readable Errors"
status: closed
source: bdr-verification
bdr: BDR-015
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-015 not satisfied on main: Structured, Machine-Readable Errors

Spec: [`docs/bdr/015-structured-errors.md`](../../../docs/bdr/015-structured-errors.md)

## Expected

Every rejected request returns an error with a machine-readable code, a human-readable message and the relevant request context, in a consistent shape on every interface.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.

## Failing acceptance criteria

- [ ] AC-1: REST errors use the body `{"error": {"code": string, "message": string, ...context}}`.
- [ ] AC-2: Game-rule errors include `gameId` and `currentVersion` in the error context.
- [ ] AC-3: The same rule violation produces the same `code` through REST, the CLI with `--json`, and MCP.
- [ ] AC-4: An MCP rule violation is reported as a tool-level error result (with the error flag set), not as a protocol failure.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Progress (6f4394c, 2026-11-19, session 5b5d3e77)

- AC-1 and AC-2 (REST shape, gameId and currentVersion on rule errors): asserted by `wantError` across `tests/api_test.go`.
- AC-3: MCP returns the REST error object byte for byte for rule errors (`TestMCPRuleErrorsMatchREST`). Since 6f4394c it also returns the same code for schema-level failures (`TestMCPSchemaErrorsCarryErrorObject`). The CLI prints the server's error body with `--json` (9e6ef2b).
- AC-4: MCP failures are `isError` tool results, never protocol errors.
- Remaining before accepting: one contract test that runs the same violation through REST, CLI `--json` and MCP side by side (BDR-023).

- **Update (9e55ba4):** the cross-interface contract test now exists. `TestContractSameViolationSameError` runs six rule violations through REST, CLI `--json` and MCP and gets the same error object every time. All four ACs are now covered by tests, so BDR-015 can be verified and accepted.

## Resolution (8f8e519, 2026-11-19, BDR verify loop d5b9c49b)

Verified black-box and accepted. Scenario 1: REST returns 409 `NOT_YOUR_TURN` with `message`, `retryable`, `gameId` and `currentVersion: 1`. Scenario 2: REST, `connect4 move --json` (exit 3) and MCP `make_move` return the same `COLUMN_FULL` object. MCP reports it as an `isError` tool result, not a JSON-RPC error. 401, 404, 422, unknown routes and malformed JSON keep the `{"error":{...}}` shape.
