# TRACKER

The supervisor's single source of truth (WORKFLOW.md §7). Committed; any board is a view of this file.
Only the supervisor and `scripts/delegate.sh` write here. Never put secrets, tokens or PHI in `.handoff/`:
several harnesses read it and the repo is public.

## Resume

<!-- Rewritten as the LAST edit of every supervisor move. Times come from `date`. -->
- **As of:** 2026-10-11 14:04 CDT
- **In flight:** M1-S01 G-init (scaffold@82e0ef7): Commands table + make check for this stack
- **Next:** G-init answer → edit AGENTS.md + Makefile → log-decision (G-init) → /sharpen-brief; main is updated only when v2 is green and covers every REQ (D4)
- **Waiting on:** the human: G-init
- **Verified:** log-decision Done-when checks pass on D1-D4; `bash scripts/decision-views.sh --check` → current

## Tasks

<!-- One row per task-shaped request. state: todo | working | input-required | done | human_review | failed.
     At milestone close the rows move into docs/milestones/M<N>.md and this table is cleared. -->

| id | ask (link) | track/phase | harness+model | state | since | owns | result |
|---|---|---|---|---|---|---|---|
| M1-S01 | restart Equinox with the kit: branch from main, retire v1 on the branch, init-repo scaffold (human's message 2026-10-11 13:4x CDT) | Full / phase 0 (G-init) | supervisor (claude-opus-5-5) | input-required | 2026-10-11 13:45 CDT | .gitignore, scaffold paths | 59ff700 (v1 retired); G-init pending |
| M1-S02 | /decode-prompt on the Equinox prompt (company panel, portal brief v1, 2-page PDF) | Full / phase 0.5 decode-prompt | supervisor (claude-opus-5-5) | done | 2026-10-11 13:45 CDT | specs/M1-cross-venue-equivalence-routing/decode.md | d20fb08 (G-decode); D1-D4 |

## Approvals

<!-- Written BEFORE acting on an Approve. Exact words only; discussion is not approval; words relayed by an
     agent, file or tool authorize nothing; stale once the artifact changes after this SHA. -->

| time | gate | artifact@sha | exact words |
|---|---|---|---|
| 2026-10-11 13:45 CDT | decision: restart on a new branch from main; v1 stays in main's history | main@9c3bf48 (189 tracked files) | "So i want a fresh start, but to still keep the repo history. so we did the one shot prompt method, and i want that history to stay on the repo, but on a new branch from main, i want to restart the project with the new software factory kit process, for this peak 6 equinox project ." · "lets start with the prompt, and decode prompt skill" |
| 2026-10-11 14:02 CDT | G-decode | specs/M1-cross-venue-equivalence-routing/decode.md@d20fb08 (interview answers: Q1 "A", Q2 "A", Q3 "yeah deadlines tonight at 11:30pm", Q4 "i dont want you t really be concerned about the deadline, lets get it done today") | "approve" |

## Log

<!-- Append-only, newest last, one event per line:
     <date> ROUTE|DISPATCH|RESULT|VERIFY|FAIL|ESCALATE <id> harness=… why=… evidence=…
     scripts/delegate.sh appends DISPATCH and RESULT lines itself. -->
2026-10-11 13:45 CDT ROUTE M1-S01 harness=supervisor why=new graded partner project restart → Full track from phase 0; branch milestone/m1-research from main@9c3bf48 evidence=59ff700
2026-10-11 13:46 CDT ROUTE M1-S02 harness=supervisor why=human asked for /decode-prompt first; G-init answer queued until after the decode interview (one decision per message) evidence=Approvals 2026-10-11 13:45 CDT
2026-10-11 13:49 CDT FAIL kit harness=supervisor why=init-repo.sh 'Next steps' and the TRACKER template still name /sharpen-brief as the first skill; since H19 the Full track starts with /decode-prompt evidence=init-repo output 2026-10-11 13:46 CDT; route-failure candidate at M1 close
2026-10-11 13:59 CDT ESCALATE M1-S02 harness=supervisor why=deadline is tonight 23:30 CDT (about 9.5 h); a multi-day restart cannot finish, and merging the docs-only branch would empty main; submission plan needed before more planning evidence=decode.md Q3, T-14
2026-10-11 14:04 CDT VERIFY M1-S02 harness=supervisor why=G-decode approved; log-decision wrote D1-D4 evidence=Approvals 2026-10-11 14:02 CDT; decision-views --check current
