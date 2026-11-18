# BDR-019: Play Through the `connect4` CLI With Machine-Readable Output

## Status

Accepted

## Behavior

An agent can create, join, inspect and play a game using the `connect4` command-line tool. With `--json` it gets clean JSON output, and every failure exits non-zero.

## Context

Many agents can only drive tools through a shell. The CLI must be scriptable: predictable commands, parseable output and meaningful exit codes.

Traces to: SPEC.md CLI-001..007; OUTCOMES.md 1, 6.

## Acceptance Criteria

- AC-1: `connect4 create`, `connect4 join <gameId>`, `connect4 state <gameId>` and `connect4 move <gameId> --column <n>` each perform their operation against the server.
- AC-2: With `--json`, standard output contains only a single valid JSON document, with no banners, colors or progress text.
- AC-3: Successful commands exit with status 0. Failed commands exit with a non-zero status.
- AC-4: With `--json`, a failed command's standard output contains an `error` object with the machine-readable error code.

## Verification

### Scenario 1: Scripted game creation

- **Given** `connect4` is configured with a valid agent credential
- **When** the agent runs `connect4 create --mode agent_vs_agent --json`
- **Then** the exit status is 0 and standard output parses as JSON containing a `gameId`

### Scenario 2: Failure in JSON mode

- **Given** an in-progress game `G` with red to move, and the yellow agent's credential configured
- **When** the yellow agent runs `connect4 move G --column 3 --json`
- **Then** the exit status is non-zero
- **And** standard output parses as JSON with `error.code` = `NOT_YOUR_TURN`
