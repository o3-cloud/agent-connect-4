# BDR-030: A Human Can Start Playing Within Seconds

## Status

Proposed

## Behavior

A first-time visitor can go from opening the app to making their first move against an agent in a few interactions, with no sign-up or configuration beyond choosing an opponent.

## Context

OUTCOMES.md 6 asks that "a human can start playing in seconds". The thresholds below make that claim falsifiable. These thresholds were adopted on 2026-11-18.

Traces to: OUTCOMES.md 2, 6; SPEC.md UI-001, UI-002.

## Acceptance Criteria

- AC-1: From the app's start page, a new visitor can make their first move against an agent in at most 3 interactions (for example: choose Human vs Agent, create the game, click a column).
- AC-2: No account creation, API key or configuration file is needed to start a human-vs-agent game in the browser.
- AC-3: On the reference deployment, the game page is ready to accept the first move within 2 seconds of creating the game.

## Verification

### Scenario 1: First move in three interactions

- **Given** a fresh browser session that has never visited the app
- **When** the user opens the start page, selects Human vs Agent, clicks Create and clicks column 3
- **Then** a disc appears in column 3 and the turn passes to the agent
- **And** no more than 3 interactions were needed after the page loaded
