---
id: M3-D02
milestone: M3
skill: draw-diagrams
covers: []
after: []
harness: claude
mode: workspace-write
hitl: false
parallel: true
budget: 1 run, 15 min
slug: hld-container
milestone_branch: milestone/m3-design
---
Owns: docs/architecture/diagrams/hld-container.mmd

## Ask

Draw hld-container diagram

Draw hld-container from docs/architecture/hld.md.

- Why claude: Claude subagent is the default worker (CLAUDE.md Dispatch); Codex is not on PATH in this session.

## Done-when

draw-diagrams' own Done-when checks pass, and `bash scripts/scope-guard.sh .handoff/runs/M3-D02/task.md milestone/m3-design` exits 0.

## Must not change

- every path except docs/architecture/diagrams/hld-container.mmd, above all docs/architecture/*.md

## Consumes / Produces

- Consumes: none
- Produces: none

## Context manifest

- give: this file; AGENTS.md; docs/architecture/hld.md#3-containers-c4-level-2, docs/architecture/hld.md#4-connections, docs/architecture/hld.md#5-trust-boundaries--threats
- never give: other runs/, any worker's self-report or result.md, .handoff/TRACKER.md

## Return

- Commit docs/architecture/diagrams/hld-container.mmd on the branch your worktree was started on, with trailer `Task: M3-D02`; never push or merge.
- Write $HANDOFF_DIR/runs/M3-D02/result.md from specs/_templates/result.md.
