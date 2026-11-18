# Behavior Decision Records

Behavior contracts for this stack.
Each BDR captures a single capability the system agrees to uphold,
with acceptance criteria a black-box observer can confirm.

## Source

BDR-001 to BDR-033 were extracted from [`SPEC.md`](../../SPEC.md) and
[`OUTCOMES.md`](../../OUTCOMES.md). Each BDR's Context section lists the
requirement IDs and outcomes it traces to. They are all `Proposed` until they
are verified end to end ([`bdr.md`](../specs/practices/bdr.md) rule 13).

Not extracted, because they describe implementation rather than external behavior
(they belong in ADRs or tests): NFR-007 (engine testable in isolation) and
DATA-003 (atomic persistence transaction). The observable effects of DATA-003
are covered by BDR-006, BDR-011 and BDR-018.

## Index

| BDR | Title | Status |
| --- | ----- | ------ |
| [001](001-create-game.md) | Create a Game | Accepted |
| [002](002-join-and-start-game.md) | Join a Game and Start Play | Accepted |
| [003](003-read-game-state.md) | Read Authoritative Game State | Accepted |
| [004](004-list-games.md) | List Visible Games | Accepted |
| [005](005-submit-legal-move.md) | Submit a Legal Move | Accepted |
| [006](006-reject-invalid-move.md) | Reject Invalid Moves Without Changing State | Accepted |
| [007](007-enforce-turn-and-membership.md) | Enforce Turn Order and Player Membership | Accepted |
| [008](008-detect-win.md) | Detect a Win and End the Game | Accepted |
| [009](009-detect-draw.md) | Detect a Draw | Accepted |
| [010](010-resign-game.md) | Resign a Game | Accepted |
| [011](011-reject-stale-and-concurrent-moves.md) | Reject Stale and Concurrent Moves | Proposed |
| [012](012-idempotent-move-retry.md) | Idempotent Move Retries | Proposed |
| [013](013-authenticate-agent-operations.md) | Authenticate Agent Operations and Protect Credentials | Proposed |
| [014](014-rate-limit-clients.md) | Rate-Limit Excess Requests Per Client | Proposed |
| [015](015-structured-errors.md) | Structured, Machine-Readable Errors | Proposed |
| [016](016-move-history.md) | Ordered Move History | Proposed |
| [017](017-replay-completed-game.md) | Replay a Completed Game Deterministically | Proposed |
| [018](018-survive-server-restart.md) | Preserve Games Across Server Restarts | Proposed |
| [019](019-cli-play-and-json-output.md) | Play Through the `connect4` CLI With Machine-Readable Output | Accepted |
| [020](020-cli-watch-game.md) | Watch a Game From the CLI | Proposed |
| [021](021-mcp-tools.md) | Play Through MCP Tools | Proposed |
| [022](022-self-describing-interfaces.md) | Agents Can Discover How to Play | Proposed |
| [023](023-cross-interface-consistency.md) | Identical Rules on Every Interface | Proposed |
| [024](024-autonomous-agent-match.md) | Two External Agents Complete a Game Autonomously | Proposed |
| [025](025-agent-failure-isolation.md) | Isolate Agent Failures | Proposed |
| [026](026-human-plays-agent-in-browser.md) | Human Plays an Agent in the Browser | Proposed |
| [027](027-live-spectating.md) | Watch Games Live Without Refreshing | Proposed |
| [028](028-browser-reconnect-recovery.md) | Recover After a Browser Disconnect | Proposed |
| [029](029-responsive-layout.md) | Usable on Mobile and Desktop | Proposed |
| [030](030-start-playing-quickly.md) | A Human Can Start Playing Within Seconds | Proposed |
| [031](031-move-latency-and-capacity.md) | Move Latency and Concurrent Game Capacity | Proposed |
| [032](032-operational-event-logs.md) | Structured Operational Logs | Proposed |
| [033](033-containerized-deployment.md) | Run From a Container With External Configuration | Proposed |

## Authoring

Copy [`000-template.md`](000-template.md), assign the next monotonic number, and open a PR.
