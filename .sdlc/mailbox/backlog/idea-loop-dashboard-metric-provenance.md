---
title: "Count progress on the dashboard only when it reaches the reviewed remote"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Count progress on the dashboard only when it reaches the reviewed remote

## Problem

`history.jsonl` around the reset:

| at (Z) | issuesClosed | bdrOnMain | mainAheadOfOrigin | branchAheadOfMain | commits24h |
|---|---|---|---|---|---|
| 15:13 | 2 | 0 | 1 | 7 | 9 |
| 15:58 | 4 | 33 | 11 | 0 | 12 |
| 16:13 | 4 | 33 | **0** | 0 | 1 |

- `bdrOnMain` rose from 0 to 33 because commits merged into *local* main, which was 11 commits ahead and unreviewed. That counts as "Climbing".
- `mainAheadOfOrigin` dropped to 0 only because `origin/main` no longer exists. `collect.py` turns the git error into `"0"` (`git(...) or "0"`). So the human-in-the-loop item "unpushed commits made directly on main" quietly disappeared, even though nothing had been pushed.
- `commits24h` fell from 12 to 1, which would read as Slipping if anything used it, yet no metric flagged the reset.

## Proposal

- In `collect.py`, treat a missing `origin` or `origin/main` as a high-severity human-in-the-loop item, not as 0.
- Measure "on main" against `origin/main`, and only for changes that arrived through a merged PR (`gh pr list --state merged`), with local-only progress shown separately.
- Add the repo identity (root SHA) to each snapshot, and break the trend line when it changes, rather than comparing across unrelated histories.

## Acceptance

- [ ] With the remote removed, the dashboard shows a high-severity item, not a zero.
- [ ] The verdict can't reach Climbing on local-only, unreviewed commits.

## Evidence (2026-11-18 13:33, improvement loop run 4)

Scope reduced to match the owner decision (local main, no remote). Keep the "missing remote is an explicit state, not 0" part. `collect.py` now has a comment saying unpushed commits "are expected while origin is the template", but that is no longer accurate: there is no origin at all. Drop the "count only via merged PR" part until a remote exists.

## Evidence (2026-11-18 17:25, improvement loop run 7)

A stability gate was met in 18 minutes. `loops-adopt-coordination-protocol` closes after "3 consecutive clean dashboard snapshots", and those were taken at 21:34, 21:45 and 21:52Z: two of them ad hoc, between hourly runs. The gate was meant to show that the rules hold over time, so state such gates in elapsed time or scheduled runs ("clean across 3 hourly runs" or "for 3 hours"), not in snapshot count.
