---
title: "Add `make ci` that runs exactly what GitHub Actions runs"
status: proposed
type: loop
outcomes: [5]
spec: [docs/specs/quality/command-verification.md]
created: 2026-11-18
---

# Add `make ci` that runs exactly what GitHub Actions runs

## Problem

On 2026-07-05, between 09:27 and 09:41, main took six back-to-back `fix:` commits (`6dd65ac` to `0d1c336`) chasing CI failures in Go version, golangci-lint v2, govulncheck, commitlint config, commitlint fetch depth and subject-case. Each failure was found only after pushing. `make check` doesn't run commitlint or the Docker build, but CI does, so local runs and CI disagree.

## Proposal

- Add a `make ci` target that runs the same steps as `.github/workflows/ci.yml`: install-tools, fmt-check, test, coverage, vet, lint, vuln, commitlint over `origin/main..HEAD`, and docker-build when Docker is available. Then have the workflow call `make ci`, so there is a single source of truth.
- Loops run `make ci` before pushing and report any step they skipped (for example, "docker unavailable").

## Acceptance

- [ ] `ci.yml` calls `make ci` and contains no copy of the steps.
- [ ] A loop never pushes a change that fails `make ci` locally.

## Evidence (2026-11-18 13:33, improvement loop run 4)

With no remote, there is no CI at all, so `.github/workflows/ci.yml` never runs. `make check` is now the only quality gate, which makes it more important that it covers everything CI did (commitlint, docker-build).
