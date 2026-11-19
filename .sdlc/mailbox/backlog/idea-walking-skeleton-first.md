---
title: "Deliver a walking skeleton: engine → service → REST → one agent, end to end"
status: promoted
type: delivery
outcomes: [1, 3, 5]
spec: [SPEC §20 Phase 1, RULE-*, GAME-*, TURN-*, API-001..008]
bdr: [BDR-001, BDR-002, BDR-003, BDR-005, BDR-008]
created: 2026-11-18
---

# Deliver a walking skeleton first

## Problem

All 33 BDR issues are open because main is still the stack-base-go template. Building the work layer by layer (all of the engine, then all of the API...) delays the first point at which a whole game can be played.

## Proposal

Build the thinnest slice that lets two scripted HTTP clients finish a game:

1. `internal/engine`: a pure board, `Drop(col)`, win and draw checks, `LegalMoves()`. Port the cases from `prototype/engine.js` tests.
2. `internal/game`: a service with an in-memory store, a mutex per game, versions and turns.
3. `internal/api`: Huma (ADR-008) routes for create, join, state and move, using the structured error envelope from the start.
4. A Go integration test in which two clients play to a win.

After that, add one capability at a time behind the slice: persistence, auth, idempotency, CLI, MCP, UI.

## Acceptance

- [ ] `go test ./...` includes an end-to-end test where two HTTP clients play a full game to `completed`.
- [ ] The engine package imports nothing from HTTP or storage (NFR-007).
- [ ] BDR-001, 002, 003, 005, 008 issues can be closed.

## Notes

Unblock first: `main-build-go-toolchain.md`, `main-check-govulncheck-stdlib.md`, `main-unknown-routes-return-200.md`.

## Evidence (2026-11-18, run 2)

- The two build blockers are fixed on main: 08957ee pins `go 1.24.0` and `toolchain go1.27.4`, and `main-build-go-toolchain` and `main-check-govulncheck-stdlib` are closed. Still open: `main-unknown-routes-return-200` (needs human approval in `hitl-notes.json`) and `main-toolchain-followups`.
- `docs/consolidate-agents-md`, which holds the BDRs, still has `go 1.24`. Rebase it onto main before starting the skeleton branch, or the build blocker comes back there.

## Promoted (2026-11-18)

Now tracked as `issues/delivery-walking-skeleton.md`, with acceptance criteria updated to the current main (blockers resolved, BDRs on main, SPEC.md §16 error codes).
