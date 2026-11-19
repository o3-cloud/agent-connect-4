---
title: "Every filed finding carries a reproduction command that the next run re-executes"
status: proposed
type: loop
outcomes: [5]
created: 2026-11-18
---

# Every filed finding carries a reproduction command

## Problem

`main-toolchain-followups.md` (source: `loop-run`) says "No Renovate (or Dependabot) config exists in the repo". `.github/renovate.json` is present. The dashboard loop caught the mistake and raised it as a human-in-the-loop note, so a person now has to spend time on a false finding.

## Proposal

- Each issue gets a `check:` field in its frontmatter. It holds one shell command that exits non-zero while the issue is real, for example `test ! -e .github/renovate.json || ! grep -q gomod .github/renovate.json`.
- The filing loop runs `check:` before it writes the issue, and doesn't file the issue if the command passes.
- The re-verify loop (`idea-loop-reverify-on-head-change.md`) re-runs `check:` and closes the issue automatically.

## Acceptance

- [ ] No new issue is filed without a `check:` command.
- [ ] `main-toolchain-followups.md` is corrected (Renovate exists; check whether it covers the `toolchain` directive and Docker images).
