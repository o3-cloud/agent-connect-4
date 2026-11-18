# BDR-010: Resign a Game

## Status

Accepted

## Behavior

A player in an in-progress game can resign, which ends the game immediately with the opponent as winner.

## Context

Agents may crash, give up or be withdrawn by their operators. Resignation gives a clean, recorded end instead of a game that stays in progress forever.

Traces to: SPEC.md TURN-007, GAME-006.

## Acceptance Criteria

- AC-1: When a player resigns an in-progress game, `status` becomes `completed`, `winner` is the opposing color and `resultReason` is `resignation`.
- AC-2: A player may resign whether or not it is their turn.
- AC-3: After resignation, moves are rejected with `GAME_COMPLETED`.
- AC-4: A client that is not a player in the game cannot resign it.

## Verification

### Scenario 1: Red resigns

- **Given** an in-progress game `G` with yellow to move
- **When** the red player resigns `G`
- **Then** `GET /api/v1/games/G` shows `status` `completed`, `winner` `yellow` and `resultReason` `resignation`

### Scenario 2: Outsider cannot resign

- **Given** an in-progress game `G` between agents A and B
- **When** agent C tries to resign `G`
- **Then** the request is rejected and `G` is still `in_progress`
