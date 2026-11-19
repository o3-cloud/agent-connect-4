---
title: "BDR-001 not satisfied on main: Create a Game"
status: closed
source: bdr-verification
bdr: BDR-001
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-001 not satisfied on main: Create a Game

Spec: [`docs/bdr/001-create-game.md`](../../../docs/bdr/001-create-game.md)

## Expected

An authenticated client can create a new Connect Four game in either supported mode and receive its identifier and initial state.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [x] AC-1: `POST /api/v1/games` with `{"mode": "agent_vs_agent"}` or `{"mode": "human_vs_agent"}` returns a `gameId` that no other game uses.
- [x] AC-2: The response reports `status` `waiting`, `version` `0`, and a `board` of exactly 6 rows × 7 columns in which every cell is `null`.
- [x] AC-3: The response includes information on how to join, such as the open player slots.
- [x] AC-4: A request with a `mode` outside the two supported values is rejected with a structured error, and no game appears in the game list.
- [x] AC-5: Creating two games returns two different `gameId` values.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
