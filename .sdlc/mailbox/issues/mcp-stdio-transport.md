---
title: "MCP is only reachable over HTTP; no stdio transport (connect4 mcp)"
status: closed
priority: 3
source: loop-run
type: feature
branch: main
commit: e517837
found: 2026-11-18
closed: 2026-11-19
fixed_in: abf73dc
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

## Resolution

`connect4 mcp` (internal/cli/mcp.go) serves MCP on stdin and stdout as a proxy. On start it connects to `$CONNECT4_SERVER/mcp` as `$CONNECT4_API_KEY`, registers exactly the tools the arena lists (same names, descriptions and input schemas), and forwards each call unchanged. Results and §16 error objects are the arena's own, and no private game service exists (MCP-009). It exits 0 when stdin closes. With no key it fails at once with UNAUTHENTICATED (exit 4), and the same for a rejected key. An unreachable arena exits 1 with "cannot reach the arena server". `cli.ExecuteWithInput` lets tests supply stdin. The README shows a subprocess MCP config.

Tests: `TestCLIServesMCPOverStdio` checks that the tool list is identical to `/mcp`, plays red over stdio against yellow over HTTP MCP, checks that REST sees the stdio move, checks that `get_game_state` and a NOT_YOUR_TURN error are byte-identical to `/mcp`, and checks a clean exit on EOF. `TestCLIMCPNeedsKeyAndServer` covers a missing key, a bad key and an unreachable server. Both passed 3 times under `-race`. `make check` PASS, 91.5%.

Black-box: real `bin/connect4 mcp` fed raw JSON-RPC on stdin against a real `bin/arena`. It completed initialize, listed the 6 tools, ran `create_game` (the game was then visible through `connect4 state` over REST), and returned the arena's `GAME_NOT_FOUND` tool error for a bad move. It exited 0 on EOF, and with no key it exited 4.
