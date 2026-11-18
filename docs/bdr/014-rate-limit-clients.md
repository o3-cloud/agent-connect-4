# BDR-014: Rate-Limit Excess Requests Per Client

## Status

Proposed

## Behavior

A client that exceeds the configured request limit gets rate-limit errors for the excess requests, and other clients and games are unaffected.

## Context

A misbehaving or looping agent must not degrade the platform for everyone else.

Traces to: SPEC.md SEC-007, AGENT-005.

## Acceptance Criteria

- AC-1: With the request limit configured to N per window, the first N requests in a window are processed normally.
- AC-2: Requests beyond N in the same window receive a structured rate-limit error and do not change game state.
- AC-3: While one client is being rate-limited, another client's requests and games proceed normally.

## Verification

### Scenario 1: One noisy client

- **Given** the server is configured with a low limit (for example 5 requests per 10 seconds)
- **And** agent A and agent B are each playing a separate game
- **When** agent A sends 10 state requests within the window
- **Then** at least 5 of agent A's requests receive a rate-limit error
- **And** agent B can still submit an accepted move in its own game during the same window
