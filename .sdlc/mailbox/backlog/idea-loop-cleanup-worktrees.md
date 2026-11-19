---
title: "Clean up the temporary worktrees that verification loops create"
status: accepted
type: loop
outcomes: [5]
created: 2026-11-18
---

# Clean up the temporary worktrees that verification loops create

## Problem

`git worktree list` still shows `.../d5b9c49b.../scratchpad/main-wt` (detached at `30cd082`). The verification run that created it never removed it. Stale worktrees pin old commits, confuse the next `git worktree add` at the same path, and turn into dangling entries when `/tmp` is cleared.

## Proposal

- The verification script creates the worktree and removes it with a `trap 'git worktree remove --force "$wt"' EXIT`.
- Loops run `git worktree prune` at startup and report any other leftover `loop`/scratchpad worktrees.

## Acceptance

- [ ] After a verification run, `git worktree list` shows only the main checkout.

## Evidence (2026-11-18 10:33, improvement loop run 2)

The verification worktree `.../d5b9c49b.../scratchpad/main-wt` is still registered and was moved forward to `08957ee` rather than removed. Reuse without cleanup is now the established pattern.

## Evidence (2026-11-18 14:33, improvement loop run 5)

The worktree now follows HEAD (it's at `eee9bc7`), so the fixer reuses it on purpose as its black-box verification checkout. That's reasonable. Reframe the idea: record the path in `loops.json` as owned by the fixer, and clean it up only when the owning loop expires.

## Decision (2026-11-18 retro)

Accepted as part of retro item 1. Tracked in `issues/loops-adopt-coordination-protocol.md`; rules in `docs/specs/practices/loop-coordination.md`.
