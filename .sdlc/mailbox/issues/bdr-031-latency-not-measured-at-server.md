---
title: "BDR-031 AC-1 asks for move latency measured at the server, but the server records none"
status: open
priority: 4
source: loop-run
type: observability
bdr: [BDR-031, BDR-032]
branch: main
commit: 3acdb0d
found: 2026-11-20
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-20T09:09Z
spec: [SPEC.md NFR-002, SPEC.md NFR-006]
---

# BDR-031 AC-1 asks for move latency measured at the server, but the server records none

## Observed

BDR-031 (Accepted) AC-1: "at least 95% of valid move requests receive a response within 200 ms, measured at the server from request receipt to response". The server logs no request timing: `oplog.Middleware` logs only 5xx responses, and the game service's `move_accepted` record has no duration. Every BDR-031 measurement so far, including `TestJournaledConcurrentGamesMeetMoveLatency` and the journal-fsync-serializes-all-games black-box runs, was taken at the client. An operator of the reference deployment (Docker) cannot check AC-1 from the server.

## Expected

Each REST move request leaves a server-side record of how long it took, so AC-1's p95 can be computed from the server's own log.

## Suggested fix

In `oplog.Middleware`, log one `move_request` record per `POST /api/v1/games/{id}/moves` with `gameId`, `status` and `durationMs`, timed from when the handler receives the request to when the response is written. No headers, so no credential (SEC-005). Have the 100-game load test compute p95 from these records.
