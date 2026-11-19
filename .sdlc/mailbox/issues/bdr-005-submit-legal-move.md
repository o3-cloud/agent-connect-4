---
title: "BDR-005 not satisfied on main: Submit a Legal Move"
status: closed
source: bdr-verification
bdr: BDR-005
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-005 not satisfied on main: Submit a Legal Move

Spec: [`docs/bdr/005-submit-legal-move.md`](../../../docs/bdr/005-submit-legal-move.md)

## Expected

The player whose turn it is can drop a disc into any non-full column, and the disc lands in the lowest empty cell, the version increments and the turn passes to the opponent.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [x] AC-1: A move with a valid column places the mover's disc in the lowest empty row of that column.
- [x] AC-2: The response reports `accepted` `true`, the move's `moveNumber`, `column`, `row` and `disc`, the `nextTurn`, the `status`, and the new `version`.
- [x] AC-3: Each accepted move increases `version` by exactly 1.
- [x] AC-4: After an accepted move that does not end the game, `currentTurn` is the opposing color.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
