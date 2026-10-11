---
id: M3-D04
milestone: M3
skill: draw-diagrams
covers: []
after: []
harness: claude
mode: workspace-write
hitl: false
parallel: true
budget: 1 run, 15 min
slug: erd
milestone_branch: milestone/m3-design
---
Owns: docs/architecture/diagrams/erd.mmd

## Ask

Draw erd diagram

Draw erd from docs/architecture/lld.md.

- Why claude: Claude subagent is the default worker (CLAUDE.md Dispatch); Codex is not on PATH in this session.

## Done-when

draw-diagrams' own Done-when checks pass, and `bash scripts/scope-guard.sh .handoff/runs/M3-D04/task.md milestone/m3-design` exits 0.

## Must not change

- every path except docs/architecture/diagrams/erd.mmd, above all docs/architecture/*.md

## Consumes / Produces

- Consumes: none
- Produces: none

## Context manifest

- give: this file; AGENTS.md; docs/architecture/lld.md#1-data-model
- never give: other runs/, any worker's self-report or result.md, .handoff/TRACKER.md

## Return

- Commit docs/architecture/diagrams/erd.mmd on the branch your worktree was started on, with trailer `Task: M3-D04`; never push or merge.
- Write $HANDOFF_DIR/runs/M3-D04/result.md from specs/_templates/result.md.
