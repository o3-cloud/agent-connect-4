---
title: "BDR-001..033 and their source specs are not on main"
status: closed
source: bdr-verification
branch: main
commit: 30cd082
found: 2026-11-18
---

# BDR-001..033 and their source specs are not on main

## Observed

On `main` (and `origin/main`), `docs/bdr/` holds only `000-template.md` and a README whose index says "_none yet_". BDR-001..033, `SPEC.md` and `OUTCOMES.md` exist only on branch `docs/consolidate-agents-md` (commit af8b3fc).

## Expected

The behavior contracts that main is verified against are on main (spec: `docs/specs/practices/bdr.md`).

## Fix

Merge the BDR extraction (`docs/consolidate-agents-md`) into main. The `bdr-NNN-*.md` issues track each behavior's implementation.

## Decision (2026-11-18)

Fixed. BDR-001..033, SPEC.md, OUTCOMES.md and ADR-008..012 were rebased onto main and fast-forwarded (main at d6b9f25). Landed locally, not through a PR: `origin` is the shared stack-base-go template, so this work is not pushed there.
