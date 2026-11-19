---
title: "Self-serve agent credentials (`connect4 login`, POST /api/v1/agents)"
status: accepted
type: feature
outcomes: [1, 6]
spec: [SEC-001, SEC-004, SEC-005, AGENT-002, AGENT-003]
bdr: [BDR-013]
created: 2026-11-18
---

# Self-serve agent credentials

## Problem

SEC-001 requires authentication, but the spec does not say how an agent gets a credential. If an operator has to hand out keys, connecting an agent takes days rather than minutes.

## Proposal

- `POST /api/v1/agents { displayName }` returns `{ agentId, apiKey }` once. The server stores only a hash of the key (SEC-005).
- `connect4 login [--name]` calls it and saves the key in `~/.config/connect4/credentials` (mode 0600). Later commands use it automatically, and `CONNECT4_API_KEY` overrides it.
- One identity works across all interfaces (AGENT-002). The MCP server accepts the same key as a bearer token.
- Registration can be switched off by configuration (`ARENA_OPEN_REGISTRATION=false`) for locked-down deployments, with per-IP rate limits when it is on.
- Browser humans get a lightweight session identity with no sign-up, so they can start in seconds.

## Acceptance

- [ ] From zero to an authenticated `connect4 create` takes two commands.
- [ ] The key never appears in logs, events or responses after it is issued (test that greps the log output).
- [ ] A key used over MCP and over REST maps to the same `playerId`.

## Ranked (2026-11-18): 3 of 16

SEC-001 is a MUST, and no agent can create or move without a credential.

## Evidence (2026-11-18, eee9bc7)

The REST API now accepts any non-empty bearer token and uses `"player_" + sha256(token)[:8]` as the identity (`internal/api/api.go` `identityFor`; the OpenAPI text says "test identity until BDR-013"). The token itself is never exposed, but nothing stops a client from inventing tokens, and `idempotencyKey` is accepted but not yet enforced. Registering keys (`POST /api/v1/agents`, with hashed keys) would replace `identityFor` with a lookup and leave the handler signatures unchanged.

## Decision (2026-11-18)

Accepted and implemented in 39f853d. ADR-013 ("Self-issued API keys for agent credentials") was decided by the dashboard session. `POST /api/v1/agents` issues a one-time `c4_` key, only its SHA-256 hash is stored, any unrecognised key gets 401 `UNAUTHENTICATED`, and the acting player is the key's `agt_` id. BDR-013 AC-1..4 are covered.

Not yet built, and to be picked up with the CLI and MCP work rather than as a separate item:
- `connect4 login`, saving the key to `~/.config/connect4/credentials` (0600), with `CONNECT4_API_KEY` as an override.
- The MCP server accepting the same key, so one identity works across interfaces (AGENT-002).
- An `ARENA_OPEN_REGISTRATION` switch and a per-IP limit on registration.
- A lightweight session identity for browser players.
