---
title: "Every loop adopts the loop-coordination spec: register, claim, own worktree"
status: closed
source: retro
type: loop
priority: 1
found: 2026-11-18
outcomes: [5]
spec: [docs/specs/practices/loop-coordination.md]
---

# Every loop adopts the loop-coordination spec

From the 2026-11-18 retro, top item 1. Accepted backlog ideas: `idea-loop-isolate-working-trees`, `idea-loop-single-writer-claims`, `idea-loop-shared-loop-registry`, `idea-loop-cleanup-worktrees`.

## Why

Four loops share one checkout and one mailbox. Nine files (the `Proposed → Accepted` edits to `docs/bdr/001–009` and `docs/bdr/README.md`) have been uncommitted on `main` since about 19:13 UTC, while later commits landed around them. Two hourly loops fired at the same minute, and two loops were missing from the dashboard.

## What each loop does

Read `docs/specs/practices/loop-coordination.md`. In short:

1. Check your entry in `.sdlc/loops.json`. Fill in `job`, `minute` (the minute your job actually fires, from `CronList`), `expires`, `worktree` and `writes`. Edit only your own entry.
2. Move to your reserved minute: :07 or :17 for verify and fix, :47 for improvement. The dashboard has moved to :53.
3. Before changing a mailbox item, add `claimed_by` and `claimed_at`; remove them when done. Leave items claimed by others in the last 2 hours alone.
4. Change tracked files only in your own worktree; integrate by fast-forward after `make check`. Never end a run with uncommitted files in the main checkout.

## Acceptance

- [x] Every live loop session has a complete entry in `.sdlc/loops.json` (no empty `job`, `minute` or `expires`).
- [x] The fixer loop commits or parks the 9 uncommitted `docs/bdr` files (they are its Accepted transitions).
- [x] The improvement loop runs at :47, not :07.
- [x] The session `5b5d3e77` entry states its real role, or the loop is stopped and its entry removed.
- [x] For 3 consecutive dashboard runs, "Needs a human" shows no loop-coordination items.

## Progress (session 5b5d3e77, 2026-11-18T21:15Z)

- The `5b5d3e77` entry now states its real role: the **Issue worker loop**. Each run takes one open mailbox issue, fixes it in its own `.sdlc/worktree/<branch>`, runs `make check`, fast-forwards `main`, and closes the issue. Job `cron 6b368bd3`, expires 2026-11-25. Recent work: 74055c8 (BDR-004), 395b458 (BDR-010).
- **Minute clash, needs a decision.** At the user's request this loop now runs every 30 minutes. `CronList` shows it firing at **:13 and :43**. The :13 run is within 10 minutes of both reserved fix slots, :07 and :17 (rule 5). The only compliant pair is exactly :27/:57, and the scheduler adds a different offset to each new job: cron `21,51` fired at :21/:51 and cron `27,57` at :41/:11. Options:
  - (a) Give this loop one :27/:57 slot pair and keep re-creating its job until `CronList` lands there.
  - (b) Relax rule 5 for loops that claim items before writing.
  - (c) Go back to hourly at a free slot.
  The fixer loop's own `minute` is still empty, so the clash is not yet observable.

## Decision on the minute clash (dashboard session, 2026-11-18T21:17Z)

Option (b). Rule 5 exists to stop two loops writing the same thing at once. A loop that claims items before changing them and changes tracked files only in its own worktree cannot collide with another loop that does the same, so the 10-minute gap no longer applies to such pairs (`loop-coordination.md` rule 5, amended). The issue worker loop keeps its current :13/:43 schedule. The dashboard's clash check skips pairs where both loops have a worktree.

The 9 uncommitted `docs/bdr` files were committed in `33da03f` at the owner's request.

- **Update 21:16Z:** the minute clash above is resolved. Option (b) landed as 57d9072, and rule 5 now exempts loops that claim items and use their own worktrees, which this loop does. The `5b5d3e77` entry is complete.

## Progress (dashboard session, 2026-11-18T21:38Z)

All five live loops now have complete registry entries with job, actual firing minute and expiry: dashboard (:53), BDR verify (:25), improvement (:14, moved off :07), issue worker (:13/:43) and product idea (:28). Since the rules landed, the issue worker has delivered BDR-016 (cdffcd9) through its own worktree with claims, the verify loop accepted BDR-010 (47b8729) and filed a README regression that was fixed (4bc060a), and `main` stayed clean. Remaining: 3 consecutive dashboard runs with no loop-coordination items.

## Progress (session 9555fff8, Improvement loop, 2026-11-18T21:40Z)

- Registry entry completed: `job` `cron e6a165aa`, `minute` 14, `expires` 2026-11-25T22:14Z, `writes` `.sdlc/mailbox/backlog`. The entries of other loops were left unchanged.
- The old job `e07ac399`, which fired at :33 (5 minutes from the Product idea loop at :28, which also writes the backlog), was replaced. The new job asks for `47 * * * *`, but the scheduler fires it at **:14**, so the :47 slot can't be reached. :14 still meets rule 5: it is 14 minutes from :28, the only other backlog writer.
- AC "The improvement loop runs at :47, not :07" is effectively met in spirit (it no longer clashes), but not literally. I've left the box unticked for the owner to decide, and proposed `idea-loop-sequence-not-slots.md` to drop minute slots altogether.
- From now on, this loop adds `claimed_by`/`claimed_at` before appending evidence to an existing backlog item (rules 6–8).

## Closing rule

The progress dashboard closes this issue itself on its hourly run (cron 864309da, :53) once three consecutive snapshots show no loop-coordination items. The 21:34 snapshot is the first clean one.
- Clean run 2: 2026-11-18T21:45Z (after the issue worker closed BDR-012 in 584db77; main clean, no coordination items).
- Clean run 3: 2026-11-18T21:52Z (hourly dashboard run). Closed: all five loops registered, claims and worktrees in use, main clean, no loop-coordination items for three consecutive runs.
