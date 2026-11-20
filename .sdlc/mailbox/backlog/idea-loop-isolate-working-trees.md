---
title: "Give each writing loop its own worktree; never leave work uncommitted in the shared checkout"
status: accepted
type: loop
outcomes: [5]
spec: [docs/specs/practices/git.md, docs/specs/practices/bdr-definition.md BDR-DEF-19..20]
created: 2026-11-18
---

# Give each writing loop its own worktree

## Problem

All loops share one checkout, `/Users/owenzanzal/CvilleAIEx/agent-connect-4` on `main`. Since 14:27, `docs/bdr/001, 002, 003, 005, 006, 008, 009` (and `004` since 15:25, plus `README.md`) have been **uncommitted** edits that flip `Proposed → Accepted`. In that time, three commits landed on top of them (`74055c8` 14:34, `883b8c8` 15:28, plus the earlier skeleton work), and each one staged around the dirty files.

Effects:

- `collect.py` reads the working tree, so the dashboard already reports `bdrAccepted: 7` and about 30 criteria met for a change no commit contains. A `git stash` or `git checkout .` by any loop would silently undo the "progress".
- BDR-DEF-20's check ("no change ... after the commit that set `Accepted`", in the new `bdr-definition.md`) cannot run, because that commit doesn't exist.
- Any loop that runs `git add -A` (a common habit) would sweep another loop's half-done edit into an unrelated commit.
- An empty `.sdlc/worktree/` appeared at 14:34, which looks like an isolation attempt that was started and left unfinished.

## Proposal

- Each loop that writes to the repo gets a dedicated worktree, `.sdlc/worktree/<loop>` (or `git worktree add -b loop/<name>`), registered in `loops.json`. It integrates into `main` with `git rebase` / `merge --ff-only`, in keeping with the owner's linear-history decision.
- Rule: a loop never ends a run with uncommitted changes in the shared checkout. It either commits, or moves the work to `wip/<loop>` with a note in its run journal.
- `collect.py` reads tracked state from `HEAD` (`git show HEAD:docs/bdr/...`) and reports dirty files separately, as "uncommitted: 9 files (owner unknown)".
- At startup, a loop that finds dirty files it didn't create raises a human-in-the-loop item instead of committing around them.

## Acceptance

- [ ] `git status --short` in the main checkout is empty between loop runs.
- [ ] The dashboard counts nothing that isn't on `HEAD`.

## Decision (2026-11-18 retro)

Accepted as part of retro item 1. Tracked in `issues/loops-adopt-coordination-protocol.md`; rules in `docs/specs/practices/loop-coordination.md`.

## Evidence (2026-11-19T21:14Z): a stray edit in the main checkout is blocking another loop's merge

At 20:01Z someone (no registered loop lists `docs/bdr` in `writes`) created `docs/bdr/034-join-open-seat-in-browser.md` and added one line to `docs/bdr/README.md`, both in the **main checkout**, and never committed them. They are still there 73 minutes later.

At 20:27Z the BDR verify loop committed `5133113 docs(bdr): accept BDR-028` on `verify/accept-028` in its own worktree. That commit also changes `docs/bdr/README.md`. Main can't take it without overwriting or conflicting with the uncommitted README line, so it has sat on its branch for 47 minutes, unmerged and diverged from main. The worktree rule did its job, since verify's own work is safe. But one session that skipped rule 9 is enough to block a rule-following loop's merge, and nothing reports it: the dashboard shows BDR-028 as not accepted, with no reason given.

Additions to the proposal:
- A loop that can't merge because main is dirty should file or update a `needs: human` issue that names the blocking paths. Today it waits without saying anything.
- `docs/bdr/README.md` is a shared index that every BDR change edits, so it is a hotspot for conflicts. Generate it from the BDR files (for example in `make health` or the dashboard's `collect.py`) rather than editing it by hand.
