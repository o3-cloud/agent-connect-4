# ADR-010: Cobra for the `connect4` CLI

## Status

Accepted

## Context and Problem Statement

Agents must be able to play through a command-line tool named `connect4` (`SPEC.md` CLI-001..008). It has subcommands (`create`, `join`, `state`, `move`, `watch`, …) with positional arguments and flags such as `--column` and `--json`. With `--json` it must write clean machine-readable output to stdout, and on failure it must exit non-zero (CLI-006, CLI-007). Agents discover a CLI by reading its help text, so help and usage output matter as much as parsing. `SPEC.md` §19 recommends Commander.js, but [ADR-003](003-go-language.md) makes Go the primary language.

Which library should implement the CLI?

## Decision Drivers

- **Agent ergonomics**: generated help, usage and shell completion that an agent can explore with `--help`
- **Nested subcommands**: `connect4 <verb> <gameId> [flags]` now, with room for `connect4 mcp` ([ADR-009](009-mcp-go-sdk.md))
- **Output control**: commands write to injectable writers, so tests can assert the exact stdout and stderr (CLI-006)
- **Testability**: commands can run in-process with arguments supplied in the test (NFR-007, `unit-testing.md`)
- **Dependency policy**: widely used, maintained, permissively licensed

## Considered Options

- **Option A**: Cobra (`github.com/spf13/cobra`)
- **Option B**: Standard library `flag` package with a hand-written subcommand dispatcher
- **Option C**: `urfave/cli`
- **Option D**: Commander.js, as `SPEC.md` §19 recommends

## Pros and Cons of the Options

### Option A: Cobra

- Pro: the de facto standard for Go CLIs (used by `kubectl`, `gh`, `hugo`), so agents and contributors already know its conventions
- Pro: nested subcommands, POSIX flags, generated help and usage, and shell completion out of the box
- Pro: `SetOut`, `SetErr` and `SetArgs` make in-process tests easy
- Con: more structure than a small CLI strictly needs; brings `spf13/pflag`

### Option B: Standard library `flag`

- Pro: no dependency
- Con: no subcommands, help generation or completion; we would rebuild what Cobra provides

### Option C: urfave/cli

- Pro: capable, and a smaller API
- Con: less common than Cobra in the Go ecosystem, so agents have seen fewer examples of its conventions

### Option D: Commander.js

- Con: conflicts with [ADR-003](003-go-language.md); users would need a Node.js runtime instead of a single binary

## Decision Outcome

Chosen option: **Option A, Cobra**, because it is the most familiar CLI framework in Go and its help, completion and testability features fit an agent-first CLI.

We will:

1. Implement `connect4` as a Cobra root command, with one file per subcommand that `SPEC.md` §8 names.
2. Make the CLI a client of the REST API ([ADR-008](008-huma-for-http-api-and-openapi.md)) using a typed Go client. It never imports the game engine to decide outcomes (server authority).
3. Add a persistent `--json` flag. When it is set, commands write only JSON to stdout, including error objects, and write diagnostics to stderr (CLI-006, CLI-007).
4. Set `SilenceUsage` and `SilenceErrors` and map errors to exit codes centrally, so failures never print decorative output into JSON mode.
5. Read the server URL and credentials from flags and environment variables, never from positional arguments, so secrets do not appear in shell history (SEC-005).

## Consequences

### Positive

- Agents can discover every command and flag through `connect4 --help` and `connect4 <cmd> --help`
- Golden-output tests can assert exact JSON and text output in-process
- Shell completion is free

### Negative

- Two new modules (`cobra`, `pflag`) to keep updated
- `make build` must now produce two binaries (server and `connect4`), and the Makefile and Dockerfile must change to match ([ADR-001](001-five-command-interface.md))

## Dependency Review

| Item | Finding (2026-11-18) |
|------|----------------------|
| Module | `github.com/spf13/cobra` (+ `github.com/spf13/pflag`) |
| License | Apache-2.0 |
| Maintenance | Mature and stable; latest release v1.10.2 (2025-12-04); very widely used |
| Footprint | Do not adopt `viper` unless a later ADR justifies it |

## Compliance

| Spec | Enforcement |
|------|-------------|
| `SPEC.md` CLI-006/007 | Golden tests: `--json` stdout parses as JSON for success and error cases; failures exit non-zero |
| `SPEC.md` SEC-005 | Test: credentials never appear in CLI stdout or stderr |
| `security/dependency-management.md` | Version pinned in `go.mod` / `go.sum` |

## References

- **internal** [ADR-001](001-five-command-interface.md) — Five Command Project Interface
- **internal** [ADR-003](003-go-language.md) — Go as Primary Language
- **internal** [ADR-008](008-huma-for-http-api-and-openapi.md) — Huma for HTTP API Routing and Code-First OpenAPI
- **internal** [ADR-009](009-mcp-go-sdk.md) — MCP Go SDK for the MCP Server
- **spec**: `SPEC.md` CLI-001..008, SEC-005
- **external**: https://github.com/spf13/cobra
- **external**: https://cobra.dev/
