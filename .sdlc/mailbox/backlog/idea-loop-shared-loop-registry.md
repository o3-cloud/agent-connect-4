---
title: "One loop registry with staggered cadences"
status: accepted
type: loop
outcomes: [5]
created: 2026-11-18
---

# One loop registry with staggered cadences

## Problem

Two session-only hourly loops both fire at minute :07: the dashboard (`cron 387f1c4c`) and this improvement loop (`cron e07ac399`). Both read and write `.sdlc/`. `.sdlc/dashboard/loops.json` lists only the dashboard loop. The verification loop and this one show up only as "observed" mailbox writers. No one place says which loops exist, who runs them, when they fire, or when they expire. Each session-only cron job also expires silently after 7 days.

## Proposal

- Make `.sdlc/loops.json` the single registry. Each entry records the loop's name, purpose, owner session, cron id, cadence, inputs, outputs (the folders it writes), `expires`, and `last_run`.
- Each loop registers or updates its own entry on every run, and the dashboard reads the registry.
- Stagger the start minutes so that dependent loops run in order (for example :07 verify, :17 build, :37 dashboard, :47 improve). Then the dashboard sees fresh results, and the improvement loop sees the dashboard.
- Flag entries whose `expires` is less than 24 hours away.

## Acceptance

- [ ] Every running loop appears in the registry with a cron id and an expiry.
- [ ] No two loops that write the same folder fire in the same minute.

## Evidence (2026-11-18 13:33, improvement loop run 4)

Partly adopted. `.sdlc/dashboard/loops.json` now lists 2 loops with their session ids (the dashboard is `4e529a7f`, the fixer is `d5b9c49b`). This improvement loop (`9555fff8`, cron `e07ac399`, hourly at :07) is missing. It runs in the same minute as the dashboard, and the registry has no cron ids or expiry dates.

## Decision (2026-11-18 retro)

Accepted as part of retro item 1. Tracked in `issues/loops-adopt-coordination-protocol.md`; rules in `docs/specs/practices/loop-coordination.md`.
