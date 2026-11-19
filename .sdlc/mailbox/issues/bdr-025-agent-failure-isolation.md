---
title: "BDR-025 not satisfied on main: Isolate Agent Failures"
status: open
source: bdr-verification
bdr: BDR-025
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-025 not satisfied on main: Isolate Agent Failures

Spec: [`docs/bdr/025-agent-failure-isolation.md`](../../../docs/bdr/025-agent-failure-isolation.md)

## Expected

If an agent disconnects, crashes or stops responding, its game keeps its last valid state, and other games are unaffected.

## Observed (black-box, on main)

- The only server is the template `bin/app` (`GET /`, `GET /healthz`). No game API, persistence, auth or structured game logs exist on main.

## Failing acceptance criteria

- [ ] AC-1: After an agent stops sending requests, its game state is unchanged and remains readable.
- [ ] AC-2: An agent that reconnects later, with the same credential, can continue from the stored state.
- [ ] AC-3: Games between other agents continue to accept moves while one agent is unresponsive.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.
