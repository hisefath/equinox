# AGENTS.md — <project name>

<One sentence: what this project is and who it is for.> Built with software-factory-kit 0.1 (spec-driven, one milestone
per merge request). Every harness (Claude Code, Codex, Cursor, Antigravity) reads this file.

## Commands
| Purpose | Command |
|---|---|
| Setup | `<e.g. composer install && cp .env.example .env && php artisan key:generate>` |
| Run locally | `<e.g. php artisan serve>` |
| **All checks (the one green/red gate)** | `make check` |
| Tests only | `<e.g. php artisan test>` |
| Lint / format | `<e.g. vendor/bin/pint --test>` |
| Deploy | see `docs/runbooks/deploy.md` (human-only) |

## Where things live
| What | Path |
|---|---|
| Artifact index | `docs/README.md` |
| Product requirements (living) | `docs/product/prd.md` |
| High-level / low-level design | `docs/architecture/hld.md`, `docs/architecture/lld.md` |
| Diagrams (Mermaid source) | `docs/architecture/diagrams/*.mmd` (embedded by `scripts/embed-diagrams.sh`) |
| API contract | `docs/architecture/contracts/openapi.yaml` |
| Decisions | `docs/decisions/DECISION_JOURNAL.md` (+ `NNNN-*.md` for one-way decisions) |
| Milestone specs, acceptance, tasks | `specs/M<N>-<slug>/` (immutable after merge) |
| Milestones | `MILESTONES.md`, reports in `docs/milestones/M<N>.md` |
| Shared agent state | `.handoff/` (tracker, inbox, run results, reviews) |

## Rules for every agent
**Always**
- Write only the paths your skill or task file declares (`Owns:`). Phase skills stop at their STOP.
- Run `make check` and paste its output before claiming anything is done.
- Stage files explicitly (`git add <paths>`), never `git add -A` / `git add .`.
- Get times from `date`, counts from commands.
- Use ids: FR-### NFR-### SC-### AE-### (PRD), D<n> (decisions), M<N>-T### (tasks), R<n> (review rows).

**Ask first** (the supervisor asks the human; workers write the question into `result.md` and stop)
- Adding or upgrading a dependency; changing a public API, data model or migration; anything touching auth or money.

**Never**
- Edit `tests/acceptance/**` unless you are running `define-acceptance` (QA-owned).
- Push, merge, deploy, change repo visibility, or run `curl | bash`.
- Put secrets, tokens, PHI or real personal data in code, logs, commits, screenshots or `.handoff/`.
- Disable TLS verification.

## Task and result contract
Workers receive one task file (`Owns:`, Done-when, context manifest) and must write
`$HANDOFF_DIR/runs/<id>/result.md` (`status: done | human_review | relaunch`, base/head SHA, changed paths, pasted
test output, questions). Templates: `specs/_templates/task.md`, `specs/_templates/result.md`.

## Git
- `main` is protected; one branch + one merge request per milestone: `milestone/m<N>-<slug>`.
- One worktree + branch per task: `m<N>/t###-<slug>`; squashed onto the milestone branch as
  `feat(m<N>/t###): <summary>` with trailer `Task: M<N>-T###`.
- Canonical remote: Gauntlet GitLab (`origin`); GitHub is a push mirror (no PRs there).

## Project specifics
<Stack, conventions, gotchas, test data. Keep this file under ~100 lines: one line per rule, pointers not essays.>
