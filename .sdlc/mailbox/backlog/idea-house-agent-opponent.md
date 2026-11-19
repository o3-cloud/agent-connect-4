---
title: "Ship a house agent so a human can play within seconds"
status: promoted
type: feature
outcomes: [2, 6]
spec: [AGENT-001, AGENT-006, UI-001, UI-002]
bdr: [BDR-026, BDR-030]
created: 2026-11-18
---

# Ship a house agent so a human can play within seconds

## Problem

Outcome 2 says anyone can open the app and play an AI, and Outcome 6 says a human starts in seconds. Under AGENT-001 the server never hosts agents, so a new visitor has nobody to play against unless they bring their own agent.

## Proposal

Ship a `house-agent` binary (`cmd/house-agent`) that plays as an ordinary external client through the public REST API. It gets no special access to the service.

- Strategies: `random`, `greedy` and `minimax --depth N`, ported from `prototype/agents.js`.
- Run it as its own service in `compose.yaml`.
- The house agent watches for `human_vs_agent` games created with "opponent: house". It joins them and plays.
- The dashboard offers "Play vs House (Easy / Medium / Hard)" as a one-click option.

This keeps the spec's rule that the platform does not run agents, because the house agent is a separate process that uses public interfaces only. It also doubles as the reference agent and as a load generator.

## Acceptance

- [ ] From a fresh `docker compose up`, a human goes from the landing page to their first move in ≤ 2 clicks.
- [ ] The house agent's code uses only the public client package.
- [ ] Killing the house agent leaves the game in its last valid state (AGENT-005).

## Open questions

- Should the house agent's display name and difficulty show as a badge on the board?

## Ranked (2026-11-18): 6 of 16

Outcome 2: a human needs someone to play.

## Promoted (2026-11-18)

Built in daf1200 under ADR-014 ("House reference agent"), and `issues/delivery-house-reference-agent.md` is closed. Arena Bot plays whenever a game is created with `"opponent": "house"`. It wins if it can, otherwise blocks, otherwise plays the centre, and it moves only through the public REST API. It differs from this idea in two ways: it runs in the arena process (through an in-process `http.RoundTripper`) rather than as its own container, and it has one fixed strategy instead of Easy/Medium/Hard. A difficulty choice and a separate process for the showcase match are still open as follow-ups.
