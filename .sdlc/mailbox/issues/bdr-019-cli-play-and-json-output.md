---
title: "BDR-019 not satisfied on main: Play Through the `connect4` CLI With Machine-Readable Output"
status: closed
priority: 1
unblocks: [BDR-012, BDR-013, BDR-016]
source: bdr-verification
bdr: BDR-019
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-019 not satisfied on main: Play Through the `connect4` CLI With Machine-Readable Output

Spec: [`docs/bdr/019-cli-play-and-json-output.md`](../../../docs/bdr/019-cli-play-and-json-output.md)

## Expected

An agent can create, join, inspect and play a game using the `connect4` command-line tool. With `--json` it gets clean JSON output, and every failure exits non-zero.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.

## Failing acceptance criteria

- [x] AC-1: `connect4 create`, `connect4 join <gameId>`, `connect4 state <gameId>` and `connect4 move <gameId> --column <n>` each perform their operation against the server.
- [x] AC-2: With `--json`, standard output contains only a single valid JSON document, with no banners, colors or progress text.
- [x] AC-3: Successful commands exit with status 0. Failed commands exit with a non-zero status.
- [x] AC-4: With `--json`, a failed command's standard output contains an `error` object with the machine-readable error code.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Priority raised (progress dashboard, 2026-11-18T22:45Z)

Three BDRs are built and verified over REST but cannot be accepted until this interface exists: BDR-012 (the MCP `make_move` `idempotency_key` argument), BDR-013 AC-4 (no API keys in CLI or MCP output) and BDR-016 AC-4 (`connect4 history`, MCP `get_game_history`). This issue now unblocks the most acceptances, so it moves to priority 1. Use the ADR-013 API keys: the CLI reads `CONNECT4_API_KEY` and the MCP server takes the same key.

## Accepted (9e6ef2b, b568d0b, 2026-11-18)

`connect4 register|create|join|state|move|history|resign` now drive the REST API with `CONNECT4_SERVER` and `CONNECT4_API_KEY`. `--json` prints exactly one JSON document, and failures exit with the SPEC.md §16 code (the `error` object goes to stdout in JSON mode). Tests: `tests/cli_play_test.go`. Verified with the built `bin/connect4` against a running `bin/arena`: both BDR scenarios, a full game to a red win, an unknown game (exit 5) and a made-up key (exit 4). `make check` passes (coverage 90.5%). BDR-019 is Accepted. `connect4 watch` (CLI-008) is BDR-020.
