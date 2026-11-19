---
title: "Re-verify open issues when main moves, and close the ones that are fixed"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Re-verify open issues when main moves, and close the ones that are fixed

## Problem

37 issues record `commit: 30cd082`, but local main is now at `08957ee`. Nothing in the loop re-checks an issue after main changes, so issues go out of date silently and never close on their own. 36 are open and 2 are closed; the closed ones were closed by hand.

## Proposal

- Each issue keeps `last_verified_commit:`.
- When main's HEAD differs from `last_verified_commit`, the verification loop re-runs that issue's reproduction steps. If they pass, it sets `status: closed` and adds `closed_by: <sha>`. If they still fail, it bumps `last_verified_commit`.
- If HEAD hasn't changed since the last run, the loop does nothing and records the run as `noop: true`.

## Acceptance

- [ ] After a commit that fixes an issue, the next loop run closes it with evidence.
- [ ] Two runs on an unchanged HEAD make no file changes.

## Evidence (2026-11-18 11:33, improvement loop run 3)

37 issues still say `commit: 30cd082`, which no longer exists in this repo (see `idea-loop-detect-history-reset.md`). Only the two new issues cite `4218fba`.

## Evidence (2026-11-18 14:33, improvement loop run 5)

Closing an issue doesn't update its body. `bdr-001` and `bdr-005` are now `status: closed`, but their "Observed (black-box, on main)" sections still say "`POST /api/v1/games` -> `404` ... No game API exists". When a loop closes an issue, it should replace "Observed" with a "Verified" section (commit, command, result), so the file doesn't contradict its own status.
