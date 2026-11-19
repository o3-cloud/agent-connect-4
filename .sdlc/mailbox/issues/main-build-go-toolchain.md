---
title: "make build fails on main: go.mod `go 1.24` names a nonexistent toolchain"
status: closed
source: bdr-verification
branch: main
commit: 30cd082
found: 2026-11-18
---

# make build fails on main: go.mod `go 1.24` names a nonexistent toolchain

## Observed

Following README "Getting Started" step 1 with a local Go older than 1.24 (here go1.22.2, `GOTOOLCHAIN=auto`):

```
$ make build
go: downloading go1.24 (darwin/arm64)
go: download go1.24 for darwin/arm64: toolchain not available
make: *** [build] Error 1
```

`make check` fails the same way at `go vet`, and `make health` reports `go vet failed` / `go test failed`.

## Expected

`make build` succeeds for a new contributor (spec: `docs/specs/interface/commands.md` build; ADR-001).

## Cause and fix

`go.mod` declares `go 1.24`. The auto-toolchain switch looks for release `go1.24`, which does not exist (the first release is `go1.24.0`). Declare a full version (`go 1.24.0`) and/or add a `toolchain go1.24.x` line. Builds succeed with `GOTOOLCHAIN=go1.24.0`.

## Resolution

Fixed on main in 08957ee (2026-11-18): `go 1.24.0` + `toolchain go1.27.4`; Dockerfile builder `golang:1.27.4-alpine`. Verified `make build`, `make run`, `make check` (PASS, govulncheck clean), `make health`, `make clean` from local go1.22.2 with GOTOOLCHAIN=auto.
