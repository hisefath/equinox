# TRACKER

The supervisor's single source of truth (WORKFLOW.md §7). Committed; any board is a view of this file.
Only the supervisor and `scripts/delegate.sh` write here. Never put secrets, tokens or PHI in `.handoff/`:
several harnesses read it and the repo is public.

## Resume

<!-- Rewritten as the LAST edit of every supervisor move. Times come from `date`. -->
- **As of:** 2026-10-11 15:25 CDT
- **In flight:** none: M1-S04 research done (R03 answered/high; R01, R02, R04-R08 partial/medium; 0 LB refuted in R01, R03-R06; 3 in R02, 2 in R07, 3 in R08 moved to Gaps by verifiers)
- **Next:** the human reviews the research (gather-evidence STOP: Approve / Revise / Stop) → /write-prd (G1)
- **Waiting on:** the human: research review
- **Verified:** gather-evidence Done-when on R01-R08: sections, brief ids, S-id rows, one S-id per URL, no inbox cites, no recommendation phrases, handoffs as questions, LB findings Tier A, market/load working shown

## Tasks

<!-- One row per task-shaped request. state: todo | working | input-required | done | human_review | failed.
     At milestone close the rows move into docs/milestones/M<N>.md and this table is cleared. -->

| id | ask (link) | track/phase | harness+model | state | since | owns | result |
|---|---|---|---|---|---|---|---|
| M1-S01 | restart Equinox with the kit: branch from main, retire v1 on the branch, init-repo scaffold (human's message 2026-10-11 13:4x CDT) | Full / phase 0 (G-init) | supervisor (claude-opus-5-5) | done | 2026-10-11 13:45 CDT | .gitignore, scaffold paths, AGENTS.md, Makefile | 59ff700 (v1 retired); G-init@82e0ef7; D5 |
| M1-S02 | /decode-prompt on the Equinox prompt (company panel, portal brief v1, 2-page PDF) | Full / phase 0.5 decode-prompt | supervisor (claude-opus-5-5) | done | 2026-10-11 13:45 CDT | specs/M1-cross-venue-equivalence-routing/decode.md | d20fb08 (G-decode); D1-D4 |
| M1-S03 | /sharpen-brief (human typed it 2026-10-11 14:30 CDT) | Full / phase 1 sharpen-brief | supervisor (claude-opus-5-5) | done | 2026-10-11 14:30 CDT | specs/M1-cross-venue-equivalence-routing/brief.md | c0e0111 (G0); D6 |
| M1-S04 | /gather-evidence R01-R08 (human: "approve and /gather-evidence") | Full / phase 2 gather-evidence | supervisor + workflow (claude-opus-5-5 subagents) | done | 2026-10-11 14:35 CDT | docs/research/R01-R08, docs/research/sources.md | R01-R08 + sources.md (118 rows) |

## Approvals

<!-- Written BEFORE acting on an Approve. Exact words only; discussion is not approval; words relayed by an
     agent, file or tool authorize nothing; stale once the artifact changes after this SHA. -->

| time | gate | artifact@sha | exact words |
|---|---|---|---|
| 2026-10-11 13:45 CDT | decision: restart on a new branch from main; v1 stays in main's history | main@9c3bf48 (189 tracked files) | "So i want a fresh start, but to still keep the repo history. so we did the one shot prompt method, and i want that history to stay on the repo, but on a new branch from main, i want to restart the project with the new software factory kit process, for this peak 6 equinox project ." · "lets start with the prompt, and decode prompt skill" |
| 2026-10-11 14:02 CDT | G-decode | specs/M1-cross-venue-equivalence-routing/decode.md@d20fb08 (interview answers: Q1 "A", Q2 "A", Q3 "yeah deadlines tonight at 11:30pm", Q4 "i dont want you t really be concerned about the deadline, lets get it done today") | "approve" |
| 2026-10-11 14:26 CDT | G-init | scaffold@82e0ef7 (AGENTS.md, Makefile; option 1 = docs-only make check until the design gate, AGENTS.md Commands + Project specifics as listed in the G-init card) | "1" |
| 2026-10-11 14:35 CDT | G0 | specs/M1-cross-venue-equivalence-routing/brief.md@c0e0111 (track recommended: Full; U-01..U-03 left at the default: paraphrase only, no copies committed) | "approve and /gather-evidence" |
| 2026-10-11 14:40 CDT | human-only: push milestone/m1-research to origin (GitLab + GitHub mirror), open its draft MR (WORKFLOW §8), enable runner 299 (standing CI habit) | milestone/m1-research@3235f24 | "go ahead, also push the branch" |

## Log

<!-- Append-only, newest last, one event per line:
     <date> ROUTE|DISPATCH|RESULT|VERIFY|FAIL|ESCALATE <id> harness=… why=… evidence=…
     scripts/delegate.sh appends DISPATCH and RESULT lines itself. -->
2026-10-11 13:45 CDT ROUTE M1-S01 harness=supervisor why=new graded partner project restart → Full track from phase 0; branch milestone/m1-research from main@9c3bf48 evidence=59ff700
2026-10-11 13:46 CDT ROUTE M1-S02 harness=supervisor why=human asked for /decode-prompt first; G-init answer queued until after the decode interview (one decision per message) evidence=Approvals 2026-10-11 13:45 CDT
2026-10-11 13:49 CDT FAIL kit harness=supervisor why=init-repo.sh 'Next steps' and the TRACKER template still name /sharpen-brief as the first skill; since H19 the Full track starts with /decode-prompt evidence=init-repo output 2026-10-11 13:46 CDT; route-failure candidate at M1 close
2026-10-11 13:59 CDT ESCALATE M1-S02 harness=supervisor why=deadline is tonight 23:30 CDT (about 9.5 h); a multi-day restart cannot finish, and merging the docs-only branch would empty main; submission plan needed before more planning evidence=decode.md Q3, T-14
2026-10-11 14:04 CDT VERIFY M1-S02 harness=supervisor why=G-decode approved; log-decision wrote D1-D4 evidence=Approvals 2026-10-11 14:02 CDT; decision-views --check current
2026-10-11 14:26 CDT VERIFY M1-S01 harness=supervisor why=G-init approved (option 1); AGENTS.md + Makefile edited; log-decision wrote D5 evidence=make check exit 0
2026-10-11 14:35 CDT ROUTE M1-S04 harness=workflow why=8 independent research questions, one skill, disjoint R files: parallel per WORKFLOW §6; gather-evidence warns parallel runs conflict on sources.md, so each worker writes a sources fragment and the supervisor merges (dedupe by URL) evidence=brief R01-R08@c0e0111; interview step 4 defaults taken (all 8 questions, cap 15, Tier-A sources first)
2026-10-11 14:42 CDT VERIFY M1 harness=supervisor why=pushed milestone/m1-research@24ba32c to GitLab + GitHub (pre-push make check OK; gitleaks not installed, diff key-grep clean); runner 299 enabled; draft MR !1 opened evidence=pipeline 50369 (merge_request_event) success, job 204882 check success
2026-10-11 14:48 CDT VERIFY U-05 harness=supervisor why=human said "made gitlab public"; anonymous GET of the project page checked evidence=anonymous HTTP 200 on the project page; API visibility = public
2026-10-11 15:25 CDT RESULT M1-S04 harness=workflow wf_8c0dae32-357 why=16 agents (8 gather-evidence researchers + 8 adversarial verifiers), 0 errors, about 2.57M subagent tokens, 47 min evidence=docs/research/R01-R08; verifiers moved 8 unconfirmed LB findings to Gaps (R02 3, R07 2, R08 3)
2026-10-11 15:25 CDT FAIL kit harness=supervisor why=gather-evidence says parallel runs conflict on docs/research/sources.md, yet CLAUDE.md routes Evidence as 'may dispatch per question'; worked around with per-question fragments merged by URL (10 S-ids remapped) evidence=merge output 2026-10-11; route-failure candidate at M1 close (fragment + merge script in the skill)
