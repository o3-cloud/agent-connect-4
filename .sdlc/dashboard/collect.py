#!/usr/bin/env python3
"""Collect a progress snapshot for the Arena Loop Progress dashboard.

Reads the loop mailbox (.sdlc/mailbox), the BDRs (docs/bdr), OUTCOMES.md and git,
appends one snapshot to history.jsonl, and writes the dashboard's JSON files to out/.
Run from the repository root:  python3 .sdlc/dashboard/collect.py
"""
import datetime as dt
import json
import os
import re
import subprocess
from pathlib import Path

ROOT = Path.cwd()
HERE = Path(__file__).resolve().parent
OUT = HERE / "out"
MAILBOX = ROOT / ".sdlc" / "mailbox"
NOW = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
STAMP = NOW.isoformat().replace("+00:00", "Z")
CODE = Path(__file__).read_text()

OUTCOMES = {
    1: "Any AI agent can play",
    2: "Humans can play against AI",
    3: "AI agents compete autonomously",
    4: "Every game is observable",
    5: "Reliable and fair",
    6: "Effortless experience",
}


def git(*args):
    try:
        return subprocess.run(["git", *args], cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    except subprocess.CalledProcessError:
        return ""


def front_matter(text):
    m = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    meta = {}
    if not m:
        return meta, text
    for line in m.group(1).splitlines():
        if ":" in line:
            k, v = line.split(":", 1)
            meta[k.strip()] = v.strip().strip('"')
    return meta, text[m.end():]


def listish(value):
    return [x.strip() for x in value.strip("[]").split(",") if x.strip()] if value else []


def mtime(p):
    t = dt.datetime.fromtimestamp(p.stat().st_mtime, dt.timezone.utc).replace(microsecond=0)
    return t.isoformat().replace("+00:00", "Z")


# ---- BDRs (docs/bdr as committed on HEAD; uncommitted edits are not progress) ----
bdrs = {}
for p in sorted(Path(l) for l in git("ls-tree", "--name-only", "HEAD", "docs/bdr/").splitlines()
                if re.match(r"\d{3}-.*\.md$", Path(l).name)):
    if p.name.startswith("000"):
        continue
    text = git("show", f"HEAD:{p.as_posix()}")
    num = p.name[:3]
    title = re.search(r"^# BDR-\d+: (.+)$", text, re.M)
    status = re.search(r"## Status\s+(\S.*)", text)
    traces = re.search(r"Traces to:.*?OUTCOMES\.md ([0-9, ]+)", text)
    bdrs[f"BDR-{num}"] = {
        "bdr": f"BDR-{num}",
        "title": title.group(1) if title else p.stem,
        "status": status.group(1).strip() if status else "Unknown",
        "outcomes": [int(x) for x in re.findall(r"\d", traces.group(1))] if traces else [],
        "acTotal": len(re.findall(r"^- AC-\d+:", text, re.M)),
    }
main_bdr_count = len([l for l in git("ls-tree", "--name-only", "main", "docs/bdr/").splitlines()
                      if re.search(r"/\d{3}-", l) and "/000-" not in l])

# ---- mailbox issues ----
issues = []
for p in sorted((MAILBOX / "issues").glob("*.md")):
    meta, body = front_matter(p.read_text())
    done = len(re.findall(r"^- \[[xX]\]", body, re.M))
    ac_done = len(re.findall(r"^- \[[xX]\] AC-\d+", body, re.M))
    todo = len(re.findall(r"^- \[ \]", body, re.M))
    status = meta.get("status", "open")
    issues.append({
        "id": p.stem,
        "title": meta.get("title", p.stem),
        "status": status,
        "source": meta.get("source", ""),
        "bdr": meta.get("bdr", ""),
        "branch": meta.get("branch", ""),
        "found": meta.get("found", ""),
        "closedApprox": mtime(p)[:10] if status == "closed" else "",
        "acDone": done,
        "acCriteriaDone": ac_done,
        "acTotal": done + todo,
        "hitl": meta.get("needs", "") == "human" or meta.get("hitl", "") == "true",
        "claimedBy": meta.get("claimed_by", ""),
        "claimedAt": meta.get("claimed_at", ""),
    })

# ---- backlog ----
backlog = []
for p in sorted((MAILBOX / "backlog").glob("*.md")):
    meta, _ = front_matter(p.read_text())
    backlog.append({
        "id": p.stem,
        "title": meta.get("title", p.stem),
        "status": meta.get("status", "proposed"),
        "type": meta.get("type", ""),
        "outcomes": ", ".join(listish(meta.get("outcomes", ""))),
        "created": meta.get("created", ""),
        "claimedBy": meta.get("claimed_by", ""),
        "claimedAt": meta.get("claimed_at", ""),
    })

# BDR progress = checked acceptance criteria on its tracking issue (or all, once Accepted).
by_bdr = {i["bdr"]: i for i in issues if i["bdr"]}
bdr_rows = []
for b in bdrs.values():
    iss = by_bdr.get(b["bdr"])
    accepted = b["status"] == "Accepted"
    ac_done = b["acTotal"] if accepted else min(iss["acCriteriaDone"], b["acTotal"]) if iss else 0
    bdr_rows.append({
        "bdr": b["bdr"], "title": b["title"], "status": b["status"],
        "outcomes": ", ".join(map(str, b["outcomes"])),
        "issue": iss["status"] if iss else "none",
        "acDone": ac_done, "acTotal": b["acTotal"],
        "satisfied": accepted or bool(iss and iss["status"] == "closed"),
    })

outcome_rows = []
for n, name in OUTCOMES.items():
    rel = [r for r in bdr_rows if str(n) in r["outcomes"].split(", ")]
    outcome_rows.append({
        "outcome": n, "name": name, "bdrs": len(rel),
        "bdrsSatisfied": sum(r["satisfied"] for r in rel),
        "acDone": sum(r["acDone"] for r in rel), "acTotal": sum(r["acTotal"] for r in rel),
        "backlogItems": sum(str(n) in b["outcomes"].split(", ") for b in backlog),
    })

# ---- git / loop activity ----
branch = git("rev-parse", "--abbrev-ref", "HEAD")
ahead_origin = git("rev-list", "--count", "origin/main..main") or "0"
branch_ahead = git("rev-list", "--count", f"main..{branch}") if branch != "main" else "0"
commits_24h = git("rev-list", "--count", "--all", "--since=24 hours ago") or "0"
mailbox_files = list(MAILBOX.rglob("*.md"))
last_mailbox = max((mtime(p) for p in mailbox_files), default="")

# ---- snapshot ----
snap = {
    "at": STAMP,
    "issuesOpen": sum(i["status"] in ("open", "in_progress", "blocked") for i in issues),
    "issuesInProgress": sum(i["status"] == "in_progress" for i in issues),
    "issuesClosed": sum(i["status"] == "closed" for i in issues),
    "issuesTotal": len(issues),
    "bdrTotal": len(bdr_rows),
    "bdrProposed": sum(r["status"] == "Proposed" for r in bdr_rows),
    "bdrAccepted": sum(r["status"] == "Accepted" for r in bdr_rows),
    "bdrSatisfied": sum(r["satisfied"] for r in bdr_rows),
    "bdrOnMain": main_bdr_count,
    "acDone": sum(r["acDone"] for r in bdr_rows),
    "acTotal": sum(r["acTotal"] for r in bdr_rows),
    "backlogTotal": len(backlog),
    "backlogProposed": sum(b["status"] == "proposed" for b in backlog),
    "mainAheadOfOrigin": int(ahead_origin),
    "branchAheadOfMain": int(branch_ahead or 0),
    "commits24h": int(commits_24h),
}
for o in outcome_rows:
    snap[f"o{o['outcome']}Done"] = o["acDone"]
    snap[f"o{o['outcome']}Satisfied"] = o["bdrsSatisfied"]
history_path = HERE / "history.jsonl"
with history_path.open("a") as f:
    f.write(json.dumps(snap) + "\n")
history = [json.loads(l) for l in history_path.read_text().splitlines() if l.strip()]

# ---- human-in-the-loop: automatic signals + judgement notes kept by the loop ----
hitl = []
for i in issues:
    if i["hitl"] and i["status"] == "open":
        hitl.append({"item": i["title"], "kind": "Flagged issue", "ask": "Decision requested in the issue", "ref": i["id"], "severity": "high"})
for b in backlog:
    if b["type"] == "spec-change" and b["status"] == "proposed":
        hitl.append({"item": b["title"], "kind": "Spec change", "ask": "Approve or reject this change to SPEC.md", "ref": b["id"], "severity": "medium"})
if snap["bdrOnMain"] < snap["bdrTotal"]:
    hitl.append({"item": f"{snap['bdrTotal'] - snap['bdrOnMain']} BDRs are not on main", "kind": "Merge",
                 "ask": f"Review and merge branch {branch} ({snap['branchAheadOfMain']} commits) through a PR",
                 "ref": "main-bdrs-not-on-main", "severity": "high"})
# Unpushed commits on main are expected while origin is the template; see hitl-notes.json.
notes_path = HERE / "hitl-notes.json"
if notes_path.exists():
    hitl += [n for n in json.loads(notes_path.read_text()) if n.get("open", True)]
for n, h in enumerate(hitl):
    h["n"] = n + 1

# ---- tracked loops (loops.json: name, session, role, cadence, mailbox sources) ----
SESSIONS = Path("/private/tmp/claude-501") / ("-" + str(ROOT).strip("/").replace("/", "-"))
lock = {}
try:
    lock = json.loads((ROOT / ".claude" / "scheduled_tasks.lock").read_text())
except (OSError, ValueError):
    pass


def pid_alive(pid):
    try:
        os.kill(int(pid), 0)
        return True
    except (OSError, ValueError, TypeError):
        return False


def live_sessions():
    """Map session id -> pid for Claude processes running in this repo.

    A session's temp folder gets a `started` file when the process starts, so a live
    `claude` process whose cwd is in this repo and whose start time is within 15 s of
    that file's mtime is that session's process.
    """
    found = {}
    try:
        pids = subprocess.run(["pgrep", "-x", "claude"], capture_output=True, text=True).stdout.split()
    except OSError:
        return found
    starts = {}
    for s in SESSIONS.iterdir() if SESSIONS.exists() else []:
        if (s / "started").exists():
            starts[s.name] = (s / "started").stat().st_mtime
    for pid in pids:
        cwd = subprocess.run(["lsof", "-a", "-p", pid, "-d", "cwd", "-Fn"], capture_output=True, text=True).stdout
        if str(ROOT) not in cwd:
            continue
        lstart = subprocess.run(["ps", "-o", "lstart=", "-p", pid], capture_output=True, text=True).stdout.strip()
        try:
            t = dt.datetime.strptime(lstart, "%a %b %d %H:%M:%S %Y").timestamp()
        except ValueError:
            continue
        best = min(starts.items(), key=lambda kv: abs(kv[1] - t), default=None)
        if best and abs(best[1] - t) <= 15:
            found[best[0]] = pid
    return found


live = live_sessions()
loops = []
REGISTRY = json.loads((ROOT / ".sdlc" / "loops.json").read_text())
for lp in REGISTRY:
    if lp.get("noLoop"):
        continue  # a known session that runs no loop; registered so it is not flagged
    sdir = SESSIONS / lp["session"]
    stamps = [p.stat().st_mtime for p in sdir.rglob("*")] if sdir.exists() else []
    last = (dt.datetime.fromtimestamp(max(stamps), dt.timezone.utc).strftime("%Y-%m-%d %H:%M") + " UTC") if stamps else ""
    if lp["loop"] == "Progress dashboard":
        status, last = "running", STAMP[:16].replace("T", " ") + " UTC"  # this run is that loop
    elif lp["session"] in live:
        status = "running"
    elif lock.get("sessionId") == lp["session"] and pid_alive(lock.get("pid")):
        status = "running"
    else:
        status = "stopped" if sdir.exists() else "not found"
    mine = [i for i in issues if i["source"] in lp.get("sources", [])]
    ideas = [b for b in backlog if lp.get("backlogType") and b["type"] == lp["backlogType"]]
    if mine:
        activity = f"{len(mine)} issues filed, {sum(i['status'] == 'closed' for i in mine)} closed"
    elif ideas:
        activity = f"{len(ideas)} backlog ideas, {sum(b['status'] != 'proposed' for b in ideas)} decided"
    elif lp["loop"] == "Progress dashboard":
        activity = f"{snap['issuesTotal']} issues, {snap['acDone']}/{snap['acTotal']} criteria tracked"
    else:
        activity = lp.get("note", "no mailbox output attributed")
    loops.append({"loop": lp["loop"], "session": lp["session"][:8], "role": lp["role"], "cadence": lp["cadence"] + (f" at :{lp['minute']:02d}" if isinstance(lp.get("minute"), int) else "") + (f" ({lp['job']})" if lp.get("job") else ""),
                  "job": lp.get("job", ""), "status": status, "lastSeen": last, "detail": activity})

# ---- loop-coordination checks (docs/specs/practices/loop-coordination.md) ----
def hitl_add(item, kind, ask, ref, severity):
    hitl.append({"item": item, "kind": kind, "ask": ask, "ref": ref, "severity": severity})


# The dashboard's own outputs are committed by this loop at the end of each run, so they are not counted here.
dirty = [p for p in (l.split(None, 1)[1] for l in git("status", "--porcelain", "--untracked-files=no").splitlines() if l.strip())
         if not p.startswith(".sdlc/dashboard/")]
if dirty:
    hitl_add(f"{len(dirty)} uncommitted files in the main checkout: " + ", ".join(dirty[:4]) + (" ..." if len(dirty) > 4 else ""),
             "Loop coordination", "Ask the owning loop to commit or park them on wip/<loop>; they are not counted as progress",
             "loop-coordination rule 11-12", "high")
registered = {lp["session"] for lp in REGISTRY}
for sid in sorted(set(live) - registered):
    hitl_add(f"Live session {sid[:8]} is not in .sdlc/loops.json", "Loop coordination",
             "Register it, or confirm it runs no loop", "loop-coordination rule 1", "medium")
timed = [lp for lp in REGISTRY if isinstance(lp.get("minute"), int)]
for a_i, a in enumerate(timed):
    for b in timed[a_i + 1:]:
        gap = abs(a["minute"] - b["minute"]) % 60
        isolated = all(lp.get("worktree") and lp.get("claims", True) for lp in (a, b))  # rule 5 exemption
        if min(gap, 60 - gap) < 10 and set(a.get("writes", [])) & set(b.get("writes", [])) and not isolated:
            hitl_add(f"{a['loop']} and {b['loop']} start within 10 minutes and write the same folder", "Loop coordination",
                     "Move one of them to another minute", "loop-coordination rule 5", "medium")
for lp in REGISTRY:
    exp = lp.get("expires", "")
    if exp:
        left = dt.datetime.fromisoformat(exp.replace("Z", "+00:00")) - NOW
        if left < dt.timedelta(hours=24):
            hitl_add(f"{lp['loop']} expires {exp[:16].replace('T', ' ')} UTC", "Loop coordination",
                     "Renew the job (re-run /loop) or let it end", "loop-coordination rule 2", "medium" if left.total_seconds() > 0 else "high")
for it in issues + backlog:
    if it.get("claimedAt"):
        try:
            age = NOW - dt.datetime.fromisoformat(it["claimedAt"].replace("Z", "+00:00"))
        except ValueError:
            continue
        if age > dt.timedelta(hours=2):
            hitl_add(f"Stale claim on {it['id']} by {it['claimedBy'][:8]}", "Loop coordination",
                     "The claim is over 2 hours old; another loop may take it over", "loop-coordination rule 7", "low")
owned, owned_prefixes = set(), []
for lp in REGISTRY:
    w = lp.get("worktree", "").split(" ")[0]
    if not w:
        continue
    if "<" in w:  # a pattern such as .sdlc/worktree/<branch>: one worktree per run
        owned_prefixes.append(str((ROOT / w.split("<")[0]).resolve()) + "/")
    else:
        owned.add(str((ROOT / w).resolve()))
for line in git("worktree", "list", "--porcelain").splitlines():
    if line.startswith("worktree "):
        wt = str(Path(line[9:]).resolve())
        if wt != str(ROOT.resolve()) and wt not in owned and not any(wt.startswith(p) for p in owned_prefixes):
            hitl_add(f"Worktree {wt} has no owner in .sdlc/loops.json", "Loop coordination",
                     "Register it to its loop or remove it", "loop-coordination rule 13", "low")
for n, h in enumerate(hitl):
    h["n"] = n + 1
snap["uncommittedFiles"] = len(dirty)
history[-1]["uncommittedFiles"] = len(dirty)
history_path.write_text("".join(json.dumps(h) + "\n" for h in history))

meta = [{"label": "collect.py", "value": CODE, "type": "code", "lang": "python"},
        {"label": "Collected", "value": STAMP}]
OUT.mkdir(exist_ok=True)
for name, rows in {
    "history": history, "issues": issues, "bdrs": bdr_rows, "backlog": backlog,
    "outcomes": outcome_rows, "hitl": hitl, "loops": loops,
}.items():
    (OUT / f"{name}.json").write_text(json.dumps({"rows": rows, "meta": meta}, indent=1))
print(json.dumps(snap))
