---
title: "README CLI walkthrough joins only one player, so move and resign fail with GAME_NOT_STARTED"
status: closed
source: bdr-verification
type: defect
branch: main
commit: e517837
found: 2026-11-18
closed: 2026-11-19
spec: [CLI-001, interface/commands.md]
bdr: [BDR-019, BDR-030]
---

# README CLI walkthrough joins only one player, so move and resign fail with GAME_NOT_STARTED

## Observed

A new user who runs `make run` and pastes the "Playing with the `connect4` CLI" block from `README.md` as written:

```sh
eval "$(bin/connect4 register --name Alpha | tail -1)"
GAME=$(bin/connect4 create)
bin/connect4 join $GAME --disc red --name Alpha
bin/connect4 move $GAME --column 3
bin/connect4 state $GAME
bin/connect4 history $GAME --json
bin/connect4 resign $GAME
```

gets:

```
Error: GAME_NOT_STARTED: the game starts when both players have joined   # move
...                                                                       # state: empty board, "waiting, version 0"
Error: GAME_NOT_STARTED: the game starts when both players have joined   # resign, exit 6
```

Only red joins, so the game never leaves `waiting`. The `move` comment ("uses the current version") and the `resign` step can never succeed. The REST walkthrough above it registers and joins two players, and works.

## Expected

The README's CLI block plays a real move and ends with the game `completed`, as the REST block does.

## Fix

Register a second agent and join it as yellow before moving. For example, keep each key in its own variable and pass it per command:

```sh
ALPHA=$(bin/connect4 register --name Alpha --json | jq -r .apiKey)
BETA=$(bin/connect4 register --name Beta --json | jq -r .apiKey)
GAME=$(CONNECT4_API_KEY=$ALPHA bin/connect4 create)
CONNECT4_API_KEY=$ALPHA bin/connect4 join $GAME --disc red --name Alpha
CONNECT4_API_KEY=$BETA  bin/connect4 join $GAME --disc yellow --name Beta
CONNECT4_API_KEY=$ALPHA bin/connect4 move $GAME --column 3
```

Adjust this to however `register --json` actually reports the key. Add a test that runs the README block, so the walkthrough cannot drift from the CLI again (the REST block broke the same way in 39f853d).

## Still reproduces (f375e07, 2026-11-19)

The block is unchanged and still joins only Alpha. It now also runs `bin/connect4 watch $GAME --json`, which waits for moves in a game that never starts. A user pasting the block hangs at that line and never reaches `resign`. The house-agent block below it now works (0/20 failures).

## Resolution

The README's CLI block now registers Alpha (key in `CONNECT4_API_KEY`) and Beta (key in `$BETA`, passed with `--api-key`), joins both, moves, reads state and history, resigns as Beta, and runs `watch`, which prints the result and exits on the finished game. It needs no `jq`: Beta's key comes from the last line of `register` with `cut`. A note says to run `watch` in a second terminal to follow a live game.

`TestREADMECLIWalkthroughRuns` (tests/readme_test.go) builds `bin/connect4`, extracts every `sh` block in the README's CLI section and runs them in order with `set -eu` against a test server, starting with no `CONNECT4_API_KEY`. It must see `game_completed`. It failed on the old README with `GAME_NOT_STARTED` and passes now, so the walkthrough cannot drift from the CLI again. The Arena Bot block in the same section runs too.

Black-box: the same blocks, pasted into `sh` against the real `bin/arena`, exit 0. The walkthrough game ended `completed, red wins (resignation)`, and Arena Bot replied in the house game.

Also fixed: `.sdlc/worktree/` is now in `.gitignore`. After `.sdlc` became tracked (6e8699e), loop worktrees showed up as untracked files in the main checkout.
