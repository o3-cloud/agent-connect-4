# ADR-008: Huma for HTTP API Routing and Code-First OpenAPI

## Status

Accepted

## Context and Problem Statement

Connect Four Arena exposes a versioned JSON REST API under `/api/v1` (`SPEC.md` API-001..008) and must publish machine-readable OpenAPI documentation for it (NFR-005). The API is the main contract for external agents, and the CLI ([ADR-010](010-cobra-for-cli.md)) is a client of it. A hand-maintained OpenAPI file drifts from the handlers, while generating handlers from a spec adds a codegen step and a second source of truth. `SPEC.md` §19 recommends Node.js + Fastify, but [ADR-003](003-go-language.md) makes Go the primary language.

What should we use for HTTP routing, request validation, and OpenAPI generation?

## Decision Drivers

- **Single source of truth**: Go request/response types should define both the behaviour and the OpenAPI document (NFR-005)
- **Input validation**: externally supplied input must be validated before it reaches game state (SEC-006)
- **Structured errors**: errors must use one consistent JSON shape (API-007, `SPEC.md` §16)
- **Standard library first**: prefer `net/http` and Go 1.22+ `ServeMux` routing over a heavyweight framework
- **Testability**: handlers should be testable without a network listener (NFR-007)
- **Dependency policy**: a maintained library with a permissive license (`dependency-management.md` rule 10)

## Considered Options

- **Option A**: Huma v2 (`github.com/danielgtaylor/huma/v2`) on the standard library router
- **Option B**: Standard library `net/http` with a hand-written OpenAPI document
- **Option C**: Spec-first code generation (`oapi-codegen`) from a hand-written OpenAPI document
- **Option D**: Node.js + Fastify, as `SPEC.md` §19 recommends

## Pros and Cons of the Options

### Option A: Huma v2

- Pro: OpenAPI 3.1 and JSON Schema are generated from Go structs, so docs cannot drift from the handlers
- Pro: declarative validation (struct tags) for path, query and body input
- Pro: works on top of the Go 1.22+ `ServeMux` through the `humago` adapter (or chi and others), so there is no lock-in to a router
- Pro: `humatest` exercises operations in-process
- Con: its default errors are RFC 9457 Problem Details, not the `{"error": {...}}` shape that `SPEC.md` §16 requires, so we have to override them
- Con: adds a framework layer that contributors need to learn

### Option B: Standard library + hand-written OpenAPI

- Pro: no new dependency
- Con: the OpenAPI document and the handlers are maintained separately and will drift
- Con: validation and error encoding would be hand-written for every endpoint

### Option C: oapi-codegen (spec-first)

- Pro: the OpenAPI document is the explicit contract
- Con: adds a codegen step and generated code to review; YAML becomes the source of truth instead of Go
- Con: changing an endpoint means editing YAML, regenerating, then editing handlers

### Option D: Fastify

- Con: conflicts with [ADR-003](003-go-language.md); a second runtime and toolchain for the server

## Decision Outcome

Chosen option: **Option A, Huma v2**, because it makes Go types the single source of truth for routing, validation and the OpenAPI document, and it still runs on the standard library router.

We will:

1. Register every public `/api/v1` operation with Huma, using the standard library `ServeMux` through the `humago` adapter.
2. Serve the generated OpenAPI 3.1 document and do not hand-edit it (NFR-005).
3. Override Huma's error constructor (`huma.NewError`) so every error response uses the `SPEC.md` §16 shape: `{"error": {"code", "message", ...context}}` (API-007).
4. Register browser HTML routes ([ADR-011](011-htmx-for-frontend.md)) and the MCP endpoint ([ADR-009](009-mcp-go-sdk.md)) on the same `ServeMux`, outside Huma, so the OpenAPI document covers only the JSON API.
5. Keep handlers as thin adapters over the shared game service. Game rules live in a transport-free engine package (NFR-007, MCP-009).

## Consequences

### Positive

- The OpenAPI document always matches the deployed API, so agents and the CLI client can rely on it
- Validation happens at the edge, consistently, before game state is touched (SEC-006)
- In-process operation tests with `humatest` keep coverage high without network setup (`unit-testing.md`)

### Negative

- Overriding the error format is needed and must be covered by contract tests, or Problem Details responses could leak through
- Huma's conventions (input/output structs, `Body` fields) take some learning

### Neutral

- The router stays swappable (`humago` ↔ `humachi`) without changing operation code

## Dependency Review

| Item | Finding (2026-11-18) |
|------|----------------------|
| Module | `github.com/danielgtaylor/huma/v2` |
| License | MIT |
| Maintenance | Active; latest release v2.41.0 (2026-11-17) |
| Footprint | Use only the core plus the `humago` adapter; no router dependency beyond the standard library |

## Compliance

| Spec | Enforcement |
|------|-------------|
| `SPEC.md` NFR-005 | Contract test asserts the served OpenAPI document lists every `/api/v1` operation |
| `SPEC.md` API-007 | Contract test asserts error bodies match the `{"error": {...}}` shape for each error code |
| `security/dependency-management.md` | Version pinned in `go.mod` / `go.sum` |

## References

- **internal** [ADR-003](003-go-language.md) — Go as Primary Language
- **internal** [ADR-009](009-mcp-go-sdk.md) — MCP Go SDK for the MCP Server
- **internal** [ADR-010](010-cobra-for-cli.md) — Cobra for the `connect4` CLI
- **internal** [ADR-011](011-htmx-for-frontend.md) — htmx for the Browser UI
- **spec**: `SPEC.md` API-001..008, NFR-005, NFR-007, SEC-006
- **external**: https://github.com/danielgtaylor/huma
- **external**: https://huma.rocks/
