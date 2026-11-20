---
title: "LLM-friendly game state: text board, perspective fields, threats"
status: proposed
rank: 7
type: feature
outcomes: [1, 6]
spec: [GAME-005, AGENT-004]
bdr: [BDR-003]
created: 2026-11-18
---

# LLM-friendly game state

## Problem

A 6×7 array of `null`/"red"/"yellow" is fine for code but easy for an LLM to misread: it confuses rows, gets gravity wrong, or loses track of which colour it is. Bad moves make AI play look broken to the people watching.

## Proposal

Add optional fields to the game state on every interface (via `?view=agent`, or by default on MCP):

- `boardText`: an ASCII grid with column numbers along the bottom, `R`/`Y`/`.` cells and the bottom row last.
- `you`: `{ disc, isYourTurn }`, worked out from the caller's credentials.
- `columnHeights` and `legalMoves`, which are already there.
- `lastMove`: `{ column, row, disc }`.

These fields are derived views only. The canonical `board` doesn't change, and determinism is unaffected.

## Acceptance

- [ ] `boardText` round-trips: parsing it gives back `board`.
- [ ] `you` never shows another player's identity details (SEC-005).
- [ ] An LLM eval shows fewer illegal or nonsensical moves with `view=agent` than with the plain board.

## Out of scope

Hints about threats or best moves; that would be "sophisticated AI", which the outcomes don't target yet.

## Ranked (2026-11-18): 12 of 16

Improves AI play quality; additive.

## Ranked (2026-11-20, 0cc40e1): 8 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. Helps LLM agents read the board. Worth doing with onboarding's MCP prompt.
