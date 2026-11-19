---
title: "Toolchain pin follow-ups: Renovate config and unverified Docker build"
status: closed
source: loop-run
branch: main
commit: 08957ee
found: 2026-11-18
closed: 2026-11-19
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

## Resolution

- **Docker build (item 2):** fixed and verified in 20f873e. The builder is `golang:1.27.4-trixie` pinned by digest, which also answers the rule 8 note. `make docker-smoke` passed locally on arm64 and amd64. CI run 44603243149 on f390929 passed "Build and smoke-test Docker image" on linux/amd64, with setup-go downloading go1.27.4 from the `toolchain` pin.
- **Renovate (item 1):** already covered by `.github/renovate.json`, as the earlier update notes.
- **Found while verifying:** the same CI run's "Scan dependencies" step did not scan. govulncheck v1.1.4, the Makefile pin, panicked (`unexpected expr: *ast.KeyValueExpr` in x/tools v0.29.0 `go/ssa`) on Go 1.27 standard library code. `|| echo "::warning::govulncheck found vulnerabilities"` plus `continue-on-error` hid the crash as a misleading warning. Local `make check` passed only because a separately installed govulncheck v1.9.0, built with go1.27.3, was on PATH. Built with go1.26, v1.9.0 also fails ("method must have no type parameters"), because govulncheck type-checks with the Go it was built with. Fixed in this commit:
  - `make install-tools` installs govulncheck v1.9.0 with `GOTOOLCHAIN=$(go env GOVERSION)`, so it is built with the module's toolchain (go1.27.4).
  - CI's "Scan dependencies" is now a real gate (`make vuln`, no `continue-on-error`), as dependency-management rule 14 and the README's quality gates require.
  - Verified locally: the Makefile line builds govulncheck with go1.27.4, and `make vuln` reports "No vulnerabilities found." `make check` PASS, 91.5%. Docker was not running, so I did not repeat this in a Linux container. The next CI run on main confirms it there.
