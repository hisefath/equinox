# Docs index

THE map of this repo's artifacts: one line per artifact type → where it lives, which skill writes it, and the
human approval gate that accepts it (software-factory-kit WORKFLOW.md §4). If an artifact is not reachable from this
page, agents will not find it. `garden-docs` keeps this file in sync; it lists paths, never counts.

`M<N>` = milestone number · `<slug>` = short kebab name · `T###` = task number · ids never get renumbered.

## Plan (Diamond 1: discover + define)

| Artifact | Path | Written by | Gate |
|---|---|---|---|
| Constitution (rules for every agent) | `AGENTS.md`, `CLAUDE.md`, `.claude/settings.json` | `scripts/init-repo.sh`, then human-only edits | G-init |
| Scaffolding (task/result contracts, helper scripts, build entry, git wiring) | `specs/_templates/task.md`, `specs/_templates/result.md`, `scripts/{scope-guard,delegate,embed-diagrams}.sh`, `Makefile`, `.githooks/pre-push`, `.gitignore` entries | `scripts/init-repo.sh` | G-init |
| Milestone brief | `specs/M<N>-<slug>/brief.md` | `sharpen-brief` | G0 (Full track) |
| Research notes + source list | `docs/research/R<NN>-<slug>.md`, `docs/research/sources.md` | `gather-evidence` | feeds G1 |
| Inherited-code audit (brownfield) | `docs/audit/AUDIT.md` | `audit-codebase` | G-audit (before G1) |
| Product requirements (living PRD) | `docs/product/prd.md` | `write-prd`, `clarify-spec` | **G1** |

## Design (Diamond 2: develop)

| Artifact | Path | Written by | Gate |
|---|---|---|---|
| UX flows and screens | `docs/design/flows.md`, `docs/design/screens.md` | `map-ux-flows` | G-ux |
| High-level design (HLD) | `docs/architecture/hld.md` | `design-hld` | **G2** |
| Low-level design (LLD) | `docs/architecture/lld.md` | `design-lld` | **G3** |
| API contract | `docs/architecture/contracts/openapi.yaml` | `design-lld` | G3 |
| Brownfield change plan | `specs/M<N>-<slug>/change-plan.md` | `design-lld` | G3 |
| Diagrams (Mermaid source of truth) | `docs/architecture/diagrams/<type>.mmd`, embedded under `<!-- diagram: <type> -->` by `scripts/embed-diagrams.sh` | `draw-diagrams` | inside G2 / G3 |
| Root architecture summary (when graded) | `ARCHITECTURE.md` | `garden-docs` | rides the MR |

## Deliver (Diamond 2: deliver)

| Artifact | Path | Written by | Gate |
|---|---|---|---|
| Acceptance criteria + failing tests | `specs/M<N>-<slug>/acceptance.md`, `tests/acceptance/**` | `define-acceptance` | folded into G4 |
| Task files (one per task) | `specs/M<N>-<slug>/tasks/T###.md` (from slice-tasks' `assets/template.md` FILE 1; `specs/_templates/task.md` is the same shape) | `slice-tasks` | **G4** |
| Milestone index | `MILESTONES.md` | `slice-tasks` (row), supervisor (status) | — |
| Task code | commits on `m<N>/t###-<slug>`, squashed onto `milestone/m<N>-<slug>` | `implement-task` | automated: scope-guard + `make check` |
| Bug spec + regression test | `specs/bugs/<slug>.md` | `fix-bug` | rides G5 |
| Verifier review | `.handoff/reviews/M<N>-pass<k>.md` (input: `.handoff/reviews/M<N>-diff.patch`) | `verifier` subagent | human sets each row fix / file / keep / discard |
| Security evidence | `docs/security/CHECKLIST.md` | `check-security` | inside G5 |
| Milestone report (+ MR body) | `docs/milestones/M<N>.md` | `report-milestone` | **G5** |
| Deploy runbook, health, smoke test | `docs/runbooks/deploy.md`, `docs/ops/HEALTH.md`, `scripts/smoke.sh` | `ship-milestone` | G-live |

## Records, learning, demo

| Artifact | Path | Written by | Gate |
|---|---|---|---|
| Decision journal (+ one-way ADRs) | `docs/decisions/DECISION_JOURNAL.md`, `docs/decisions/NNNN-<slug>.md` | `log-decision` | none (it is the record) |
| Supervisor state | `.handoff/TRACKER.md` (Resume, Tasks, Approvals, Log) | supervisor, `scripts/delegate.sh` | — |
| Dispatch records | `.handoff/runs/<id>/{brief.md,cmd.txt,result.md}` (`events.jsonl` gitignored) | `scripts/delegate.sh`, the worker | automated |
| Pasted research digests | `.handoff/inbox/` | the human | candidate sources only |
| Durable learnings | `docs/learnings/<slug>.md` | `route-failure` | G-L5 |
| Demo script, posts, recording | `docs/demo/SCRIPT.md`, `docs/demo/POSTS.md`, `docs/demo/SMOKE.md`, `docs/demo/RECORDING.md`, `docs/demo/out/` (`docs/demo/raw/` gitignored) | `write-demo-script`, `record-demo` | G-demo |
| Superseded material | `docs/archive/` | `garden-docs` | rides the MR |

## Rules

- `docs/product/prd.md` is living; `specs/M<N>-<slug>/` is frozen once its milestone merges.
- Templates for the task and result contracts: `specs/_templates/task.md`, `specs/_templates/result.md`.
- Private reflections and quiz notes never live here: `~/software-factory-kit-private/<repo>/`.
