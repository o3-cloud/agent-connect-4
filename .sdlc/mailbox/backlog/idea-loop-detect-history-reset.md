---
title: "Detect a git history reset or lost remote, and stop the loops before they build on it"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Detect a git history reset or lost remote

## Problem

At 11:06 local time (16:06Z) on 2026-11-18, the repo was re-initialized. `git reflog` holds a single entry, `commit (initial): init`. `main` is now one commit, `4218fba`. The earlier history is gone from this clone: 12 commits on main including PR #1 merge `30cd082`, `08957ee`, the BDR merge, the SPEC error-code commit `d6b9f25`, and branch `docs/consolidate-agents-md`. `git remote -v` is empty.

None of the four loops noticed. The verification loop re-ran against `4218fba` and filed `main-tracked-claude-lock-file.md` (the init commit tracks `.claude/scheduled_tasks.lock`), but it never mentioned that history had been replaced.

The reset may have been deliberate. Even so, the loops should say so and not carry on as if nothing happened.

## Proposal

- Every loop records a repo identity in the run journal or registry: the root commit SHA (`git rev-list --max-parents=0 HEAD`), the `origin` URL, and main's HEAD.
- At startup, if the root commit or remote changed, or if the previous main HEAD isn't an ancestor of the current one, the loop does three things:
  1. Writes `issues/main-history-rewritten.md` with `needs: human`, the old and new SHAs, and how to recover (`git fsck --lost-found`, GitHub `origin`, `/private/tmp/.../main-wt`).
  2. Marks every issue that cites a now-unreachable SHA `stale: history-rewritten`.
  3. Exits as a no-op until a human acknowledges the change.

## Acceptance

- [ ] A forced reset of main produces exactly one human-in-the-loop issue within one loop cycle.
- [ ] No loop files new findings against the new history until that issue is acknowledged.

## Open question for a human

Was the 11:06 reset intentional, for example to detach from the `stackable-specs` origin? If not, the old objects can still be recovered from the GitHub remote and from the `main-wt` worktree's object store.

## Answer (2026-11-18)

Yes, intentional. The project owner re-initialized the repo to detach it from the `stackable-specs/stack-base-go` template, and has no remote for now. All content from the old history is in `4218fba init`. Commit SHAs cited in older issues (`08957ee`, `c89a57e`, `d6b9f25`, `30cd082`) no longer exist, but their changes are all present. The detection proposal stays open as a loop improvement.

## Evidence (2026-11-18 13:33, improvement loop run 4)

**Open question answered:** the reset was intentional. `hitl-notes.json` says it was "re-initialized ... as its own history ... so it is no longer tied to the stack-base-go template". The idea still stands. No loop *noticed* the reset; the human recorded it. A root-commit check would have made the loops flag it instead of carrying on.
