---
title: "README house-agent flow fails 9/10: move runs before Arena Bot has joined (GAME_NOT_STARTED)"
status: closed
source: bdr-verification
type: defect
branch: main
commit: 6f4394c
found: 2026-11-18
closed: 2026-11-19
fixed_in: f375e07
spec: [ADR-014, AGENT-007]
bdr: [BDR-026, BDR-030]
---

# README house-agent flow fails 9/10: move runs before Arena Bot has joined

## Observed

The README says "Arena Bot takes the other slot once you join and replies within a second", and gives:

```sh
GAME=$(bin/connect4 create --mode human_vs_agent --opponent house)
bin/connect4 join $GAME --disc red --name You --type human
bin/connect4 move $GAME --column 3 && bin/connect4 state $GAME
```

I ran these three commands as written 10 times against a fresh `bin/arena` (with a registered human key). The `move` failed 9 times:

```
Error: GAME_NOT_STARTED: the game starts when both players have joined
```

The game then sits forever with red to move, because the human's first move was rejected and the script went on. When I polled instead, Arena Bot joined 0.06 s after the human and replied 0.06 s after red's move. So the bot works, but it joins asynchronously after the human's `join` has already returned `200` with `status: waiting`.

## Expected

A human who follows the README can move right after joining. The `join` response should already show the game `in_progress` with Arena Bot seated, or the README should say to wait for that.

## Fix options

1. **Preferred:** seat Arena Bot synchronously inside the human's join (or at create time), so the join response is already `in_progress`. Nothing then depends on timing, and a human in the browser (BDR-026) sees the bot immediately.
2. Alternatively, have `connect4 move` wait briefly on `GAME_NOT_STARTED` for a house game, or make the README poll `state` until `in_progress`.

Add a test that does `join` then `move` with no sleep and expects `200`. The existing `tests/house_test.go` waits for the bot, which hides the race.

## Still reproduces (925cd38, 2026-11-18)

After the browser UI (9a7a343) landed, the README flow failed 20 out of 20 runs. A human using the browser (BDR-026) only avoids it if the page waits for `in_progress` before enabling moves.

## Still reproduces (d9b031c, 2026-11-19)

The README flow still fails: `move` straight after `join` exits 6 with `GAME_NOT_STARTED`.

## Resolution

Fixed in f375e07 with option 1 (seat Arena Bot synchronously). `game.Service` gained `WithAwaitingOpponent`: when a join leaves a game with one player, `Join` calls the hook before returning and then returns the game as the hook left it. Arena Bot's `Seat` (wired in `internal/app`) joins house games through the public REST API, as ADR-014 requires, then plays in the background. All interfaces join through `Service.Join`, so REST, CLI, MCP and the browser all get `in_progress` straight from the human's join. The browser's one-second wait loop is gone.

Tests (red before, green after): `TestHouseAgentIsSeatedBeforeJoinReturns` (20 REST games, both discs, move with no wait), `TestHouseAgentCLIJoinThenMove` (the README flow, 10 times), and `TestHouseAgentSeatedBeforeMCPJoinReturns`. A mutation that unwires the hook fails all of these. `make check` PASS, 91.2%.

Black-box against the real `bin/arena`: the README's three commands succeeded 20/20 (they failed 9/10 before), Arena Bot replied in column 3, and browser **Play vs AI** lands on a page with `data-status="in_progress"`.
