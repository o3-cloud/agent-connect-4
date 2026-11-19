---
title: "Always-on showcase match so the dashboard is never empty"
status: proposed
rank: 16
type: feature
outcomes: [4, 6]
spec: [UI-001, UI-011, LIVE-002, LIVE-003]
bdr: [BDR-027, BDR-030]
created: 2026-11-18
---

# Always-on showcase match

## Problem

A first-time visitor to a new deployment sees an empty game list. Outcome 4 (games can be watched) and Outcome 6 (effortless) are both easier to grasp when something is already happening on screen.

## Proposal

- An optional compose profile (`docker compose --profile showcase up`) runs two house agents (for example Minimax vs Greedy), each on a different interface (REST vs the MCP bridge). They start a new agent-vs-agent game whenever the previous one finishes.
- The dashboard pins the current showcase game at the top as a small live board with a "Watch" link, and offers "Play the winner" as a call to action.
- The showcase games are tagged `showcase: true` so they can be filtered out of listings and cleaned up after N completed games.

This also gives free end-to-end coverage: if the showcase match stalls, live updates, an agent interface or the engine is broken. That can be alerted on through the operational logs (NFR-006).

## Acceptance

- [ ] With the showcase profile on, the dashboard shows a game in progress within 5 s of startup.
- [ ] The showcase agents use only public interfaces, and the showcase is off by default.

## Ranked (2026-11-18): 16 of 16

Nice first impression; needs the house agent and the UI.
