---
title: "Long-poll 'wait for my turn' for REST, CLI and MCP agents"
status: proposed
rank: 3
type: feature
outcomes: [1, 3, 6]
spec: [AGENT-004, LIVE-001, CLI-008, MCP-004]
bdr: [BDR-020, BDR-021, BDR-024]
created: 2026-11-18
---

# Long-poll "wait for my turn"

## Problem

Agents that use REST, CLI or MCP can't hold a WebSocket open, so they have to poll `get_game_state` in a loop. That costs tokens for LLM agents, runs into rate limits (SEC-007), and adds latency. MCP tool calls are request/response only.

## Proposal

Add one blocking primitive, built on the LIVE-001 event bus and exposed the same way on every interface:

| Interface | Shape |
|---|---|
| REST | `GET /api/v1/games/{id}/state?waitForTurn=true&sinceVersion=N&timeout=30s` |
| CLI | `connect4 wait <gameId> [--timeout 30s] --json` |
| MCP | `wait_for_turn { game_id, since_version, timeout_seconds }` |

It returns as soon as one of these happens: it is the caller's turn, the game completes, or the timeout expires (the call then returns current state and `timedOut: true`).

## Acceptance

- [ ] An agent can play a whole game with exactly one `wait` call and one `move` call per turn.
- [ ] The wait returns within 100 ms after the opponent's move commits.
- [ ] Waiting calls don't count against the move rate limit.
- [ ] Results match across REST, CLI and MCP (NFR-004).

## Notes

This goes beyond the spec, so it needs a BDR, and possibly an addition to SPEC §17 (MCP tools).

## Ranked (2026-11-18): 7 of 16

Removes busy polling for every agent interface.

## Evidence (2026-11-18, 4baf071)

Nothing on main pushes events: there is no event bus, SSE or WebSocket (LIVE-001..004 are not implemented). Every client polls:
- The house agent polls `GET /games/{id}` every **25 ms** (`pollInterval`), which is about 40 requests per second for each live game. At NFR-003's 100 games that is about 4,000 requests per second of polling, and once SEC-007 rate limits arrive, the house agent could throttle itself.
- `connect4 watch` (4baf071) polls on a fixed `--interval`.

The LIVE-001 event bus is a MUST anyway, and the browser's WebSocket (BDR-027) needs it. Building it first and then exposing a "wait until my turn" call on REST, CLI and MCP removes all three polling loops. Moved to rank 5.

## Evidence (2026-11-18, 925cd38)

The browser UI (BDR-026, accepted) is now a fourth client that polls: `internal/web/templates/game.html` refreshes the board with `hx-trigger="every 1s"`. LIVE-002 requires a WebSocket subscription, and ADR-011 plans htmx's WebSocket extension. That needs the same event bus as a "wait until my turn" call for agents. So one server-side component, the LIVE-001 publisher, replaces four polling loops: the house agent, `connect4 watch`, the browser, and any external agent.
## Evidence (2026-11-19, d9b031c)

Rate limiting has landed: 600 requests a minute for each agent or IP address, across `/api/v1` and `/mcp`. Arena Bot and the browser UI are exempt, so the 25 ms house-agent polling noted earlier doesn't throttle itself. External agents have no exemption, though, and still no way to wait except polling:
- An agent that polls once a second (`connect4 watch`'s default `--interval`) and plays a 21-move game uses about 1 request a second while it waits. That's fine for one game, but an agent playing 10 or more games at once hits `RATE_LIMITED`.
- The rate limit forces a trade-off: poll slower and answer later, or poll faster and get throttled. LLM agents on MCP pay for tokens on every poll as well.
- A "wait until my turn" call, held open on the server and counted as one request until it returns, removes the trade-off. That is also why this idea says waiting calls shouldn't count against the move budget.

Moved to rank 3. It and the LIVE-001 event bus are now the main gap for Outcome 3 (agents play autonomously) at any scale.

## Evidence (2026-11-19, f375e07)

Polling has now caused a real defect. Arena Bot polled to notice that the human had joined, and took its seat a few milliseconds after the human's join had already returned `waiting`. As a result, the README flow (create, join, move) failed with `GAME_NOT_STARTED` 9 times out of 10 (`issues/house-agent-join-race-readme.md`). The fix adds a one-off service hook, `WithAwaitingOpponent`, that `Join` calls synchronously. That's an event subscription built for a single case.

Arena Bot still polls every 25 ms for its turns once it is seated. A general LIVE-001 publisher (events on join, start, move and complete) would cover this hook, the house agent's turn loop, the browser's 1 s refresh, `connect4 watch` and a "wait until my turn" call for external agents. It's worth building before a second one-off hook is needed.

