# ADR-013: Self-Issued API Keys for Agent Credentials

## Status

Accepted

Decided on 2026-11-18 by the progress-dashboard session, under the project owner's standing instruction to make the best decision the evidence supports. It unblocks BDR-013 and BDR-007. Supersede it with a new ADR if the owner chooses differently.

## Context and Problem Statement

`SPEC.md` requires authenticated agent operations (SEC-001), authorization from a server-side identity (SEC-002, SEC-004), rejection of unauthenticated or unrecognised credentials with `401 UNAUTHENTICATED` (SEC-003, §16), and credentials that never appear in output (SEC-005). It never says how an agent *gets* a credential. Today `main` accepts any non-empty bearer token as an identity. That passes the walking-skeleton tests but fails BDR-013 AC-3 ("invalid credentials are rejected"), which in turn keeps BDR-007 from being accepted.

How does an external agent obtain a credential the server can recognise?

## Decision Drivers

- **Effortless for agents** (Outcome 6): a developer can connect an agent in minutes, with no human approval step
- **Any agent can play** (Outcome 1): the same mechanism works from REST, the `connect4` CLI and MCP
- **Recognisable credentials**: the server must be able to tell a credential it issued from a made-up one (SEC-003)
- **No leaks**: the secret is never stored or echoed in plain text after issuance (SEC-005)
- **Out of scope**: accounts, rankings and social features (OUTCOMES.md "not optimizing for yet")

## Considered Options

- **Option A**: Self-issued opaque API keys from a public registration endpoint
- **Option B**: Operator-provisioned keys from configuration or an admin command
- **Option C**: OAuth 2.0 / OpenID Connect through an external identity provider
- **Option D**: Keep accepting any bearer token as the identity

## Pros and Cons of the Options

### Option A: Self-issued API keys

- Pro: an agent gets a working key with one request, which meets Outcome 6
- Pro: the server stores only a hash, so a made-up or leaked-and-revoked key is rejected
- Pro: the bearer header is identical across REST, CLI and MCP
- Con: anyone can mint identities, so abuse control relies on rate limits (BDR-014)

### Option B: Operator-provisioned keys

- Pro: the operator knows every agent
- Con: a human must act before an agent can play, which contradicts Outcomes 1 and 6

### Option C: OAuth / OIDC

- Pro: industry-standard delegated identity
- Con: needs an identity provider, client registration and token refresh in every agent; far beyond the initial release

### Option D: Any bearer token

- Pro: no work
- Con: fails SEC-001 and SEC-003; anyone can impersonate any player by guessing their token string

## Decision Outcome

Chosen option: **Option A, self-issued API keys**.

- `POST /api/v1/agents` with `{"displayName": "...", "type": "agent" | "human"}` needs no credentials. It returns `201` with `{"agentId": "agt_…", "displayName": …, "apiKey": "c4_…"}`. The key is 32 random bytes, base64url-encoded, behind the `c4_` prefix.
- The response that issues the key is the only time the server returns it. The server stores a SHA-256 hash of the key, never the key itself.
- Every authenticated operation takes `Authorization: Bearer <apiKey>`. A missing, malformed or unrecognised key returns `401 UNAUTHENTICATED`. The acting player is the `agentId` bound to the key (SEC-004).
- Keys live in the same store as games, so they become durable when persistence lands (BDR-018).
- The registration endpoint is rate-limited per client address once BDR-014 lands.
- Tests and reference agents register a key first. The any-token test identity is removed.
- The browser UI (Phase 3) uses the same endpoint once per visitor and keeps the key in an `HttpOnly` cookie, so humans never see it.

## Consequences

### Positive

- BDR-013 AC-3 and BDR-007 AC-3 become testable and can pass.
- The CLI (`connect4 register`) and MCP server (`register` tool) can share one onboarding step.

### Negative

- Identities are cheap to create. Rate limiting (BDR-014) is the only abuse control in the initial release.
- A lost key cannot be recovered; the agent registers again under a new identity.

### Neutral

- `SPEC.md` gains API-009 (Agent Registration). The OpenAPI document gains the endpoint when it is built.

## Compliance

- An HTTP test registers two agents, plays a game with their keys, and asserts that a made-up `c4_` key gets `401 UNAUTHENTICATED` on create, join and move.
- A test collects game state, history, the game list and spectator events, and asserts that neither key string appears (BDR-013 AC-4).
- No code path logs the `Authorization` header or the issued key.

## References

- `SPEC.md` SEC-001..005, §16 Error Codes, Phase 2 "Agent credentials"
- [BDR-013](../bdr/013-authenticate-agent-operations.md), [BDR-007](../bdr/007-enforce-turn-and-membership.md), [BDR-014](../bdr/014-rate-limit-clients.md)
- `prototype/service.js`: models issued credentials the same way
