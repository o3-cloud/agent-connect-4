---
title: "Moves rejected before the game service are not logged as move_rejected"
status: open
priority: 3
source: loop-run
bdr: BDR-032
branch: main
commit: d30a814
found: 2026-11-19
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
