---
title: "BDR-021 not satisfied on main: Play Through MCP Tools"
status: closed
priority: 1
unblocks: [BDR-012, BDR-013, BDR-016]
source: bdr-verification
bdr: BDR-021
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-021 not satisfied on main: Play Through MCP Tools

Spec: [`docs/bdr/021-mcp-tools.md`](../../../docs/bdr/021-mcp-tools.md)

## Expected

An MCP-capable agent can discover the game tools, and create, join, inspect and play games, through the platform's MCP server.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.

## Failing acceptance criteria

- [ ] AC-1: Listing the MCP server's tools returns `create_game`, `join_game`, `get_game_state`, `make_move`, `list_games` and `get_game_history`, each with an input schema.
- [ ] AC-2: `make_move` with `game_id`, `column`, `expected_version` and `idempotency_key` places a disc exactly as a REST move would.
- [ ] AC-3: The player identity for MCP calls comes from the MCP connection's authentication, not from tool arguments.
- [ ] AC-4: Calls with invalid arguments return a tool-level error and do not change game state.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Priority raised (progress dashboard, 2026-11-18T22:45Z)

Three BDRs are built and verified over REST but cannot be accepted until this interface exists: BDR-012 (the MCP `make_move` `idempotency_key` argument), BDR-013 AC-4 (no API keys in CLI or MCP output) and BDR-016 AC-4 (`connect4 history`, MCP `get_game_history`). This issue now unblocks the most acceptances, so it moves to priority 1. Use the ADR-013 API keys: the CLI reads `CONNECT4_API_KEY` and the MCP server takes the same key.

## Resolution (e517837, 2026-11-18)

Fixed on main by e517837 `feat(mcp): serve the six game tools over Streamable HTTP at /mcp`. It adds `internal/mcpserver` on `github.com/modelcontextprotocol/go-sdk` v1.8.0, pinned per ADR-009.

- **AC-1:** the six tools are listed with descriptions and typed input schemas. The required arguments match SPEC.md §17 exactly.
- **AC-2:** `make_move` places a disc through `Service.MoveWithKey`, the same path REST uses, and honours `idempotency_key`. The MCP and REST views of board, version, moves, turn and players are identical.
- **AC-3:** `/mcp` requires an issued API key (`401` for a missing or forged key). The acting player is the key's agent, and no tool takes an identity argument.
- **AC-4:** invalid arguments are tool-level errors (`isError`, not protocol errors) and leave the game unchanged. Game-rule errors carry the same §16 `{"error": {...}}` object as REST, byte for byte (tested with `NOT_YOUR_TURN`).

Verification:
- `make check` PASS (91.0% after rebasing onto the CLI from 9e6ef2b). New tests are in `tests/mcp_test.go`.
- Black-box run against `bin/arena` with raw JSON-RPC over curl (not the Go SDK): 401 without a key or with a bad key; `tools/list` shows the six tools with the required arguments.
- Scenario 2: `make_move` column 3 succeeded, and REST shows `board[5][3]` red at version 1.
- Scenario 3: a missing column gave `isError: true`.
- A rule error gave `isError: true` with the structured NOT_YOUR_TURN object.

Follow-ups filed:
- `mcp-schema-errors-lack-error-object.md`
- `mcp-stdio-transport.md` (ADR-009 item 3)
