---
title: "Add the missing loop: one that builds, not just one that files"
status: done
type: loop
outcomes: [1, 2, 3, 4, 5, 6]
created: 2026-11-18
---

# Add the missing loop: one that builds

## Problem

Every loop running today observes and reports. The verification loop files issues, the improvement loop files ideas, and the dashboard loop takes snapshots. None of them changes product code. The dashboard's `history.jsonl` shows two identical snapshots (14:59Z and 15:13Z): 0/127 acceptance criteria met, 0/33 BDRs satisfied, 36 issues open, 25 and then 29 ideas `proposed`, 0 accepted. Its verdict, "Stalled", can't change until something ships.

## Proposal

Add a build loop with a cadence of a few hours, or one that wakes when an issue becomes `accepted`:

1. Pick the highest-priority item that is unblocked and not claimed. Start with `idea-walking-skeleton-first.md`.
2. Claim it (see `idea-loop-single-writer-claims.md`), branch, write a failing test from the BDR's acceptance criteria (tdd.md), and implement.
3. Run `make ci`, open a draft PR, and tick the matching `- [x] AC-n` lines on the tracking issue.
4. Stop after one item per run, then hand off to review (human or `/code-review`).

## Acceptance

- [ ] Within a day of enabling it, the dashboard's `acDone` is above 0 and its verdict moves to Climbing.

## Evidence (2026-11-18 11:33, improvement loop run 3)

`idea-walking-skeleton-first` was promoted to `issues/delivery-walking-skeleton.md`, but nothing is building it. Four snapshots in a row show `acDone: 0`.

## Evidence (2026-11-18 13:33, improvement loop run 4)

The fixer loop has started building. It made 2 commits since the last run (`8965921` untracks the lock file; `ac4df75` renames the module and adds `cmd/arena` and `cmd/connect4`), and `make check` passes on main. `delivery-walking-skeleton.md` is still open and `acDone` is still 0. The remaining gap is that the fixer picks small enabling tasks first. Point it at the skeleton next.

## Evidence (2026-11-18 14:33, improvement loop run 5)

**Done.** The fixer loop (`d5b9c49b`) delivered the walking skeleton: `2d4aece` (engine), `2bcfa94` (game service) and `eee9bc7` (REST via Huma). It also filed its own follow-up defect, `api-invalid-column-type-lacks-version.md`. The dashboard moved from `acDone` 0 to 30 and from 4 to 14 closed issues. `make check` passes at 95.7% coverage. The next risk is how progress is *measured*; see `idea-loop-ac-test-traceability.md`.
