---
title: "Optional turn clock so abandoned games still reach an outcome"
status: rejected
type: spec-change
outcomes: [3, 5]
spec: [AGENT-005, GAME-006, TURN-007]
bdr: [BDR-025]
created: 2026-11-18
---

# Optional turn clock

## Problem

AGENT-005 keeps the last valid state when an agent crashes, but nothing ever ends the game. Outcome 3 says games reach a valid outcome. With unattended AI-vs-AI play, dead games will pile up on the dashboard and the human opponent will be left waiting.

## Proposal

- Add an optional `turnTimeoutSeconds` on create (default off, so current spec behaviour is unchanged).
- If the active player hasn't moved within the timeout, the server completes the game with `resultReason: "timeout"` and the opponent wins. It publishes a completion event (LIVE-004).
- The state shows `turnDeadline` (ISO time) so agents and the UI can display a countdown.
- A separate idea: a `waiting` game nobody joins expires after N hours (`status: abandoned`, or it is hidden from the list).

## Spec impact

This adds a `timeout` value to `resultReason` (SPEC §15) and needs a new BDR and an amendment to GAME-006. Deterministic replay still works because the timeout is recorded as a terminal event.

## Acceptance

- [ ] A game whose agent stops responding completes within `timeout + 1s`, with the right winner.
- [ ] The timer survives a server restart, using a persisted deadline (DATA-006).

## Decision (2026-11-18)

Rejected by the project owner: no turn clock. Games without a move stay `in_progress`; resignation (BDR-010) remains the way to end an abandoned game.
