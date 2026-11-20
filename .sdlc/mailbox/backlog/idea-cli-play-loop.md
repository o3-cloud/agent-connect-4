---
title: "`connect4 play --exec`: turn any program into an agent"
status: proposed
rank: 6
type: feature
outcomes: [1, 3, 6]
spec: [CLI-001..008, AGENT-006]
bdr: [BDR-019, BDR-024]
created: 2026-11-18
---

# `connect4 play --exec`: turn any program into an agent

## Problem

To connect an agent today, a developer has to write the whole game loop: join, poll, check the turn, build a move with an idempotency key, retry on `STALE_STATE`. That takes far longer than the "minutes" promised in Outcome 6.

## Proposal

Add a CLI command that runs the loop and hands off only the decision:

```sh
connect4 play <gameId> --disc yellow --name "my-bot" --exec "python my_bot.py"
```

- On each turn, the CLI writes the game state as JSON to the child process's stdin.
- The child writes a column number to stdout. The CLI submits it with `expectedVersion` and a generated idempotency key.
- The CLI handles retries, stale state and the end of the game, and exits 0 on win, loss or draw.
- With `--exec` left out, the CLI plays interactively in the terminal (a human-friendly mode).
- `connect4 play --new --mode agent_vs_agent` creates the game and prints the join command for the opponent.

Any language, any model, a shell one-liner (`--exec 'shuf -i 0-6 -n1'`) all work, which is AGENT-006 in practice.

## Acceptance

- [ ] Two `connect4 play --exec` processes complete a game with no human input (AT-002 variant).
- [ ] If the child process exits non-zero or prints an illegal column, the CLI reports it clearly and the game state stays valid.
- [ ] `--json` emits one JSON line per event (newline-delimited).

## Ranked (2026-11-18): 8 of 16

Outcome 6: an agent in minutes.

## Evidence (2026-11-19, 7e2e0a3)

The same turn loop (read state, check whose turn it is, choose from `legalMoves`, submit with `expectedVersion`, sleep, repeat) now exists three times: the house agent (`internal/houseagent`, 25 ms poll), and the CLI and MCP agents in `tests/agent_match_test.go` (20 ms poll, "to stay inside the default rate limit"). Every external developer will write it a fourth time. `connect4 play --exec` would put the loop in one place that's tested, with retries on `STALE_STATE`, `Retry-After` handling, and later a switch to wait-for-turn, all done once.

## Ranked (2026-11-20, 0cc40e1): 7 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. `connect4 play` would make "an agent in minutes" a single command, after onboarding and wait-for-turn.
