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

## Evidence (2026-11-20T04:14Z, improvement loop)

Two sessions that aren't in the registry now own worktrees, and their work overlaps:

- `8e59714e`: `scratchpad/wt-bdr` on `bdr/034-036-browser-seats`, commit `5be86ea` (20:49 local), which adds BDR-034 to BDR-036. Its draft of BDR-034 was the uncommitted file that blocked main from 20:01Z to 01:49Z. It moved the draft into its own worktree and committed it there, so the work was **not lost**, though the improvement loop had worried it might be.
- `bd9e2cc7`: `scratchpad/seat-wt` on `feat/browser-join-seat`, commit `c054545` (21:53 local), "let a visitor take an open seat (BDR-034)".

`feat/browser-join-seat` is **not** based on `bdr/034-036-browser-seats`, and BDR-034 isn't on main. So the code is being written against a BDR that no reviewer or loop can see on main. Neither session claims a mailbox item, and the BDR verify loop can't check AC coverage for a BDR it has never seen. If these are the owner's interactive sessions, that's fine, but the registry should still have a lightweight entry for them (`loop: "owner interactive"`, `worktree`, `writes`), so the dashboard can tell an owner's work apart from a stray session and loops avoid editing the same files.
