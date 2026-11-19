---
title: "A browser click is applied to the latest board, not the board the player saw (BDR-028 AC-2)"
status: closed
source: bdr-verification
type: defect
bdr: BDR-028
branch: main
commit: 00884db
found: 2026-11-19
closed: 2026-11-19
fixed_in: 3e11fb6
spec: [SPEC.md TURN-*, LIVE-005]
---

# A browser click is applied to the latest board, not the board the player saw

Related: [bdr-028-browser-reconnect-recovery](bdr-028-browser-reconnect-recovery.md), which is claimed by session 5b5d3e77. I filed this separately so I didn't edit a claimed item.

## Observed

I re-verified BDR-028 in Chrome. I made the page's XHRs fail (offline), made two moves through the API (v4 → v6), then restored the XHRs.

- **AC-1: pass.** With no reload, the page caught up to v6 within 0.7 s, showing both missed moves.
- **AC-3: pass.** The page's `data-version`, board and turn then matched the API.
- **AC-2: fail.**
  - While offline, the column buttons stay active, and nothing on the page says the connection is lost.
  - A click sends no version. `internal/web/web.go` `move` calls `service.Move(id, agent, column, g.Version)` with the server's *current* version, so a click chosen from a stale board is applied to the new board.

## Reproduce

This uses the same request a button click sends, with the player's key as the `c4_session` cookie:

```sh
# human red + agent yellow; play 0,6,0,6 -> v4, red to move; the page renders data-version="4"
# while the page is stale, the API plays red 3, yellow 0 -> v6 (red to move again)
curl -XPOST localhost:8080/games/$G/moves/1 -b "c4_session=$KEY" -H 'HX-Request: true'
# -> 200; history gains move 7: red column 1, placed on a v6 board the player never saw
```

## Expected

A move picked from an out-of-date board is not applied. The page shows the current board first (BDR-028 AC-2, scenario 1: "before the human can submit a move").

## Fix

1. Render the version into each column button, for example `hx-vals='{"expectedVersion": {{$.Version}}}'`, and have `move` pass it to `service.Move`. A stale click then gets `STALE_STATE`, and the handler re-renders the current board.
2. Optionally, disable the buttons on `htmx:sendError` or `offline`, and re-enable them after the next successful board swap.
3. Add a test: render at v4, advance to v6 through the service, POST a click carrying v4, and expect no new move and the v6 board.

## Resolution

Fixed in 3e11fb6 by session 5b5d3e77, under the claimed item [bdr-028-browser-reconnect-recovery](bdr-028-browser-reconnect-recovery.md). That work was in progress when this was filed, and it does fix options 1 to 3:

1. Each column button carries `hx-vals='{"version": N}'` (the field is `version`, a form value on the browser route). `move` moves only when N is current and passes N to `service.Move`, which re-checks it atomically. A stale or missing version moves nothing and re-renders the current board with "The board changed while you were away. This is the current position."
2. On `htmx:sendError` or `htmx:responseError`, the page shows "Reconnecting… moves are paused until the board is up to date." and disables the column buttons. The next successful board swap clears both. I verified this in Chrome by stopping and restarting a proxy in front of the arena.
3. `TestBrowserDoesNotMoveFromAStaleBoard` renders v0, advances to v2 through the API, clicks with v0, and expects no move and the v2 board with the notice.
