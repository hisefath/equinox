<!--
One merge request per milestone (WORKFLOW.md §8). report-milestone fills this from docs/milestones/M<N>.md.
Fill every section; write "n/a — <why>" instead of deleting one. Paste evidence, never summarize it.
Title: M<N> <goal in a few words>
-->

## TL;DR

<One line: what this milestone changes for a user or a reviewer, in plain words.>

## What

- Milestone: **M<N>** — <one-sentence goal> · branch `milestone/m<N>-<slug>` · report `docs/milestones/M<N>.md`
- Specs: `specs/M<N>-<slug>/` (brief, acceptance, tasks — immutable after merge)

| Task | Squash commit | What changed (one line) |
|---|---|---|
| M<N>-T001 | `<sha>` | |

## Why (FR ids)

<!-- Which requirements this milestone delivers. Every FR/AE listed in a task's covers: must appear here. -->
- FR-### — <requirement title> (`docs/product/prd.md#fr-###`) — covered by M<N>-T### · AE-###
- Not delivered (planned but cut, with reason): <none | FR-### because …>

## Decisions (D-ids)

<!-- Decisions made at gates during this milestone, from docs/decisions/DECISION_JOURNAL.md. -->
- D<n> <title> — <choice in one line> (Type 1 | Type 2)

## Verification

Run by the supervisor on the milestone head, not pasted from a worker:

```text
$ git rev-parse --short HEAD
<sha>
$ make check
<paste the output tail, including the test counts>
$ echo $?
0
```

- [ ] `scripts/scope-guard.sh <task-file> milestone/m<N>-<slug>` exited 0 for every task (paste the OK lines)
- [ ] This MR's pipeline is green on the human's runner: <pipeline link>
- [ ] Acceptance tests in `tests/acceptance/**` were written before the code and now pass (counts from the output above)
- [ ] `scripts/embed-diagrams.sh --check` is clean for every changed design doc

## Verifier dispositions

Review file(s): `.handoff/reviews/M<N>-pass<k>.md` — verifier ≠ author; the verifier never saw worker self-reports.

| Row | Axis | Severity | Disposition (fix / file / keep / discard) | Where handled (commit, issue, or reason) |
|---|---|---|---|---|
| R1 | | | | |

## Security

- Checklist: `docs/security/CHECKLIST.md` @ `<sha>` — unaccepted high/critical findings: <0>
- Secret scan: <gitleaks version + result | NOT INSTALLED — diff checked by eye>
- Dependency scan: <tool + result | n/a — no dependency changes>
- Accepted risks (each with its `.handoff/TRACKER.md ## Approvals` row): <none>

## Rollback

- One command: `git revert -m 1 <merge sha>` (milestones merge with a merge commit, so one revert removes one milestone)
- Data / migrations / config: <none | how to reverse them, in order>

## Agent disclosure

| Harness | Model | Used for (skills / task ids) |
|---|---|---|
| Claude Code | <model id> | supervisor, <skills> |
| Codex | <model id> | <skills / task ids> |

Human approvals for every gate are recorded verbatim in `.handoff/TRACKER.md ## Approvals`.
Nothing graded its own work: the verifier, scope-guard and `make check` ran independently of the authors.
