---
title: "Portable game fixtures: move sequences and expected outcomes shared by every test layer"
status: proposed
rank: 4
type: quality
outcomes: [5]
spec: [RULE-005..010, NFR-001, NFR-004, DATA-005, AT-004, AT-005]
bdr: [BDR-005, BDR-006, BDR-008, BDR-009, BDR-017, BDR-023]
created: 2026-11-18
---

# Portable game fixtures

## Problem

The test knowledge we have (win in every direction, draws, full-column cases, edge diagonals) lives in `prototype/arena.test.js`. `prototype/` is git-ignored, and the repo has just been re-initialised, so that knowledge exists only on this machine. Engine tests, contract tests, replay tests and the load test's invariant checks would each re-type the same move sequences.

## Proposal

- Add `testdata/games/*.json`, one file per scenario:
  ```json
  { "name": "diagonal-up-right-win", "moves": [0,1,1,2,2,3,2,3,3,6,3],
    "expect": { "status": "completed", "winner": "red", "resultReason": "connect_four",
                "finalBoardText": "...", "rejects": [] } }
  ```
- Rejection scenarios add steps such as `{ "col": 0, "expectError": "COLUMN_FULL" }`, using SPEC §16 codes.
- Port every case from `prototype/arena.test.js` and the sequences in BDR-008 and BDR-009 into this folder now, so they are committed.
- Each layer loads the same fixtures:
  - the engine unit tests;
  - the cross-interface contract suite, once per driver;
  - the replay test (fixture moves → stored game → replay equals `finalBoardText`);
  - a notation round-trip, if `idea-shareable-replay-and-notation` is adopted.
- Add a fuzz test (`go test -fuzz`) that plays random legal sequences and checks the SPEC §15 invariants. Each new failure it finds is saved as a fixture.

## Acceptance

- [ ] Every engine rule in RULE-001..010 has at least one fixture, and a test fails if a RULE ID has none.
- [ ] Engine and REST tests load the same fixture files.
- [ ] The fuzz corpus runs in `make check` (with a short `-fuzztime`).

## Ranked (2026-11-18): 1 of 16

Build alongside the walking skeleton; the prototype's test cases exist only on this machine.

## Evidence (2026-11-18, eee9bc7)

The walking skeleton landed. `tests/engine_test.go` writes its win, draw and full-column cases as inline Go literals (about 9 move sequences), and `tests/api_test.go` builds its own sequences again. There is still no `testdata/`. Moving these cases into shared fixtures now is a small change, and it gives the persistence, replay and contract tests the same cases to reuse.
