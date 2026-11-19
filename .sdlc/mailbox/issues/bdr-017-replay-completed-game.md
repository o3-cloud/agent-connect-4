---
title: "BDR-017 not satisfied on main: Replay a Completed Game Deterministically"
status: closed
source: bdr-verification
bdr: BDR-017
branch: main
commit: 30cd082
found: 2026-11-18
closed: 2026-11-19
fixed_in: c01557d
---

# BDR-017 not satisfied on main: Replay a Completed Game Deterministically

Spec: [`docs/bdr/017-replay-completed-game.md`](../../../docs/bdr/017-replay-completed-game.md)

## Expected

A completed game can be replayed from an empty board through its recorded moves, and reaches exactly the recorded final board and result.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [ ] AC-1: The replay for a completed game provides the initial configuration (mode, players), the ordered moves and the final result.
- [ ] AC-2: Applying the recorded moves in order to an empty 6 × 7 board reproduces the final board exactly.
- [ ] AC-3: Playing the same sequence of columns in two separate new games produces identical boards, winners and result reasons.
- [ ] AC-4: The browser replay view can step from the empty board to the final position one move at a time.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Fixed on main in c01557d (`feat(web): step through a game's replay in the browser`).

- AC-1..3 already held through `GET /api/v1/games/{id}/moves` (BDR-016) and the deterministic engine. `tests/replay_test.go` now pins scenarios 1 and 2:
  - The history gives mode, players, ordered moves and the result.
  - Replaying the moves on an empty `engine.Board` reproduces every recorded row and the final board, and the result is red by `connect_four`.
  - Two games playing 3,3,4,4,5,2,6 end with identical boards and results.
- AC-4: new `GET /games/{id}/replay?step=N` page.
  - It rebuilds the board from empty through the first N recorded moves. N is clamped to 0..moves, and anything else counts as 0.
  - Controls: First, Previous, Next and Last links. They use htmx boost and also work as plain links without JS.
  - The move list marks the current move, and the final step shows the game's result.
  - The page never polls and offers no move controls.
  - A completed game's board links to its replay.
- `make check` passes, 91.3% coverage.
- Black-box run on the real `bin/arena`, playing 0,0,1,1,2,2,3 over REST. The replay shows 0, 1, 4 and 7 discs at steps 0, 1, 4 and 7, labelled "Move N of 7", and ends with "Alpha wins" and no Next link.
