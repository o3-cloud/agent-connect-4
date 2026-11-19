---
title: "BDR-007 not satisfied on main: Enforce Turn Order and Player Membership"
status: closed
source: bdr-verification
bdr: BDR-007
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-007 not satisfied on main: Enforce Turn Order and Player Membership

Spec: [`docs/bdr/007-enforce-turn-and-membership.md`](../../../docs/bdr/007-enforce-turn-and-membership.md)

## Expected

Only the authenticated player who holds the current turn can make a move; moves from the opponent, from non-players or from unauthenticated callers are rejected without changing state.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.

## Failing acceptance criteria

- [x] AC-1: A move from the player who does not hold the turn is rejected with error code `NOT_YOUR_TURN`.
- [x] AC-2: A move from an authenticated client that is not a player in the game is rejected.
- [x] AC-3: A move without valid credentials is rejected.
- [x] AC-4: Including another player's identifier in the move body does not let the caller move as that player.
- [x] AC-5: After any of these rejections, the board, turn and version are unchanged.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Progress (eee9bc7, 2026-11-18)

Passing over HTTP: AC-1 (`NOT_YOUR_TURN` 409 with `currentVersion`), AC-2 (non-participant gets `FORBIDDEN` 403), AC-4 (a `playerId` in the body is rejected as an unknown field, so nobody can move as another player), AC-5 (state unchanged after each rejection). AC-3 is only partly met: a missing or non-Bearer `Authorization` header is rejected with `UNAUTHENTICATED` 401, but any non-empty bearer token is accepted as a test identity. It closes with BDR-013 (real credentials).

## Independent verification (eee9bc7, 2026-11-18, bdr-verification loop)

Confirmed the progress note above over HTTP against the running `bin/arena`. AC-1, AC-2, AC-4 and AC-5 pass, and a request with no `Authorization` header gets `401`. The BDR stays `Proposed`, because a made-up bearer token is still accepted as a player (AC-3 "without valid credentials"). It can move to `Accepted` once BDR-013 lands.

## Accepted (de4e8c8, 2026-11-18)

AC-3 now passes: `39f853d` replaced the any-token identity with issued API keys (ADR-013). Verified against a running `bin/arena` with curl: opponent out of turn `409 NOT_YOUR_TURN`, non-player `403 FORBIDDEN`, missing or made-up key `401 UNAUTHENTICATED`, a `playerId` in the body `400 INVALID_ARGUMENT`, and version, turn and board unchanged afterwards. BDR-007 is Accepted.
