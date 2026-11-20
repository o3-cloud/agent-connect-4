---
title: "Update the coordination rules now that .sdlc is tracked: every mailbox write is a tracked change on main"
status: proposed
type: loop
outcomes: [4, 5]
spec: [docs/specs/practices/loop-coordination.md rules 6-12]
created: 2026-11-19
---

# Update the coordination rules now that `.sdlc` is tracked

## Problem

At 07:01 local (12:01Z) on 2026-11-19 the owner committed `6e8699e Add .sdlc dir`: 114 files, covering the whole mailbox (46 backlog items and 51 issues), `.sdlc/loops.json`, and the dashboard's inputs and generated output (`history.jsonl`, `out/*.json`, `page/*`). Only `.sdlc/worktree/` is still ignored. That settles [[idea-loop-version-the-mailbox]]: a fresh clone can now read the mailbox.

But `loop-coordination.md` was written when `.sdlc` was git-ignored, and several of its rules now conflict with each other:

- **Rule 9** ("never edits tracked files in the main checkout") now covers every claim, release, evidence append, new backlog item, registry update and dashboard snapshot. Every loop does these in the main checkout, because the mailbox is shared state and a worktree copy would be invisible to the other loops until it merges.
- **Rule 11** ("end every run clean or committed") is broken by every run of every loop unless that loop commits.
- **Rule 12** tells a loop that finds uncommitted changes it didn't make to file `needs: human` and leave them. Now that mailbox edits are tracked, every loop will find another loop's uncommitted mailbox edits almost every run.
- **The loops adapted in different ways.** The issue worker now commits each claim and resolution straight to main (`0711be5 chore(sdlc): claim ...`, `3daca75 chore(sdlc): record fixed_in ...`). The other loops (dashboard, BDR verify, product ideas, and this improvement loop) have no rule for this yet. The dashboard's next snapshot will leave `history.jsonl` and 7 `out/*.json` files modified on main every hour.
- **Generated output is now tracked.** `out/*.json` and `page/*` are rebuilt from the mailbox each run, so committing them adds about 8 files of churn per hour to main's history and makes merge conflicts likely.
- **Commit races.** Two loops committing to main in the same minute can collide on `.git/index.lock`. And `git commit -a`, or a plain `git commit` after another loop's `git add`, would sweep up the other loop's staged mailbox edits. That is the "sweep edits into unrelated commits" failure this spec was written to prevent.

## Proposal

1. **Add a mailbox exception to rule 9.** Files under `.sdlc/mailbox/` and a loop's own `.sdlc/loops.json` entry are edited in the main checkout, and each write is committed straight away with a **path-limited** commit: `git commit -m "chore(sdlc): <verb> <item>" -- <exact paths>`. Never `-a`, and never a bare `git commit`.
2. **Add one helper for this,** `scripts/mailbox-commit <message> <path>...`. It retries on `index.lock` (3 tries with backoff), refuses paths outside `.sdlc/`, and refuses if a path has changes the loop didn't make (it compares against the hash recorded at claim time). Every loop calls it instead of improvising.
3. **Stop tracking generated dashboard output.** Ignore `.sdlc/dashboard/out/` and `.sdlc/dashboard/page/` again, since `collect.py` rebuilds them. Keep `history.jsonl` tracked (it is real history) and commit it once per snapshot.
4. **Narrow rule 12** to uncommitted changes *outside* `.sdlc/mailbox`. An uncommitted mailbox edit older than one cadence counts as a stale claim, not a decision for a human.
5. **Record the convention once** in `docs/specs/practices/git.md`: `chore(sdlc):` commits for mailbox traffic, so `git log --invert-grep --grep='^chore(sdlc)'` shows product history only.

## Acceptance

- [ ] After every loop has run once, `git status --short` on main is empty.
- [ ] No commit on main mixes `.sdlc/mailbox` paths with paths owned by another loop.
- [ ] `git ls-files .sdlc/dashboard/out .sdlc/dashboard/page` is empty, or the owner records that keeping them tracked is intentional.
- [ ] `loop-coordination.md` rules 9, 11 and 12 no longer conflict for mailbox writes.

## Note from the improvement loop

This loop's own writes have the same problem: this file is untracked on main until it is committed. This loop doesn't commit without the owner's go-ahead, so it is left untracked and listed in the run report.

## Evidence (2026-11-19T13:14Z, improvement loop run 19)

The dashboard has started committing as well: `243ea6b chore(sdlc): dashboard snapshot 2026-11-19T12:53Z`. The commit was scoped correctly. It touched only `.sdlc/dashboard/` and left this loop's untracked file alone, so proposal 1 is already in practice for two of the five loops. As predicted in proposal 3, 7 of its 10 files are generated output (`out/*.json`, `assets.json`): 69 of 78 changed lines are derived data, and the snapshot's real content is one line of `history.jsonl`. At hourly cadence that comes to about 24 churn commits a day. Rules 9, 11 and 12 in `loop-coordination.md` have not been amended yet.

## Evidence (2026-11-19T14:19Z, improvement loop run 20)

The workaround isn't holding. The dashboard committed its 12:53Z snapshot (`243ea6b`). Its next snapshot, taken at 14:00Z rather than its usual :53, was **not** committed, and 8 tracked files (`history.jsonl` and 7 `out/*.json`) are left modified on main. So a single loop commits on one run and not the next. Without a rule and a helper (proposals 1 and 2), each run does whatever its session happens to remember.

At the same time, the issue worker (`:08`) has made no claim or commit for two runs (13:08Z and 14:08Z) while 10 issues are open and unclaimed. Main has been dirty since 12:19Z, first from this loop's untracked file and then from the dashboard's files. That may be why it is holding back, but no `needs: human` issue was filed, so the cause can't be confirmed. Whatever the cause, it has now missed two runs, which is the threshold in [[idea-loop-one-role-per-session-and-watchdog]].

This loop is holding back too: rather than append evidence to tracked backlog files and make main dirtier, it writes only to this untracked file.

## Evidence (2026-11-19T15:14Z, improvement loop run 21): a correction, and a real race

**Correction.** My 14:19Z claim that the issue worker had "missed two runs" was wrong. It claimed `move-rejections-before-service-unlogged` at 14:25Z (`9968416`), six minutes after I checked, and fixed it by 14:29Z (`59ee859`). It then fixed `bdr-025` at 15:10Z. A dirty main didn't stop it. This is the second time this loop has called a loop stuck from a single observation (the first was 03:28Z; see [[idea-loop-one-role-per-session-and-watchdog]]). The rule this loop now follows: never report a loop as stalled until the `last_run` heartbeat (watchdog proposal 2) shows it, or until three runs have passed with no activity.

The dashboard's uncommitted 14:00Z snapshot was folded into its next commit (`cd105ac`, 14:52Z). So the dashboard commits at :53, and the 14:00Z run was an off-schedule extra.

**The predicted commit race happened.** At 14:26Z the BDR verify loop committed `8f8e519` and `57d6223` to main while the issue worker was mid-fix. The worker had to rebase, and the `fixed_in` SHA it had already committed (`8e4fd8f`) pointed at the pre-rebase commit. It took a corrective commit (`860ef37 correct fixed_in ... after rebase`). Any SHA recorded before the final rebase is unsafe. The helper in proposal 2 should record `fixed_in` only after the product commit has landed on main, by reading `git rev-parse main` after the fast-forward.

**Who commits:** the issue worker, the dashboard and BDR verify commit their own `.sdlc` writes. The product idea loop (`:28`) left `idea-load-test-with-agents.md` modified at 14:28Z, and this loop leaves its file untracked. That's 3 of the 5 loops, a working convention that no rule has written down yet.

## Evidence (2026-11-19T20:14Z): rule 12 is being worked around in practice

`f390929 chore(sdlc): commit pending backlog ideas and evidence notes` (14:54 local, next to the dashboard's 14:53 snapshot) committed four mailbox files written by *other* loops: this loop's two untracked ideas and the product idea loop's edits to `idea-load-test-with-agents.md` and `idea-wait-for-turn.md`. The message names each file and its source, and the commit lists exact paths, so it was careful. But rule 12 says not to commit changes you didn't make. That makes it the first sweep commit, the failure this spec was written to prevent, and it happened only because the rule had become unworkable for mailbox files. Proposal 4 (narrow rule 12 to non-mailbox paths) would make this legitimate. A related gap: the claim-and-commit helper should record which loop wrote each file, because a sweep commit hides that.

A different case is now open on main: `docs/bdr/034-join-open-seat-in-browser.md` (new) and `docs/bdr/README.md` (modified), last touched 20:01Z and idle for 13 minutes. These are **product docs, not mailbox files**, edited in the main checkout. Rule 9 still applies to them, and this is the case rule 12 is for. No loop's registry entry lists `docs/bdr` in `writes`.

## Evidence (2026-11-20T05:14Z): the product idea loop is the one loop still not committing

Four of the five loops now commit their own `.sdlc` writes with path-limited commits. This loop started with `a5eca37` and `6afb3e9`. The product idea loop (`432272f0`, `:28`) still doesn't. Its 04:28Z run left **12 backlog files** modified on main (87 lines added, 13 removed), unchanged 46 minutes later. Most of the change is a re-ranking: it rewrites `rank:` in each item's frontmatter, adds the same line ("All 33 BDRs are accepted (5477d60), so ranks now follow ...") to 11 files, and sets one `status: promoted`. Since `6e8699e` its edits have reached git only when another loop's sweep commit (`f390929`) included them.

Two additions to the proposal:
- Give the product idea loop the same commit step: add a `commit:` line to its registry entry, or have the dashboard flag any loop whose writes are uncommitted for more than one cadence.
- **Keep ranking in one file.** Store backlog ranks in a single index (for example `.sdlc/mailbox/backlog/RANKING.md`) rather than in every item's frontmatter. Re-ranking then changes one file instead of 12, and it can't collide with another loop's claim or evidence edit on the same item.
