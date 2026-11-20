---
title: "Arena load test: 100 concurrent agent-vs-agent games in CI"
status: proposed
rank: 11
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

## Evidence (2026-11-19, 8f8e519)

BDR-031 is accepted. The BDR verify loop ran a one-off black-box test against `bin/arena`: 200 agents, each with its own key and the default rate limits, played 100 games at once to completion. That was 2126 moves, with client-measured p50 2.4 ms, p95 10.1 ms and max 38.5 ms. All 100 games finished, every final board matched its replayed history, and every move was numbered once and credited to the right player.

**Correction:** the previous evidence note said 100 games "from one IP address would be throttled almost at once". That was wrong. The limiter counts per API key, and only falls back to IP for requests without a valid key (d9b031c). Agents that each have their own key are never throttled by IP, as this run showed.

What's left of this idea is a *repeatable* check. The run's harness isn't committed, so nothing catches a regression later. Two cases matter most: SQLite persistence (ADR-015) adding a transaction to every move, and the event bus replacing polling. Committing the harness as `cmd/arena-load` and running a short version in CI covers both. That makes it a hardening task, not a gap in the MUST requirements, so it moves from rank 5 to rank 12.


## Ranked (2026-11-20, 0cc40e1): 3 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. Up from 12. A manual 100-game run found that the journal broke BDR-031's p95 on macOS by 2.5× (507 ms, `journal-fsync-serializes-all-games`, fixed in c2cf04f). That is the second time a one-off script caught what no committed check would. A committed `cmd/arena-load` with a short CI run, plus server-side request durations in the oplog (that issue's note), keeps BDR-031 true as storage changes.

## Evidence (2026-11-20, aaa4cfa)

The server now times each REST move request in the oplog, which closes `bdr-031-latency-not-measured-at-server`. The ranking note above asked for exactly that. The committed harness can now report BDR-031's p95 *as measured at the server*, from the oplog, rather than estimating it from the client. **Correction:** the ranking note above said no committed check exists. That was wrong. `tests/load_test.go` `TestJournaledConcurrentGamesMeetMoveLatency` was committed with the group-commit fix (c2cf04f). It plays 100 concurrent journaled games through REST on every full `make check` (skipped under `-short`), and it now asserts p95 both at the client and at the server. That *is* this idea's repeatable check.

What's left is narrow: a black-box run against the real binary or container (the in-process `httptest` server skips the network, the Docker volume and the platform fsync cost that caused the macOS regression), and agents over CLI and MCP rather than REST only. That is hardening, not a gap. **Rank moves from 3 to 11 (last)**, and ideas 4–11 each move up one rank.
