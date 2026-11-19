# ADR-015: SQLite (Pure Go) With the Move Log as the Source of Truth

## Status

Proposed

Drafted on 2026-11-19 by the issue-worker loop (session 5b5d3e77) for BDR-018. It needs the project owner's acceptance: no loop has been asked to decide persistence on the owner's behalf.

## Context and Problem Statement

Everything the arena knows lives in process memory:

- games, boards and move history (`game.Service`, a map of `record`s);
- idempotency results for retried moves (`record.replays`, API-008);
- issued API keys, stored as SHA-256 hashes (`identity.Registry`);
- Arena Bot's identity, which it registers again every time the process starts (ADR-014).

A restart loses all of it: every game, every result, and every agent's credentials. That breaks DATA-001, DATA-002, DATA-003 and DATA-006, AT-010, and BDR-018 (games survive a restart). It also blocks BDR-033 AC-2, a persistence location set by configuration (NFR-008). ADRs 008–012 replaced the SPEC §19 stack for HTTP, MCP, CLI, UI and CSS, but none of them covers storage. SPEC §19 names SQLite as the initial database, with PostgreSQL later.

How should the arena persist games, moves, idempotency results and credentials?

## Decision Drivers

- **Atomic moves (DATA-003, TURN-004, TURN-006)**: a move, its game-state change and its idempotency record commit together or not at all, and a stale `expectedVersion` can never commit.
- **Deterministic restore (DATA-005, DATA-006)**: state after a restart is exactly the state before it, and replay uses the same code path as live play.
- **Static, CGO-free builds**: the image is `scratch` with `CGO_ENABLED=0` (ADR-004, `delivery/docker.md`).
- **Configuration from outside the image (NFR-008, BDR-033)**: the data location is an environment variable, and the container keeps it on a volume.
- **Dependency policy (ADR-007, `dependency-management.md`)**: every new module is pinned, reviewed and scanned, so fewer and better-maintained dependencies are better.
- **Follow SPEC §19** unless there is a reason not to, and keep PostgreSQL possible later.
- **Not optimizing for scale yet** (OUTCOMES.md): BDR-031's load targets are met in memory today.

## Considered Options

- **Option A**: SQLite through `modernc.org/sqlite` (pure Go)
- **Option B**: SQLite through `github.com/mattn/go-sqlite3` (cgo)
- **Option C**: An append-only JSON Lines journal of events, written with the standard library and replayed on start
- **Option D**: bbolt, an embedded key-value store
- **Option E**: PostgreSQL from day one

## Pros and Cons of the Options

### Option A: SQLite, pure Go (`modernc.org/sqlite`)

- Good: what SPEC §19 asks for. Real transactions give DATA-003 directly: insert the move, `UPDATE games SET version = version + 1 WHERE id = ? AND version = ?`, and record the idempotency result, all in one transaction.
- Good: no cgo, so the static `scratch` image and cross-compilation stay as they are.
- Good: SQL keeps PostgreSQL a small step away, behind the same `Store` interface.
- Good: WAL mode allows concurrent readers alongside the single writer, which suits a game server's read-heavy load.
- Bad: a large module graph (`modernc.org/libc` and friends, transpiled C). That is more to pin, review and scan, and the binary grows by several MB.
- Bad: slower than cgo SQLite. That doesn't matter at this scale.

### Option B: SQLite with cgo (`mattn/go-sqlite3`)

- Good: the reference SQLite, fast and widely used.
- Bad: needs cgo and a C toolchain in the builder. That breaks `CGO_ENABLED=0`, complicates the `scratch` image and cross-compilation, and slows CI.

### Option C: JSON Lines event journal (standard library)

- Good: no new dependencies. Events (`game_created`, `player_joined`, `move_accepted`, `player_resigned`, `agent_registered`) are appended with `fsync` and replayed through `game.Service` on start, so restore uses the same code as live play.
- Good: matches the "move log is the source of truth" idea, and the journal doubles as an audit trail.
- Bad: we would hand-write the parts a database provides: atomic multi-record writes (one event per line, with torn last lines detected and dropped), compaction, a crash-safe file format and tests for all of it. That is code we own and must get right.
- Bad: further from SPEC §19, and a later move to PostgreSQL is a rewrite rather than a driver swap.

### Option D: bbolt

- Good: pure Go, small, transactional.
- Bad: key-value only. Listing by status, history and replay need indexes we would maintain by hand, and there is no SQL path to PostgreSQL.

### Option E: PostgreSQL now

- Good: the eventual production database in SPEC §19.
- Bad: a second service to run for every developer, test and container, which works against Outcome 6 ("a developer can connect an agent in minutes"). It also breaks `make run` as a single binary.

## Decision Outcome

Recommended option: **Option A, SQLite through `modernc.org/sqlite`**, because it is what SPEC §19 specifies. It gives DATA-003's single transaction without hand-written crash safety, and it keeps the CGO-free `scratch` image. Its cost is a larger dependency graph, which ADR-007's pinning and `govulncheck` gate already cover. Option C is the fallback if the owner wants no new dependencies.

If accepted, the implementation:

1. **Extracts a `game.Store` interface** from `game.Service`. The current maps become `memstore`, used by tests and as the default when no database is configured. The service keeps the rules and the per-game locking, and the store keeps the data. Every existing test must still pass against `memstore`.
2. **Adds `sqlstore`** behind the same interface, with the tables `games` (configuration, players, status, version, result, timestamps), `moves` (`game_id`, `move_number`, `player_id`, `column`, `row`, `created_at`, primary key `(game_id, move_number)`), `idempotency` (`game_id`, `player_id`, `key`, `result_json`) and `agents` (`agent_id`, `key_hash`, `display_name`, `type`, `created_at`). Only key hashes are stored, never keys (SEC-005).
3. **Commits each move in one transaction**: insert into `moves`, run the version-guarded `UPDATE games`, and insert into `idempotency`. A guarded update that changes zero rows is `STALE_STATE` (DATA-003, TURN-006).
4. **Derives boards by replaying `moves` through the engine** on load and caches them in memory. Restore (DATA-006) and replay (DATA-004, DATA-005) then share the live code path.
5. **Is configured by `ARENA_DB_PATH`**: unset keeps memory-only behaviour for development and tests, and setting it enables SQLite in WAL mode. The compose file and README mount a volume for it (NFR-008, BDR-033 AC-2).
6. **Keeps Arena Bot's identity**: the house agent looks up its existing agent before registering, so a restart doesn't leave a new "Arena Bot" identity for every boot (backlog note, daf1200).

## Consequences

- Positive: BDR-018 and AT-010 become testable. A test can stop and restart the server on the same database file and compare `GET /api/v1/games/{id}` before and after, then move with the old `expectedVersion`.
- Positive: agents keep their API keys across deploys.
- Positive: BDR-033 AC-2 (persistence location set by configuration) follows directly.
- Negative: a new direct dependency, with a large transitive graph, to pin and keep current. Renovate and the CI `govulncheck` gate cover it.
- Negative: the store interface adds one layer between the service and its data, and the race tests must cover both stores.
- Neutral: memory-only remains the default for `go test` and `make run` without `ARENA_DB_PATH`, so the developer experience doesn't change unless persistence is wanted.

## Verification

- A restart test (BDR-018 Scenario 1, AT-010) runs the arena on a temporary `ARENA_DB_PATH`, plays to version 5, stops and restarts it, and checks that the game JSON, history and a completed game's result are identical. It then plays the move at `expectedVersion` 5.
- The existing concurrency test (one move commits out of many concurrent submissions with the same `expectedVersion`) runs against `sqlstore` under `-race`.
- An idempotent retry after a restart returns the recorded result rather than playing again (BDR-012).
- A key issued before the restart still authenticates after it.
