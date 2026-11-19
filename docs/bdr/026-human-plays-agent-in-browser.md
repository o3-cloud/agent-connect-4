# BDR-026: Human Plays an Agent in the Browser

## Status

Accepted

## Behavior

A person can open the web app, start a game against an AI agent, and play it to a result entirely through the browser.

## Context

Humans challenging agents is a headline outcome, and it must need no technical knowledge: no CLI, no API calls.

Traces to: SPEC.md UI-001..007, UI-009, AT-001, Definition of Done 4; OUTCOMES.md 2.

## Acceptance Criteria

- AC-1: From the dashboard, a user can choose the Human vs Agent mode, see its setup options, and create the game.
- AC-2: On the user's turn, clicking a non-full column places their disc in that column's lowest empty cell, as the server reports it.
- AC-3: While it is not the user's turn, the board does not accept moves from the user.
- AC-4: The page shows both players, their disc colors, whose turn it is and the game status.
- AC-5: When the game ends, the page shows the winner or draw and accepts no further moves.

## Verification

### Scenario 1: First move as red

- **Given** a human-vs-agent game with both slots filled and the human as red
- **When** the human clicks column 3 in the browser
- **Then** a red disc appears in the bottom cell of column 3, and the turn indicator shows yellow
- **And** the agent reading state through any interface sees `board[5][3]` = `red` and `currentTurn` = `yellow`

### Scenario 2: Board locked on the agent's turn

- **Given** a human-vs-agent game where it is the agent's turn
- **When** the human clicks any column
- **Then** no move is submitted and the board is unchanged

### Scenario 3: Result shown

- **Given** a game the agent has just won
- **When** the human views the game page
- **Then** the page states that the agent won, and clicking a column has no effect
