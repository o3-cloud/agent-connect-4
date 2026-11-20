---
title: "BDR-018 not satisfied on main: Preserve Games Across Server Restarts"
status: open
source: bdr-verification
bdr: BDR-018
branch: main
commit: 30cd082
found: 2026-11-18
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

## Progress (2026-11-19, session 5b5d3e77): blocked on a decision

The Arena now exists, but everything it knows is in process memory: games, history, idempotency results and API keys. A restart loses all of it, so AC-1 to AC-4 still fail. Fixing this needs a storage choice that no ADR has made, and this loop has no mandate to make it for the owner.

**Decision needed:** accept, amend or reject [ADR-015](../../../docs/adr/015-jsonl-journal-persistence.md) (Proposed). It recommends SQLite through `modernc.org/sqlite`, which is pure Go, follows SPEC §19 and keeps the `CGO_ENABLED=0` `scratch` image. The move log is the source of truth, each move commits in one transaction, and `ARENA_DB_PATH` sets the location (unset means memory only). It also compares cgo SQLite, a stdlib JSON Lines journal (the fallback if no new dependency is wanted), bbolt and PostgreSQL.

Once ADR-015 is accepted, the next issue-worker run implements it as the ADR describes (Store interface, then `sqlstore`, then the restart test) and closes this issue. BDR-033 AC-2 (persistence location set by configuration) follows from the same work.

## Decision (2026-11-20, project owner)

ADR-015 is **Accepted with Option C**: an append-only JSON Lines journal written with the standard library, set by `ARENA_JOURNAL_PATH`, with no new dependency. This was the fallback the ADR offered, not Option A, which it had recommended. `needs: human` is cleared. The next issue-worker run implements the ADR's steps 1-6 in order (`game.Store`, `memstore`, `journalstore`, replay on start, torn-tail recovery, Arena Bot lookup), adds the restart test, and closes this issue. BDR-033 AC-2 then follows, using `ARENA_JOURNAL_PATH` on a volume.
