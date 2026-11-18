---
id: bdr-definition
layer: practices
type: spec
extends: [bdr, bdr-ears]
---

# What a BDR Is, in EARS

## Purpose

`bdr` defines Behavior Decision Records as a list of rules. This spec restates that definition as EARS requirements, so that a reviewer or a loop can check each property of a BDR as pass or fail. It adds no new rules. Where this spec and `bdr` disagree, `bdr` wins and this spec is corrected.

## Definition

A Behavior Decision Record (BDR) is a Markdown file that records one externally observable capability the system agrees to uphold. It states the capability in one sentence, says why it exists, lists three to five acceptance criteria a black-box observer can confirm, and gives Given / When / Then scenarios that turn into integration tests. An ADR records *how* the system is built; a BDR records *what it must do for users*.

## References

- **spec** `bdr`: the rules this spec restates (the trace column below)
- **spec** `bdr-ears`: the EARS grammar for a BDR's acceptance criteria
- **spec** `madr`: the sibling format for architectural decisions

## Conventions

- The system name is `the BDR` for requirements about one record and `the BDR log` for requirements about the directory of records. Each requirement uses exactly one of them.
- "Merge" means the BDR lands on the main branch.
- Each requirement has an id `BDR-DEF-nn` and traces to a `bdr` rule.

## Requirements

### Location and naming

| Id | Requirement | Pattern | `bdr` rule |
|---|---|---|---|
| BDR-DEF-01 | The BDR log shall keep every BDR in one directory, `docs/bdr/`, named in the repository README. | Ubiquitous | 1 |
| BDR-DEF-02 | The BDR shall have a file name of the form `NNN-kebab-title.md`, where `NNN` is a three-digit, zero-padded sequence number. | Ubiquitous | 2 |
| BDR-DEF-03 | When an author creates a BDR, the BDR log shall assign it the next unused sequence number. | Event-driven | 3 |
| BDR-DEF-04 | If a sequence number has been used by a merged or retired BDR, then the BDR log shall keep that number bound to that BDR. | Unwanted behaviour | 3 |
| BDR-DEF-05 | The BDR log shall keep an index file that lists every BDR by number, title and current status. | Ubiquitous | 18 |

### Structure

| Id | Requirement | Pattern | `bdr` rule |
|---|---|---|---|
| BDR-DEF-06 | The BDR shall begin with a level-1 heading `# BDR-NNN: Short Title` whose number matches its file name. | Ubiquitous | 4 |
| BDR-DEF-07 | The BDR shall contain a `## Status` section holding exactly one of `Proposed`, `Accepted`, `Rejected`, `Deprecated` or `Superseded by BDR-NNN`. | Ubiquitous | 5 |
| BDR-DEF-08 | The BDR shall contain a `## Behavior` section that states the capability in one sentence. | Ubiquitous | 6 |
| BDR-DEF-09 | The BDR shall contain a `## Context` section that names the user need or external constraint behind the behaviour. | Ubiquitous | 7 |
| BDR-DEF-10 | The BDR shall contain an `## Acceptance Criteria` section with three to five criteria written as EARS requirements (`bdr-ears`). | Ubiquitous | 8, 12 |
| BDR-DEF-11 | The BDR shall contain a `## Verification` section with Given / When / Then scenarios concrete enough to copy into a test file. | Ubiquitous | 9 |
| BDR-DEF-12 | The BDR shall refer to another BDR as `BDR-NNN` with a relative Markdown link to its file. | Ubiquitous | 16 |

### Content

| Id | Requirement | Pattern | `bdr` rule |
|---|---|---|---|
| BDR-DEF-13 | The BDR shall state its Behavior, Acceptance Criteria and Verification as falsifiable claims about externally observable output. | Ubiquitous | 11 |
| BDR-DEF-14 | If a criterion or scenario names an internal class, function, framework call, schema or other implementation detail, then the BDR shall move that content to an ADR before merge. | Unwanted behaviour | 6, 8, 10 |
| BDR-DEF-15 | If a criterion or scenario cannot be checked by an external integration test without reading source code, then the BDR shall rewrite it before merge. | Unwanted behaviour | 10 |
| BDR-DEF-16 | If the BDR uses a subjective qualifier such as "fast" or "user-friendly", then the BDR shall give that qualifier a measurable threshold. | Unwanted behaviour | 11 |
| BDR-DEF-17 | If the behaviour needs more than five core criteria, then the BDR shall move the extra criteria to a follow-up BDR. | Unwanted behaviour | 12 |

### Lifecycle

| Id | Requirement | Pattern | `bdr` rule |
|---|---|---|---|
| BDR-DEF-18 | When an author creates a BDR, the BDR shall have status `Proposed`. | Event-driven | 5 |
| BDR-DEF-19 | When the behaviour has been verified end to end outside the test harness (for example with `curl`, an observability check or a manual run), the BDR shall change its status from `Proposed` to `Accepted`. | Event-driven | 13 |
| BDR-DEF-20 | While the BDR has status `Accepted`, the BDR shall keep its Behavior, Acceptance Criteria and Verification sections unchanged. | State-driven | 14 |
| BDR-DEF-21 | If an Accepted BDR's contract needs to change, then the BDR log shall record the change in a new BDR that supersedes it. | Unwanted behaviour | 14 |
| BDR-DEF-22 | When a new BDR supersedes an earlier one, the BDR shall have its status set to `Superseded by BDR-NNN` in the same pull request. | Event-driven | 15 |
| BDR-DEF-23 | When a BDR is rejected, the BDR log shall keep its file with status `Rejected` and a short rationale. | Event-driven | 17 |

In BDR-DEF-22, "the BDR" is the earlier, superseded record.

### Pull requests

| Id | Requirement | Pattern | `bdr` rule |
|---|---|---|---|
| BDR-DEF-24 | Where the repository has a pull request template, the BDR log shall be backed by a "Which BDR does this verify?" prompt in that template. | Optional feature | 19 |
| BDR-DEF-25 | When a pull request changes externally observable behaviour, the BDR log shall be cited in that pull request by the number of each BDR it implements or extends. | Event-driven | 19 |

## Checking

Most requirements can be checked mechanically against the files in `docs/bdr/`:

- **File checks:** BDR-DEF-01, 02, 05, 06 and 07 hold when the file name, heading, status line and index entry all agree.
- **Sections:** BDR-DEF-08 to 11 hold when the four sections exist, in any order.
- **Criteria count and grammar:** BDR-DEF-10 holds when the section has three to five `- AC-n:` lines and each matches the `bdr-ears` pattern.
- **Accepted contracts:** BDR-DEF-20 holds when `git log -p` shows no change to those three sections after the commit that set `Accepted`.

BDR-DEF-13 to 16 and BDR-DEF-19 need a reviewer: whether something is observable, implementation-free or measurable, and whether verification really happened outside the harness, are judgements, not text patterns.
