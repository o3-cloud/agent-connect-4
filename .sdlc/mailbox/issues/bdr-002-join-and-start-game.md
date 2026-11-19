---
title: "BDR-002 not satisfied on main: Join a Game and Start Play"
status: closed
source: bdr-verification
bdr: BDR-002
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-002 not satisfied on main: Join a Game and Start Play

Spec: [`docs/bdr/002-join-and-start-game.md`](../../../docs/bdr/002-join-and-start-game.md)

## Expected

A participant can claim an open red or yellow slot in a waiting game, and the game starts with red to move once both slots are filled.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [x] AC-1: Joining an open slot assigns the participant to that color, and later state reads show the participant's display name in that slot.
- [x] AC-2: After exactly one slot is filled, the game's `status` remains `waiting` and `currentTurn` is `null`.
- [x] AC-3: When the second slot is filled, `status` becomes `in_progress` and `currentTurn` is `red`.
- [x] AC-4: Attempting to join a slot that is already taken is rejected with a structured error, and the slot's occupant is unchanged.
- [x] AC-5: The joining participant's identity comes from their credential. A player identifier supplied in the request body has no effect on who occupies the slot.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
