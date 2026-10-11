# AGENTS.md — equinox

Equinox: a feasibility prototype for PEAK6 Labs that detects equivalent prediction markets across two venues and simulates explained, deterministic routing between them. Built with software-factory-kit 0.1 (spec-driven, one milestone
per merge request). Every harness (Claude Code, Codex, Cursor, Antigravity) reads this file.

## Commands
| Purpose | Command |
|---|---|
| Setup | Go 1.27 (`brew install go`); no other dependencies (standard library only, D13) |
| Build | `go build -o bin/equinox ./cmd/equinox` |
| Run locally (recorded venue data) | `./bin/equinox serve` then open http://localhost:8080 |
| Run locally (live venue data) | `./bin/equinox serve -live` (live calls go through the system curl, see Project specifics) |
| One refresh / one order | `./bin/equinox refresh`, `./bin/equinox route -pair <id> -side yes -qty 100 -region US` |
| **All checks (the one green/red gate)** | `make check` (gofmt, go vet, go test, diagram embeds, decision review) |
| Tests only | `go test ./...` |
| Lint / format | `gofmt -l cmd internal` |
| Deploy | later (D21: run locally first); see `docs/runbooks/deploy.md` when it exists (human-only) |

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
- Clean restart (v2) under software-factory-kit (D1). The v1 one-shot build lives in `main@9c3bf48`; cite it only as a labelled prior experiment ("v1, one-shot"), re-verify its facts, never quote its numbers for v2, never copy its code or design forward without a gate choosing it (D3).
- `main` is updated only when v2's `make check` is green and covers every REQ; until then milestone MRs stack on the restart branch (D4).
- Stack: Go 1.27 standard library only (D13); no database (D15); one service + command line in `cmd/equinox` (D14).
- This Mac's outbound firewall (Little Snitch) blocks newly built binaries from the internet, so live venue calls go through the system `curl` (`fetch.Curl`); set `EQUINOX_NATIVE_HTTP=1` elsewhere. Tests never touch the network: they replay `testdata/recordings` (D17).
- Partner documents are uncleared for this public repo: paraphrase them with page/section refs, never quote them.
- Local v1 leftovers (`/DECISION_JOURNAL.md`, `/bin/`, `/data/`, `/research/`) are gitignored; never commit them.
