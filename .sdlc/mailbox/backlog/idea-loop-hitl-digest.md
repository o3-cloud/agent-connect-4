---
title: "Send human decisions to the human as one digest"
status: proposed
type: loop
outcomes: [5, 6]
created: 2026-11-18
---

# Send human decisions to the human as one digest

## Problem

`hitl-notes.json` holds 5 open decisions. Two are spec thresholds and error codes, one is an incorrect Renovate claim, one is ADR status, and one is the catch-all route. The dashboard adds more automatically: BDRs not on main, and the unpushed commit on main. They are visible only if someone opens the dashboard artifact. Meanwhile, the loops keep working around the blockers, for example by re-filing BDR failures that wait on the merge decision.

## Proposal

- Once a day, or when a `severity: high` item appears, send a single `PushNotification` digest that lists each decision, a recommended default, and the command or file edit that resolves it.
- Loops that are blocked by an open decision record `blocked_on: <hitl ref>` and stop, instead of producing more output.
- Resolving a decision (`open: false`) wakes the blocked loops on their next tick.

## Acceptance

- [ ] The user hears about each high-severity decision within one loop cycle.
- [ ] A loop blocked on a decision records a no-op run.
