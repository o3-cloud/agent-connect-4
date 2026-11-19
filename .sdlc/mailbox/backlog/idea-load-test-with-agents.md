---
title: "Arena load test: 100 concurrent agent-vs-agent games in CI"
status: proposed
rank: 5
type: quality
outcomes: [5]
spec: [NFR-002, NFR-003, AGENT-005, TURN-004]
bdr: [BDR-031, BDR-025]
created: 2026-11-18
---

# Arena load test

## Problem

NFR-002 (p95 under 200 ms) and NFR-003 (100 games at once) have no defined load profile or harness, so the MVP can't show it meets them.

## Proposal

- `cmd/arena-load`: starts N pairs of random agents (reusing the house-agent strategies) against a target URL. It reports p50, p95 and p99 move latency, the error rate by code, and the number of finished games.
- It injects faults: 5% of agents disconnect partway through a game, and some duplicate retries are sent. The test checks that the failures stay contained and that idempotency holds.
- It checks invariants after the run: each finished game's history replays to the stored board, and occupied cells equal the move count.
- A CI job runs a short profile (100 games, about 60 s) against the container image, with `go test -race` on the server.
- The "MVP load profile" goes into a BDR, so NFR-002 becomes measurable.

## Acceptance

- [ ] The CI job fails if p95 move latency is ≥ 200 ms or any invariant is broken.
- [ ] The results are uploaded as a CI artifact (JSON and a short markdown summary).

## Ranked (2026-11-18): 15 of 16

NFR-002/003 proof; Phase 4 hardening.

## Evidence (2026-11-19, 20f873e)

26 of 33 BDRs are now accepted. Of the 7 that remain, BDR-031 (move latency and capacity, NFR-002 and NFR-003, both MUST) is the only one that no other idea or issue gives a way to prove. The others map to: BDR-018 → `idea-adr-persistence-sqlite-move-log` (rank 1); BDR-027/028 → `idea-wait-for-turn` and the LIVE-001 event bus (rank 3); and BDR-015, 032 and 033 → open issues. Moved from rank 15 to rank 5.

Parts that now exist and can be reused: the agent loops in `tests/agent_match_test.go` (7e2e0a3) for load agents, the JSON-line logs (d30a814) for counting errors, and `ARENA_RATE_LIMIT` (d9b031c), which must be raised or exempted for a load run. Otherwise 100 games from one IP address would be throttled almost at once. Run it after persistence lands, so the latency figures include the SQLite transaction.

