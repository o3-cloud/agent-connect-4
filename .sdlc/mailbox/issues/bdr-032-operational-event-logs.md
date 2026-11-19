---
title: "BDR-032 not satisfied on main: Structured Operational Logs"
status: closed
source: bdr-verification
bdr: BDR-032
branch: main
commit: 30cd082
found: 2026-11-18
closed: 2026-11-19
fixed_in: d30a814
---

# BDR-032 not satisfied on main: Structured Operational Logs

Spec: [`docs/bdr/032-operational-event-logs.md`](../../../docs/bdr/032-operational-event-logs.md)

## Expected

The server emits a structured log record for every game creation, player join, accepted move, rejected move, game completion and unexpected server error.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [ ] AC-1: Each listed event produces exactly one machine-parseable log record (for example JSON lines on the server's standard output) that names the event type and game identifier.
- [ ] AC-2: Rejected-move records include the error code.
- [ ] AC-3: No log record contains a credential value.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Fixed on main in d30a814 (`feat(api): log game events and server errors as JSON lines`).

- The game service writes `game_created`, `player_joined`, `move_accepted`, `move_rejected` (with `code`), `move_replayed`, `player_resigned` and `game_completed`, each with `gameId`. REST, CLI, MCP, the browser UI and Arena Bot all go through it, so each event is logged exactly once.
- New `internal/oplog` middleware: `server_error` for any 5xx response or panic. A panic now returns the §16 INTERNAL error rather than dropping the connection. `http.ErrAbortHandler` passes through.
- `bin/arena` writes the records as JSON lines on stdout. Records carry `agt_` IDs, never keys or headers.
- Tests: `tests/oplog_test.go` (scenario 1, win, resignation, rejection codes, idempotent replay, MCP, panic, 5xx, abort). `make check` passes, 91.1% coverage.
- Black-box run on the real `bin/arena`: create, two joins, column 3, column 9. The log has 1 `game_created`, 2 `player_joined`, 1 `move_accepted` and 1 `move_rejected` (`INVALID_COLUMN`), all for the game's ID. Every line parses as JSON, and neither key appears in the log.

Follow-up: `move-rejections-before-service-unlogged.md` (moves rejected by auth, request validation or the rate limiter are not logged as `move_rejected`).
