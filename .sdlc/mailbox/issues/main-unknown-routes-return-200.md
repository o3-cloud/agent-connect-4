---
title: "Unknown routes on main return 200 instead of 404"
status: closed
source: bdr-verification
branch: main
commit: 30cd082
found: 2026-11-18
---

# Unknown routes on main return 200 instead of 404

## Observed

`internal/app/app.go` registers `GET /` as a catch-all, so any GET path answers `200 text/plain "stack-base-go"`:

```
GET /api/v1/games        -> 200 stack-base-go
GET /api/v1/openapi.json -> 200 stack-base-go
GET /mcp                 -> 200 stack-base-go
```

A client probing for the API (BDR-022, BDR-033 AC-1) gets a success status and a non-JSON body instead of a clear failure.

## Expected

Unknown paths return 404, as a structured error once the API exists (BDR-015). Register the root as `GET /{$}`.

## Resolution

Fixed on main in c89a57e (2026-11-18): root registered as `GET /{$}`. Tests added for `/unknown`, `/api/v1/games`, `/mcp` -> 404. Verified black-box: `/` 200, `/healthz` 204, `/api/v1/games`, `/api/v1/openapi.json`, `/mcp` 404; `make check` PASS. Structured JSON 404s remain with BDR-015.
