---
title: "Invalid PORT logs server_started, then fails with 'lookup tcp/abc' and exit 1, never naming PORT"
status: open
source: bdr-verification
type: defect
branch: main
commit: 53967bd
found: 2026-11-19
claimed_by: 5b5d3e77-65ad-4b95-a4f4-99ac7cf90b0f
claimed_at: 2026-11-19T23:10Z
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
