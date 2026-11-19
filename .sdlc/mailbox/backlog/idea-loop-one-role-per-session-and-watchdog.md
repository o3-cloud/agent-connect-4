---
title: "One role per loop session, plus a watchdog that notices when a loop stops running"
status: proposed
type: loop
outcomes: [4, 5]
spec: [docs/specs/practices/loop-coordination.md rules 1-3, 9]
created: 2026-11-18
---

# One role per loop session, plus a watchdog

## Problem

At 23:44Z the **Progress dashboard** session (`4e529a7f`) claimed `issues/bdr-026-human-plays-agent-in-browser.md`. It created `.sdlc/worktree/browser-play` on `feat/browser-play` and started building the browser UI: it downloaded htmx 2.0.4 and the Tailwind 4.1.11 CLI and checked their sha256 sums. Its registry entry still says `role: "Snapshots issues, BDRs and backlog hourly ..."`, `writes: [".sdlc/dashboard"]`, `worktree: ""`.

Results:

- **The dashboard stopped running.** The last `history.jsonl` snapshot is from 22:52Z. Its :53 runs at 23:53Z and 00:53Z didn't happen (checked at 01:21Z). Session cron jobs fire only when the session's REPL is idle, so a session doing a long build job starves its own scheduled loop.
- **Nobody noticed.** The dashboard is the loop that checks the other loops (unregistered sessions, stale claims, expiring jobs). Nothing checks the dashboard.
- **The registry is wrong.** Rules 2, 3 and 9 require the entry to list the loop's worktree and the folders it writes. A loop reading the registry would assume nobody is working on `docs/`, `internal/` and similar paths from that session.
- The `bdr-026` claim (23:44Z) goes stale at 01:44Z. If the build is still running, another loop may then take it over and the two will collide.
- No commits on main since `daf1200` (18:31 local). The issue worker's `feat/bdr-020-cli-watch` and the dashboard's `feat/browser-play` are both open. Two hours without a merge isn't unusual on its own, but no loop reports it.

## Proposal

1. **One role per session.** A session that runs a scheduled loop does only that loop's work. If the dashboard (or any loop) finds work for itself outside its role, it files or claims the issue *for* the right loop (`assign: issue-worker`), or the human starts a new session for it. Add this as a rule in `loop-coordination.md`.
2. **Liveness heartbeat.** Each loop writes `last_run` (ISO time) to its own registry entry at the end of every run.
3. **Mutual watchdog.** At startup, *every* loop checks the registry. Any entry whose `last_run` is more than 2× its cadence behind, including the dashboard's, gets a `needs: human` issue `loops-<name>-not-running.md` (one per loop, updated rather than duplicated).
4. **Long claims refresh.** A loop working on a claim for longer than an hour refreshes `claimed_at`, so that a live claim never looks stale.
5. The dashboard reports "hours since last commit on main" next to its open feature branches.

## Acceptance

- [ ] If any loop misses 2 runs, an issue appears within one cycle of another loop.
- [ ] Every session's actual worktrees and write paths match its registry entry.

## Evidence (2026-11-19 03:28Z, improvement loop run 10)

**Correction to proposal 4.** Refreshing a claim must require **progress**, not just activity. At 03:27Z the `bdr-026` claim (priority 1) was refreshed by `4e529a7f`. Meanwhile, `feat/browser-play` has **0 commits** ahead of main, and its worktree holds about 4 hours of uncommitted work: `internal/web/` is untracked, 4 tracked files are changed (+44/−11), and the last file change was at 20:55 local. That breaks loop-coordination rule 11 ("End every run with your worktree either clean or committed ... `wip/<loop>`"), and a time-based refresh keeps the item locked indefinitely. Revised rule: a claim may be refreshed only together with a commit (to `wip/<loop>` or the feature branch) made since the last refresh. Otherwise it goes stale after 2 hours as normal.

The dashboard also still isn't snapshotting on schedule: one snapshot at 01:21Z, then none at 01:53Z or 02:53Z.

The issue worker (`5b5d3e77`, every 30 min) last committed at 20:54 local (`6f4394c`), with 19 unclaimed open issues available. Without `last_run` in the registry, it's impossible to tell whether it's idle, blocked or gone.

## Evidence (2026-11-19 05:14Z, improvement loop run 11): correction

My 03:28Z entry was taken **7 minutes before** the work landed. `9a7a343 feat(web): play Arena Bot in the browser` merged at 22:35 local (03:35Z), and BDR-026 was accepted at 23:24. The 03:27Z claim refresh came right before the merge, so it was not a stuck session keeping a lock. The rule I proposed still stands (a refresh needs a commit since the last one, which a `wip/` commit every hour would have shown). But the "locked indefinitely" framing was wrong for this case. The dashboard is snapshotting again (04:25, 04:30 and 04:52Z).

A lesson for this loop: a single observation of a "stuck" branch is weak evidence. Check again on the next run before calling something stuck.
