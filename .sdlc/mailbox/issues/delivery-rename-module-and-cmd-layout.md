---
title: "Rename the Go module and add a cmd/ layout before the walking skeleton"
status: closed
source: backlog
type: delivery
priority: 1
branch: main
commit: 8965921
found: 2026-11-18
outcomes: [1, 6]
spec: [ADR-001, ADR-010, CLI-001, docs/specs/interface/commands.md]
---

# Rename the Go module and add a cmd/ layout

Promoted from `backlog/idea-rename-module-and-cmd-layout.md`.

## Why

`go.mod` still declares the template's module path, `github.com/stackable-specs/stack-base-go`, and `make build` builds `./src` into `bin/app`. The walking skeleton will add imports in every new package, so the rename is cheapest before it lands. The `connect4` CLI (CLI-001, ADR-010) also needs a home.

## Scope

- Module path: `connect4arena` (a short local path; there is no remote yet).
- `cmd/arena/`: the server (moved from `src/`). `cmd/connect4/`: the CLI, a Cobra root command with help only for now (ADR-010).
- Makefile: `build` produces `bin/arena` and `bin/connect4`; `run` starts `bin/arena`. The five commands keep their meaning (ADR-001).
- Dockerfile: build both binaries and ship both in the image.
- Remove `src/`, and update `AGENTS.md`, `README.md` and `.cursorrules` where they name the old path or layout.

## Acceptance

- [x] `grep -r stack-base-go` finds nothing outside text that describes where the project came from.
- [x] `make build` produces `bin/arena` and `bin/connect4`, and `bin/connect4 --help` exits 0.
- [x] `make check` passes.
- [ ] The Dockerfile builds both binaries (verify with `make docker-build` when Docker is running; otherwise note it as unverified).

## Resolution

Fixed on main in ac4df75 (2026-11-18). Module `connect4arena`; server in `cmd/arena`, Cobra CLI in `cmd/connect4` backed by `internal/cli` (tested in-process: help, no-args help, unknown subcommand exits non-zero). `make build` -> `bin/arena`, `bin/connect4`; `bin/connect4 --help` exit 0; `make run` serves `/` 200, `/healthz` 204, unknown 404; `make check` PASS (100% coverage, govulncheck clean); `make health` no issues. Remaining `stack-base-go` mentions describe origin (README, AGENTS.md) or are accepted ADR history (003, 006).

Unverified: `make docker-build` (Docker not running). Tracked in `main-toolchain-followups.md`.
