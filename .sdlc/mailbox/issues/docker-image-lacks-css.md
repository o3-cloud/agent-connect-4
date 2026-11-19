---
title: "Docker image serves the browser UI without its stylesheet"
status: closed
source: progress-dashboard
type: defect
priority: 2
found: 2026-11-19
closed: 2026-11-19
fixed_in: 20f873e
spec: [ADR-012, delivery/docker.md]
bdr: [BDR-026, BDR-033]
---

# Docker image serves the browser UI without its stylesheet

`Dockerfile` runs `go build` directly, so `internal/web/static/app.css` (built by `make css` with the pinned Tailwind CLI, ADR-012) does not exist in the build stage and is not embedded. The page works but is unstyled in the container.

## Fix

In the builder stage, download the pinned Tailwind CLI for the build platform, verify it against the checksum in the Makefile, and run `make css` before `go build` (or run `make build`). Keep the version and checksums in one place (the Makefile).

## Acceptance

- [x] `make docker-build` produces an image whose `/static/app.css` returns 200 with the board styles.

## Resolution

Fixed in 20f873e. The builder stage now runs `make tailwind` (downloads and verifies the pinned CLI against the Makefile checksums) and `make build` (which runs `make css` first), so `app.css` is embedded. The builder moved to `golang:1.27.4-trixie`, pinned by digest, because the Tailwind CLI needs glibc; the runtime stage is still `scratch`. `.dockerignore` now excludes a host-built `app.css` and `.tools`.

Verified: new `make docker-smoke` (scripts/docker-smoke.sh) was red on the old image (`/static/app.css` 404) and is green on the new one. It runs the image read-only with all capabilities dropped and checks `/healthz` and the `.board`/`.cell` styles. Both the arm64 image and an amd64 image (under emulation, which also checks the linux-x64 checksum) passed. CI now runs `make docker-smoke` instead of `make docker-build`. `make check` PASS.
