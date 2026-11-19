---
title: "examples/: Python REST agent, shell CLI agent and MCP agent config"
status: proposed
rank: 11
type: docs
outcomes: [1, 3, 6]
spec: [SPEC §19 examples/, SPEC §20 Phase 2 "Reference agents"]
bdr: [BDR-024]
created: 2026-11-18
---

# Reference agent examples

## Problem

SPEC §19 and Phase 2 call for reference agents, and no BDR tracks them. Developers learn fastest by copying a working agent.

## Proposal

`examples/` with agents that each fit in one short file and use only public interfaces:

- `python-rest-agent/agent.py`: standard library only (`urllib`), with the turn loop, a random or greedy strategy, and retry on `STALE_STATE`.
- `shell-cli-agent/agent.sh`: about 15 lines using `connect4 wait` and `connect4 move --json` with `jq`.
- `mcp-agent/`: `.mcp.json` for Claude Code and other MCP clients, plus a system prompt that tells an LLM to play a full game.
- `README.md`: "Your first agent in 5 minutes".

CI runs `python-rest-agent` against `shell-cli-agent` to a finished game, which tests the examples and AT-002 together.

## Acceptance

- [ ] Each example runs against `docker compose up` with only the steps in its README.
- [ ] A CI job plays a full cross-interface game between two examples.

## Ranked (2026-11-18): 11 of 16

Spec Phase 2; mostly falls out of ranks 4, 6 and 8.

## Evidence (2026-11-19, 7e2e0a3)

BDR-024 scenario 1 is now tested (`tests/agent_match_test.go`): a CLI agent and an MCP agent each run in a goroutine and play a full game, reading only the game state. That proves the platform supports agents playing on their own. But those agents exist only as test code, and there is still no `examples/`. A developer looking for "my first agent" has nothing to copy. The test's agent loops are a good starting point for `examples/shell-cli-agent` and `examples/mcp-agent`, provided they are rewritten to run against a live server rather than an in-process handler.
