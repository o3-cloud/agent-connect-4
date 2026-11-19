---
title: "BDR-012 not satisfied on main: Idempotent Move Retries"
status: closed
source: bdr-verification
bdr: BDR-012
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-012 not satisfied on main: Idempotent Move Retries

Spec: [`docs/bdr/012-idempotent-move-retry.md`](../../../docs/bdr/012-idempotent-move-retry.md)

## Expected

Retrying a move with the same idempotency key returns the originally recorded result and never places a second disc.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [ ] AC-1: A second request from the same player in the same game with the same idempotency key returns the same move result (`moveNumber`, `column`, `row`, `disc`) as the first.
- [ ] AC-2: The retry does not place a disc, change the turn or increase the version.
- [ ] AC-3: The retry succeeds even if the game state has changed since the original request.
- [ ] AC-4: The same key used by a different player, or in a different game, is not treated as a retry.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution (584db77, 2026-11-18)

Fixed on main by 584db77 `feat(game): replay moves retried with the same idempotency key`. It adds `game.Service.MoveWithKey`, and `POST /api/v1/games/{gameId}/moves` now enforces `idempotencyKey`.

- **AC-1:** resending a key returns the recorded result field for field, including the original timestamp.
- **AC-2:** no disc is placed, and the turn and version are unchanged.
- **AC-3:** the replay works after the game has moved on, and even if the retry's `expectedVersion` is stale.
- **AC-4:** keys are scoped to one player in one game.
- Only accepted moves are recorded, so a rejected move retried with its key is evaluated afresh. An outsider using someone's key still gets `403 FORBIDDEN`. 16 concurrent retries with one key place exactly one disc.

Verification:
- `make check` PASS (95.7%), with new tests in `tests/idempotency_test.go`.
- A black-box run against `bin/arena` replayed scenario 1: two identical requests got identical 200 responses (move 1, column 3, row 5), the game stayed at version 1 with one move, and a keyless repeat got `409 NOT_YOUR_TURN`.

Not covered:
- The MCP `make_move` `idempotency_key` argument is part of BDR-021.
- Replay records live in memory and will need persisting with BDR-018.
