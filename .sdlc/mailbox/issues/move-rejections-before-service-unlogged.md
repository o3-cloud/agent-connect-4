---
title: "Moves rejected before the game service are not logged as move_rejected"
status: closed
priority: 3
source: loop-run
bdr: BDR-032
branch: main
commit: d30a814
found: 2026-11-19
closed: 2026-11-19
fixed_in: 9fe1c57
---

# Moves rejected before the game service are not logged as move_rejected

BDR-032 AC-1 asks for a record for every rejected move. Since d30a814, `move_rejected` is written by the game service, so it covers every rule rejection (INVALID_COLUMN, COLUMN_FULL, NOT_YOUR_TURN, STALE_STATE, FORBIDDEN, GAME_NOT_FOUND, ...).

A move request that never reaches the service leaves no record:

- `UNAUTHENTICATED` (missing or unknown key, REST or MCP)
- `INVALID_ARGUMENT` from request validation (for example, a missing `column` or `expectedVersion`)
- `RATE_LIMITED`

## Suggested fix

Log `move_rejected` with `code` and `gameId` for these too, at the REST move handler's error path, the MCP `make_move` schema check, and the rate limiter. Never log the presented key. Do not log the same request twice: the service already logs rule rejections.

## Reproduce

```sh
make build && PORT=18133 ./bin/arena &
curl -s -H 'Content-Type: application/json' -d '{"column":3,"expectedVersion":0}' http://127.0.0.1:18133/api/v1/games/game_x/moves
# 401 UNAUTHENTICATED; stdout has no move_rejected record
```

## Resolution

Every move request refused before the game service now leaves exactly one `move_rejected` record with `gameId` and `code`, plus `playerId` when the key is valid. The presented key is never logged. `game.Service.RecordRejectedMove` owns the record format, and each edge site calls it where the refusal happens, so nothing is logged twice:

- REST `UNAUTHENTICATED`: the `submitMove` handler, when the key is missing or unknown.
- REST validation (`INVALID_ARGUMENT`, and `INVALID_COLUMN`/`GAME_NOT_FOUND` for a non-integer column): `newHumaErrorWithContext`, for the `submitMove` operation.
- MCP `make_move` failing its argument checks: a reject hook in `addTool`.
- MCP `make_move` without a valid key: the `/mcp` handler, which peeks at the JSON-RPC body (`mcpserver.MoveGameID`) before the bearer check.
- `RATE_LIMITED` (REST and MCP): the rate limiter.

Tests (red before, green after): `TestLogRecordsMovesRejectedBeforeTheService`, `TestLogRecordsRateLimitedMoves` and `TestLogRecordsMCPMovesRejectedBeforeTheService`. They assert exact counts, so a double record fails. Mutations removing the rate-limit and MCP-auth records were caught. `make check` PASS, 91.5%.

Black-box against the real `bin/arena` (`ARENA_RATE_LIMIT=8`): the issue's repro, a missing field, an MCP call with no key and two rate-limited moves each logged once (11 refused requests, 11 records). The API key appeared 0 times in the log.
