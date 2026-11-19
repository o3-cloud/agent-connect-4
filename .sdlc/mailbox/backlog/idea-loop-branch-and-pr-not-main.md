---
title: "Loops commit on a branch and open a PR, never directly on main"
status: deferred
deferred_until: a git remote exists
type: loop
outcomes: [5]
spec: [docs/specs/practices/git.md]
created: 2026-11-18
---

# Loops commit on a branch and open a PR, never directly on main

## Problem

`08957ee fix: pin go 1.24.0 and patched go1.27.4 toolchain` was committed straight onto local `main` (author `owen.zanzal`, which differs from the usual `Owen Zanzal`). It has not been pushed, and `origin/main` is still `30cd082`. CI has never run on that change, and the follow-up issue `main-toolchain-followups.md` says so. Meanwhile, `docs/consolidate-agents-md` is 5 commits ahead with no PR, so main has no BDRs (`main-bdrs-not-on-main.md`).

## Proposal

- Loop instructions: if HEAD is `main`, create `loop/<slug>-<date>` before committing. Push it and open a draft PR with `gh pr create --draft`. Refuse to commit on `main`.
- Add a Claude Code `PreToolUse` hook that blocks `git commit` when the current branch is `main`.
- An hourly loop reports unmerged branches that are ahead of main and older than 2 hours.

## Acceptance

- [ ] A hook blocks a commit on main.
- [ ] Each loop change shows up as a PR with CI results.

## Evidence (2026-11-18 11:33, improvement loop run 3)

The remote is now gone (`git remote -v` is empty), so no loop can open a PR at all. The new `init` commit was made directly on main and tracks `.claude/scheduled_tasks.lock` (see `issues/main-tracked-claude-lock-file.md`).

## Evidence (2026-11-18 13:33, improvement loop run 4)

**Deferred by an owner decision.** `hitl-notes.json` records: "Project owner: no remote for now. Work lands on local main with linear history; revisit when a remote is added." The fixer loop now follows that rule (`8965921` and `ac4df75` are on main with clear messages that cite the spec and reference what they resolve). Revisit when a remote is added. Until then, the local gate is `make check` plus `idea-loop-unverified-gates-ledger.md`.
