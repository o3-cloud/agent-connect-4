---
title: "Accessible board: keyboard play, screen-reader move announcements, colour-blind discs"
status: proposed
rank: 9
type: feature
outcomes: [2, 6]
spec: [UI-003, UI-006, UI-007, UI-013]
bdr: [BDR-026, BDR-029]
created: 2026-11-18
---

# Accessible board

## Problem

Outcome 2 says anyone can play without technical knowledge. A red/yellow board you can only click with a mouse leaves out keyboard users, screen-reader users and people with red/green colour blindness.

## Proposal

- Keyboard: `←/→` choose a column, `Enter`/`Space` drop a disc, `0–6`/`1–7` drop straight into that column (the prototype already does this). The focus ring shows the column under consideration and a preview disc.
- Screen readers: the board is an ARIA grid. Each move is announced in an `aria-live="polite"` region, e.g. "Yellow (Minimax Bot) played column 4, row 3. Your turn."
- Colour-blind support: discs carry a pattern or glyph as well as a colour, and contrast is WCAG AA.
- `prefers-reduced-motion` turns off the disc-drop animation.
- htmx (ADR-011): swapped fragments keep focus and live-region announcements.

## Acceptance

- [ ] A full game against the house agent is completable with keyboard only.
- [ ] axe-core (Playwright) reports no serious violations on dashboard, game and replay pages.

## Ranked (2026-11-18): 14 of 16

Outcome 2 for all humans; do with the UI phase.

## Evidence (2026-11-18, 925cd38)

The first browser UI has started on this: the status line is an `aria-live="polite"` region, and each column button has `aria-label="Drop a disc in column N"`. Still missing: arrow-key and number-key play, announcing each move (column, row, player) rather than only the status, a pattern as well as a colour on the discs, `prefers-reduced-motion`, and an axe-core check. With the 1-second polling swap, the live region may repeat itself every second. Check this with a screen reader once WebSocket updates replace the polling.

## Ranked (2026-11-20, 0cc40e1): 10 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. "Anyone can play" (outcome 2). Not a release criterion on its own, but cheap to start.
