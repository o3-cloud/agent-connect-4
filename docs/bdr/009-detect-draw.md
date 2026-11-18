# BDR-009: Detect a Draw

## Status

Proposed

## Behavior

When the board fills without any line of four, the system ends the game as a draw with no winner.

## Context

Games must always reach a valid outcome, including when nobody wins.

Traces to: SPEC.md RULE-008, GAME-006, RULE-010.

## Acceptance Criteria

- AC-1: When the 42nd disc fills the board and no line of four exists, `status` becomes `completed`, `winner` is `null` and `resultReason` is `draw`.
- AC-2: After a draw, `legalMoves` is empty and further moves are rejected with `GAME_COMPLETED`.
- AC-3: The move history of a drawn game contains exactly 42 moves.

## Verification

### Scenario 1: Full board draw

- **Given** a new in-progress game
- **When** the columns 0,1,0,1,0,1,2,3,2,3,2,3,4,5,4,5,4,5,6,0,6,0,6,0,1,2,1,2,1,2,3,4,3,4,3,4,5,6,5,6,5,6 are played in order
- **Then** after the 42nd move, `status` is `completed`, `winner` is `null` and `resultReason` is `draw`
- **And** the game's move history lists 42 moves
