---
title: "BDR-024 not satisfied on main: Two External Agents Complete a Game Autonomously"
status: closed
source: bdr-verification
bdr: BDR-024
branch: main
commit: 30cd082
found: 2026-11-18
closed: 2026-11-19
fixed_in: 7e2e0a3
---

# BDR-024 not satisfied on main: Two External Agents Complete a Game Autonomously

Spec: [`docs/bdr/024-autonomous-agent-match.md`](../../../docs/bdr/024-autonomous-agent-match.md)

## Expected

Two independent external agents, possibly on different interfaces, can create, join and play a game to a valid result with no human involvement.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [ ] AC-1: Agent A creates a game and joins it through the CLI, and agent B joins through MCP, with no human action.
- [ ] AC-2: Both agents can discover their turn and the legal moves from game state alone.
- [ ] AC-3: The game reaches `completed` with a winner or a draw.
- [ ] AC-4: A spectator browser shows the moves while the game is in progress.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main. The behaviour already worked once the CLI (BDR-019), MCP (BDR-021) and browser spectating (BDR-026) were in place. 7e2e0a3 (`test(agents): a CLI agent and an MCP agent play a game on their own`) pins scenario 1 as `TestAutonomousCLIAgentVsMCPAgent`:

- AC-1: A runs `create --mode agent_vs_agent` and `join --disc red` through the CLI, and B calls `join_game` as yellow over MCP.
- AC-2: each agent runs in its own goroutine. It reads only the game state (`status`, `currentTurn`, `legalMoves`, `version`) to find its turn and choose a column.
- AC-3: the game reaches `completed` within 42 moves, with discs alternating red and yellow from red.
- AC-4: B waits for its first move until an anonymous browser has seen the game page `in_progress` with A's move, both players and the live refresh. This makes the check deterministic.

Checks:
- Ran 3 times under `-race`, passing each time.
- Mutation check: renaming the page's `data-move` attribute makes the test fail.
- `make check` passes, 91.4% coverage.

Black-box run against the real `bin/arena` with two independent processes:
- Agent A is a bash loop around `bin/connect4 state/move`. Agent B is raw MCP JSON-RPC over curl (`initialize`, `join_game`, `get_game_state`, `make_move`), polling every 200 ms.
- A curl spectator saw the page `in_progress` with moves shown 5 times.
- The game completed with red winning by `connect_four` in 7 moves, discs r,y,r,y,r,y,r. The server log has 7 `move_accepted` records.

Note: an agent polling flat out, every few milliseconds, would hit the default 600 requests per minute limit. Push events (BDR-027) would remove the need to poll.
