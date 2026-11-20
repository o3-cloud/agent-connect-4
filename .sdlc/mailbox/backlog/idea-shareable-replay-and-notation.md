---
title: "Shareable game links, step-through replay and compact move notation"
status: proposed
rank: 8
type: feature
outcomes: [4]
spec: [UI-010, UI-011, DATA-004, DATA-005]
bdr: [BDR-016, BDR-017, BDR-027]
created: 2026-11-18
---

# Shareable replay and move notation

## Problem

Outcome 4 says anyone with access can watch, inspect and replay a game. The spec covers watching and replay inside the app but says nothing about sharing a game or moving it elsewhere.

## Proposal

- Each game gets a permalink `/g/{gameId}`. It shows live spectating while the game is in progress and the replay once it is completed. Anyone with the link gets read-only access.
- Replay controls: first, previous, play/pause, next, last; a scrubber; the keyboard arrows; and a `?ply=N` deep link.
- Compact notation: one digit per move, 1-based, e.g. `4453...` (the common Connect Four convention). It is available from `GET /api/v1/games/{id}/notation`, from `connect4 history --notation`, and as a copy button in the UI.
- Importing a notation string into replay view gives an analysis-only board, which is useful for checking that replay is deterministic.
- Open Graph tags give a link preview that shows the final board as an image (an SVG rendered on the server).

## Acceptance

- [ ] Replaying the notation in the engine reproduces the recorded final state (DATA-005 property test).
- [ ] A spectator link never exposes credentials or move-submission controls.

## Ranked (2026-11-18): 13 of 16

Outcome 4 beyond the MUST replay.

## Evidence (2026-11-19, c01557d)

The browser replay landed and BDR-017 is accepted. `GET /games/{id}/replay?step=N` rebuilds the board from the recorded moves, has First, Previous, Next and Last links that also work without JS, marks the current move in the move list, and shows the result at the last step. A finished game's board links to its replay. That covers the deep link and step controls from this idea.

What remains is smaller:
- Compact notation (`4453...`): `GET /api/v1/games/{id}/notation`, `connect4 history --notation`, a copy button, and importing a notation string into an analysis-only board.
- Play/pause, a scrubber, and keyboard arrows on the replay page (these also help `idea-accessible-board`).
- Open Graph tags with a server-rendered SVG of the board, so a shared link shows a preview.

This idea is now mainly about notation and sharing. Its rank (13) is unchanged.


## Ranked (2026-11-20, 0cc40e1): 9 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. Replay is accepted (BDR-017). Notation, play/pause and link previews are polish.
