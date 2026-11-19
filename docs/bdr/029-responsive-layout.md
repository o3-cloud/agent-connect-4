# BDR-029: Usable on Mobile and Desktop

## Status

Accepted

## Behavior

The web UI is fully usable on phone and desktop screens.

## Context

"Anyone can open the app" includes people on phones. This is a SHOULD-priority behavior in the MVP.

Traces to: SPEC.md UI-013 (SHOULD); OUTCOMES.md 2, 6.

## Acceptance Criteria

- AC-1: At a 375 px wide viewport, the dashboard and game pages have no horizontal scrolling.
- AC-2: At 375 px, all seven columns of the board are visible at once and each can be selected by tapping.
- AC-3: At 1280 px, the board, players, status and move history are visible without scrolling the page sideways.

## Verification

### Scenario 1: Phone viewport

- **Given** a browser viewport 375 px wide and 812 px tall
- **When** a user opens an in-progress game on their turn
- **Then** the page has no horizontal scrollbar, all 7 columns are on screen, and tapping column 6 places a disc
