---
title: "Nothing tests that Arena Bot plays a game it was seated in at start-up only once"
status: closed
priority: 5
source: loop-run
type: test-gap
bdr: BDR-018
branch: main
commit: b5317c5
found: 2026-11-20
closed: 2026-11-20
fixed_in: fe90185
spec: [docs/adr/014-house-reference-agent.md, docs/adr/015-jsonl-journal-persistence.md]
---

# Nothing tests that Arena Bot plays a game it was seated in at start-up only once

## Observed

`resumeHouseGames` (internal/app/app.go, cb37d1d) first resumes the bot in every in-progress game it plays, then seats it in waiting house games that a player joined just before a crash. Seating starts the bot's play loop and makes the game `in_progress`. If the two passes were swapped, the bot would run two play loops in that game. No test fails when they are swapped: `TestHouseBotTakesTheSeatItMissedInACrash` only checks that the bot takes the seat.

## Expected

A test that fails if Arena Bot plays any game with more than one loop after a restart.

## Suggested fix

Two loops both send a move for the same turn, so one is refused (`STALE_STATE` or `NOT_YOUR_TURN`) and logged as `move_rejected` with the bot's player id. A single loop never sends a stale move. Restore a journal that ends between the human's join and the bot's, play several human moves, and assert the log has no `move_rejected` event for the bot.

## Resolution

`TestHouseBotSeatedAtStartUpPlaysOnce` (tests/restart_test.go) writes a journal holding a human agent (registered with the hash of a key the test knows), a house game, and the human's `player_joined`, but no join for the bot. It opens the arena with a log sink, plays up to six bot turns through the human's key, and asserts that no `move_rejected` record is logged for the game.

- On main: passes 5 runs out of 5.
- With the waiting-games pass moved before the resume pass in `resumeHouseGames`: fails 10 runs out of 10. The second loop's move is refused as `NOT_YOUR_TURN`, because it read state before the first loop's move landed.
- `make check` PASS, 89.5%.

No production code changed.
