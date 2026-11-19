---
title: "Make loop shell commands non-interactive (rm -i alias killed a run)"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Make loop shell commands non-interactive

## Problem

In session `4e529a7f`, a background task hung at `remove docs/bdr/_index.txt?` and had to be killed. The user's zsh profile sets `alias rm='rm -i'`, and the agent's shell loads that profile. Any unattended loop that deletes files will hang the same way.

## Proposal

- Loop prompts and helper scripts use `command rm -f`, or `\rm`, and run inside scripts (`bash script.sh`), where aliases don't expand.
- Add guidance to AGENTS.md: "Shell aliases from the user profile are active. Use `command <tool>` for rm/cp/mv in automation."
- Optionally add a `PreToolUse` hook that rewrites a bare `rm ` to `command rm `, or warns about it.

## Acceptance

- [ ] No loop run blocks waiting on a y/n prompt.
