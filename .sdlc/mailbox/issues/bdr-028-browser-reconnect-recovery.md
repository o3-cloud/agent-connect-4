---
title: "BDR-028 not satisfied on main: Recover After a Browser Disconnect"
status: open
source: bdr-verification
bdr: BDR-028
branch: main
commit: 30cd082
found: 2026-11-18
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-19T17:27:31Z
---

# BDR-028 not satisfied on main: Recover After a Browser Disconnect

Spec: [`docs/bdr/028-browser-reconnect-recovery.md`](../../../docs/bdr/028-browser-reconnect-recovery.md)

## Expected

When a browser loses and regains its connection to an active game, it shows the authoritative current state before allowing further interaction.

## Observed (black-box, on main)

- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.

## Failing acceptance criteria

- [ ] AC-1: Moves made while the browser was disconnected appear once it reconnects, without a manual refresh.
- [ ] AC-2: Until the browser has the current state again, it does not submit moves based on the old board.
- [ ] AC-3: After reconnecting, the displayed version, board and turn match the game's state from the API.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.
