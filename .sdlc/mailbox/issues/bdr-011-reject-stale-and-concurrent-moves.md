---
title: "BDR-011 not satisfied on main: Reject Stale and Concurrent Moves"
status: closed
source: bdr-verification
bdr: BDR-011
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-011 not satisfied on main: Reject Stale and Concurrent Moves

Spec: [`docs/bdr/011-reject-stale-and-concurrent-moves.md`](../../../docs/bdr/011-reject-stale-and-concurrent-moves.md)

## Expected

A move is accepted only if it names the game's current version, so at most one move is ever committed per turn, even when requests arrive at the same time.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [ ] AC-1: A move whose `expectedVersion` differs from the current version is rejected with error code `STALE_STATE`, and the error reports the current version.
- [ ] AC-2: A rejected stale move does not change the board, turn or version.
- [ ] AC-3: When two moves for the same turn and version are sent at the same time, at most one is accepted, and the other receives a conflict error (`STALE_STATE` or `NOT_YOUR_TURN`).
- [ ] AC-4: After any race, the number of discs on the board equals the number of moves in the history.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution (74d1fc2, 2026-11-18)

The service already satisfied BDR-011. Optimistic versioning (TURN-006) and a per-game lock (TURN-004) landed in 2bcfa94/eee9bc7. 74d1fc2 adds black-box proof in `tests/concurrency_test.go`.

- **AC-1:** a stale `expectedVersion` returns `409 STALE_STATE` with `currentVersion`, `gameId` and `retryable: true`.
- **AC-2:** board, turn, version, moves and legal moves are unchanged.
- **AC-3:** across 10 rounds of two simultaneous red moves, exactly one is accepted and the other gets `STALE_STATE` or `NOT_YOUR_TURN`.
- **AC-4:** after 56 racing requests from both players, discs, moves and version are equal, and turns alternate.
- Mutation check: removing the per-game lock makes `TestAPIRaceKeepsBoardAndHistoryInStep` fail under `-race`, which is how `make check` runs the tests.

Verification:
- `make check` PASS (95.7%).
- A black-box run against `bin/arena` reproduced scenario 1 (409 STALE_STATE, currentVersion 3, still version 3, yellow to move).
- Scenario 2 run 10 times with parallel curl: one 200 and one 409 every time, ending at version 1 with 1 disc and 1 move.
