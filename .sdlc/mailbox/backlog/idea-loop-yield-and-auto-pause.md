---
title: "Measure each loop's yield and pause loops that stop paying for themselves"
status: proposed
type: loop
outcomes: [5, 6]
created: 2026-11-18
---

# Measure each loop's yield and pause loops that stop paying for themselves

## Problem

Five Claude Code sessions now run loops against this repo. Between them they fire about 6 times an hour, around the clock, until 2026-11-25. Their yields differ widely:

- **Issue worker** (`5b5d3e77`): about 15 commits today, including the skeleton, auth, history and resign. It moved the dashboard from 0 to 42 criteria met.
- **BDR verify** (`d5b9c49b`): accepted 10 BDRs and found real defects (for example `api-invalid-column-type-lacks-version`).
- **Improvement loop** (this one, `9555fff8`): 6 runs, 24 `idea-loop-*` files. 4 accepted, 1 done, 1 deferred, the rest untriaged. Later runs mostly appended evidence.
- **Product ideas** (`432272f0`): adds to a backlog that is already over its cap.

No loop records how long it ran or what it cost, and nothing pauses a loop whose output nobody uses.

## Proposal

- Each run appends `{loop, at, duration_s, noop, outputs: {commits, issues_closed, issues_filed, ideas_filed, evidence_appends}}` to `.sdlc/runs.jsonl` (this extends `idea-loop-keep-run-journal.md`).
- The dashboard shows yield per loop for the last 24 hours, plus the share of its ideas or issues that were accepted or closed.
- **Auto-backoff:** after 3 runs in a row with `noop: true`, or when less than 20% of a loop's output was triaged in 24 hours, the loop doubles its interval, up to 4 hours. Any new upstream event resets it.
- Discovery loops (improvement, product ideas) pause while the backlog is over its cap, and resume once triage brings it back under.

## Acceptance

- [ ] The dashboard shows each loop's runs, no-ops and outputs for the last 24 hours.
- [ ] A discovery loop with untriaged output backs off without anyone stepping in.

## Evidence (2026-11-19 03:28Z, improvement loop run 10)

Catch-up firings pile up. Because session cron jobs wait for the REPL to be idle, this loop's job `e6a165aa` delivered **three** prompts back to back (two together, then a third partway through the run) after the session had been busy. Each would have re-run the whole review. A loop should drop a firing if it ran less than half a cadence ago, and treat firings queued in a burst as a single run.
