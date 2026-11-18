# ADR-014: A House Reference Agent for Human vs Agent Games

## Status

Accepted

Decided on 2026-11-18 by the progress-dashboard session, under the project owner's standing instruction to make the best decision the evidence supports. Supersede it with a new ADR if the owner chooses differently.

## Context and Problem Statement

Outcome 2 says anyone can open the app, start a game against an AI agent, and play without technical knowledge. BDR-026 covers the human's side of that game, and `SPEC.md` Phase 2 lists "reference agents", but no requirement says an opponent will be there. A human who is not a developer cannot bring their own agent, so a browser UI alone would leave them waiting in a `human_vs_agent` game that nobody joins.

Where does the AI opponent come from when a human starts a game?

## Decision Drivers

- **Outcome 2 and Outcome 6**: a human starts playing in seconds, with no setup
- **Fairness** (Outcome 5, AGENT-001, AGENT-002): every agent, including ours, plays under the same rules through a public interface, with no back door into game state
- **Not optimizing for strength** (OUTCOMES.md): the opponent should be credible, not sophisticated
- **Testability**: BDR-026 and BDR-030 need a predictable opponent for browser tests

## Considered Options

- **Option A**: A house reference agent that runs inside the arena process and plays through the public REST API with its own API key
- **Option B**: Wait in the lobby until an external agent joins
- **Option C**: A built-in opponent that calls the game service directly

## Pros and Cons of the Options

### Option A: House agent over the public API

- Pro: a human always has an opponent within a second of creating the game
- Pro: it is a real client of the same API (ADR-013 key, same rules, same rate limits), so it proves the fairness the platform claims
- Pro: it doubles as the reference agent that Phase 2 asks for and that BDR-024 tests can reuse
- Con: one more component running in the server process

### Option B: Wait for an external agent

- Pro: no new code
- Con: with no agents online, the human waits indefinitely, which fails Outcome 2

### Option C: Direct calls into the game service

- Pro: simplest
- Con: it bypasses authentication and the API contract, so it is not evidence that external agents are treated the same way

## Decision Outcome

Chosen option: **Option A, a house reference agent over the public REST API**.

- When a game is created in `human_vs_agent` mode with `"opponent": "house"`, the arena starts the house agent for that game. The browser UI sends this by default.
- The house agent registers once at startup (`POST /api/v1/agents`, display name "Arena Bot", type `agent`), joins the open slot, and moves only through `POST /api/v1/games/{id}/moves`, using `legalMoves` and `version` from game state (AGENT-004).
- Strategy, in order: win now if it can; otherwise block the opponent's immediate win; otherwise play the legal column closest to the centre. It is deterministic for a given position, so browser tests are repeatable.
- It replies within 1 second of the human's move, which fits BDR-027's 2-second live-update budget.
- A game created without `"opponent": "house"` waits for an external agent, as today.

## Consequences

### Positive

- Outcome 2 becomes reachable as soon as the BDR-026 page exists.
- The reference agent becomes a working example for developers connecting their own agents (Outcome 6).

### Negative

- The house agent's failures must not affect other games (AGENT-005). It runs one goroutine per game, and a crash forfeits only that game.

### Neutral

- `SPEC.md` gains AGENT-007 (House Opponent) and an `opponent` field on create.

## Compliance

- An HTTP test creates a `human_vs_agent` game with `"opponent": "house"`, joins as the human, and observes the house agent join and reply to each human move through the public API within 1 second.
- A test gives the house agent a position with an immediate win, and another with a block it must make, and checks its column.
- The house agent's API key never appears in any output (SEC-005).

## References

- OUTCOMES.md 2, 5, 6; `SPEC.md` AGENT-001..006, Phase 2 "Reference agents"
- [ADR-013](013-self-issued-agent-api-keys.md), [BDR-026](../bdr/026-human-plays-agent-in-browser.md), [BDR-024](../bdr/024-autonomous-agent-match.md), [BDR-030](../bdr/030-start-playing-quickly.md)
