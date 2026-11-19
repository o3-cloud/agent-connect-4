---
title: "BDR-016 not satisfied on main: Ordered Move History"
status: closed
source: bdr-verification
bdr: BDR-016
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-016 not satisfied on main: Ordered Move History

Spec: [`docs/bdr/016-move-history.md`](../../../docs/bdr/016-move-history.md)

## Expected

Any authorized client can retrieve the complete, ordered history of accepted moves for a game, and the game result once it has one.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [ ] AC-1: `GET /api/v1/games/{gameId}/moves` returns the accepted moves in order, each with move number, player identifier, color, column, row and timestamp.
- [ ] AC-2: Move numbers start at 1 and increase by 1 with no gaps or duplicates.
- [ ] AC-3: Rejected moves never appear in the history.
- [ ] AC-4: For a completed game, the history response (and the MCP `get_game_history` result) includes the winner and result reason.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution (cdffcd9, 2026-11-18)

Fixed on main by cdffcd9 `feat(api): serve ordered move history at GET /games/{gameId}/moves` (OpenAPI operation `getGameHistory`).

- **AC-1:** the response lists the accepted moves in order. Each move has `moveNumber`, `playerId`, `disc`, `column`, `row` and `timestamp`. The response also has `gameId`, `mode`, `status`, `players` and `version`.
- **AC-2:** move numbers run 1..n with no gaps.
- **AC-3:** rejected moves are absent.
- **AC-4:** `winner` and `resultReason` are set once the game is completed and null before.
- An unknown game returns `404 GAME_NOT_FOUND`. A history read needs no credentials, the same as a game-state read, and it never contains API keys (SEC-005).

Verification:
- `make check` PASS (95.6%), with new tests in `tests/history_test.go`.
- A black-box run against `bin/arena` reproduced scenario 1 exactly: columns 3, 3 and 4 accepted, column 9 rejected with 422, then history moves 1–3 as red/yellow/red, columns 3/3/4, rows 5/4/5.

The MCP `get_game_history` part of AC-4 remains under BDR-021. The CLI `connect4 history` remains under BDR-019.

## CLI part done (9e6ef2b, 2026-11-18)

`connect4 history <gameId>` lists moves in order and the result, and `--json` returns the history document. Only the MCP `get_game_history` part of AC-4 remains (BDR-021, claimed by the issue worker).
