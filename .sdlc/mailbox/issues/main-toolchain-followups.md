---
title: "Toolchain pin follow-ups: Renovate config and unverified Docker build"
status: open
source: loop-run
branch: main
commit: 08957ee
found: 2026-11-18
---

# Toolchain pin follow-ups: Renovate config and unverified Docker build

## Observed

08957ee pinned `toolchain go1.27.4` in `go.mod` and moved the Dockerfile builder to `golang:1.27.4-alpine`. Two things were left open:

1. No Renovate (or Dependabot) config exists in the repo, so the toolchain pin and the `golang` image tag will go stale and govulncheck will start failing `make check` again when the next stdlib CVE lands.
2. Docker was not available in the run that made the change, so `make docker-build` with `golang:1.27.4-alpine` has not been verified. CI (`actions/setup-go` v5.5.0 with `go-version-file: go.mod`) has also not run on the new pin.

## Expected

The toolchain and base image are kept current automatically (spec: `docs/specs/security/dependency-management.md`), and the container image builds (spec: `docs/specs/delivery/docker.md`).

## Fix

- Add `renovate.json` covering the `gomod` (incl. `toolchain` directive) and `dockerfile` managers.
- Run `make docker-build` and a CI run on main; fix any failures.

## Update (2026-11-18)

- Item 1 was wrong: `.github/renovate.json` exists and extends `config:recommended`, which enables the `gomod` manager (it updates the `go` and `toolchain` directives) and the `dockerfile` manager. No config change needed.
- Item 2 is still open: the Docker daemon was not running, so `make docker-build` with `golang:1.27.4-alpine` is still unverified. Run it once Docker is available.

## Update 2026-11-18 (ac4df75)

The Dockerfile now builds `bin/arena` and `bin/connect4` with `-mod=readonly` and ships both in `/app`; `HEALTHCHECK` runs `/app/arena -healthcheck`. `make docker-build` is still unverified (Docker not running). Note dependency-management rule 8: the base image is pinned by tag (`golang:1.27.4-alpine`), not by digest.
