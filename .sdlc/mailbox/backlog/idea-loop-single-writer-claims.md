---
title: "Coordinate concurrent loops with claims on mailbox items"
status: accepted
type: loop
outcomes: [5]
created: 2026-11-18
---

# Coordinate concurrent loops with claims on mailbox items

## Problem

Seven sessions touched this repo in about 3 hours. `.claude/scheduled_tasks.lock` is held by session `5b5d3e77`, while this session `9555fff8` also runs an hourly loop. They all write to `.sdlc/mailbox/` without coordinating. One script writes to the wrong directory (`file_issues.py`'s docstring still says `.sdlc/issues`; see the memory note about `mailbox/issues`), and two loops can pick up the same backlog item.

## Proposal

- When a loop starts work on an item, it adds `claimed_by: <session-id>` and `claimed_at:` to the item's frontmatter, and removes them when it finishes.
- Claims older than 2 hours are treated as stale and can be taken over.
- Each mailbox subfolder gets a short `README.md` that states the schema and the folder's single purpose, so scripts don't drift.

## Acceptance

- [ ] Two simultaneous loops never edit the same item.
- [ ] Every mailbox folder documents its frontmatter schema.

## Decision (2026-11-18 retro)

Accepted as part of retro item 1. Tracked in `issues/loops-adopt-coordination-protocol.md`; rules in `docs/specs/practices/loop-coordination.md`.
