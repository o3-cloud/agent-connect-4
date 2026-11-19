---
title: "BDR-013 not satisfied on main: Authenticate Agent Operations and Protect Credentials"
status: closed
source: bdr-verification
bdr: BDR-013
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-013 not satisfied on main: Authenticate Agent Operations and Protect Credentials

Spec: [`docs/bdr/013-authenticate-agent-operations.md`](../../../docs/bdr/013-authenticate-agent-operations.md)

## Expected

Creating, joining and moving require valid credentials, and credentials never appear in any response, history, event or log that the system exposes.

## Observed (black-box, on main)

- `make build` now produces `bin/connect4` (ac4df75), but it is a help-only Cobra root: `connect4 create`, `connect4 play` and `--json` all fail with `unknown command` / `unknown flag` (exit 1, plain-text error). No game subcommands exist on main.
- `GET /mcp` -> `404` (re-verified on 4218fba); no MCP server or tools exist on main.
- `GET /` -> `200 text/plain "stack-base-go"` (re-verified on 4218fba); no browser UI exists on main.
- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [x] AC-1: Creating a game without credentials is rejected, and no game is created.
- [x] AC-2: Joining a game without credentials is rejected, and the slot stays open.
- [x] AC-3: Submitting a move without credentials, or with invalid ones, is rejected, and the game is unchanged.
- [x] AC-4: The credential value used in a test never appears in game state, move history, game lists, live updates sent to spectators, or CLI and MCP output.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Unblocked (2026-11-18, ADR-013)

The spec never said how an agent obtains a credential, so this issue had no buildable design. [ADR-013](../../../docs/adr/013-self-issued-agent-api-keys.md) decides it:

- `POST /api/v1/agents` (no auth) with `{"displayName", "type"}` → `201 {"agentId", "displayName", "apiKey": "c4_…"}`; the key is returned only once and stored only as a SHA-256 hash.
- `Authorization: Bearer <apiKey>` on create, join, move and resign; a missing, malformed or unrecognised key → `401 UNAUTHENTICATED`. The acting player is the key's `agentId`.
- Remove the any-token test identity; tests register keys first.

## Scope to close this issue

- [x] Registration endpoint and hashed key store, with HTTP tests.
- [x] A made-up `c4_` key gets `401` on create, join and move (AC-1..3).
- [x] Key strings never appear in state, history, the game list or error bodies (AC-4).
- [x] Re-run BDR-007: its AC-3 should now pass, so BDR-007 can be accepted too.

## Progress (39f853d, 2026-11-18)

Built: `POST /api/v1/agents` issues `c4_` keys stored as SHA-256 hashes; create, join, move and resign accept only issued keys. Verified with curl against a running `bin/arena`: made-up and forged `c4_` keys get `401 UNAUTHENTICATED`, and no key appears in game state or the game list. Tests: `tests/auth_test.go`.

Still open: AC-4 also covers live spectator updates and CLI and MCP output, which do not exist yet. Re-check AC-4 when each lands; then accept BDR-013.

## CLI part of AC-4 done (9e6ef2b, 2026-11-18)

`TestCLIPlaysAGameWithJSONOutput` asserts that neither API key appears in CLI output for create, state, history or move. Still needed for AC-4: MCP output (BDR-021, in progress) and live spectator updates (BDR-027).

## Progress (e517837, 2026-11-18, session 5b5d3e77)

AC-4, MCP part: `/mcp` now exists and requires an issued key (401 otherwise). `TestMCPListAndHistory` asserts that no API key appears in `list_games` or `get_game_history` results. AC-4 still needs a check for live spectator updates (BDR-027) and for CLI output (9e6ef2b).

## Independent verification (e517837, 2026-11-18, bdr-verification loop)

Black-box against a running `bin/arena`: AC-1 to AC-3 pass for REST, and an unauthenticated MCP connection is rejected. For AC-4, no API key appeared in REST state, history or lists, in `connect4 state/history` output (human and `--json`), or in MCP `get_game_state`, `get_game_history`, `list_games` and `tools/list` results. Only live spectator updates are still unverifiable, because BDR-027 is not built. Accept BDR-013 when that lands.

## Accepted (2606fc3, 8bb5de4, 2026-11-19)

AC-4 verified on every current surface: `tests/leak_test.go` (REST, MCP, CLI, `connect4 watch`, browser spectator page) and a black-box run with the built binaries (15 KB of output, 18 watch updates, 0 key hits). `make check` passes (coverage 90.8%, govulncheck clean). BDR-027's WebSocket push must keep `TestCredentialsNeverLeakToAnySurface` passing.
