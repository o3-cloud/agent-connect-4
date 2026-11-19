---
title: "Track skipped quality gates in a ledger so 'unverified' doesn't become permanent"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Track skipped quality gates in a ledger

## Problem

The Docker image build has been skipped three times in a row because "Docker was not available":

- `08957ee` (toolchain pin, now lost in the reset)
- `main-toolchain-followups.md` item 2, still open
- `ac4df75` (module rename and `cmd/` layout), which changed the Dockerfile substantially: two binaries, a new health-check path

Each time, the commit message or issue mentions it and moves on. With no remote there is also no CI, so `make check` (which passes on `ac4df75`) is the only gate. Nothing ever runs `make docker-build`, and the debt grows silently with every Dockerfile change.

## Proposal

- `make check` prints `SKIP docker-build (daemon not running)` instead of leaving it out, and appends `{gate, commit, reason, date}` to `.sdlc/mailbox/unverified.jsonl`.
- At startup, each loop checks `docker info`. If Docker is up, or can be started (`colima start` / `open -a Docker`), it runs the skipped gates first and clears the ledger entries they cover.
- The dashboard shows how many gates are unverified and the oldest skipped commit. After 2 or more skips of the same gate, it raises a human-in-the-loop item ("start Docker once so the loop can verify").

## Acceptance

- [ ] Every skipped gate leaves a ledger entry.
- [ ] The first run with Docker available clears the ledger, or files a real failure.

## Evidence (2026-11-19 05:14Z, improvement loop run 11)

The skipped Docker gate has now let a real defect through. `issues/docker-image-lacks-css.md` (priority 2): `9a7a343` added a Tailwind build step (`make css`, a pinned CLI in `.tools/`, generated `internal/web/static/app.css`, which is git-ignored). The `Dockerfile` still runs plain `go build`, so the container serves the browser UI **unstyled**. It was found by reading the code, not by running `make docker-build`, which still hasn't run once in this repo's history. That makes 4 Dockerfile-affecting changes with the gate skipped (`08957ee`, `ac4df75`, `9a7a343`, plus the toolchain follow-up). Raise the human-in-the-loop item "start Docker once" now.

## Evidence (2026-11-19T10:14Z, improvement loop run 16)

Docker is now running on this machine (`docker info` succeeds). At 10:08Z the issue worker (`5b5d3e77`) claimed `issues/docker-image-lacks-css.md` and is working on `fix/docker-image-css`. It changes `Dockerfile`, `Makefile` and `ci.yml`, and adds a new `scripts/docker-smoke.sh`. This is the first time the Docker gate can be run instead of skipped. Once the fix merges, the ledger's Docker entry should change from "skipped" to "verified at <sha>", and `make docker-build` plus the smoke script should run on every Dockerfile change.

## Evidence (2026-11-19T11:19Z, improvement loop run 17): Docker gate closed

`20f873e` (05:14 local) fixed the defect. The new `make docker-smoke` failed on the old image and passed on the new one, for both arm64 and amd64. CI now runs it in place of `make docker-build`. Separately, the BDR verify loop built the image from main and found BDR-033 AC-1 and AC-3 met. AC-2 is only partly met, blocked on persistence (BDR-018), not on Docker. **The Docker gate has gone from "skipped" to "verified at 20f873e".** It was blocked for about 24 hours on one human action, starting Docker. This is the case the ledger is meant to make visible: the blocked gate hid a real defect, and once the human action was done, the loops closed it within about an hour. Docker is no longer a reason this idea is needed. The idea still holds for other gates that loops skip.
