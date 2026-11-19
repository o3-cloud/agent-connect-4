---
title: "BDR-025 not satisfied on main: Isolate Agent Failures"
status: closed
source: bdr-verification
bdr: BDR-025
branch: main
commit: 30cd082
found: 2026-11-18
closed: 2026-11-19
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

## Resolution

The Arena was implemented after this issue was filed, and BDR-025 was accepted in bd81d4a, but nothing pinned the behaviour and the issue was never closed. `TestAgentFailureIsIsolatedToItsGame` (tests/isolation_test.go) now follows Scenario 1:

- Agent A plays G1 over MCP until it is A's turn. Then A's MCP session closes, and a move request A started sending (headers and half the body) stalls on an open connection.
- While A hangs, C and D play G2 to a red win (AC-3), and B's attempt to move for A gets NOT_YOUR_TURN.
- G1's state is byte-for-byte unchanged while A hangs and after the stalled connection closes, and its history still has 2 moves (AC-1).
- A comes back with the same key, over REST, and moves from version 2 to 3 (AC-2).

It passed 3 times under `-race`. In a mutation where the cut-off move arrives whole, the test fails with "G1 changed while A hung". `make check` PASS, 91.5%.

Black-box against the real `bin/arena`: A's `connect4 watch` process was killed with `kill -9`, and a half-sent move was held open with `nc` for 3 s. Meanwhile C and D finished G2 (`completed, red wins (connect_four)`), and G1's `--json` state was identical before, during and after. A then moved with the same key (`Move 3: red in column 3`).
