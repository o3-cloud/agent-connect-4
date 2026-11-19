# Arena Loop Progress — hourly runbook

Dashboard: https://claude.ai/artifact/7gWGC88ig223niRe3xH4yN

Each run of the hourly `/loop` job does this:

1. From the repo root: `python3 .sdlc/dashboard/collect.py`. It appends a snapshot to `history.jsonl` and rewrites `out/*.json`.
2. For each dataset id in `assets.json` (`history`, `issues`, `bdrs`, `backlog`, `outcomes`, `hitl`, `loops`):
   upload `out/<id>.json` as an asset, `update` `datasets/<id>` with the new `source.url` and `updated.at`,
   delete the old asset, and record the new id in `assets.json`.
3. Read `data/users/me` (`check`, `runs`) and fix anything that failed.
4. Report the verdict (Climbing, Stalled, Slipping), what moved, and the open decisions in `hitl.json`.

## Human-in-the-loop items

- Automatic: open mailbox issues with `needs: human` (or `hitl: true`) in their front matter, proposed backlog
  items with `type: spec-change`, BDRs missing from main, and unpushed commits made directly on main.
- Judgement notes: `hitl-notes.json`. Set `"open": false` once an item is resolved.

Other loops can raise a decision for a human by adding `needs: human` to an issue's front matter.

## Measures

- **Criteria met**: ticked `- [x] AC-n` lines on each `bdr-NNN-*` tracking issue, or all of a BDR's criteria once it is `Accepted`.
- **Verdict**: compares the last two snapshots on closed issues, accepted BDRs, satisfied BDRs, BDRs on main and criteria met.
  Any rise with no fall means Climbing, any fall means Slipping, and no change means Stalled.
- New open issues count as discovery, not regression.

## Tracked loops

The loop registry is `.sdlc/loops.json`, defined by `docs/specs/practices/loop-coordination.md`. Each loop keeps its own entry: session, cron job, the minute it actually fires, expiry, worktree, the folders it writes, and how to attribute its output (mailbox issue `sources`, or a backlog `backlogType`). This dashboard runs as cron `864309da` and fires at :53.

Status: a session is `running` when a live `claude` process has its working directory in this repo and started within 15 s of the session folder's `started` file. Last active is the newest change in the session's temp folder.

## Coordination checks

Each run adds a "Needs a human" item for: uncommitted tracked files in the main checkout, live sessions missing from the registry, registered loops that write the same folder within 10 minutes of each other, loops expiring within 24 hours, claims older than 2 hours, and worktrees no loop owns. BDR status is read from `HEAD`, so uncommitted edits never count as progress. Each snapshot also records `uncommittedFiles`.

## Pending closures

- `loops-adopt-coordination-protocol`: closed 2026-11-18T21:52Z after three consecutive clean runs.
- BDR-013: re-check AC-4 when spectator updates, the CLI or MCP land; then the verify loop accepts it.
