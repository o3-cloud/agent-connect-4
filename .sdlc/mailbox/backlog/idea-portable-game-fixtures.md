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

## Evidence (2026-11-20, 492da18)

ADR-015 makes restore *replay the journal through `game.Service`*. That means a game's move sequence is now also its stored form, so these fixtures gain a fourth consumer: the restart test. For each fixture, play it on a temporary `ARENA_JOURNAL_PATH`, restart, and check that the restored game equals the fixture's `expect` (status, winner, `finalBoardText`, history). The torn-last-line case from the ADR's Verification is a cheap addition: append each fixture's journal cut at every byte offset of its last line and check that the restore keeps every complete move. Building the fixtures before or with BDR-018 gives that test its cases for free. Rank stays 4. It is the cheapest way to make the journal's "deterministic restore" claim (DATA-005, DATA-006) checkable across many games instead of one.

## Evidence (2026-11-20, 41a8494)

The journal has shipped with its own restart tests (version 5, completed and resigned games, idempotent retry, torn last line). So the note above is now about *breadth*: those tests restore a handful of hand-built games, and the fixtures would restore every win direction, the draw and each rejection through the same journal. Still worth doing, still rank 4. It is no longer on BDR-018's critical path.

## Ranked (2026-11-20, 0cc40e1): 5 of 11 product ideas

All 33 BDRs are accepted (5477d60), so ranks now follow the OUTCOMES.md release criteria rather than open BDRs. Breadth for the engine, the contract suite and the journal restore. No longer on any BDR's critical path.
