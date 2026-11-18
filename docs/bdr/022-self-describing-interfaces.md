# BDR-022: Agents Can Discover How to Play

## Status

Proposed

## Behavior

A developer or agent with no prior knowledge can find out every operation, input and output from the platform itself, through its OpenAPI document, CLI help and MCP tool listing.

## Context

"A developer can connect an agent in minutes" depends on discovery, not on reading source code. Each interface should describe itself in a form a machine can consume.

Traces to: SPEC.md NFR-005, MCP-001, CLI-001; OUTCOMES.md 1, 6 ("discover how to play", "connect an agent in minutes").

## Acceptance Criteria

- AC-1: The server publishes a machine-readable OpenAPI document that describes every public `/api/v1` endpoint, including request bodies, responses and error shape.
- AC-2: `connect4 --help` lists every command, and `connect4 <command> --help` lists that command's arguments and flags.
- AC-3: The MCP tool listing describes each tool's purpose and input schema (see [BDR-021](021-mcp-tools.md)).
- AC-4: An agent that uses only the published OpenAPI document can create, join and complete a game without other documentation.

## Verification

### Scenario 1: OpenAPI coverage

- **Given** a running server
- **When** a client fetches the published OpenAPI document
- **Then** it parses as valid OpenAPI and includes operations for creating, listing, reading, joining, moving in and reading the history of games

### Scenario 2: CLI help

- **Given** the `connect4` binary
- **When** a user runs `connect4 --help`
- **Then** the output names `create`, `join`, `state` and `move`, and exits with status 0
