---
title: "Plan for the laptop sleeping: missed runs, a burst of runs on wake, and off-schedule runs that skip their commit"
status: proposed
type: loop
outcomes: [4, 5]
spec: [docs/specs/practices/loop-coordination.md rule 5]
created: 2026-11-19
---

# Plan for the host sleeping

## Problem

All five loops are session cron jobs on one Mac. When the Mac sleeps, none of them run, and when it wakes they all fire at once.

- **Runs missed.** `pmset -g log` shows the machine asleep, then waking from deep idle at 12:25:42 local (17:25:42Z) on 2026-11-19. There were no commits and no snapshots between 15:25Z and 17:25Z. This loop's 16:14Z run fired at 17:26Z, and the dashboard's 15:53Z and 16:53Z runs collapsed into one snapshot at 17:25:59Z.
- **Runs burst together on wake.** The dashboard and this loop both fired within 30 seconds of the wake. Rule 5's minute spacing (:08, :14, :25, :28, :53) only works while the machine is awake. On wake, the queued jobs fire together, so the commit race seen at 14:26Z (`860ef37`, see [[idea-loop-mailbox-is-tracked-now]]) becomes more likely.
- **An off-schedule run may skip its commit.** The 14:00:07Z snapshot, taken after an earlier wake, was never committed by its own run. It was swept into the next on-schedule commit (`cd105ac`, 14:52Z). *Correction, 17:27Z:* the 17:25:59Z snapshot *was* committed (`6fbad18`, about a minute later). I checked while that run was still going. So there is one confirmed case, not two.
- **Sleep looks like a stuck loop.** With no heartbeat, a gap caused by sleep looks the same as a stuck loop. This loop misread one gap twice (03:28Z and 14:19Z). The watchdog in [[idea-loop-one-role-per-session-and-watchdog]] would raise false `loops-*-not-running` alarms after every night the laptop sleeps.

## Proposal

1. **Wake jitter.** At the start of a run, if the scheduled time is more than 10 minutes in the past (a fire caught up after sleep), wait for the loop's own offset (for example, its registry `minute` mod 15, in minutes) before writing anything. This spreads the post-wake burst across about 15 minutes.
2. **Run once after a gap.** A loop that wakes to several missed runs does one run, not one per missed slot.
3. **Commit on every run.** The commit step must not depend on whether the run is on-schedule. Whatever triggered the run, it ends clean (rule 11).
4. **Separate sleep from stuck.** The watchdog compares a loop's `last_run` against the host's last wake time (`pmset -g log | grep ' Wake '`) and treats a gap that falls inside a sleep as expected.
5. **Optionally, for overnight runs:** `caffeinate -i` for the session, or move the loops to cloud routines. That is an owner decision about power use and cost.

## Acceptance

- [ ] After a wake, no two loops commit within the same 2 minutes.
- [ ] After a wake, `git status --short` on main is empty once every loop has run.
- [ ] A night of sleep produces no `not-running` alarm.

## Note for observers (including this loop)

After a wake, every loop is mid-run at the same moment. A check taken then sees half-finished work: uncommitted snapshots, claims not yet committed. This loop drew a wrong conclusion that way three times (03:28Z, 14:19Z, 17:26Z). Before reporting "uncommitted" or "stuck", wait until no `.sdlc` file has changed for 5 minutes, or check again on the next run.

## Evidence (2026-11-19T19:23Z, improvement loop): claims expire while the host is asleep

The issue worker claimed `bdr-028-browser-reconnect-recovery` at 17:27:31Z (`73aae0d`) and created `.sdlc/worktree/bdr-028-stale-board-moves`. The Mac went to sleep **10 seconds later** (17:27:41Z, per `pmset`). It then dozed, with only short maintenance dark-wakes, until 19:22:28Z. The claim's 2-hour stale timer ran out at 19:27Z, five minutes after wake. The worker had done less than a minute of awake work on it. Any loop that wakes and checks claims by wall-clock time would treat this claim as abandoned and take it over, and two loops would then be working on the same item in two worktrees.

Proposal addition:

6. **Measure claim staleness in awake time.** A claim is stale only if 2 hours have passed *and* the host has been awake for at least 2 hours since `claimed_at`, or, more simply, the claim's age minus the time spent asleep in that window exceeds 2 hours (`pmset -g log` gives the sleep intervals). A cheaper fallback: no loop may take over a stale claim within 30 minutes of a wake.
