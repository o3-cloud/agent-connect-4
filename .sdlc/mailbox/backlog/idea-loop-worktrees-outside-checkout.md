---
title: "Keep loop worktrees outside the main checkout, or exclude them from every scan"
status: proposed
type: loop
outcomes: [5]
spec: [docs/specs/practices/loop-coordination.md rule 9, docs/specs/interface/commands.md]
created: 2026-11-18
---

# Keep loop worktrees outside the main checkout

## Problem

The issue worker follows loop-coordination rule 9 by placing its worktree at `.sdlc/worktree/<branch>`, which is **inside** the main checkout. Right now that's `.sdlc/worktree/bdr-021-mcp-tools` on `feat/bdr-021-mcp-tools`. Git ignores it (`.sdlc/`), but filesystem walks don't:

- **`make health` is wrong.** It reports `Files: 444, Go files: 60, Test files: 30`. Git tracks 109 files, 20 Go files and 10 test files. `scripts/health.py` walks `ROOT.rglob("*")` and skips only `.git` and `bin`, so it counts the worktree copy, `prototype/` and `.sdlc/`.
- **`make fmt-check` on main checks another loop's unfinished work.** `gofmt -l .` and `goimports -l .` recurse into `.sdlc/worktree/**`; unlike `./...` package patterns, they don't skip dot-directories. That's 40 extra `.go` files today. An unformatted file halfway through an edit in the worker's branch would fail `make check` for every other loop working on main.
- **Repo-wide greps and `find` from any loop** (including this one) return each file twice. Evidence counts, such as "N files cite SHA X", get inflated.

## Proposal

Pick one, then amend rule 9 to match:

1. **Move worktrees out of the tree** (preferred). Use `../agent-connect-4.worktrees/<branch>`, or the session scratchpad, as the BDR verify loop already does with `.../scratchpad/main-wt`.
2. **Exclude them everywhere.** Change `fmt-check` to run gofmt and goimports only on `git ls-files '*.go'`. Have `health.py` count `git ls-files` instead of `rglob`. Document "exclude `.sdlc/`" for ad-hoc greps.

Do option 2 either way: counting tracked files is right whatever the worktree layout.

## Acceptance

- [ ] `make health` file counts equal `git ls-files | wc -l` (and the same for the Go and test counts).
- [ ] A deliberately unformatted file in another loop's worktree doesn't fail `make fmt-check` on main.
