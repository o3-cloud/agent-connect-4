# BDR-017: Replay a Completed Game Deterministically

## Status

Accepted

## Behavior

A completed game can be replayed from an empty board through its recorded moves, and reaches exactly the recorded final board and result.

## Context

Replays let anyone confirm how a game ended and trust its result. Determinism means the same moves always produce the same game, whichever interface made them.

Traces to: SPEC.md DATA-004, DATA-005, NFR-001, UI-010, AT-009; OUTCOMES.md 4, 5.

## Acceptance Criteria

- AC-1: The replay for a completed game provides the initial configuration (mode, players), the ordered moves and the final result.
- AC-2: Applying the recorded moves in order to an empty 6 × 7 board reproduces the final board exactly.
- AC-3: Playing the same sequence of columns in two separate new games produces identical boards, winners and result reasons.
- AC-4: The browser replay view can step from the empty board to the final position one move at a time.

## Verification

### Scenario 1: Replay reproduces the result

- **Given** a completed game `G` that red won with columns 0,0,1,1,2,2,3
- **When** a client fetches the replay of `G` and applies its moves to an empty board
- **Then** the resulting board equals `G`'s final board, and the winner is `red` by `connect_four`

### Scenario 2: Determinism across games

- **Given** two new in-progress games `G1` and `G2`
- **When** the column sequence 3,3,4,4,5,2,6 is played in both games
- **Then** `G1` and `G2` have identical boards, `winner` and `resultReason`

### Scenario 3: Step through in the browser

- **Given** a completed game with 7 moves
- **When** a user opens its replay and steps forward one move at a time
- **Then** after step 1 the board shows one disc, and after step 7 the board matches the final board
