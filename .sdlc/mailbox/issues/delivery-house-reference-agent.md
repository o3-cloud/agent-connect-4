---
title: "House reference agent (Arena Bot) joins and plays human_vs_agent games"
status: closed
source: progress-dashboard
type: delivery
priority: 1
found: 2026-11-18
outcomes: [2, 5, 6]
spec: [AGENT-007, AGENT-001..006, ADR-014]
bdr: [BDR-026, BDR-030]
unblocks: [bdr-026-human-plays-agent-in-browser]
---

# House reference agent (Arena Bot)

Outcome 2 is at 0 of 11 criteria. Even with a browser page, a human who starts a `human_vs_agent` game has nobody to play: nothing guarantees an agent joins. [ADR-014](../../../docs/adr/014-house-reference-agent.md) decides the fix.

## Scope

1. `POST /api/v1/games` accepts an optional `"opponent": "house"` (only with `mode: human_vs_agent`; otherwise `400 INVALID_ARGUMENT`).
2. An `internal/houseagent` package: registers once at startup through `POST /api/v1/agents` ("Arena Bot"), then for each such game joins the open slot and plays through the public REST API only (no calls into `internal/game`).
3. Strategy: win now, else block the opponent's immediate win, else the legal column nearest the centre (3, 2, 4, 1, 5, 0, 6). Deterministic.
4. One goroutine per game; a panic or error forfeits only that game (AGENT-005).

## Acceptance

- [x] An HTTP test creates `human_vs_agent` with `"opponent": "house"`, joins as the human, and sees Arena Bot join and reply to each move within 1 second, through to `completed`.
- [x] Unit tests: the bot takes an immediate win, and blocks an immediate loss.
- [x] `"opponent": "house"` with `agent_vs_agent` returns `400 INVALID_ARGUMENT`.
- [x] Arena Bot's API key never appears in any response.
- [x] `make check` passes.

## Done (daf1200, 2026-11-18)

`internal/houseagent` (Arena Bot) plays through the REST API with its own key. Tests: `tests/house_test.go` (full game against a human with every reply within 1 s, strategy cases, and the agent_vs_agent rejection). `make check` passes (coverage 90.1%, race detector on). Verified with the built binaries: a human joined red, Arena Bot took yellow and replied in 0.04 to 0.11 s, and won in 8 moves. No API key appears in output. `connect4 create --opponent house` and the README show how to play it.
