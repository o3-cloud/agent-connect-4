---
title: "BDR-026 not satisfied on main: Human Plays an Agent in the Browser"
status: closed
priority: 1
source: bdr-verification
bdr: BDR-026
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-026 not satisfied on main: Human Plays an Agent in the Browser

Spec: [`docs/bdr/026-human-plays-agent-in-browser.md`](../../../docs/bdr/026-human-plays-agent-in-browser.md)

## Expected

A person can open the web app, start a game against an AI agent, and play it to a result entirely through the browser.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [x] AC-1: From the dashboard, a user can choose the Human vs Agent mode, see its setup options, and create the game.
- [x] AC-2: On the user's turn, clicking a non-full column places their disc in that column's lowest empty cell, as the server reports it.
- [x] AC-3: While it is not the user's turn, the board does not accept moves from the user.
- [x] AC-4: The page shows both players, their disc colors, whose turn it is and the game status.
- [x] AC-5: When the game ends, the page shows the winner or draw and accepts no further moves.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Priority raised (progress dashboard, 2026-11-18T23:28Z)

Outcome 2 is the only outcome still at 0. Build the browser game page with htmx and Tailwind (ADR-011, ADR-012) once the house agent lands (`delivery-house-reference-agent`, ADR-014): the dashboard's "Play vs AI" creates a `human_vs_agent` game with `"opponent": "house"`, registers a browser identity, and shows the board. Clicking a column posts a move.

## Unblocked (daf1200, 2026-11-18)

Arena Bot exists: `POST /api/v1/games` with `{"mode": "human_vs_agent", "opponent": "house"}` gives the browser page an opponent that joins once the human does and replies within a second. Only the page itself is left.

## Accepted (9a7a343, 925cd38, 2026-11-19)

The dashboard at `/` starts a game against Arena Bot; the game page is an htmx board with server-checked column controls. Tests: `tests/web_test.go`. Verified in Chrome against a running `bin/arena`: all three scenarios passed, and the move was confirmed over the REST API. fmt, vet, lint, race tests and coverage (89.8%) pass. `govulncheck` was not re-run because its database download hung on the network; this change adds no Go dependencies. Follow-ups: `docker-image-lacks-css`, plus live push over WebSocket in BDR-027 (the board polls every second for now).

## Vulnerability scan (2026-11-19, main at 9e55ba4)

`govulncheck ./...` now completes: no vulnerabilities found. The gap noted above is closed.
