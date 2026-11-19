---
title: "Compute 'criteria met' from passing tests tagged with BDR/AC ids, not hand-ticked boxes"
status: proposed
type: loop
outcomes: [5]
spec: [docs/specs/practices/bdr.md rules 13-14]
created: 2026-11-18
---

# Compute "criteria met" from passing tests

## Problem

Now that real code exists, the progress number is what matters. Today it is assembled by hand:

- `collect.py` counts ticked `- [x] AC-n` lines on each tracking issue, or **all** of a BDR's criteria once its status is `Accepted`.
- The working tree right now has uncommitted edits that flip BDR-001, 002, 003, 005, 006, 008 and 009 from `Proposed` to `Accepted` (`git diff --stat`: 7 BDRs plus the README). That one edit moves the dashboard by about 30 criteria.
- The tests already cite their BDRs (`tests/api_test.go` has 9 `BDR-00N ... AC-n` comments and `engine_test.go` has 5), but `game_test.go` and `cli_test.go` have none, and nothing reads these comments.
- bdr.md rule 13 allows `Accepted` only "after independent end-to-end verification ... outside the test harness". The skeleton issue records one black-box game, but not one check per BDR.

So a status flip, a tick, or a stale checkbox can move the metric without any test proving the behavior.

## Proposal

1. Use a convention: each test that verifies a criterion carries a comment `// verifies: BDR-001/AC-1, BDR-001/AC-2`, or uses subtests named `BDR-001/AC-1`.
2. `collect.py` (or `scripts/sdlc/ac_report.py`) runs `go test -json ./tests/...`, joins passing tests to their tags, and reports, for each AC, whether it is **covered**, **passing**, or **untested**.
3. "Criteria met" counts only ACs that are passing. `Accepted` additionally requires a black-box record: a `## Verified` block in the tracking issue with the curl transcript and commit, per bdr.md rule 13.
4. `make health` lists any `Accepted` BDR that has an untested AC.

## Acceptance

- [ ] The dashboard's `acDone` equals the number of tagged, passing AC tests.
- [ ] Flipping a BDR to `Accepted` without tests behind it raises a health warning.

## Evidence (2026-11-18 15:33, improvement loop run 6)

The dashboard now reports `bdrAccepted: 7` and `acDone: 34`. The 7 accepted BDRs come from *uncommitted* status edits (see `idea-loop-isolate-working-trees.md`). `883b8c8` added `bdr-definition.md`, which turns bdr.md rule 13 into **BDR-DEF-19** (Accepted only after end-to-end verification outside the test harness). That makes the rule checkable. A health check should report every `Accepted` BDR that has no recorded black-box verification.
