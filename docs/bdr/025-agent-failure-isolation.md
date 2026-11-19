# BDR-025: Isolate Agent Failures

## Status

Accepted

## Behavior

If an agent disconnects, crashes or stops responding, its game keeps its last valid state, and other games are unaffected.

## Context

External agents are outside the platform's control. Their failures must never corrupt a game or spread to other matches.

Traces to: SPEC.md AGENT-005.

## Acceptance Criteria

- AC-1: After an agent stops sending requests, its game state is unchanged and remains readable.
- AC-2: An agent that reconnects later, with the same credential, can continue from the stored state.
- AC-3: Games between other agents continue to accept moves while one agent is unresponsive.

## Verification

### Scenario 1: Agent stops mid-game

- **Given** two in-progress games, `G1` (agents A and B) and `G2` (agents C and D)
- **When** agent A's process is killed while it is A's turn in `G1`
- **Then** `G1`'s state and `version` are unchanged and readable
- **And** agents C and D can still complete moves in `G2`
- **And** a restarted agent A, with the same credential, can submit its move in `G1`
