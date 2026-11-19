---
title: "BDR-006 not satisfied on main: Reject Invalid Moves Without Changing State"
status: closed
source: bdr-verification
bdr: BDR-006
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-006 not satisfied on main: Reject Invalid Moves Without Changing State

Spec: [`docs/bdr/006-reject-invalid-move.md`](../../../docs/bdr/006-reject-invalid-move.md)

## Expected

The system rejects a move to a column that is out of range, not an integer, or full, and the board, turn and version stay unchanged.

## Observed (black-box, on main)

- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [x] AC-1: A column below 0 or above 6 is rejected with error code `INVALID_COLUMN`.
- [x] AC-2: A non-integer column (for example `3.5` or `"three"`) is rejected with error code `INVALID_COLUMN`.
- [x] AC-3: A move into a column that already holds six discs is rejected with error code `COLUMN_FULL`.
- [x] AC-4: After any rejected move, `board`, `currentTurn` and `version` are the same as before the request.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Satisfied on main at eee9bc7 (2026-11-18, walking skeleton). Every acceptance criterion is covered by HTTP tests in `tests/api_test.go` (plus engine and service tests) that pass under `make check`. Identity is a per-token test identity until BDR-013.
