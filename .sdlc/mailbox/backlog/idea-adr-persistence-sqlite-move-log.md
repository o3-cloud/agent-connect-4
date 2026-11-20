---
title: "ADR-015: SQLite (pure Go) with an append-only move log as the source of truth"
status: promoted
type: architecture
outcomes: [4, 5]
spec: [DATA-001..006, TURN-004, TURN-005, API-008, NFR-001, NFR-008, SPEC §19 "Initial database: SQLite"]
bdr: [BDR-011, BDR-012, BDR-016, BDR-017, BDR-018]
created: 2026-11-18
---

# ADR-015: persistence

## Problem

ADRs 008–012 replaced the stack in SPEC §19 for HTTP, MCP, CLI, UI and CSS, but none covers persistence. DATA-001..006, AT-010 (restart) and idempotency (API-008) all depend on that choice. The walking skeleton starts with an in-memory store, and the next step needs a decision.

## Proposal

Write ADR-015 recommending:

- **SQLite through `modernc.org/sqlite`** (pure Go). This keeps `CGO_ENABLED=0` builds and the distroless/alpine image. It uses WAL mode, and the file path comes from configuration (`ARENA_DB_PATH`, NFR-008), with a compose volume.
- **The move log is the source of truth.** Tables are `games` (config, players, status, version, result), `moves` (game_id, move_number, player_id, column, row, ts; primary key `(game_id, move_number)`), and `idempotency` (game_id, player_id, key, response_json).
- **One transaction per move.** It inserts the move, updates `games.version` with `WHERE version = :expected` (an optimistic check that also satisfies TURN-004 and TURN-006), and records the idempotency result. All three commit together (DATA-003).
- **Board state is derived** by replaying `moves` through the engine, with a cache in memory per game. Replay (DATA-005) and restart recovery (DATA-006) then use the same code path as live play, so they can't drift apart.
- Keep a `Store` interface so PostgreSQL stays an option (SPEC §19). Don't build it now.

Options to compare in the ADR: SQLite with cgo (`mattn`), BoltDB/bbolt, a JSON file per game, and PostgreSQL from day one.

## Acceptance

- [ ] ADR-015 is accepted and AGENTS.md lists it.
- [ ] A restart test kills the server process in the middle of a game, restarts it, and gets back the identical state and history (AT-010).
- [ ] A concurrency test (100 goroutines sending the same `expectedVersion`) commits exactly one move under `-race` (AT-007).

## Ranked (2026-11-18): 2 of 16

Next decision after the skeleton; DATA-*, restart (AT-010) and idempotency depend on it.

## Evidence (2026-11-18, eee9bc7)

`internal/game/service.go` keeps games in `map[string]*record` inside `Service`, and there is no store interface yet. Before SQLite goes in, the first step is to pull out a `Store` interface (`Create`, `Get`, `AppendMove(gameID, expectedVersion, move, idempotency)`, `List`), with the current map as the in-memory version. The SQLite version then only has to pass the same tests.

## Evidence (2026-11-18, 47b8729)

- ADR-013 is now taken by self-issued agent API keys, so this proposal is renumbered to ADR-014.
- New state now lives only in memory: issued keys (`internal/identity`), move history (cdffcd9) and resignations (BDR-010, accepted). A restart loses every agent's credentials as well as every game, so BDR-018 and AT-010 can't pass, and agents would have to register again after every deploy. Add an `api_keys(agent_id, key_hash, display_name, created_at)` table to the schema.
- Moved to rank 1. Most of the remaining MUST criteria (BDR-011, 012, 017, 018) depend on this decision.

## Evidence (2026-11-18, 584db77)

Idempotent retry (API-008) now works, but the replay results are kept in `record.replays`, a map in memory on each game (`internal/game/service.go`). If the server restarts between a move and the agent's retry, the result is lost and the retry is treated as a new move. The `idempotency` table already in this proposal fixes that, as long as the move and its idempotency row commit in the same transaction (DATA-003). That gives BDR-012 and BDR-018 the same dependency on this decision.

## Evidence (2026-11-18, daf1200)

ADR-014 is now the house reference agent (20e5f13), so this proposal is renumbered again, to **ADR-015**. To stop this happening again, whoever writes it should claim the next free number in `docs/adr/README.md` first. The house agent registers a fresh key every time the process starts, so even with persistent storage it should look up its existing agent rather than register again. Otherwise every restart leaves an old Arena Bot identity in finished games.

## Promoted (2026-11-19, 2517b88)

Now `docs/adr/015-sqlite-persistence.md` (status Proposed, awaiting the project owner), linked from `issues/bdr-018-survive-server-restart.md` as `needs: human`. The ADR keeps this idea's core: pure-Go `modernc.org/sqlite`, a `game.Store` interface, one transaction per move with a version-guarded update. It adds `ARENA_DB_PATH` and compares five options. It already covers this idea's two remaining points: an `agents` table of key hashes with Arena Bot keeping its identity (decision 6), and a restart test on the same `ARENA_DB_PATH` that checks game JSON, history, results, idempotent retries and old keys (Confirmation). Nothing from this idea is left outside the ADR.

Rank 1 is released. The next ranks move up: Go client package, wait-for-turn, game fixtures.

## Decided (2026-11-20, 492da18)

The project owner accepted ADR-015 with **Option C, a standard-library JSON Lines journal** (`docs/adr/015-jsonl-journal-persistence.md`, configured by `ARENA_JOURNAL_PATH`), not the SQLite this idea recommended. The title above is kept as the original proposal. The parts of this idea that survive are the move log as the source of truth, the `game.Store` interface, persisting key hashes and Arena Bot's identity, and a real restart test. SQLite stays recorded as the upgrade path. `bdr-018-survive-server-restart` no longer needs a human, and the issue worker can implement it.

## Built (2026-11-20, 41a8494)

`internal/journal` implements ADR-015. The tests cover a real restart of the binary, torn and corrupt journals, a failed write, Arena Bot's identity and resume, and the old key. BDR-018 and BDR-033 issues are closed and await BDR verification. The one gap found so far is filed as `issues/house-game-pending-seat-lost-on-restart.md`: whether a game is a house game is not journaled. Nothing left here.
