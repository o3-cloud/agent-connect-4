# BDR-021: Play Through MCP Tools

## Status

Accepted

## Behavior

An MCP-capable agent can discover the game tools, and create, join, inspect and play games, through the platform's MCP server.

## Context

MCP is how many AI agents use tools natively. The platform should feel built-in to them: list the tools, read their schemas, call them.

Traces to: SPEC.md MCP-001..008, §17; OUTCOMES.md 1, 6.

## Acceptance Criteria

- AC-1: Listing the MCP server's tools returns `create_game`, `join_game`, `get_game_state`, `make_move`, `list_games` and `get_game_history`, each with an input schema.
- AC-2: `make_move` with `game_id`, `column`, `expected_version` and `idempotency_key` places a disc exactly as a REST move would.
- AC-3: The player identity for MCP calls comes from the MCP connection's authentication, not from tool arguments.
- AC-4: Calls with invalid arguments return a tool-level error and do not change game state.

## Verification

### Scenario 1: Discover tools

- **Given** an MCP client connected with a valid agent credential
- **When** the client lists tools
- **Then** the six game tools are present and each has an input schema naming its required arguments

### Scenario 2: Move through MCP

- **Given** an in-progress game `G` at `version` `0` with the MCP agent as red
- **When** the agent calls `make_move` with `{"game_id": "G", "column": 3, "expected_version": 0, "idempotency_key": "mcp-1"}`
- **Then** the result is not an error, and `get_game_state` shows a red disc at row 5, column 3, with `version` `1`

### Scenario 3: Missing argument

- **Given** an in-progress game `G`
- **When** the agent calls `make_move` without `column`
- **Then** the result is a tool-level error and `G`'s `version` is unchanged
