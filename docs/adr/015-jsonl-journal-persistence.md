# ADR-015: Append-Only JSON Lines Journal as the Source of Truth

## Status

Accepted (2026-11-20, project owner)

Drafted on 2026-11-19 by the issue-worker loop (session 5b5d3e77) for BDR-018, recommending Option A. On 2026-11-20 the project owner chose Option C instead: a standard-library JSON Lines journal, with no new dependency.

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

Chosen option: **Option C, an append-only JSON Lines event journal written with the standard library**. The project owner chose it over Option A. It adds no dependency (ADR-007), keeps the CGO-free `scratch` image, and makes the move log the source of truth literally: state on start is whatever replaying the journal through `game.Service` produces. The cost is that we own the crash safety a database would otherwise give us. The implementation and its tests must cover that explicitly. SQLite (Option A) is still the upgrade path if the journal outgrows itself.

The implementation:

1. **Extracts a `game.Store` interface** from `game.Service`. The current maps become `memstore`, used by tests and as the default when no journal is configured. The service keeps the rules and the per-game locking, and the store keeps the data. Every existing test must still pass against `memstore`.
2. **Adds `journalstore`** behind the same interface. It appends one JSON object per line to a single file. Each object has the fields `seq`, `at`, `type` and `data`. The event types are `game_created`, `player_joined`, `move_accepted` (which carries the idempotency key and the recorded result), `player_resigned`, `game_finished` and `agent_registered`. `agent_registered` stores only the key hash, never the key (SEC-005).
3. **Makes each move one event, so it commits atomically**: the move, its version bump and its idempotency record are one line. `game.Service` checks `expectedVersion` under the per-game lock before appending, so a stale move is never written (DATA-003, TURN-006). The append is followed by `fsync`, and the HTTP response is sent only after `fsync` returns.
4. **Restores by replaying the journal through the engine** on start, using the same code path as live play (DATA-004..006). If the last line is torn (no trailing newline, or invalid JSON), it is truncated away and the truncation is logged. Corruption anywhere else stops start-up with an error naming the line. The server never silently skips an event.
5. **Is configured by `ARENA_JOURNAL_PATH`**. Leaving it unset keeps the memory-only behaviour for development and tests. The compose file and README mount a volume for it (NFR-008, BDR-033 AC-2).
6. **Keeps Arena Bot's identity**: on start the house agent looks up its replayed `agent_registered` event before registering, so a restart doesn't create a new "Arena Bot" identity on every boot (backlog note, daf1200).
7. **Defers compaction**. At current scale the journal is replayed in full on every start. If start-up time becomes a problem, snapshotting will be a later ADR.

### Implementation notes (2026-11-20, issue-worker loop, session 5b5d3e77)

How the code meets steps 1 to 7, where it differs from their wording:

- **Store shape (step 1).** `game.Service` keeps its live state in memory in both modes, as before. `game.Store` is the interface the service journals through: `Append(type, at, data)`, which returns once the event is on disk. Without `ARENA_JOURNAL_PATH` there is no store, which is the memory-only mode the ADR calls `memstore`. Replay goes through `game.Service.Restore` and `identity.Registry.Restore`. `internal/journal` is the `journalstore`.
- **Atomic moves (step 3).** Under the game's lock, the service checks the move, drops the disc on a copy of the board, appends `move_accepted` (column, `expectedVersion`, idempotency key and the resulting `version`) with fsync, and only then applies the copy. A failed write returns `INTERNAL` and changes nothing.
- **Group commit (2026-11-20).** One fsync per append, taken in turn by every game, put a 100-game load at a 535 ms move p95 on macOS, where Go's `Sync` is `F_FULLFSYNC`. Appends now write their line under the journal's lock, and one fsync covers every line written so far. Each `Append` still returns only once its own line is on disk. If an fsync fails, the lines it did not cover are cut off, their appends fail, and the journal refuses further writes, because after a failed fsync the kernel may have dropped those pages. Measured p95 with group commit: 23 ms (`TestJournaledConcurrentGamesMeetMoveLatency`).
- **No `game_finished` event.** A finished game is fully determined by its last `move_accepted` or `player_resigned`, so replay derives it, and the `version` check on every replayed move catches any divergence.
- **Arena Bot (step 6).** The journal holds key hashes only, so the bot cannot reuse its old key. `agent_registered` carries `"house": true` for the bot, and on its first use after a start the registry issues it a new key for the same agent (`agent_key_reissued`). The arena then resumes the bot in every in-progress game it plays.
- **Not restored.** A house game that was created but never joined before a restart loses its pending Arena Bot seat. It expires after 10 minutes anyway. Tracked in `house-game-pending-seat-lost-on-restart.md`.

## Consequences

- Positive: BDR-018 and AT-010 become testable. A test can stop and restart the server on the same journal file and compare `GET /api/v1/games/{id}` before and after, then move with the old `expectedVersion`.
- Positive: agents keep their API keys across deploys.
- Positive: BDR-033 AC-2 (persistence location set by configuration) follows directly.
- Positive: no new dependency, and the journal doubles as an audit trail of every game.
- Negative: crash safety (fsync on every append, recovery from a torn last line) is our code, and its tests are our responsibility.
- Negative: a later move to SQL or PostgreSQL means writing an importer for the journal. A driver swap won't do it.
- Negative: the store interface adds one layer between the service and its data, and the race tests must cover both stores.
- Neutral: memory-only remains the default for `go test` and `make run` without `ARENA_JOURNAL_PATH`, so the developer experience doesn't change unless persistence is wanted.

## Verification

- A restart test (BDR-018 Scenario 1, AT-010) runs the arena on a temporary `ARENA_JOURNAL_PATH`, plays to version 5, stops and restarts it, and checks that the game JSON, history and a completed game's result are identical. It then plays the move at `expectedVersion` 5.
- The existing concurrency test (one move commits out of many concurrent submissions with the same `expectedVersion`) runs against `journalstore` under `-race`.
- A journal whose last line is cut off mid-object restores every complete event, logs the truncation, and accepts the next move. A journal corrupted mid-file refuses to start.
- An idempotent retry after a restart returns the recorded result rather than playing again (BDR-012).
- A key issued before the restart still authenticates after it.
