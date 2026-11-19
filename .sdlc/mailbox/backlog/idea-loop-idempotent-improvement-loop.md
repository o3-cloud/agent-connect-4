---
title: "Make the hourly improvement loop idempotent: fingerprint inputs and dedupe ideas"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Make the hourly improvement loop idempotent

## Problem

The "propose ways to improve the loops" loop runs every hour and writes one file per idea. Without a guard, each run reworks the same evidence and adds near-duplicate ideas under new names.

## Proposal

- At the start of each run, compute a fingerprint of the inputs: main's HEAD, the current branch's HEAD, the mailbox file list and mtimes, and the newest session start time. Store it in the last run journal entry (see `idea-loop-keep-run-journal.md`). If the fingerprint hasn't changed, record a no-op and stop.
- Use stable `idea-loop-<slug>.md` file names, and skip any slug that already exists (the writer already does this). When there is new evidence for an existing idea, append it under `## Evidence (YYYY-MM-DD)` instead of creating a new file.
- Respect the backlog cap in `idea-loop-mailbox-lifecycle-in-health.md`.

## Acceptance

- [ ] Two consecutive runs with no repo activity change no files.

## Evidence (2026-11-18 10:33, improvement loop run 2)

`.sdlc/dashboard/collect.py` appends a snapshot to `history.jsonl` on every run, even when nothing has changed (the 14:59Z and 15:13Z snapshots are identical). The same fingerprint guard should apply there.
