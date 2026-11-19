---
title: "BDR-020 not satisfied on main: Watch a Game From the CLI"
status: closed
priority: 1
unblocks: [BDR-013, BDR-027]
source: bdr-verification
bdr: BDR-020
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-020 not satisfied on main: Watch a Game From the CLI

Spec: [`docs/bdr/020-cli-watch-game.md`](../../../docs/bdr/020-cli-watch-game.md)

## Expected

An agent can follow a game from the command line and receive each newly accepted move until the game completes or the command is stopped.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.

## Failing acceptance criteria

- [ ] AC-1: `connect4 watch <gameId>` prints each move accepted after it starts, in order.
- [ ] AC-2: The command exits with status 0 once the game is completed, after printing the result.
- [ ] AC-3: With `--json`, each update is one JSON document per line.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Priority raised (progress dashboard, 2026-11-18T23:17Z)

Next after BDR-021 (MCP). Build the server side first: a per-game event stream (LIVE-001, LIVE-003, LIVE-004) over WebSocket as SPEC.md LIVE-002 specifies, with each event carrying `gameId`, type, `version` and state, and never an API key (SEC-005). Then make `connect4 watch` its first client. This gives BDR-013 AC-4 its spectator surface to test, and BDR-027 (browser live spectating) its server half.

## Resolution (4baf071, 2026-11-18)

Fixed on main by 4baf071 `feat(cli): follow a game live with connect4 watch`.

- **AC-1:** `connect4 watch <gameId>` polls `GET /api/v1/games/{id}/moves` (`--interval`, default 1s). It prints every move accepted after it starts, in order, and none from before.
- **AC-2:** after the result it exits 0. This covers a win, a resignation, or a game that was already completed.
- **AC-3:** with `--json`, each update is one JSON event per line (`watching`, `move`, `game_completed`), shaped like LIVE-001.
- An unknown game exits 5 (`GAME_NOT_FOUND`), and a zero interval exits 2.

Verification:
- `make check` PASS (90.5%), with new tests in `tests/cli_watch_test.go`.
- Black-box run of scenario 1 with the real `bin/connect4 watch --json` against `bin/arena`: a game one move from a red win printed `watching` at v6, then the column-0 move at v7, then `game_completed` with winner red, and exit 0.

Note: watch polls the move history. When BDR-027 adds push events, watch can subscribe instead, keeping the same output format.
