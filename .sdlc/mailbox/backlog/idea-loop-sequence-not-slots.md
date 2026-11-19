---
title: "Order loops by events or a single tick, not by wall-clock minute slots"
status: proposed
type: loop
outcomes: [5]
spec: [docs/specs/practices/loop-coordination.md rule 5]
created: 2026-11-18
---

# Order loops by events or a single tick, not by minute slots

## Problem

`loop-coordination.md` rule 5 reserves minute slots (:07/:17 for verify and fix, :47 for improvement, :53 for the dashboard). The session scheduler adds a different, unpredictable offset to each new cron job, so the slots can't be hit:

| Job asked for | Actually fires |
|---|---|
| `7 * * * *` (improvement, e07ac399) | :33 |
| `47 * * * *` (improvement, e6a165aa) | :14 |
| `7 * * * *` (BDR verify) | :25 |
| `7 * * * *` (product ideas) | :28 |
| `47 * * * *` (product ideas, attempt) | :08 |
| `21,51` / `27,57` (issue worker) | :21/:51, then :41/:11 |

Loops have spent several runs deleting and re-creating jobs to chase a slot. Rule 5 was then amended (`57d9072`) to excuse most pairs. Meanwhile the real dependency (verify → fix → dashboard → improve, so each one sees fresh results) is still left to timing luck.

## Proposal

Pick one of these:

1. **Event chaining.** Each loop ends its run by appending `{loop, at, head}` to `.sdlc/events.jsonl`. Downstream loops watch the file with `Monitor` and run when their upstream loop finishes, keeping an hourly cron only as a fallback heartbeat.
2. **Single orchestrator.** One cron job in one session runs the stages in order (verify → work one issue → collect dashboard → improvement review), each as a sub-agent with its own worktree. The registry then describes stages instead of separate schedules.

With either option, drop minute reservations from rule 5, and keep only "claim + own worktree".

## Acceptance

- [ ] No loop needs to re-create its cron job to land on a particular minute.
- [ ] The dashboard snapshot always comes after that hour's verify and worker runs.

## Evidence (2026-11-18 17:25, improvement loop run 7)

The offset also varies from run to run, not only from job to job. Job `e6a165aa` was registered as firing at :14 (per `CronList`), but this run started at **:25**. The registry's `minute` field is at best approximate, so a 10-minute spacing rule built on it can't be relied on.
