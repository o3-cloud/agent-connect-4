---
title: "Nothing tests that Arena Bot plays a game it was seated in at start-up only once"
status: open
priority: 5
source: loop-run
type: test-gap
bdr: BDR-018
branch: main
commit: b5317c5
found: 2026-11-20
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-20T08:08Z
spec: [docs/adr/014-house-reference-agent.md, docs/adr/015-jsonl-journal-persistence.md]
---

# Nothing tests that Arena Bot plays a game it was seated in at start-up only once

## Observed

`resumeHouseGames` (internal/app/app.go, cb37d1d) first resumes the bot in every in-progress game it plays, then seats it in waiting house games that a player joined just before a crash. Seating starts the bot's play loop and makes the game `in_progress`. If the two passes were swapped, the bot would run two play loops in that game. No test fails when they are swapped: `TestHouseBotTakesTheSeatItMissedInACrash` only checks that the bot takes the seat.

## Expected

A test that fails if Arena Bot plays any game with more than one loop after a restart.

## Suggested fix

Two loops both send a move for the same turn, so one is refused with `version_conflict` and logged as `move_rejected` with the bot's player id. A single loop never sends a stale move. Restore a journal that ends between the human's join and the bot's, play several human moves, and assert the log has no `move_rejected` event for the bot.
