---
title: "BDR-029 not satisfied on main: Usable on Mobile and Desktop"
status: closed
source: bdr-verification
bdr: BDR-029
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-029 not satisfied on main: Usable on Mobile and Desktop

Spec: [`docs/bdr/029-responsive-layout.md`](../../../docs/bdr/029-responsive-layout.md)

## Expected

The web UI is fully usable on phone and desktop screens.

## Observed (black-box, on main)

- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [x] AC-1: At a 375 px wide viewport, the dashboard and game pages have no horizontal scrolling.
- [x] AC-2: At 375 px, all seven columns of the board are visible at once and each can be selected by tapping.
- [x] AC-3: At 1280 px, the board, players, status and move history are visible without scrolling the page sideways.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Accepted (1e14aa8, 525d1d6, 2026-11-19)

Verified in Chrome against a running `bin/arena` at 375 px and 1280 px. See the commit message of 525d1d6 for the measurements. `make check` passes (coverage 90.8%).
