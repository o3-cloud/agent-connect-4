---
title: "Uncommitted BDR-034 draft in the main checkout blocks the BDR-028 acceptance merge"
status: closed
source: bdr-verification
type: loop
needs: human
found: 2026-11-19
fixed_in: dbbefd5
spec: [docs/specs/practices/loop-coordination.md]
bdr: [BDR-028, BDR-034]
---

# Uncommitted BDR-034 draft in the main checkout blocks the BDR-028 acceptance merge

## Observed

Since about 15:01 local (20:01 UTC) on 2026-11-19, the main checkout has uncommitted changes that no loop has committed:

- `docs/bdr/034-join-open-seat-in-browser.md` (untracked): a new BDR draft
- `docs/bdr/README.md` (modified): adds the BDR-034 index row
- `.sdlc/mailbox/backlog/idea-agent-onboarding-discovery.md`, `idea-loop-isolate-working-trees.md`, `idea-loop-mailbox-is-tracked-now.md`, `idea-loop-verify-claims-before-filing.md` (modified)

The BDR verify loop verified BDR-028 against 3e11fb6 and committed its acceptance on branch `verify/accept-028` (rebased onto 2517b88, `make check` PASS). `git merge --ff-only verify/accept-028` in the main checkout aborts on two consecutive runs because the uncommitted `docs/bdr/README.md` edit would be overwritten.

Per rule 12, the verify loop has not committed, stashed or reverted these files.

## Needed

Whoever owns the draft commits it (from a worktree, per rule 9) or moves it to a `wip/` branch, so the main checkout is clean. The verify loop then fast-forwards `verify/accept-028` on its next run.

## Resolution (2026-11-20T02:20Z)

The BDR-034 draft no longer sits in the main checkout: it is committed on branch `bdr/034-036-browser-seats` (5be86ea, with BDR-035 and BDR-036), and `docs/bdr` in the main checkout is clean. `verify/accept-028` had diverged from main in the meantime, so its acceptance commit was cherry-picked onto main as dbbefd5. Merging the BDR-034..036 branch is still the owner's call.
