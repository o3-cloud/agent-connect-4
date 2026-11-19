---
title: "make check fails on main: govulncheck reports reachable stdlib vulnerabilities"
status: closed
source: bdr-verification
branch: main
commit: 30cd082
found: 2026-11-18
---

# make check fails on main: govulncheck reports reachable stdlib vulnerabilities

## Observed

With `GOTOOLCHAIN=go1.24.0`, fmt, vet, lint, test and coverage (100%) pass, then:

```
govulncheck ./...
Your code is affected by 40 vulnerabilities from the Go standard library.
  e.g. GO-2026-6859, GO-2026-6809, GO-2026-6617
  #1: src/main.go:24:30: src.main calls http.Client.Get, which eventually calls http.ProxyFromEnvironment
make: *** [vuln] Error 3
```

## Expected

`make check` prints `PASS` on main (spec: `docs/specs/interface/commands.md` check; `docs/specs/security/dependency-management.md`).

## Fix

Pin a patched toolchain in `go.mod` (`toolchain go1.<latest patched>`) and keep it current via Renovate.

## Resolution

Fixed on main in 08957ee (2026-11-18): `go 1.24.0` + `toolchain go1.27.4`; Dockerfile builder `golang:1.27.4-alpine`. Verified `make build`, `make run`, `make check` (PASS, govulncheck clean), `make health`, `make clean` from local go1.22.2 with GOTOOLCHAIN=auto.
