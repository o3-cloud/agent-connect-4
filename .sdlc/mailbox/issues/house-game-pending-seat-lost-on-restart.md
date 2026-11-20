---
title: "A house game created but not yet joined loses its Arena Bot seat across a restart"
status: open
priority: 4
source: loop-run
type: defect
bdr: BDR-018
branch: main
found: 2026-11-20
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
