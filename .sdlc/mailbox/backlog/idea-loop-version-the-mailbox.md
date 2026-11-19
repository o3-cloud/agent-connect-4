---
title: "Decide where the mailbox lives: it is git-ignored and local-only today"
status: proposed
type: loop
outcomes: [4, 5]
created: 2026-11-18
---

# Decide where the mailbox lives

## Problem

`4e6d6bb chore: ignore .sdlc directory` git-ignored the whole mailbox, and `prototype/` is ignored too. All issues, ideas and loop state live only on this laptop. A cloud `/schedule` routine, a teammate, or CI can't see them, and losing the disk loses the history. The `/loop` offer to run in the cloud is useless to these loops, because a cloud copy of the repo would have an empty mailbox.

## Proposal

Choose one, and record the choice as an ADR:

1. Commit `.sdlc/mailbox/`, keeping `runs/` and scratch files ignored.
2. Mirror issues to GitHub Issues with `gh issue create`, using labels `bdr`, `loop` and `idea`, and keep `.sdlc` as a local cache.

## Acceptance

- [ ] A fresh clone, or a cloud routine, can read the open issues and the backlog.

## Evidence (2026-11-18 11:33, improvement loop run 3)

Counterpoint: because `.sdlc/` is ignored, the mailbox **survived** the 16:06Z history reset. If you commit it, keep a copy outside the repo as well (GitHub Issues, or a separate branch or repo), so that a repo reset can't erase both the code and the record of what happened.
