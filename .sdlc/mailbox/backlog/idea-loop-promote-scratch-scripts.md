---
title: "Promote loop helper scripts from /tmp scratchpads into scripts/sdlc/"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Promote loop helper scripts into the repo

## Problem

The loops depend on scripts that live only in session scratchpads under `/private/tmp/claude-501/...`:

- `d5b9c49b/scratchpad/file_issues.py` files BDR issues. It hard-codes `today = "2026-11-18"`, and its docstring names the wrong folder.
- `4e529a7f/scratchpad/bdr/gen.py` generated BDR-001..033.
- `4e529a7f/scratchpad/smoke.cjs` and `measure.cjs` are prototype smoke and measurement checks.

Each new session rewrites them from scratch, they drift, and a reboot deletes them.

## Proposal

- Move these into `scripts/sdlc/` (`file_issues.py`, `verify_bdrs.py`). Derive the date and commit at runtime, and read paths from a single config.
- Loop prompts call `python3 scripts/sdlc/<x>.py` instead of writing ad-hoc code.
- Cover them with a small test, or at least a `--dry-run`.

## Acceptance

- [ ] A verification loop run creates no new script in a scratchpad.
