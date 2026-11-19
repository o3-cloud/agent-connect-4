---
title: "Claude Code session lock file is committed to main"
status: closed
source: bdr-verification
branch: main
commit: 4218fba
found: 2026-11-18
---

# Claude Code session lock file is committed to main

## Observed

`git ls-files .claude` on main lists `.claude/scheduled_tasks.lock`. Its content is per-machine runtime state:

```
{"sessionId":"5b5d3e77-…","pid":81108,"procStart":"Wed Nov 18 11:37:46 2026","acquiredAt":…}
```

`.gitignore` has no `.claude` entry, so the file shows up as modified whenever any session takes the scheduler lock. Every clone also inherits a stale lock that names a session and PID from another machine.

## Expected

Only source, config and docs are tracked. Local tool state stays untracked (spec: `docs/specs/practices/git.md`).

## Fix

```sh
git rm --cached .claude/scheduled_tasks.lock
printf '\n# Claude Code local state\n.claude/*.lock\n.claude/settings.local.json\n' >> .gitignore
```

Keep `.claude/settings.json`, `.claude/agents/` and similar shared config tracked if they are added later.

## Resolution

Fixed on main in 8965921 (2026-11-18): `git rm --cached .claude/scheduled_tasks.lock`; `.gitignore` adds `.claude/*.lock` and `.claude/settings.local.json`; `make health` now reports tracked-but-ignored files (red with the lock tracked, green after). `make check` PASS. Note: fast-forwarding a checkout across this commit deletes its live lock file; the owning Claude session recreates it.
