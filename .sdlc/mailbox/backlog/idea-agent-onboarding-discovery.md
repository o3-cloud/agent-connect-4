---
title: "Agent onboarding: /llms.txt, /agents quickstart and an MCP how-to-play prompt"
status: proposed
rank: 1
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


## Evidence (2026-11-19, abf73dc)

`connect4 mcp` has shipped: a stdio MCP server that proxies to `/mcp` using `CONNECT4_API_KEY`, and `TestCLIServesMCPOverStdio` checks its `tools/list` against `/mcp`. Desktop and IDE agents now have a one-command setup, and the README shows the subprocess config. Two things follow for this idea:

- The `/agents` page's MCP snippet should be the stdio form (`claude mcp add connect4 --env CONNECT4_API_KEY=… -- connect4 mcp --server <base>`), filled in with the server's own URL. Today that snippet lives only in the README, which a remote agent pointed at a base URL cannot see.
- The `how_to_play` prompt and `connect4://rules` resource are still missing. Because the stdio bridge offers exactly the tools the arena lists, adding them to `/mcp` makes them reach stdio clients too, but only if the bridge forwards prompts and resources as well as tools. Check that when this is built.

Rank stays 9.

## Ranked (2026-11-20, 0cc40e1): 1 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. Outcome 1 starts with "an agent can *discover* how to play" and outcome 6 says "connect an agent in minutes". An agent given only a base URL still has no `/llms.txt`, `/agents` page or MCP how-to-play prompt. This is now the biggest gap between the accepted BDRs and the release criteria.
