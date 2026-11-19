---
title: "Cite evidence with references that survive rebases and resets"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Cite evidence with references that survive rebases and resets

## Problem

45 files in `.sdlc/` and `docs/` cite commits that no longer exist after the 16:06Z re-init (`30cd082`, `08957ee`, `d6b9f25`, `af8b3fc`, `4e6d6bb`). Examples:

- 37 issues record `commit: 30cd082`.
- `hitl-notes.json` resolutions say "BDR-030 updated (d6b9f25)".
- `idea-walking-skeleton-first.md` says "08957ee pins go 1.24.0".

All of them now point nowhere, and a reader can't confirm any of the claims.

## Proposal

- Cite a **durable reference** first: a PR number or URL, a tag, or a file path plus a short content quote. Add the SHA only as a secondary hint.
- Have the build/PR loop tag milestones (for example `loop/bdrs-on-main`) so they can be cited.
- Add a `make health` check that lists SHAs in `.sdlc/**` that `git cat-file -e` can't resolve.

## Acceptance

- [ ] `make health` reports 0 unresolvable SHAs, or lists them.
- [ ] New issues and resolutions include at least one reference that isn't a SHA.
