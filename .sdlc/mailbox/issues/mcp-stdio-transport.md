---
title: "MCP is only reachable over HTTP; no stdio transport (connect4 mcp)"
status: open
priority: 3
source: loop-run
type: feature
branch: main
commit: e517837
found: 2026-11-18
spec: [ADR-009, SPEC.md MCP-001]
outcomes: [1, 6]
---

# Add a stdio MCP transport

ADR-009 decision item 3: "Offer a stdio transport through the CLI (for example `connect4 mcp`) for agents that launch MCP servers as subprocesses." Main has only the Streamable HTTP endpoint at `/mcp` (e517837).

## Expected

`connect4 mcp` speaks MCP on stdin/stdout and proxies each tool call to the arena server named by `$CONNECT4_SERVER`, authenticated as `$CONNECT4_API_KEY`. That way one arena stays authoritative (MCP-009). It must not start a private in-memory game service.

## Notes

- The simplest faithful design: the stdio server registers the same six tools, and each handler calls the remote `/mcp` endpoint or the REST API. Either way, results and §16 error objects must be identical to `/mcp`.
- The CLI from 9e6ef2b already has the server URL and key plumbing.
