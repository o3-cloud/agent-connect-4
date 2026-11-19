---
title: "BDR-023 not satisfied on main: Identical Rules on Every Interface"
status: closed
source: bdr-verification
bdr: BDR-023
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-023 not satisfied on main: Identical Rules on Every Interface

Spec: [`docs/bdr/023-cross-interface-consistency.md`](../../../docs/bdr/023-cross-interface-consistency.md)

## Expected

Moves made through the browser, REST API, CLI or MCP follow identical validation and win-detection rules, and act on the same shared game.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [ ] AC-1: The same move sequence submitted through REST, the CLI or MCP produces the same boards, versions, errors and results.
- [ ] AC-2: A move made through one interface is immediately visible in the state read through every other interface.
- [ ] AC-3: An agent can submit one move through the CLI and its next move through MCP with the same credential, and both are attributed to the same player.
- [ ] AC-4: The platform accepts any legal move regardless of which agent, model or language chose it.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution (9e55ba4, 2026-11-19)

Main already met BDR-023: REST, CLI and MCP all run on one `game.Service` and share the error mapping (`api.ErrorFor`). 9e55ba4 adds the contract suite `tests/contract_test.go`, which drives one agent through all three interfaces with a shared `iface` abstraction.

- **AC-1, scenario 1:** the sequence 0,0,1,1,2,2,3 played via REST, CLI and MCP gives identical games: board, moves, version 7, red winning by `connect_four`. Six violations (NOT_YOUR_TURN, STALE_STATE, INVALID_COLUMN, COLUMN_FULL, GAME_COMPLETED, FORBIDDEN) return the same code, message, retryable and currentVersion on every interface and leave the game unchanged. CLI exit statuses match §16. GAME_NOT_FOUND is the same everywhere.
- **AC-2:** each move goes through a rotating interface and is immediately visible, at the same version and view, through all three readers.
- **AC-3, scenario 2:** a CLI move and an MCP move made with the same key are attributed to one player id.
- **AC-4:** the platform validates moves only by rule, never by client. The suite runs identical moves from three different clients.
- Mutation check: removing `currentVersion` from MCP errors, or changing the CLI's STALE_STATE exit code, fails the suite.

Verification:
- `make check` PASS (89.9%), re-run after rebasing onto the browser UI (9a7a343).
- Black-box run of scenario 2 with the real `bin/connect4` and raw MCP JSON-RPC against `bin/arena`: moves 1 (CLI) and 3 (MCP) are both attributed to `agt_d2c3…`, the red player.
