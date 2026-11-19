---
title: "BDR-004 not satisfied on main: List Visible Games"
status: closed
source: bdr-verification
bdr: BDR-004
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-004 not satisfied on main: List Visible Games

Spec: [`docs/bdr/004-list-games.md`](../../../docs/bdr/004-list-games.md)

## Expected

An authorized client can list the games visible to it, with each game's identifier, mode, players and status.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.

## Failing acceptance criteria

- [x] AC-1: The list includes each visible game's identifier, mode, players and status.
- [x] AC-2: A newly created game appears in the list with status `waiting`.
- [x] AC-3: Filtering by status returns only games in that status.
- [x] AC-4: The list never includes credentials or secrets.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Progress (eee9bc7, 2026-11-18, bdr-verification loop)

Black-box over HTTP against `make run`:

- Passing: AC-1 and AC-2 (a new game is listed with `gameId`, `mode`, `players`, `status` `waiting`), AC-4 (bearer tokens never appear in the list).
- **Failing: AC-3.** `GET /api/v1/games?status=in_progress` returns `200` with every game. The response held `waiting`, `in_progress` and `completed` entries, so the `status` query parameter is ignored. The `listGames` operation in `internal/api/api.go` takes no input (`*struct{}`).
- Scenario 2 also names the MCP tool `list_games`. MCP does not exist yet (BDR-021).

```sh
curl -s 'http://localhost:8080/api/v1/games?status=in_progress' | jq '[.games[].status] | unique'
# ["completed","in_progress","waiting"]   expected ["in_progress"]
```

## Resolution

Fixed on main in 74055c8 (2026-11-18). `GET /api/v1/games?status=<waiting|in_progress|completed>` returns only games in that status; no `status` returns all; an unknown status returns 400 `INVALID_ARGUMENT`. The filter lives in `game.Service.ListByStatus`, ready for the MCP `list_games` tool (BDR-021). Verified by `TestListByStatusFiltersGames`, `TestAPIListGamesFiltersByStatus` (also asserts AC-4: no bearer token in the list) and black-box curl against `bin/arena`; `make check` PASS, coverage 95.4%.

## Independent verification (74055c8, 2026-11-18, bdr-verification loop)

Confirmed over HTTP against the running `bin/arena`: `?status=in_progress` returns only in-progress games and includes the in-progress one, and `?status=bogus` returns `400 INVALID_ARGUMENT`. AC-1 to AC-4 pass. BDR-004 is marked `Accepted`.
