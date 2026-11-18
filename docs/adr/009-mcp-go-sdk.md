# ADR-009: Official MCP Go SDK for the MCP Server

## Status

Accepted

## Context and Problem Statement

External AI agents must be able to discover and call game operations over the Model Context Protocol (`SPEC.md` MCP-001..009). The required tools are `create_game`, `join_game`, `get_game_state`, `make_move`, `list_games` and `get_game_history` (§17). Each must use the same game service as the REST API and CLI (MCP-009) and return structured tool errors (MCP-008). Agent identity must come from the MCP transport's authentication context (§17), not from tool arguments (SEC-004). `SPEC.md` §19 recommends the official TypeScript SDK, but [ADR-003](003-go-language.md) makes Go the primary language.

Which MCP implementation should the server use?

## Decision Drivers

- **Spec conformance**: track the MCP specification as it evolves, without us maintaining protocol code
- **Same process, same service**: MCP tools must call the same Go game service as the REST API (MCP-009, NFR-004)
- **Transports**: Streamable HTTP for remote agents, and stdio for local agents launched as a subprocess
- **Auth hooks**: bearer-token authentication that exposes the caller's identity to tool handlers (SEC-001/002/004)
- **Typed tools**: input schemas generated from Go types, matching how [ADR-008](008-huma-for-http-api-and-openapi.md) handles REST
- **Dependency policy**: maintained, permissively licensed (`dependency-management.md` rule 10)

## Considered Options

- **Option A**: Official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`)
- **Option B**: Community Go library (`github.com/mark3labs/mcp-go`)
- **Option C**: Hand-rolled JSON-RPC 2.0 implementation of the MCP protocol
- **Option D**: Official TypeScript SDK in a separate Node.js process, as `SPEC.md` §19 recommends

## Pros and Cons of the Options

### Option A: Official Go SDK

- Pro: maintained under the `modelcontextprotocol` organization alongside the specification; aims to implement the full spec
- Pro: generic typed tool handlers generate input and output schemas from Go structs
- Pro: ships Streamable HTTP and stdio transports and an `auth` package for bearer-token middleware
- Con: younger than some community libraries; API changes are possible between minor versions
- Con: the license is moving from MIT to Apache-2.0, so the code is under a mix of the two (both permissive)

### Option B: mcp-go

- Pro: mature and widely used
- Con: not the reference implementation; may lag behind or diverge from spec changes

### Option C: Hand-rolled

- Pro: no dependency
- Con: we would own protocol negotiation, capabilities, transports and spec updates, which adds no value for the product

### Option D: TypeScript SDK sidecar

- Con: a second runtime, plus a network hop to reach the Go game service, breaks the "same service, same process" goal (MCP-009)

## Decision Outcome

Chosen option: **Option A, the official MCP Go SDK**, because it is the reference implementation of the protocol, and it lets MCP tools call the shared Go game service directly, in the same process as the REST API.

We will:

1. Implement the six `SPEC.md` §17 tools with the SDK's typed tool handlers, each delegating to the shared game service.
2. Mount the Streamable HTTP handler on the server's `ServeMux` (for example at `/mcp`), behind bearer-token auth middleware that resolves the agent identity from the token.
3. Offer a stdio transport through the CLI (for example `connect4 mcp`, see [ADR-010](010-cobra-for-cli.md)) for agents that launch MCP servers as subprocesses.
4. Map game-service errors to tool-level errors (`isError` with the `SPEC.md` §16 error shape), never to protocol errors, for rule, validation and authorization failures (MCP-008).
5. Derive the acting player only from the authenticated context, never from tool arguments (SEC-004).

## Consequences

### Positive

- MCP, REST and CLI share one game service and one set of validation rules, so behaviour cannot diverge (NFR-004)
- Protocol upgrades come from SDK updates instead of in-house work
- Tool schemas come from Go types, which keeps them consistent with the Huma-generated REST schemas

### Negative

- SDK minor-version upgrades may need code changes; we should pin exactly and upgrade on purpose
- License review must cover both MIT and Apache-2.0 during the SDK's relicensing period

### Neutral

- A contract test suite should run the same scenarios through REST, CLI and MCP (`SPEC.md` AT-002, AT-008)

## Dependency Review

| Item | Finding (2026-11-18) |
|------|----------------------|
| Module | `github.com/modelcontextprotocol/go-sdk` |
| License | Moving from MIT to Apache-2.0; contributions are under one or the other |
| Maintenance | Active; latest release v1.8.0 (2026-09-14); official MCP organization |
| Footprint | Use the `mcp` and `auth` packages; `jsonrpc` only if we need a custom transport |

## Compliance

| Spec | Enforcement |
|------|-------------|
| `SPEC.md` MCP-008/009, NFR-004 | Cross-interface contract tests run identical move and error scenarios through REST, CLI and MCP |
| `SPEC.md` SEC-004 | Test: a tool call cannot act as a player other than the authenticated one |
| `security/dependency-management.md` | Version pinned in `go.mod` / `go.sum`; licenses checked against the allowlist |

## References

- **internal** [ADR-003](003-go-language.md) — Go as Primary Language
- **internal** [ADR-008](008-huma-for-http-api-and-openapi.md) — Huma for HTTP API Routing and Code-First OpenAPI
- **internal** [ADR-010](010-cobra-for-cli.md) — Cobra for the `connect4` CLI
- **spec**: `SPEC.md` MCP-001..009, SEC-001..004, §17
- **external**: https://github.com/modelcontextprotocol/go-sdk
- **external**: https://modelcontextprotocol.io/
