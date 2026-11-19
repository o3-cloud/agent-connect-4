---
title: "BDR-022 not satisfied on main: Agents Can Discover How to Play"
status: closed
source: bdr-verification
bdr: BDR-022
branch: main
commit: 30cd082
found: 2026-11-18
closed: 2026-11-19
fixed_in: 43322c5
---

# BDR-022 not satisfied on main: Agents Can Discover How to Play

Spec: [`docs/bdr/022-self-describing-interfaces.md`](../../../docs/bdr/022-self-describing-interfaces.md)

## Expected

A developer or agent with no prior knowledge can find out every operation, input and output from the platform itself, through its OpenAPI document, CLI help and MCP tool listing.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [ ] AC-1: The server publishes a machine-readable OpenAPI document that describes every public `/api/v1` endpoint, including request bodies, responses and error shape.
- [ ] AC-2: `connect4 --help` lists every command, and `connect4 <command> --help` lists that command's arguments and flags.
- [ ] AC-3: The MCP tool listing describes each tool's purpose and input schema (see [BDR-021](021-mcp-tools.md)).
- [ ] AC-4: An agent that uses only the published OpenAPI document can create, join and complete a game without other documentation.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Resolution

Fixed on main in 43322c5 (`docs(api): make the OpenAPI document enough to play a game`).

- AC-1: `/api/v1/openapi.json` (OpenAPI 3.1) already covered all 8 operations. Two gaps are now closed:
  - `info.description` (embedded from `internal/api/guide.md`) explains the full flow, retry rules and the `/mcp` and CLI alternatives.
  - Every operation now documents `429 RATE_LIMITED` with a `Retry-After` header.
- AC-2: `connect4 --help` lists every command, and each `<command> --help` shows its arguments and flags. This already held and is now pinned by a test.
- AC-3: `tools/list` returns 6 tools, each with a description and a described input schema. This already held, and BDR-021 tests cover it.
- AC-4: new `TestAgentPlaysUsingOnlyOpenAPIDocument`.
  - The test agent looks up every operation by `operationId` and checks every field against the document's schemas. Enum values such as `mode`, `disc` and `type` also come from the document.
  - It then registers two agents, creates and joins a game, and plays it to `completed`.
- New tests are in `tests/discovery_test.go`. `make check` passes, 91.4% coverage.
- Black-box run on the real `bin/arena`: `openapi.json`, `openapi.yaml` and `/api/v1/docs` return 200. All 8 operations list 429 with `Retry-After`, `connect4 --help` exits 0, and MCP `tools/list` shows the 6 described tools. No OpenAPI validator was available offline, so the document was checked structurally only.
