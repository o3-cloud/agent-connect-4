# Connect Four Arena

A Connect Four platform where AI agents and humans play through a REST API, the
`connect4` CLI, MCP and a browser UI. It is written in Go, started from the
`stack-base-go` template, and follows the stackable-specs methodology.

This repository implements the foundational command interface from
[`stack-base`](https://github.com/stackable-specs/stack-base): every project
exposes `build`, `run`, `check`, `health`, and `clean` through `make`.

## Architecture

This project is organized with layered specifications where each layer answers exactly one question about the system:

| Layer | Question | Spec |
|-------|----------|------|
| `interface` | How do developers interact with it? | Five-command interface |
| `language` | What is it written in? | Go |
| `platform` | Where does it run? | (TBD by ADR) |
| `delivery` | How is it shipped? | Docker, GitHub Actions |
| `practices` | How do we work? | MADR, BDR, TDD, Git, Conventional Commits |
| `quality` | How do we enforce? | Unit Testing |
| `security` | What must never break trust? | Dependency Management |

## Repository Layout

```
.
├── .github/workflows/     # CI/CD pipelines
├── docs/
│   ├── adr/               # Architectural Decision Records
│   ├── bdr/               # Behavior Decision Records
│   └── specs/             # Layered specifications
│       ├── delivery/
│       ├── interface/
│       ├── practices/
│       ├── quality/
│       └── security/
├── cmd/
│   ├── arena/             # Game server (bin/arena)
│   └── connect4/          # connect4 CLI (bin/connect4)
├── internal/              # Application packages
├── scripts/               # Project support scripts
├── tests/                 # Automated tests
├── CLAUDE.md              # Imports AGENTS.md for Claude Code
├── AGENTS.md              # AI agent context (single source of truth)
├── go.mod                 # Go module definition
├── Makefile               # Common commands
└── README.md
```

## Key Commands

The five core commands are the primary project interface:

| Command | Description | Spec Reference |
|---------|-------------|----------------|
| `make build` | Build `bin/arena` and `bin/connect4` | `interface/commands.md`, ADR-001 |
| `make run` | Build and run the arena server | `interface/commands.md`, ADR-001 |
| `make check` | Run all deterministic quality gates | `interface/commands.md`, `quality/command-verification.md` |
| `make health` | Show repository health and quality issues | `interface/commands.md` |
| `make clean` | Remove generated artifacts | `interface/commands.md` |

Utility targets such as `make test`, `make lint`, `make fmt`, `make vet`,
`make coverage`, and `make vuln` support `make check`.

## Playing over the REST API

`make run` starts the arena on port 8080 (set `PORT` to change it). The API
lives under `/api/v1`, and its OpenAPI document is at `/api/v1/openapi.json`.
Games are held in memory for now, so they are lost when the server stops.

Each player needs an API key (ADR-013). `POST /api/v1/agents` with a display
name and type returns one. The key is shown only in that response, so keep it.
Send it as `Authorization: Bearer <apiKey>` on create, join, move and resign;
a missing or unrecognised key gets `401 UNAUTHENTICATED`. Reading games needs
no key.

```sh
API=http://localhost:8080/api/v1
ALPHA=$(curl -s -X POST $API/agents -H 'Content-Type: application/json' \
  -d '{"displayName":"Alpha","type":"agent"}' | jq -r .apiKey)
BETA=$(curl -s -X POST $API/agents -H 'Content-Type: application/json' \
  -d '{"displayName":"Beta","type":"agent"}' | jq -r .apiKey)
GAME=$(curl -s -X POST $API/games -H "Authorization: Bearer $ALPHA" \
  -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' | jq -r .gameId)
curl -s -X POST $API/games/$GAME/join -H "Authorization: Bearer $ALPHA" \
  -H 'Content-Type: application/json' -d '{"disc":"red","displayName":"Alpha","type":"agent"}'
curl -s -X POST $API/games/$GAME/join -H "Authorization: Bearer $BETA" \
  -H 'Content-Type: application/json' -d '{"disc":"yellow","displayName":"Beta","type":"agent"}'
curl -s -X POST $API/games/$GAME/moves -H "Authorization: Bearer $ALPHA" \
  -H 'Content-Type: application/json' -d '{"column":3,"expectedVersion":0}'
curl -s $API/games/$GAME
curl -s $API/games/$GAME/moves   # ordered move history and result
# Either player may resign; the opponent wins
curl -s -X POST $API/games/$GAME/resign -H "Authorization: Bearer $BETA"
```

Errors use one shape, `{"error": {"code", "message", "retryable", ...}}`,
with the codes and HTTP statuses in `SPEC.md` §16.

Each client may make 600 API and MCP requests per minute by default. A request
with a valid API key counts against its agent; any other request counts against
its IP address. Excess requests get `429 RATE_LIMITED` with `retryAfterMs` and
a `Retry-After` header. Set `ARENA_RATE_LIMIT` (requests) and
`ARENA_RATE_WINDOW` (for example `10s`) to change the limit.

The server writes one JSON log record per line to standard output for each
operational event: `game_created`, `player_joined`, `move_accepted`,
`move_rejected` (with its error `code`), `move_replayed` (an idempotent
retry), `player_resigned`, `game_completed` and `server_error`. Every game
record has a `gameId`; records name players by `agt_` ID and never contain API
keys. For example, list rejected moves with `./bin/arena | jq 'select(.event == "move_rejected")'`.

## Playing in the browser

`make run`, then open http://localhost:8080. Enter a name, choose red or
yellow, and click **Play vs AI**: you play Arena Bot (ADR-014). Click the arrow
above a column to drop a disc; the board refreshes every second and only offers
moves on your turn. Anyone with the game's link can watch.

`make build` builds the stylesheet with a pinned Tailwind CLI (ADR-012), which
`make` downloads once into `.tools/` and checks against its published checksum.
The container image builds the stylesheet the same way: `make docker-build`
builds the image, and `make docker-smoke` runs it and checks that it serves
`/healthz` and `/static/app.css`.

## Playing with the `connect4` CLI

`make build` also produces `bin/connect4`. It talks to `$CONNECT4_SERVER`
(default `http://localhost:8080`) and authenticates with `$CONNECT4_API_KEY`.
Add `--json` to any command to get exactly one JSON document on stdout. A failed
command exits with the code in `SPEC.md` §16 (for example 6 for
`NOT_YOUR_TURN`), and with `--json` prints the `error` object.

A game needs two players, so the walkthrough registers two. Alpha's key goes
in `CONNECT4_API_KEY`; Beta's is passed with `--api-key`:

```sh
eval "$(bin/connect4 register --name Alpha | tail -1)"            # sets CONNECT4_API_KEY
BETA=$(bin/connect4 register --name Beta | tail -1 | cut -d= -f2)  # a second player's key
GAME=$(bin/connect4 create)
bin/connect4 join $GAME --disc red --name Alpha
bin/connect4 join $GAME --disc yellow --name Beta --api-key $BETA
bin/connect4 move $GAME --column 3      # uses the current version unless --expected-version is set
bin/connect4 state $GAME                # board and status
bin/connect4 history $GAME --json       # ordered moves and result
bin/connect4 resign $GAME --api-key $BETA   # either player may resign; the opponent wins
bin/connect4 watch $GAME --json         # one JSON event per new move; exits 0 when the game ends
```

Run `watch` in a second terminal during a game to follow it live. On a finished
game, as here, it prints the result and exits.

To play against the house agent, Arena Bot (ADR-014), create a human game with
`--opponent house`. Arena Bot takes the other slot before your `join` returns,
so the game is already in progress and you can move straight away. It replies
to each move within a second:

```sh
GAME=$(bin/connect4 create --mode human_vs_agent --opponent house)
bin/connect4 join $GAME --disc red --name You --type human
bin/connect4 move $GAME --column 3 && bin/connect4 state $GAME
```

## Playing over MCP

The server also speaks MCP (Streamable HTTP) at `/mcp`. Authenticate with the
same API key as REST, sent as `Authorization: Bearer <key>`. The key decides
which player you are, so no tool takes a player argument. The six tools are
`create_game`, `join_game`, `get_game_state`, `make_move`, `list_games` and
`get_game_history`, and each one publishes its input schema. Results match
the REST views. A game-rule failure is a tool error (`isError: true`) whose
structured content is the same `{"error": {...}}` object REST returns.

For example, in an MCP client config:

```json
{ "mcpServers": { "connect4": {
    "type": "http",
    "url": "http://localhost:8080/mcp",
    "headers": { "Authorization": "Bearer c4_..." } } } }
```

## Quality Gates

- **Format**: `gofmt` passes (zero diffs)
- **Vet**: `go vet ./...` passes
- **Lint**: `golangci-lint` passes
- **Test**: `go test -race ./...`
- **Coverage**: ≥80% (configurable)
- **Security**: `govulncheck ./...` passes

## Specs

All specs are in `docs/specs/`. Each file defines rules for a specific layer:

- **Language**: `language/go.md` — Go idioms, formatting, module hygiene
- **Interface**: `interface/commands.md` — Five-command project interface
- **Practices**: `practices/*.md` — ADRs, BDRs, TDD, git workflow
- **Quality**: `quality/unit-testing.md` — Test scope and naming
- **Quality**: `quality/command-verification.md` — Command contract verification
- **Security**: `security/dependency-management.md` — Dependency policy
- **Delivery**: `delivery/*.md` — Docker and CI/CD conventions

## stack-base Implementation

This project backfills the foundational stack-base decisions:

| ADR | Decision | Implementation |
|-----|----------|----------------|
| ADR-001 | Five Command Project Interface | `Makefile` exposes `build`, `run`, `check`, `health`, `clean` |
| ADR-002 | Adopt Make for Command Runner | The root `Makefile` is the canonical command runner |

The copied stack-base specs live under `docs/specs/interface/`,
`docs/specs/practices/command-workflow.md`, and
`docs/specs/quality/command-verification.md`.

## Workflow

1. Check relevant specs before making changes, starting with `interface/commands.md`
2. Write tests first (TDD: `practices/tdd.md`)
3. Run `make check` before committing
4. Reference specs in commit messages and PRs

## Getting Started

1. Run `make build`
2. Run `make health`
3. Run `make check`
4. Run `make run`

## Spec-to-Enforcement Traceability

| Spec Rule | Enforcement |
|-----------|-------------|
| `interface/commands.md` | Five primary Make targets |
| `command-verification.md` | CI validates the five-command interface |
| `go.md:2` | `gofmt` in pre-commit + CI |
| `go.md:3` | `goimports` in pre-commit + CI |
| `go.md:12` | `go test -race` in CI |
| `go.md:15` | `go vet ./...` in pre-commit + CI |
| `go.md:16` | `staticcheck` via golangci-lint |
| `go.md:17` | Git track `go.mod` + `go.sum` |
| `unit-testing.md` | `go test -coverprofile` with ≥80% threshold |
| `conventional-commits.md` | commitlint in pre-commit + CI |

## License

MIT
