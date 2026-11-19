---
title: "README REST walkthrough fails at step 1: made-up bearer tokens are now rejected"
status: closed
source: bdr-verification
type: defect
branch: main
commit: 47b8729
found: 2026-11-18
spec: [ADR-013, API-009, interface/commands.md]
bdr: [BDR-013, BDR-030]
---

# README REST walkthrough fails at step 1: made-up bearer tokens are now rejected

## Observed

A new user who runs `make run` and pastes the "Playing over the REST API" block from `README.md` gets nowhere. The first command creates a game with `Authorization: Bearer alpha`:

```
{"error":{"code":"UNAUTHENTICATED","message":"the API key is not recognised; get one from POST /api/v1/agents","retryable":false}}  [401]
```

`GAME` is then empty, and every later command in the block fails. The README still says "Until real credentials arrive (BDR-013), any non-empty token is a test identity". 39f853d removed that test identity.

## Expected

The README's quick-start commands work as written on main, and it explains how to get credentials.

## Fix

Start the block by registering two agents and use their keys:

```sh
API=http://localhost:8080/api/v1
ALPHA=$(curl -s -X POST $API/agents -H 'Content-Type: application/json' \
  -d '{"displayName":"Alpha","type":"agent"}' | jq -r .apiKey)
BETA=$(curl -s -X POST $API/agents -H 'Content-Type: application/json' \
  -d '{"displayName":"Beta","type":"agent"}' | jq -r .apiKey)
# then use -H "Authorization: Bearer $ALPHA" / "$BETA" in place of alpha / beta
```

Replace the "any non-empty token" paragraph with: keys come from `POST /api/v1/agents`, are shown once, and are sent as `Authorization: Bearer <apiKey>`.

## Fixed (4bc060a, 2026-11-18)

The walkthrough now registers Alpha and Beta with `POST /api/v1/agents` and uses `$ALPHA` / `$BETA`; the "any non-empty token" paragraph is replaced. I ran the block verbatim against a fresh `bin/arena` on :8080: register, create, join, join, move, state, history and resign all succeed, and the game ends `completed`. `make check` passes. (I caused this regression in 39f853d.)
