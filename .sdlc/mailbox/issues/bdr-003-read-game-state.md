---
title: "BDR-003 not satisfied on main: Read Authoritative Game State"
status: closed
source: bdr-verification
bdr: BDR-003
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-003 not satisfied on main: Read Authoritative Game State

Spec: [`docs/bdr/003-read-game-state.md`](../../../docs/bdr/003-read-game-state.md)

## Expected

Any authorized client can retrieve a game's current authoritative state, including enough information for an agent to decide whether, and where, it may move.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [x] AC-1: `GET /api/v1/games/{gameId}` returns the game identifier, board, players with their colors, `currentTurn`, `status`, `legalMoves`, `version` and `winner`.
- [x] AC-2: While the game is `in_progress`, `legalMoves` lists exactly the column indices (0–6) whose top cell is empty.
- [x] AC-3: While the game is `waiting` or `completed`, `legalMoves` is empty.
- [x] AC-4: Requesting a game identifier that does not exist returns a structured error with code `GAME_NOT_FOUND`, and no game is created or modified.
- [x] AC-5: The state shows red discs as `red`, yellow discs as `yellow` and empty cells as `null`. Row 0 is the top row and row 5 is the bottom row.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
