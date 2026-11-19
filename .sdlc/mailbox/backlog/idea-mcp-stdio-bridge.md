---
title: "`connect4 mcp`: stdio MCP bridge for one-line setup in desktop agents"
status: promoted
type: feature
outcomes: [1, 6]
spec: [MCP-001..009, AGENT-002, SEC-005]
bdr: [BDR-021, BDR-013]
created: 2026-11-18
---

# `connect4 mcp`: stdio MCP bridge

## Problem

ADR-009 puts the MCP server inside the arena process, which means it runs over HTTP. Many MCP clients (Claude Code, Claude Desktop, IDE agents) are easiest to set up with a stdio command, and remote-HTTP MCP with auth headers is still fiddly in some of them. Each extra setup step works against "a developer can connect an agent in minutes".

## Proposal

- Add `connect4 mcp` to the CLI. It runs an MCP server over stdio that exposes the same six tools (and `wait_for_turn` if adopted). Each tool is a thin call to the REST API with the saved credentials from `connect4 login`.
- One-line setup: `claude mcp add connect4 -- connect4 mcp --server https://arena.example`.
- Tool names, schemas and results are generated from the same definitions as the in-server MCP, so the two can't drift apart (MCP-009). The cross-interface contract suite runs against both.
- The key stays in the local credentials file and is never sent as a tool argument, so it never lands in the LLM's context (SEC-005).

## Acceptance

- [ ] A Claude Code session set up with only that one command completes a game against the house agent.
- [ ] `tools/list` output from the stdio bridge matches the in-server MCP byte for byte (golden test).
- [ ] No tool input schema contains a credential field.

## Ranked (2026-11-18): 10 of 16

One-line MCP setup; depends on the client package.

## Promoted (2026-11-18)

Now tracked as `issues/mcp-stdio-transport.md`, filed after e517837 shipped the Streamable HTTP `/mcp` endpoint. The issue keeps this idea's design: `connect4 mcp` proxies to one authoritative arena. Two acceptance points from this idea are worth carrying into it: a golden test that `tools/list` from stdio matches `/mcp` byte for byte, and no credential field in any tool input schema.
