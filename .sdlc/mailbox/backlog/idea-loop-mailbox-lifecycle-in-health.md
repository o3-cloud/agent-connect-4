---
title: "Show mailbox flow in `make health`: backlog → accepted → issue → closed"
status: proposed
type: loop
outcomes: [5, 6]
created: 2026-11-18
---

# Show mailbox flow in `make health`

## Problem

The backlog has 13 `proposed` ideas and nothing that moves them forward. No `accepted` or `rejected` state exists, and nothing links an idea to the issues or PRs that implement it. Hourly loops keep adding ideas, which grows the pile but doesn't improve the system.

## Proposal

- Define states: `proposed → accepted | rejected → in-progress → done`, with `links:` to issues, branches and PRs.
- Extend `scripts/health.py`, which already parses frontmatter, to print mailbox counts by status, items older than N days, and items not verified against the current HEAD.
- Cap the backlog. Once there are 15 or more `proposed` items, the improvement loop ranks the existing ideas instead of adding new ones.

## Acceptance

- [ ] `make health` prints a mailbox section.
- [ ] The backlog never holds more than the cap of unranked proposals.

## Evidence (2026-11-18 10:33, improvement loop run 2)

The backlog went from 25 to 29 items in under an hour. A separate loop added `idea-showcase-match`, `idea-error-code-catalog`, `idea-go-client-package` and `idea-mcp-stdio-bridge`, and none of the 29 is `accepted`. The cap is now the most urgent part of this idea.

## Evidence (2026-11-18 11:33, improvement loop run 3)

Partly adopted. The backlog now uses `promoted` (walking skeleton), `accepted` (`idea-error-code-catalog`) and `rejected` (`idea-stalled-game-turn-clock`), and 4 issues are closed. Still missing: the cap (the backlog has 36 files, 33 of them `proposed`) and the `make health` view.

## Evidence (2026-11-18 13:33, improvement loop run 4)

The improvement loop is itself the biggest source of unranked items: 20 of the 40 backlog files are `idea-loop-*`, and none of them has been triaged. From now on, this loop adds a new file only for a new kind of problem. Otherwise it appends evidence, and it hands its top 3 ideas to the human-in-the-loop list instead of adding more.
