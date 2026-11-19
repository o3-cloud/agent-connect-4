---
title: "BDR-028 not satisfied on main: Recover After a Browser Disconnect"
status: closed
source: bdr-verification
bdr: BDR-028
branch: main
commit: 30cd082
found: 2026-11-18
closed: 2026-11-19
fixed_in: 3e11fb6
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

## Resolution

The browser UI arrived after this issue was filed. Its game page already polled `/games/{id}/board` every second, which covers AC-1 and AC-3 once the connection returns. AC-2 had a real gap: a column control posted only the column, and the server moved against the *current* version, so a click on a stale board landed on a position the human had not seen.

Fixed:

- Each column control sends the version of the board it is drawn on (`hx-vals='{"version": N}'`). The web move handler moves only if that version is current, and the service re-checks it atomically, as `expectedVersion`. A missing or stale version moves nothing and returns the current board with "The board changed while you were away. This is the current position."
- When a board refresh fails (`htmx:sendError` or `htmx:responseError`), the page marks `#board` `data-connection="offline"`, shows "Reconnecting… moves are paused until the board is up to date." and disables the column controls. The next successful poll replaces `#board`, which clears this.

Tests (tests/reconnect_test.go), red before and green after: Scenario 1 (the page at v0, two moves through the API, a stale click moves nothing and returns v2 with the notice, the refreshed board matches the API, and a click on it moves), a move with no or a malformed version, and the page's offline handling. The existing browser tests now click with the shown board's version, which also makes the off-turn click test deterministic. `make check` PASS, 91.5%.

Verified in Chrome through a TCP proxy I could stop and start: with the proxy down, the page showed the reconnecting notice and 7/7 controls were disabled. When the proxy came back, polling resumed on its own, the notice cleared and the controls were enabled with `hx-vals` set. A click then played column 3, and Arena Bot replied.
