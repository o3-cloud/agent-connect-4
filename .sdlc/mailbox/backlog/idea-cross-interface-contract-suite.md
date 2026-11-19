---
title: "One scenario suite run against REST, CLI and MCP adapters"
status: accepted
type: quality
outcomes: [5]
spec: [NFR-004, MCP-009, AT-001..AT-010]
bdr: [BDR-023]
created: 2026-11-18
---

# Cross-interface contract suite

## Problem

Outcome 5 says every participant plays under the same rules whichever way they connect. If each interface has its own tests, small differences creep in: error codes, field names, how versions are handled.

## Proposal

- Define an `Arena` driver interface in `tests/contract`: `Create`, `Join`, `State`, `Move`, `Resign`, `History`.
- Write three drivers: a REST client, the CLI (runs the built `connect4` binary with `--json`), and an MCP client (official Go SDK, in-memory transport).
- Write each scenario once (illegal column, full column, out of turn, stale version, idempotent retry, concurrent moves, win in all four directions, draw, resignation) and run it as a table test across the three drivers.
- Assert the normalised responses are equal across drivers, as golden JSON.
- Mixed-driver tests: red plays over CLI and yellow over MCP (AT-002).

## Acceptance

- [ ] Each AT-00x scenario has a test that names its ID.
- [ ] Adding a fourth interface (for example the browser via htmx) only needs a new driver.
- [ ] It runs under `go test -race` in `make check`.

## Ranked (2026-11-18): 5 of 16

Outcome 5; cheapest while there is only one interface.

## Evidence (2026-11-18, eee9bc7)

REST is the only interface so far. Now is the cheapest time to add the `Arena` driver abstraction: wrap the existing API tests in a REST driver, so the CLI and MCP drivers only have to be plugged in later. The open issue `api-invalid-column-type-lacks-version` is the kind of difference between interfaces this suite would catch: Huma's schema validation returns a different error shape from the service.

## Evidence (2026-11-18, e517837)

All three agent interfaces now exist: REST, the CLI (9e6ef2b) and MCP over HTTP (e517837). The second interface mismatch has already appeared: `issues/mcp-schema-errors-lack-error-object.md` (MCP schema-validation errors don't carry the §16 error object). That has the same cause as the earlier REST issue, where Huma validated the request before the game service saw it. One driver-based suite would catch this whole class of bug. Moved to rank 4.

## Decision (2026-11-18)

Built in 9e55ba4 (`tests/contract_test.go`, no production code changed). One `iface` abstraction drives REST, the CLI and MCP. It covers one game played to a win through each interface, six rule violations giving identical `code`/`message`/`retryable`/`currentVersion` and CLI exit codes, visibility of a move across interfaces, and one identity across interfaces. A mutation check shows the suite catches regressions.

Still open from this idea, as small follow-ups:
- Scenarios are inline rather than loaded from shared fixtures (`idea-portable-game-fixtures`, rank 3). Loading them would bring draws, wins in all four directions and resignation into all three interfaces at no extra cost.
- `SLOT_TAKEN`, `ALREADY_JOINED`, `GAME_NOT_STARTED` and `RATE_LIMITED` aren't yet among the error cases.
- The browser isn't a driver yet.
