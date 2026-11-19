---
title: "BDR-033 not satisfied on main: Run From a Container With External Configuration"
status: open
source: bdr-verification
bdr: BDR-033
branch: main
commit: 30cd082
found: 2026-11-18
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-19T22:10Z
---

# BDR-033 not satisfied on main: Run From a Container With External Configuration

Spec: [`docs/bdr/033-containerized-deployment.md`](../../../docs/bdr/033-containerized-deployment.md)

## Expected

The application starts from its container image, with the server settings, persistence location and auth secrets supplied from outside the image.

## Observed (black-box, on main)

- `POST /api/v1/games` -> `404`; `GET /api/v1/games` -> `404` (re-verified on 4218fba). No game API exists.

## Failing acceptance criteria

- [ ] AC-1: Running the published image with documented environment variables starts a server that answers `GET /api/v1/games`.
- [ ] AC-2: Changing the configured listen port or persistence location through configuration changes the running server's behavior with no image rebuild.
- [ ] AC-3: The image contains no authentication secrets. Without configured secrets, authenticated operations are rejected.

## Reproduce

```sh
git worktree add --detach /tmp/c4-main main && cd /tmp/c4-main
make build && PORT=18089 ./bin/app &
curl -si -X POST -H 'Content-Type: application/json' -d '{"mode":"agent_vs_agent"}' http://127.0.0.1:18089/api/v1/games
```

Root cause: the Connect Four Arena is not implemented on main; main is still the stack-base-go template. Tracked by `delivery-walking-skeleton.md`.

## Progress (20f873e, 2026-11-19, BDR verify loop d5b9c49b)

I built the image with `docker build` from main and ran it:

- AC-1: met. `docker run -p 18095:8080 <image>` answers `GET /api/v1/games` with `200`, and `/static/app.css` is served with the board styles.
- AC-2: half met. `-e PORT=9000` moves the server to port 9000 without a rebuild. There is no persistence location to configure, because games are still in memory only (BDR-018). The scenario's "games survive a container restart" fails for the same reason.
- AC-3: met. The image's environment holds only `PATH`, its build history mentions no secret, and `POST /api/v1/games` without a key returns `401`. Keys are self-issued (ADR-013), so there is no auth secret to supply.

Still blocked on BDR-018 (`bdr-018-survive-server-restart.md`). Once a data directory setting exists, document it in the README and re-verify with a mounted volume.
