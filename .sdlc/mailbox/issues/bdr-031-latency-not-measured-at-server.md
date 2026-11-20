---
title: "BDR-031 AC-1 asks for move latency measured at the server, but the server records none"
status: closed
priority: 4
source: loop-run
type: observability
bdr: [BDR-031, BDR-032]
branch: main
commit: 3acdb0d
found: 2026-11-20
closed: 2026-11-20
spec: [SPEC.md NFR-002, SPEC.md NFR-006]
---

# BDR-031 AC-1 asks for move latency measured at the server, but the server records none

## Observed

BDR-031 (Accepted) AC-1: "at least 95% of valid move requests receive a response within 200 ms, measured at the server from request receipt to response". The server logs no request timing: `oplog.Middleware` logs only 5xx responses, and the game service's `move_accepted` record has no duration. Every BDR-031 measurement so far, including `TestJournaledConcurrentGamesMeetMoveLatency` and the journal-fsync-serializes-all-games black-box runs, was taken at the client. An operator of the reference deployment (Docker) cannot check AC-1 from the server.

## Expected

Each REST move request leaves a server-side record of how long it took, so AC-1's p95 can be computed from the server's own log.

## Suggested fix

In `oplog.Middleware`, log one `move_request` record per `POST /api/v1/games/{id}/moves` with `gameId`, `status` and `durationMs`, timed from when the handler receives the request to when the response is written. No headers, so no credential (SEC-005). Have the 100-game load test compute p95 from these records.

## Resolution

`oplog.Middleware` logs one `move_request` record for each `POST /api/v1/games/{gameId}/moves`, with `gameId`, the HTTP `status` the client got and `durationMs`, timed from when the handler receives the request to when it returns. Every move request is timed: accepted, refused, rate limited or panicked. A panic's INTERNAL response now goes through the status recorder, so it is timed as 500. No header is logged. The README shows how to get the p95 with `jq`.

- `TestMoveRequestsAreTimedAtTheServer`: one accepted and one refused move give two records (200, 422) with positive durations. A game read and a history read (`GET …/moves`) give none, and no record contains a key.
- `TestRateLimitedMoveRequestsAreTimed`, `TestPanickingMoveRequestIsTimedAs500`, `TestEmptyMoveResponseIsTimedAs200`.
- `TestJournaledConcurrentGamesMeetMoveLatency` now asserts the server-side p95 as well as the client-side one: server 18 ms, client 34 ms in its run.
- Mutations caught: any HTTP method timed, zero duration, no timing, raw status instead of the served one, recovery bypassing the recorder.
- `make check` PASS, 89.6%.

Black-box, 100 concurrent games × 10 moves with `ARENA_JOURNAL_PATH`, p95 from the server's own log using the README's `jq` command:

| Setup | server p50 | server p95 | client p95 |
|-------|-----------:|-----------:|-----------:|
| macOS, `bin/arena` | 11.2 ms | 24.2 ms | 37.3 ms |
| Docker image, journal on a volume | 5.1 ms | 12.4 ms | 30.2 ms |

All 1,000 move requests in each run were status 200 with 1,000 `move_accepted` records, and no log line contains a key.

Not covered: MCP `make_move` calls, which arrive as POST `/mcp` and are not timed. BDR-031 names move requests without naming an interface. Timing MCP would need the tool name from the request body.
