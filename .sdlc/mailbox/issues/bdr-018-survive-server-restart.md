---
title: "BDR-018 not satisfied on main: Preserve Games Across Server Restarts"
status: open
source: bdr-verification
bdr: BDR-018
branch: main
commit: 30cd082
found: 2026-11-18
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-19T21:08:54Z
---

# BDR-018 not satisfied on main: Preserve Games Across Server Restarts

Spec: [`docs/bdr/018-survive-server-restart.md`](../../../docs/bdr/018-survive-server-restart.md)

## Expected

Games and their committed moves survive a server restart, and play continues from exactly where it stopped.

## Observed (black-box, on main)

- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [ ] AC-1: After a restart, an in-progress game can be retrieved, with the same board, players, `currentTurn` and `version` as before.
- [ ] AC-2: Every move committed before the restart appears in the history after it.
- [ ] AC-3: The player to move can submit the next move after the restart, using the version from before the restart.
- [ ] AC-4: Completed games keep their winner and result reason across a restart.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Note (584db77, 2026-11-18, session 5b5d3e77)

Idempotency replay records (`record.replays` in `internal/game/service.go`, API-008) are in memory only. Persist them with the game, so that a retry sent after a restart still replays and does not play twice.
