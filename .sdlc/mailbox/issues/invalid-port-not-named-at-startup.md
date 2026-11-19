---
title: "Invalid PORT logs server_started, then fails with 'lookup tcp/abc' and exit 1, never naming PORT"
status: closed
source: bdr-verification
type: defect
branch: main
commit: 53967bd
found: 2026-11-19
closed: 2026-11-19
spec: [README "Running in a container"]
bdr: [BDR-033]
---

# Invalid PORT is not reported as a configuration error

## Observed

The README's container section says: "An invalid value stops the server at startup with a message naming the variable." That holds for `ARENA_RATE_WINDOW`, but not for `PORT`:

```
$ docker run --rm -e PORT=abc connect4arena:local; echo rc=$?
listen tcp: lookup tcp/abc: unknown port
{"level":"INFO","msg":"server started","event":"server_started","addr":":abc"}
rc=1

$ docker run --rm -e ARENA_RATE_WINDOW=nope connect4arena:local; echo rc=$?
ARENA_RATE_WINDOW must be a positive duration such as 10s, got "nope"
rc=2
```

With an invalid `PORT`, an operator sees a `server_started` event for a server that never listened, an error that does not mention `PORT`, and a different exit code from the other settings. Log collectors that key on `server_started` count it as a successful start.

## Expected

Validate `PORT` with the other settings before anything is logged: it must be an integer from 1 to 65535. Otherwise exit 2 with `PORT must be a port number from 1 to 65535, got "abc"`. Emit `server_started` only after the listener is bound.

Add `PORT=abc` (and `PORT=70000`) to the config tests and to `make docker-smoke`'s invalid-value check.

## Resolution

`PORT` is validated with the other settings in `app.ConfigFromEnv`. A value that is not an integer from 1 to 65535 stops the server with exit 2 and `PORT must be a port number from 1 to 65535, got "abc"`, before anything is logged. The server now binds its listener first and logs `server_started` only after that succeeds, so a port in use no longer produces a start record. `arena -healthcheck` exits 1 on a bad setting, because Docker reserves health check exit code 2.

Tests (red before, green after): `TestPortConfigFromEnv`, `TestArenaRejectsInvalidPortByName` and `TestArenaLogsServerStartedOnlyAfterBinding` (runs the real binary on a busy port). `make docker-smoke` now checks `PORT=abc` and `PORT=70000`; against an image built from the previous main it fails with the issue's symptom. `make check` PASS, 91.6%.

Black-box: `docker run --rm -e PORT=abc` and `-e PORT=70000` exit 2 naming `PORT`; `PORT=abc ./bin/arena` the same; a valid `PORT` starts, answers `/healthz` and passes `-healthcheck`.
