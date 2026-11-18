# BDR-008: Detect a Win and End the Game

## Status

Proposed

## Behavior

When a move completes four or more of the same color in a horizontal, vertical or diagonal line, the system declares that player the winner and accepts no further moves.

## Context

A trustworthy result is the point of every match. All four line directions must be detected the same way, and a finished game must stay finished.

Traces to: SPEC.md RULE-005, RULE-006, RULE-007, RULE-010, GAME-006, AT-005; OUTCOMES.md 5.

## Acceptance Criteria

- AC-1: A move completing a horizontal line of four sets `status` `completed`, `winner` to the mover's color and `resultReason` `connect_four`.
- AC-2: Vertical lines and both diagonal directions are detected the same way.
- AC-3: After the win, `currentTurn` is `null` and `legalMoves` is empty.
- AC-4: Any move submitted to a completed game is rejected with error code `GAME_COMPLETED`, and the game is unchanged.

## Verification

### Scenario 1: Horizontal win

- **Given** a new in-progress game
- **When** the columns 0,0,1,1,2,2,3 are played in order (red moves first)
- **Then** after the 7th move, `status` is `completed`, `winner` is `red` and `resultReason` is `connect_four`

### Scenario 2: Vertical win

- **Given** a new in-progress game
- **When** the columns 0,1,0,1,0,1,0 are played
- **Then** red wins on the 7th move

### Scenario 3: Diagonal wins

- **Given** a new in-progress game
- **When** the columns 0,1,1,2,2,3,2,3,3,6,3 are played
- **Then** red wins on the 11th move (rising diagonal)
- **And** in a separate game, the columns 6,5,5,4,4,3,4,3,3,0,3 also give red a win on the 11th move (falling diagonal)

### Scenario 4: No moves after the win

- **Given** a game that red has won
- **When** yellow submits a move with the current version
- **Then** the response is an error with `error.code` = `GAME_COMPLETED`
