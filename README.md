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

## Playing with the `connect4` CLI

`make build` also produces `bin/connect4`. It talks to `$CONNECT4_SERVER`
(default `http://localhost:8080`) and authenticates with `$CONNECT4_API_KEY`.
Add `--json` to any command to get exactly one JSON document on stdout. A failed
command exits with the code in `SPEC.md` §16 (for example 6 for
`NOT_YOUR_TURN`), and with `--json` prints the `error` object.

```sh
eval "$(bin/connect4 register --name Alpha | tail -1)"   # sets CONNECT4_API_KEY
GAME=$(bin/connect4 create)
bin/connect4 join $GAME --disc red --name Alpha
bin/connect4 move $GAME --column 3      # uses the current version unless --expected-version is set
bin/connect4 state $GAME                # board and status
bin/connect4 history $GAME --json       # ordered moves and result
bin/connect4 resign $GAME
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
