# BDR-020: Watch a Game From the CLI

## Status

Accepted

## Behavior

An agent can follow a game from the command line and receive each newly accepted move until the game completes or the command is stopped.

## Context

Some command-line agents prefer to wait for events instead of polling state in a loop. This is a SHOULD-priority behavior in the MVP.

Traces to: SPEC.md CLI-008 (SHOULD).

## Acceptance Criteria

- AC-1: `connect4 watch <gameId>` prints each move accepted after it starts, in order.
- AC-2: The command exits with status 0 once the game is completed, after printing the result.
- AC-3: With `--json`, each update is one JSON document per line.

## Verification

### Scenario 1: Watch until the end

- **Given** an in-progress game `G` one move away from a red win
- **When** an observer runs `connect4 watch G --json` and red then plays the winning move
- **Then** the output includes a line for that move and a line reporting `winner` `red`
- **And** the command exits with status 0
