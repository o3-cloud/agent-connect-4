---
title: "Agent onboarding: /llms.txt, /agents quickstart and an MCP how-to-play prompt"
status: proposed
rank: 9
type: feature
outcomes: [1, 6]
spec: [NFR-005, MCP-001]
bdr: [BDR-022, BDR-030]
created: 2026-11-18
---

# Agent onboarding and discovery

## Problem

Outcome 1 says an agent can find out how to play. OpenAPI (NFR-005) describes the endpoints but not the game: the rules, the turn loop, when to retry. An LLM agent pointed at the server URL has no natural starting point.

## Proposal

- `GET /llms.txt` (plain markdown): what the arena is, the rules in 10 lines, the turn loop, how to get credentials, and links to OpenAPI, the CLI install and the MCP endpoint.
- `GET /agents`: an HTML quickstart page with copy-paste snippets for curl, the CLI, Python and an MCP client config (`claude mcp add ...`). The server fills in its own base URL and an optional freshly issued key.
- MCP: a `how_to_play` prompt and a `connect4://rules` resource, so MCP clients can read the rules without leaving the protocol.
- Every structured error carries a `hint` field, for example `NOT_YOUR_TURN` → "call wait_for_turn".

## Acceptance

- [ ] An LLM agent given only the base URL and an API key completes a game (scripted eval with a Claude model through MCP and through REST).
- [ ] The `/llms.txt` content is generated from the same source as the OpenAPI descriptions, so the two don't drift apart.

## Ranked (2026-11-18): 9 of 16

Outcome 1: agents find out how to play.

## Evidence (2026-11-19, 43322c5)

BDR-022 is accepted. The OpenAPI `info.description` now walks through the whole flow (register, send the key as a Bearer token, create, join, poll `getGame`, `submitMove` with `expectedVersion`, read the result), says when to retry, and points to `/mcp` and the CLI. That covers this idea's core point that OpenAPI alone isn't enough, at least for REST clients.

Still missing on main, and the reason this idea stays open:
- `/llms.txt`: a plain-markdown entry point for an LLM agent that has only the base URL. A raw OpenAPI document is a lot to read first.
- An `/agents` quickstart page with copy-paste curl, CLI and MCP-config snippets.
- An MCP `how_to_play` prompt and a `connect4://rules` resource. Today an MCP-only client sees tool descriptions but not the flow that the OpenAPI description now explains.
- A `hint` on errors.

The flow tells agents to "poll getGame". If `idea-wait-for-turn` is adopted, this text is one of the places to update.

