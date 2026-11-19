---
title: "Rename the Go module and add a cmd/ layout before the walking skeleton lands"
status: promoted
type: architecture
outcomes: [1, 6]
spec: [CLI-001, NFR-008, docs/specs/language/go.md]
bdr: [BDR-019, BDR-033]
created: 2026-11-18
---

# Rename the module and add a cmd/ layout first

## Problem

The repo was re-initialised as its own project (one `init` commit), but it still carries the template's identity:

- `go.mod` declares `module github.com/stackable-specs/stack-base-go`.
- `make build` builds `./src` into `bin/app`. There is nowhere for the `connect4` CLI (CLI-001, ADR-010) or a house-agent binary to go.

`issues/delivery-walking-skeleton.md` is about to add `internal/engine`, `internal/game` and `internal/api`. Every import it writes would use the template path, so renaming later means touching every file.

## Proposal

Do this in one small `refactor:` commit before the skeleton branch:

- Rename the module to a project path (for example `connect4arena` or a future `github.com/<owner>/connect4-arena`). With no remote yet, a short non-URL path is fine, and `go.mod` is the only place it is defined.
- Layout:
  - `cmd/arena/`: the server (REST, MCP over HTTP, web UI)
  - `cmd/connect4/`: the CLI (Cobra)
  - `cmd/house-agent/`: added later
- Makefile: `build` produces `bin/arena` and `bin/connect4`; `run` starts `bin/arena`. The five-command interface stays as it is (ADR-001).
- Dockerfile and compose: build both binaries, and ship `connect4` in the image so `docker compose exec arena connect4 ...` works.
- Remove `src/` and update `AGENTS.md`, `README.md` and the template's `.cursorrules`.

## Acceptance

- [ ] `grep -r stack-base-go` finds nothing outside the docs that describe where the project came from.
- [ ] `make build` produces `bin/arena` and `bin/connect4`, and `bin/connect4 --help` exits 0.
- [ ] `make check` passes and the Docker image contains both binaries.

## Promoted (2026-11-18)

Now tracked as `issues/delivery-rename-module-and-cmd-layout.md`.
