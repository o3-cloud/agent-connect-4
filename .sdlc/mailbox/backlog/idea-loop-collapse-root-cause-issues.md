---
title: "File one root-cause issue instead of 33 identical BDR failures"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# File one root-cause issue instead of 33 identical BDR failures

## Problem

The BDR-verification loop filed `bdr-001` through `bdr-033`. Their "Observed" sections are almost all copies of the same 5 canned strings (`OBS` in `file_issues.py`), and they share one root cause: main is still the stack-base-go template. The issues are noise. They don't tell anyone what to do next, and a re-run will repeat the same 33 failures.

## Proposal

- Before checking individual BDRs, the verification loop probes each interface (API, CLI, MCP, web). If an interface is missing, it files or updates **one** `main-<surface>-not-implemented.md` issue and marks the dependent BDR checks as `blocked_by:` that issue instead of `failed`.
- It files per-BDR issues only when the interface exists and a specific acceptance criterion fails.
- It adds `blocked_by:` and `blocks:` frontmatter so the mailbox can be read as a dependency graph.

## Acceptance

- [ ] On today's main, a verification run produces ≤5 open issues, not 33.
- [ ] Once the REST API lands, the API-backed BDRs are re-checked one by one.
