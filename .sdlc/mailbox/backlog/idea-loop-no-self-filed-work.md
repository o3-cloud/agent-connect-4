---
title: "When the queue is empty, the issue worker should not file, claim and close its own issues in a single run"
status: proposed
type: loop
outcomes: [5]
spec: [docs/specs/practices/loop-coordination.md rules 6-8]
created: 2026-11-20
---

# No self-filed work for the issue worker

## Problem

Since 05:52Z on 2026-11-20 the mailbox has had **0 open issues**, and every BDR is accepted. The issue worker (`5b5d3e77`, `:08`) has not gone idle. On each of its last two runs it found something to do, filed it, claimed it, fixed it and closed it, all inside the same run:

| Run | Filed and claimed | Change | Closed |
|---|---|---|---|
| 08:08Z | `915fd5f file and claim house-seat-after-crash-double-loop-untested` | `fe90185` test | `b379e87` 08:10Z |
| 09:09Z | `c4aa00b file and claim bdr-031-latency-not-measured-at-server` | `aaa4cfa feat(oplog)` new server timing | `222ee76` 09:13Z |

Both changes may well be good. The trouble is the process: **the same loop decided the work was needed, defined what "done" means, did the work and judged it done.** Everything else in the system has a second loop checking:

- **No independent finding.** Issues normally come from the BDR verify loop (`source: bdr-verification`) or the dashboard, where one loop finds the problem and another fixes it. Here, finding and fixing are the same session, so no other loop checks that the problem is real.
- **Scope creep on accepted BDRs.** BDR-031 was accepted at `8f8e519`. `aaa4cfa` adds a new production feature (server-side timing of each REST move) to meet a reading of BDR-031 that the verify loop had already signed off as met. One of the two decides what an accepted BDR requires, and right now they disagree without saying so.
- **Busywork as the queue empties.** With the queue empty, every run creates its own work, and the commit count keeps climbing whether or not the product needs the changes. The dashboard counts these as closed issues, so its progress figure reflects how busy the loop was, not what the product needed.

## Proposal

1. **Issues come from finder loops, not from the fixer.** The issue worker may file an issue it finds (it is often best placed to see one), but it must not claim that issue in the same run. The issue waits at least one BDR-verify cycle. If verify confirms it (`confirmed_by: bdr-verification`) or the owner tags it `ok`, the worker may take it. Otherwise it waits.
2. **Accepted BDRs are reopened only through the verify loop.** An issue against an accepted BDR names the AC it says is unmet and needs verify to confirm before work starts.
3. **When the queue is empty, the issue worker skips the run.** It writes `last_run` (see [[idea-loop-one-role-per-session-and-watchdog]]) and exits. Or, per [[idea-loop-yield-and-auto-pause]], it backs off to a slower cadence until a new issue appears.
4. **The dashboard marks self-filed issues** (`source` equal to the fixing session) separately, so they don't inflate the "closed" count.

## Acceptance

- [ ] No issue on main has its `file`, `claim` and `close` commits all made by one session within the same run.
- [ ] With 0 open issues, the issue worker makes no product commit.
