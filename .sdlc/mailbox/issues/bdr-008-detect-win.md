---
title: "BDR-008 not satisfied on main: Detect a Win and End the Game"
status: closed
source: bdr-verification
bdr: BDR-008
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-008 not satisfied on main: Detect a Win and End the Game

Spec: [`docs/bdr/008-detect-win.md`](../../../docs/bdr/008-detect-win.md)

## Expected

When a move completes four or more of the same color in a horizontal, vertical or diagonal line, the system declares that player the winner and accepts no further moves.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [x] AC-1: A move completing a horizontal line of four sets `status` `completed`, `winner` to the mover's color and `resultReason` `connect_four`.
- [x] AC-2: Vertical lines and both diagonal directions are detected the same way.
- [x] AC-3: After the win, `currentTurn` is `null` and `legalMoves` is empty.
- [x] AC-4: Any move submitted to a completed game is rejected with error code `GAME_COMPLETED`, and the game is unchanged.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
