---
title: "With a journal on macOS, 100 concurrent games miss BDR-031's 200 ms p95: one global fsync queue"
status: open
priority: 4
source: bdr-verification
type: performance
branch: main
commit: 1af633c
found: 2026-11-20
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-20T04:10Z
spec: [docs/adr/015-jsonl-journal-persistence.md]
bdr: [BDR-031, BDR-018]
---

# One fsync queue for every game makes journaled moves slow on macOS

## Observed

My 100-concurrent-game load test (2,126 moves, all games completed, none lost, duplicated or misattributed) against `bin/arena` built from 1af633c:

| Setup | client p50 | client p95 | wall |
|-------|-----------:|-----------:|-----:|
| macOS, in memory | 1.7 ms | 6.6 ms | 1.3 s |
| macOS, `ARENA_JOURNAL_PATH` set (`make run` style) | 395 ms | **507 ms** | 9.9 s |
| Linux container, in memory | 2.6 ms | 10.7 ms | 1.3 s |
| Linux container, journal on a volume (README / compose) | 65 ms | 75 ms | 2.0 s |

The documented container deployment meets BDR-031 AC-1 (p95 ≤ 200 ms). A developer who follows the README's `ARENA_JOURNAL_PATH=arena.jsonl make run` on a Mac misses it by 2.5×.

## Cause

`journal.Append` (internal/journal/journal.go) holds `j.mu` across `Write` and `file.Sync()`. Every append from every game waits for the one before it to reach disk, so throughput is 1/fsync-time across the whole server. On darwin, Go's `Sync` is `F_FULLFSYNC` (about 4.5 ms here: 9.9 s / 2,126 moves), so 100 waiting games give a queue of about 450 ms.

## Suggested fix

Group commit: write each line under the lock, then let a single syncer flush everything written so far and wake all waiters whose line it covered. Each append still returns only after its line is on disk (ADR-015 step 3), but one fsync covers many games' moves. Add a benchmark or test that runs 100 concurrent games with a journal and asserts p95 at the server.

Note: the server logs no per-request duration, so BDR-031's "measured at the server" can only be approximated from the client on localhost.
