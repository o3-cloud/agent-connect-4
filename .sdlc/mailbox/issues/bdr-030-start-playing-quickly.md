---
title: "BDR-030 not satisfied on main: A Human Can Start Playing Within Seconds"
status: closed
source: bdr-verification
bdr: BDR-030
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-030 not satisfied on main: A Human Can Start Playing Within Seconds

Spec: [`docs/bdr/030-start-playing-quickly.md`](../../../docs/bdr/030-start-playing-quickly.md)

## Expected

A first-time visitor can go from opening the app to making their first move against an agent in a few interactions, with no sign-up or configuration beyond choosing an opponent.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [x] AC-1: From the app's start page, a new visitor can make their first move against an agent in at most 3 interactions (for example: choose Human vs Agent, create the game, click a column).
- [x] AC-2: No account creation, API key or configuration file is needed to start a human-vs-agent game in the browser.
- [x] AC-3: On the reference deployment, the game page is ready to accept the first move within 2 seconds of creating the game.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Accepted (1e14aa8, 525d1d6, 2026-11-19)

Verified in Chrome against a running `bin/arena` at 375 px and 1280 px. See the commit message of 525d1d6 for the measurements. `make check` passes (coverage 90.8%).
