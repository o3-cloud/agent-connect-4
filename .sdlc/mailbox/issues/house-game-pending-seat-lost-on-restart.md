---
title: "A house game created but not yet joined loses its Arena Bot seat across a restart"
status: closed
priority: 4
source: loop-run
type: defect
bdr: BDR-018
branch: main
found: 2026-11-20
closed: 2026-11-20
fixed_in: cb37d1d
spec: [docs/adr/015-jsonl-journal-persistence.md, docs/adr/014-house-reference-agent.md]
---

# A house game created but not yet joined loses its Arena Bot seat across a restart

## Observed

`"opponent": "house"` on create (REST) or **Play vs AI** (web) marks the game as pending for Arena Bot in `houseagent.Agent.pending`, which is held in memory only. The journal records `game_created`, but not that the game is a house game. If the server restarts after the game is created and before its first player joins, the game comes back as a plain `waiting` `human_vs_agent` game. When the human then joins, Arena Bot never takes the other seat.

Games that already have both players are fine: the bot resumes them after a restart.

## Expected

A house game keeps its pending bot seat across a restart, until the 10-minute `joinWait` it would have had anyway runs out.

## Suggested fix

Record the house opponent in the journal: either an `opponent` field on `game_created`, or a `house_seat_offered {gameId}` event. On restore, re-mark waiting house games as pending, measuring the remaining `joinWait` from the event's `at`. Test: create a house game, restart, join as a human, and expect Arena Bot as the opponent.

## Resolution

`game_created` now carries `"house": true` for a house game, created through `game.Service.CreateHouseGame` (REST `"opponent": "house"` and web **Play vs AI**). The offer is saved in the same journal line as the game, so the two can no longer be split by a crash. On restore, `resumeHouseGames` offers Arena Bot the seat again in each waiting house game through `houseagent.Agent.Offer`. The offer lasts for what is left of the 10-minute `joinWait`, counted from the event's `at`, never a fresh 10 minutes. If a player had joined and the bot had not, the bot takes the open seat during start-up. This pass runs after in-progress games are resumed, so no game gets two bot play loops. A journaled `house` game whose mode is not `human_vs_agent` does not replay. Journals written before this change replay as before: their house games come back as plain games.

- `TestHouseSeatSurvivesARestartBeforeAnyoneJoins`: create, restart, join; yellow is Arena Bot and the bot answers the first move. Red before the fix.
- `TestHouseBotTakesTheSeatItMissedInACrash`: journal ends after the human's `player_joined`; on start-up the bot takes yellow. Red before the fix.
- `TestExpiredHouseSeatIsNotOfferedAfterARestart`: `game_created` 11 minutes old; the seat is not offered.
- `TestJournalWithAHouseAgentVsAgentGameRefusesToStart`, `TestHouseOfferIsJournaledWithTheGame`.
- Mutations caught: no re-offer, fresh `joinWait`, no seat after crash, house not journaled, no mode check. Not caught: removing `Offer`'s `left <= 0` early return. The timer then fires at once and withdraws the offer anyway; the check only closes that race.
- `make check` PASS, 89.5%.
- Black-box, real `bin/arena` with `ARENA_JOURNAL_PATH`: register, create house game, SIGKILL, restart, join. The game is `in_progress` with Arena Bot as yellow, and after the human's move it is at version 2 with red to play.
