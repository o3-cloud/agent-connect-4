---
title: "BDR-027 not satisfied on main: Watch Games Live Without Refreshing"
status: closed
source: bdr-verification
bdr: BDR-027
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-027 not satisfied on main: Watch Games Live Without Refreshing

Spec: [`docs/bdr/027-live-spectating.md`](../../../docs/bdr/027-live-spectating.md)

## Expected

Anyone authorized to view a game can watch its board and move history update live as moves arrive through any interface, without being able to move.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [ ] AC-1: When a move is accepted through any interface, an open game page shows the new disc and history entry without a manual refresh.
- [ ] AC-2: When a game completes, open game pages show the final result without a manual refresh.
- [ ] AC-3: A spectator of an agent-vs-agent game has no control that submits a move.
- [ ] AC-4: In the open page, the order of updates matches the order of moves in the history.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution (9f8770f, 2026-11-19, BDR verify loop d5b9c49b)

The observations above are from before the API, CLI, MCP and UI existed. I re-verified in Chrome as an anonymous spectator, with a JS marker to prove the page never reloaded, and accepted it:

- AC-1: moves sent via MCP, REST and the CLI showed on the board and in the move list within 49-407 ms. The page polls `/games/{id}/board` every second.
- AC-2: the winning CLI move changed the page to "Alpha wins".
- AC-3: the page has no move controls and says the viewer is watching.
- AC-4: updates came in the same order as the history.
