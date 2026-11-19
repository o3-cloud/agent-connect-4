---
title: "BDR-014 not satisfied on main: Rate-Limit Excess Requests Per Client"
status: closed
source: bdr-verification
bdr: BDR-014
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-014 not satisfied on main: Rate-Limit Excess Requests Per Client

Spec: [`docs/bdr/014-rate-limit-clients.md`](../../../docs/bdr/014-rate-limit-clients.md)

## Expected

A client that exceeds the configured request limit gets rate-limit errors for the excess requests, and other clients and games are unaffected.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.

## Failing acceptance criteria

- [ ] AC-1: With the request limit configured to N per window, the first N requests in a window are processed normally.
- [ ] AC-2: Requests beyond N in the same window receive a structured rate-limit error and do not change game state.
- [ ] AC-3: While one client is being rate-limited, another client's requests and games proceed normally.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution (d9b031c, 2026-11-19)

Fixed on main by d9b031c `feat(api): rate-limit API and MCP requests per client`. It adds `internal/ratelimit`, a fixed-window counter per client, applied to `/api/v1/*` and `/mcp`.

- **AC-1:** the first N requests in a window succeed, and the window resets after it elapses.
- **AC-2:** excess requests get `429 RATE_LIMITED` (§16) with `retryable: true`, `retryAfterMs` and `Retry-After`. They never reach a handler, so a limited move leaves the game unchanged.
- **AC-3:** a request with a valid API key spends its agent's budget. Any other request, including a forged key, spends its IP's budget, so rotating made-up keys does not evade the limit. Other agents are unaffected.
- Not limited: `/healthz`, the browser UI and Arena Bot, which calls the mux in-process.
- Default: 600 requests per minute per client. `bin/arena` reads `ARENA_RATE_LIMIT` and `ARENA_RATE_WINDOW`, and exits 2 on an invalid value.

Verification:
- `make check` PASS (90.8%), with new tests in `tests/ratelimit_test.go`.
- Black-box run of scenario 1 against `ARENA_RATE_LIMIT=5 ARENA_RATE_WINDOW=10s bin/arena`: agent A's 10 state requests got 3 × 200 then 7 × 429 (`Retry-After: 10`, `retryAfterMs` 9810). Agent B's move in its own game was accepted in the same window, and A's game stayed at version 0. `ARENA_RATE_LIMIT=abc` exits 2 with a clear message.

Limitations:
- The limiter counts per process. Running several arena instances behind a load balancer would need a shared store.
- `X-Forwarded-For` is deliberately not trusted, so behind a proxy all anonymous requests share the proxy's IP budget.
