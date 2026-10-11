---
id: M1-T001
milestone: M1
skill: implement-task
covers: [FR-001, AE-001]
after: []
harness: claude
mode: workspace-write
hitl: false
parallel: false
budget: 1 run, 30 min
---
Owns: path/to/file.ext, path/to/dir-this-task-creates/*

<!--
Task contract (software-factory-kit WORKFLOW.md §7), one file per task at specs/M<N>-<slug>/tasks/T###.md; approved at G4;
it is the worker's ONLY input. This is a reference copy: slice-tasks fills each task file from FILE 1 of its own
skills/slice-tasks/assets/template.md (the canonical shape, with per-field guidance), and its analyze pass FAILs
any other shape. Delete this comment when filling by hand.

Frontmatter: exactly these 10 keys, one per line, this order (no slug:, no milestone_branch:):
  id                M<N>-T###; also the run dir, the branch m<N>/t###-<title-slug> and the `Task: M<N>-T###` trailer
  milestone         M<N>
  skill             implement-task
  covers / after    FR/AE ids this task satisfies / task ids that must be done first ([] only for T001)
  harness           claude | codex
  mode              workspace-write
  hitl              true when the worker must pause for the human (then do not dispatch it unattended)
  parallel          true only when Owns is disjoint from every other parallel task
  budget            1 run, <m> min (m <= 120); scripts/delegate.sh turns it into a hard timeout
The branch slug comes from the title line under ## Ask. The milestone branch is not in the file: run
scripts/delegate.sh from milestone/m<N>-<slug> (or pass it as the 2nd argument); the pre-push hook uses the single
milestone/m<N>-* branch.
Diagram JOB FILE variant (WORKFLOW.md §7 "Job file"): the supervisor copies this file to
.handoff/runs/M<N>-D##/task.md with id M<N>-D##, skill: draw-diagrams, harness: codex, covers/after: [], plus
slug: <type> and milestone_branch: milestone/m<N>-<slug>; Owns: exactly docs/architecture/diagrams/<type>.mmd (the
one docs/** exception); ## Ask names the type; Context manifest give: = the design row's Draw from. Commit it on
milestone_branch before scripts/delegate.sh runs.
Owns: ONE comma-separated line, no spaces inside an entry, no backticks (`*` crosses `/`). scripts/scope-guard.sh
fails any change outside it. Never tests/acceptance/**, specs/**, docs/**, .handoff/** or a dependency manifest.
Mark anything unknown with ⚠ <question> instead of guessing; a task with ⚠ is not dispatched.
-->

## Ask

<Title of at most 6 words, no bullet — delegate.sh slugifies it into the branch name>

<One sentence: the behaviour a user can watch work when this lands, in the PRD's words.>

- Slice: <tracer bullet | vertical slice | seam> — <entry (EP-n / screen) → BE-n → table → what the user sees>
- <FR-###>: "<requirement sentence, verbatim from docs/product/prd.md>"
- <AE-###>: "<Given/When/Then, verbatim>"
- Out of this task: <the nearest thing the implementer might build, and which task owns it>
- HITL: <none | the one question or human act, and when>
- Why <claude | codex>: <one line from WORKFLOW.md §6>

## Done-when

<One line, written before dispatch, that exits 0 only when the task is done: this task's AT filter, then make check.>

```sh
make test FILTER=M1_AT_001 && make check
```

- Turns green: <tests/acceptance/<path> (AT-###, AE-### / FR-###)>
- Still red afterwards, expected: <tests/acceptance/<path> — owned by M<N>-T###> | none

## Must not change

- `tests/acceptance/**` — read-only. Never edit, skip, delete or weaken a test to get green; if one looks wrong,
  stop and write it under Questions in result.md.
- `specs/**`, `docs/**`, `.handoff/**` (except your own result.md), `AGENTS.md`, `CLAUDE.md`, and every dependency
  manifest or lockfile: a new or upgraded package is an Ask-first question; write it under Questions and stop.
- <interface a neighbouring task consumes, as named in lld.md> | none
- <accepted decision: D<n> <title>> | none

## Consumes / Produces

- Consumes: <interface exactly as lld.md / openapi.yaml names it> — from <M<N>-T### | existing path:line>
- Produces: <interface exactly as named> — for <M<N>-T### | the user via <screen id>>

## Context manifest

- give: this file; `AGENTS.md`; `docs/architecture/lld.md#<section>`; `tests/acceptance/<path>` (read-only)
- never give: other tasks' `runs/`, any worker's self-report or result.md, `.handoff/TRACKER.md`, verifier reviews,
  `.env` or any secret, PHI or real personal data

## Return

- Commit on the branch your worktree was started on; never create a second branch. Every commit carries the
  trailer `Task: M1-T001`; never push, merge or edit outside Owns.
- Write `$HANDOFF_DIR/runs/M1-T001/result.md` from `specs/_templates/result.md` (status done | human_review |
  relaunch, base/head sha, changed paths, the Done-when command with its pasted output, questions). Missing
  evidence means `human_review`, never `done`.
