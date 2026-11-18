---
id: loop-coordination
layer: practices
type: spec
extends: [git]
---

# Loop Coordination

## Purpose

Several Claude Code sessions run scheduled loops against this repository at the same time: they verify BDRs, fix issues, propose improvements and report progress. When they share one checkout and one mailbox with no rules, they leave each other's edits uncommitted on `main`, sweep those edits into unrelated commits, work the same item twice, and fire in the same minute. The dashboard then counts work that no commit contains. This spec gives every loop three duties: register itself, claim before it writes, and keep the shared checkout clean.

## References

- **spec** `git`: linear history and short-lived branches
- **file** `.sdlc/loops.json`: the loop registry this spec defines
- **file** `.sdlc/dashboard/README.md`: how the progress dashboard checks these rules

## Rules

### Registry

1. Register every loop that runs against this repository in `.sdlc/loops.json` before its first scheduled run.
2. Give each registry entry these fields: `loop` (name), `session` (Claude Code session id), `role`, `job` (cron id), `minute` (minute past the hour it starts), `cadence`, `expires` (ISO date and time the job expires; session cron jobs expire 7 days after creation), `worktree` (path, or empty for a loop that does not write to the repository), `writes` (folders it writes to), and `sources` (the `source:` value on mailbox issues it files).
3. Update your own entry when your job, minute, worktree or expiry changes. Edit only your own entry.
4. Remove your entry when you stop your loop.
5. Start each loop at its own minute. Loops that write the same folder must start at least 10 minutes apart, unless both claim items before changing them (rules 6–8) and change tracked files only in their own worktrees (rule 9); claims and worktrees already prevent their writes from colliding. Record the minute the job actually fires, as `CronList` reports it: the scheduler adds a fixed offset to the cron minute. The reserved slots are :07 and :17 for verify and fix, :47 for improvement, and :53 for the progress dashboard.

### Claims

6. Before you change a mailbox issue or backlog item, add `claimed_by: <session id>` and `claimed_at: <ISO time>` to its front matter.
7. Do not change an item that another session has claimed within the last 2 hours. Treat a claim older than 2 hours as stale; you may take it over by replacing both fields.
8. Remove `claimed_by` and `claimed_at` when you finish with the item, in the same edit that records the result (for example `status: closed`).

### Working trees

9. A loop that changes tracked files works in its own git worktree, recorded in its registry entry. It never edits tracked files in the main checkout.
10. Integrate work into `main` by fast-forward or rebase only (`git` spec), after `make check` passes in your worktree.
11. End every run with your worktree either clean or committed. Park unfinished work on a `wip/<loop>` branch, never as uncommitted files.
12. If you find uncommitted changes in the main checkout that you did not make, do not commit, stash, revert or include them. Add `needs: human` to an issue that names the files, so the dashboard shows it as a decision.
13. Run `git worktree prune` at the start of each run. Remove your worktree when you remove your registry entry.

## Checking

The progress dashboard checks these rules on every run and lists any breach under "Needs a human":

- **Uncommitted files:** uncommitted tracked files in the main checkout (rule 12). Dashboard counts come from `HEAD`, so uncommitted work is never counted as progress.
- **Unregistered sessions:** a live Claude Code session in this repository that has no registry entry (rule 1).
- **Minute clashes:** two registered loops that write the same folder and start within 10 minutes of each other (rule 5).
- **Expiring loops:** a registered loop that expires within 24 hours (rule 2).
- **Stale claims:** claims older than 2 hours (rule 7).
- **Leftover worktrees:** a worktree that no registry entry owns (rule 13).
