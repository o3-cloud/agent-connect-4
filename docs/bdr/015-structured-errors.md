# BDR-015: Structured, Machine-Readable Errors

## Status

Accepted

## Behavior

Every rejected request returns an error with a machine-readable code, a human-readable message and the relevant request context, in a consistent shape on every interface.

## Context

Agents recover from errors by reading them programmatically, so error codes have to be stable and consistent across REST, CLI and MCP.

Traces to: SPEC.md API-007, CLI-007, MCP-008, §16 (failed response example).

## Acceptance Criteria

- AC-1: REST errors use the body `{"error": {"code": string, "message": string, ...context}}`.
- AC-2: Game-rule errors include `gameId` and `currentVersion` in the error context.
- AC-3: The same rule violation produces the same `code` through REST, the CLI with `--json`, and MCP.
- AC-4: An MCP rule violation is reported as a tool-level error result (with the error flag set), not as a protocol failure.

## Verification

### Scenario 1: REST error shape

- **Given** an in-progress game `G` at `version` `1` with yellow to move
- **When** red submits a move through REST
- **Then** the body is `{"error": {"code": "NOT_YOUR_TURN", "message": <non-empty>, "gameId": "G", "currentVersion": 1}}`

### Scenario 2: Same code on every interface

- **Given** an in-progress game with column 0 full and red to move
- **When** red attempts column 0 through REST, through `connect4 move G --column 0 --json`, and through the MCP tool `make_move`
- **Then** each interface reports error code `COLUMN_FULL`
- **And** the MCP result is marked as an error and its content includes `COLUMN_FULL`
