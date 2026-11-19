---
title: "One typed Go client package used by the CLI, house agent, load test and tests"
status: proposed
rank: 2
type: architecture
outcomes: [1, 5]
spec: [SPEC §19 packages/client-sdk, CLI-001, NFR-004, NFR-005]
bdr: [BDR-019, BDR-023]
created: 2026-11-18
---

# Shared Go client package

## Problem

The CLI (ADR-010), the house agent, the stdio MCP bridge, the load test and the contract-suite REST driver all need an HTTP client for `/api/v1`. If each one writes its own, they will handle auth, idempotency keys, `STALE_STATE` retries and error decoding differently. SPEC §19 has a `client-sdk` package for exactly this reason.

## Proposal

- `pkg/arenaclient`: a public package that importable third-party Go agents can use too. It is generated from the Huma OpenAPI document, or written by hand against the Huma types, with a CI check that it still matches OpenAPI.
- It has helpers built on the turn loop: `WaitForTurn`, `Move(ctx, col)` (it adds `expectedVersion` and a UUIDv7 idempotency key, and retries once when the turn is still the caller's), and `PlayUntilDone(strategy func(State) int)`.
- Errors decode into `*arenaclient.Error{Code, Retryable, ...}` from the error-code catalogue.
- The CLI, house agent, stdio MCP bridge and `arena-load` import nothing else to talk to the server.

## Acceptance

- [ ] Only `pkg/arenaclient` builds HTTP requests to `/api/v1` (enforced with a lint rule or a grep check in `make check`).
- [ ] A CI check fails when the client and the OpenAPI document disagree.
- [ ] A Go agent example in `examples/go-agent` is under 40 lines with `PlayUntilDone`.

## Ranked (2026-11-18): 4 of 16

cmd/connect4 now exists (ac4df75); settle the shared client before the CLI commands are written.

## Evidence (2026-11-18)

ac4df75 added `cmd/connect4` and `internal/cli/root.go` (a Cobra root only). No HTTP client exists yet, so this is the cheapest moment to make `pkg/arenaclient` the only code that calls `/api/v1`, before `create`, `join`, `state` and `move` are written.

## Evidence (2026-11-18, eee9bc7)

The REST endpoints now exist, and `tests/api_test.go` builds requests by hand (7 raw `http.NewRequest`/`Post` call sites). Those helpers are the beginning of `pkg/arenaclient`. Moving them there gives the CLI commands their client, and makes the API tests the client's own tests.

## Evidence (2026-11-18, e517837)

The CLI (9e6ef2b) has its own HTTP client in `internal/cli/client.go`. Because it sits under `internal/cli`, nothing else can import it. The next three consumers are the stdio MCP bridge (`issues/mcp-stdio-transport.md`), the house agent and the reference agents, and each would need another copy. Moving `client.go` into `pkg/arenaclient` now is mostly a file move, and the CLI tests then cover it. Moved to rank 2.

## Evidence (2026-11-18, daf1200)

There is now a second hand-written client: `internal/houseagent/houseagent.go` has its own `call()` (JSON encoding, bearer auth, error decoding) next to `internal/cli/client.go`. The house agent also sends moves without an `idempotencyKey`. A shared `Move()` helper would add one automatically, along with `expectedVersion`.
