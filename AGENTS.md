# Project: stack-base-go

Go base template following the stackable-specs methodology.

## Outcomes

@OUTCOMES.md

## Architecture

Layered specifications where each spec answers exactly one question about the system:
- **Interface** (commands.md): Five-command project interface
- **Language** (go.md): Go idioms, formatting, module hygiene
- **Practices** (madr.md, bdr.md, tdd.md, git.md, conventional-commits.md): Workflow and decision records
- **Quality** (unit-testing.md): Test scope and coverage
- **Security** (dependency-management.md): Dependency policy
- **Delivery** (docker.md, github-actions.md): Container and CI/CD conventions

Foundational ADRs:
- **ADR-001**: Five Command Project Interface
- **ADR-002**: Adopt Make for Command Runner

Application stack ADRs (these replace the TypeScript stack recommended in `SPEC.md` §19):
- **ADR-008**: Huma for HTTP API routing and code-first OpenAPI
- **ADR-009**: Official MCP Go SDK for the MCP server
- **ADR-010**: Cobra for the `connect4` CLI
- **ADR-011**: htmx for the browser UI (server-rendered fragments)
- **ADR-012**: Tailwind CSS (standalone CLI) for styling

## Prototype

`prototype/` holds a throwaway HTML/JS prototype of `SPEC.md`. It is a reference for behaviour and UX, not production code. It is git-ignored, so it exists only in local checkouts. Open `prototype/index.html` to run it, and run `node --test prototype/arena.test.js` for its tests. See `prototype/README.md` for details.

| File | What it models | Spec / BDR |
|------|----------------|------------|
| `prototype/engine.js` | Pure rules: board, legal moves, win/draw detection, replay | RULE-*, DATA-005; BDR-005, 006, 008, 009, 017 |
| `prototype/service.js` | Authoritative game service: lifecycle, turns, versions, idempotency, auth, rate limits, logs, events | GAME-*, TURN-*, SEC-*, LIVE-001, NFR-006; BDR-001–003, 007, 010–014, 018, 032 |
| `prototype/interfaces.js` | REST, `connect4` CLI and MCP tool adapters over the same service | API-*, CLI-*, MCP-*; BDR-015, 019, 021, 023 |
| `prototype/agents.js` | Simulated external agents (random, greedy, minimax) that play only through an interface | AGENT-*; BDR-024, 025 |
| `prototype/app.js`, `index.html`, `styles.css` | Dashboard, board, history, spectating, replay, offline/reconnect, agent console | UI-*, LIVE-002..005; BDR-026–030 |
| `prototype/arena.test.js` | Node tests for rules, errors, idempotency, concurrency, restart, cross-interface play | AT-002..AT-007, AT-010 |

Implement against `SPEC.md`, the ADRs and the BDRs. Where the prototype differs from them, the spec wins. For example, its "server" runs in the browser and uses `localStorage`.

## Key Commands

| Command | Description | Spec Reference |
|---------|-------------|----------------|
| `make build` | Build the service binary | `commands.md`, ADR-001 |
| `make run` | Build and run the service | `commands.md`, ADR-001 |
| `make check` | All quality gates | `commands.md`, `command-verification.md` |
| `make health` | Repository health summary | `commands.md` |
| `make clean` | Remove generated artifacts | `commands.md` |

Utility targets include `make test`, `make lint`, `make fmt`, `make vet`, `make coverage`, and `make vuln`.

## Quality Gates

- **Format**: `gofmt -d .` returns empty (spec: go.md:2)
- **Imports**: `goimports` groups correctly (spec: go.md:3)
- **Vet**: `go vet ./...` passes (spec: go.md:15)
- **Lint**: `golangci-lint run` passes (spec: go.md:16)
- **Race**: `go test -race ./...` passes (spec: go.md:12)
- **Coverage**: ≥80% line coverage (spec: unit-testing.md)

## Specs

All specs are in `docs/specs/`. Check relevant specs before making changes. Each file defines rules for a specific layer:

```
docs/specs/
├── delivery/
│   ├── docker.md
│   └── github-actions.md
├── interface/
│   └── commands.md
├── language/
│   └── go.md
├── practices/
│   ├── command-workflow.md
│   ├── bdr.md
│   ├── conventional-commits.md
│   ├── git.md
│   ├── madr.md
│   └── tdd.md
├── quality/
│   ├── command-verification.md
│   └── unit-testing.md
└── security/
    └── dependency-management.md
```

## Workflow

1. Check relevant specs in `docs/specs/`, starting with `docs/specs/interface/commands.md`
2. Write tests first (TDD: `practices/tdd.md`)
3. Run quality gates before committing (`make check`)
4. Reference specs in commit messages and PR descriptions

## Spec Traceability

Map spec rules to enforcement:

| Spec Rule | Enforcement |
|-----------|-------------|
| `commands.md` | Five primary Make targets |
| `command-verification.md` | CI five-command validation |
| `go.md:2` | `gofmt` in pre-commit + CI |
| `go.md:3` | `goimports` in pre-commit + CI |
| `go.md:12` | `go test -race` in CI |
| `go.md:15` | `go vet ./...` in pre-commit + CI |
| `go.md:16` | `staticcheck` via golangci-lint |
| `go.md:17` | Git track `go.mod` + `go.sum` |
| `unit-testing.md` | `go test -coverprofile` with threshold |
| `conventional-commits.md` | commitlint in pre-commit + CI |

## Adding a Spec

When adding a new spec:
1. Copy from [stackable-specs/specs](https://github.com/stackable-specs/specs)
2. Add references in AGENTS.md
3. Add quality gates that enforce the spec's rules
4. Create ADR if it's a significant decision
