---
title: "BDR-009 not satisfied on main: Detect a Draw"
status: closed
source: bdr-verification
bdr: BDR-009
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-009 not satisfied on main: Detect a Draw

Spec: [`docs/bdr/009-detect-draw.md`](../../../docs/bdr/009-detect-draw.md)

## Expected

When the board fills without any line of four, the system ends the game as a draw with no winner.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [x] AC-1: When the 42nd disc fills the board and no line of four exists, `status` becomes `completed`, `winner` is `null` and `resultReason` is `draw`.
- [x] AC-2: After a draw, `legalMoves` is empty and further moves are rejected with `GAME_COMPLETED`.
- [x] AC-3: The move history of a drawn game contains exactly 42 moves.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
