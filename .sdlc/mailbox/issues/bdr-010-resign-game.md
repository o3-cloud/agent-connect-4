---
title: "BDR-010 not satisfied on main: Resign a Game"
status: closed
source: bdr-verification
bdr: BDR-010
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-010 not satisfied on main: Resign a Game

Spec: [`docs/bdr/010-resign-game.md`](../../../docs/bdr/010-resign-game.md)

## Expected

A player in an in-progress game can resign, which ends the game immediately with the opponent as winner.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [ ] AC-1: When a player resigns an in-progress game, `status` becomes `completed`, `winner` is the opposing color and `resultReason` is `resignation`.
- [ ] AC-2: A player may resign whether or not it is their turn.
- [ ] AC-3: After resignation, moves are rejected with `GAME_COMPLETED`.
- [ ] AC-4: A client that is not a player in the game cannot resign it.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Progress (eee9bc7, 2026-11-18, bdr-verification loop)

The REST API now exists, but resignation does not. `POST /api/v1/games/{id}/resign` from the red player of an in-progress game returns `404 ROUTE_NOT_FOUND`, and the OpenAPI document has no resign operation. AC-1 to AC-4 all fail.

## Resolution (395b458, 2026-11-18)

Fixed on main by 395b458 `feat(game): let a player resign an in-progress game`.

- `game.Service.Resign` and `POST /api/v1/games/{gameId}/resign` (OpenAPI operation `resignGame`). The response is the full game state.
- AC-1: the game becomes `completed`, the opponent is `winner`, `resultReason` is `resignation`, the version goes up by one, and `currentTurn` is null.
- AC-2: either player may resign on either turn.
- AC-3: later moves and resignations get `409 GAME_COMPLETED` with `currentVersion`.
- AC-4: outsiders get `403 FORBIDDEN` and the game is unchanged. Unauthenticated callers get `401`, unknown games `404 GAME_NOT_FOUND`, and waiting games `409 GAME_NOT_STARTED` (§16).

Verification: `make check` PASS (coverage 95.8%). New tests are in `tests/resign_test.go`. A black-box curl run against `bin/arena` returned outsider 403, red resigning on yellow's turn 200 completed/yellow/resignation, GET showing the same result, and a later move 409 GAME_COMPLETED.

The CLI (`connect4 resign`) and MCP tool are not covered here. They are tracked by BDR-019 and BDR-021.
