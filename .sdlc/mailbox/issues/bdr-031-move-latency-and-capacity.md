---
title: "BDR-031 not satisfied on main: Move Latency and Concurrent Game Capacity"
status: open
source: bdr-verification
bdr: BDR-031
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-031 not satisfied on main: Move Latency and Concurrent Game Capacity

Spec: [`docs/bdr/031-move-latency-and-capacity.md`](../../../docs/bdr/031-move-latency-and-capacity.md)

## Expected

The server keeps valid moves fast while running at least 100 active games at once on the reference deployment.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [ ] AC-1: With 100 games in progress at once, at least 95% of valid move requests receive a response within 200 ms, measured at the server from request receipt to response.
- [ ] AC-2: All 100 concurrent games can be played to completion with no lost, duplicated or misattributed moves.
- [ ] AC-3: Each game's final board equals a replay of its own move history (see [BDR-017](017-replay-completed-game.md)).

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.
