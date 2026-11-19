---
title: "Keep a run journal so loops can learn from earlier runs"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Keep a run journal so loops can learn from earlier runs

## Problem

Seven Claude Code sessions ran against this repo on 2026-11-18 (06:38–09:50). No `.jsonl` transcript is left in `~/.claude/projects/-Users-owenzanzal-CvilleAIEx-agent-connect-4/`. Only `memory/` and one `tool-results/` file (a copy of SPEC.md) remain. What each run did, why it did it, and what failed can only be pieced together from leftover scratchpad files in `/private/tmp/claude-501/...`, and those are deleted on reboot.

So a retrospective loop like this one has almost nothing to work from.

## Proposal

- Each loop run appends one file to `.sdlc/mailbox/runs/<YYYY-MM-DDTHHMM>-<loop>.md`. It records the goal, the inputs (branch, HEAD SHA, mailbox counts), the actions taken, the files created or changed, failures, and a `noop: true|false` flag.
- Check the `cleanupPeriodDays` setting in Claude Code and confirm that transcripts are kept long enough for retros.

## Acceptance

- [ ] Every scheduled loop writes a run entry, including no-op runs.
- [ ] The improvement loop reads `runs/` first, before it looks at scratchpads.

## Evidence (2026-11-18 10:33, improvement loop run 2)

`.sdlc/dashboard/history.jsonl` now gives a metrics history but records no actions. A run journal is still needed to explain *why* a metric moved.
