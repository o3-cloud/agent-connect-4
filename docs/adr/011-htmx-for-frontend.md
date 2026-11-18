# ADR-011: htmx for the Browser UI

## Status

Accepted

## Context and Problem Statement

Humans play against agents, and spectators watch games, in a browser (`SPEC.md` UI-001..013). The UI must render the authoritative server state after every move (UI-004), update live when an agent moves through any interface (UI-008, LIVE-002..004), and resync after a reconnect or a missed event (UI-012, LIVE-005). The server holds all game logic (server authority), so the browser only needs to display state and send column choices. `SPEC.md` §19 recommends React + TypeScript, which would add a Node.js build toolchain and a client-side state model alongside a Go server ([ADR-003](003-go-language.md)).

How should we build the browser UI?

## Decision Drivers

- **Server authority**: the browser should not hold or compute game state; it renders what the server sends
- **Single toolchain**: avoid a Node.js build pipeline for the main application
- **Live updates**: push board and history changes to the browser without polling (LIVE-002)
- **Small surface**: a dashboard, a board, a move history and a replay view, not a large single-page app
- **Testability**: HTML fragments can be asserted in Go handler tests; end-to-end flows are tested in a real browser
- **Dependency policy**: one small, pinned, vendored asset rather than a large npm tree

## Considered Options

- **Option A**: htmx with server-rendered HTML fragments (Go `html/template`)
- **Option B**: React + TypeScript single-page app, as `SPEC.md` §19 recommends
- **Option C**: Plain JavaScript (as in the throwaway `prototype/`)
- **Option D**: Go-to-WebAssembly front end

## Pros and Cons of the Options

### Option A: htmx

- Pro: the server returns HTML for the board, history and status, so the authoritative state is rendered once, on the server
- Pro: a single small script with no build step, vendored and embedded in the Go binary
- Pro: live updates through htmx's WebSocket or SSE extensions, which swap server-rendered fragments into the page
- Con: rich client-side interaction (such as the disc-drop animation or hover previews) needs small hand-written scripts or CSS
- Con: the team has to keep a clear set of partial templates, or the HTML endpoints become ad hoc

### Option B: React + TypeScript

- Pro: rich client-side interactivity, and a large ecosystem
- Con: a second language and a Node.js build toolchain; client-side state duplicates server state and can drift from it
- Con: much larger dependency footprint to review under `dependency-management.md`

### Option C: Plain JavaScript

- Pro: no dependency
- Con: we would hand-write DOM diffing, request plumbing and WebSocket handling

### Option D: Go → WebAssembly

- Con: large downloads, immature UI tooling, poor fit for a document-style interface

## Decision Outcome

Chosen option: **Option A, htmx**, because it keeps rendering and state on the Go server, which fits server authority, and it avoids a JavaScript build toolchain.

We will:

1. Render pages and partials with Go `html/template`, which escapes untrusted input such as agent display names by default.
2. Vendor one pinned htmx release (and any extensions we use) into the repository, record its checksum, embed it with `go:embed`, and serve it ourselves. We will not load it from a CDN at runtime (`dependency-management.md` rules 5, 9 and 18).
3. Push board, history and status fragments to subscribed browsers over a WebSocket (LIVE-002) using an htmx extension. Each fragment carries the state version, so the client can detect a gap and do a full refetch (LIVE-005, UI-012).
4. Submit moves as normal htmx requests (`hx-post`) that return the re-rendered board. The server enables column controls only for the human who holds the current turn (UI-003, UI-005).
5. Keep the small amount of custom client behaviour (the drop animation, keyboard shortcuts) in one small, plain script.

## Consequences

### Positive

- One language and one binary. The UI ships embedded in the server, with no `node_modules` for the application
- The browser cannot drift from server state; every update is a server-rendered fragment
- HTML responses are easy to test in Go handler tests, and Playwright covers the end-to-end flows (`SPEC.md` AT-001, AT-008, AT-009)

### Negative

- htmx 4.x is a recent major version (4.0.0 released 2026-08-28) and 2.x is still being patched. We must choose one major line and check that the extensions we use support it
- A UI built from fragments needs discipline about template structure and swap targets

### Neutral

- The REST JSON API ([ADR-008](008-huma-for-http-api-and-openapi.md)) stays the agent contract; the HTML routes serve browsers only and are not part of the OpenAPI document

## Dependency Review

| Item | Finding (2026-11-18) |
|------|----------------------|
| Package | htmx (`bigskysoftware/htmx`) |
| License | 0BSD (Zero-Clause BSD) |
| Maintenance | Active; latest release v4.0.2 (2026-11-18); v2.0.11 is still patched (2026-11-01) |
| Footprint | One vendored script, plus the WebSocket extension that matches the chosen major version |

## Compliance

| Spec | Enforcement |
|------|-------------|
| `SPEC.md` UI-008, LIVE-002..005 | Playwright test: a move made through MCP or the CLI appears in an open browser without a refresh |
| `security/dependency-management.md` rules 5, 9 and 18 | Vendored file with a recorded checksum; CI verifies the checksum; no runtime CDN |

## References

- **internal** [ADR-003](003-go-language.md) — Go as Primary Language
- **internal** [ADR-008](008-huma-for-http-api-and-openapi.md) — Huma for HTTP API Routing and Code-First OpenAPI
- **internal** [ADR-012](012-tailwind-css.md) — Tailwind CSS for Styling
- **spec**: `SPEC.md` UI-001..013, LIVE-001..005
- **external**: https://htmx.org/
- **external**: https://github.com/bigskysoftware/htmx
