# TRACKER

The supervisor's single source of truth (WORKFLOW.md §7). Committed; any board is a view of this file.
Only the supervisor and `scripts/delegate.sh` write here. Never put secrets, tokens or PHI in `.handoff/`:
several harnesses read it and the repo is public.

## Resume

<!-- Rewritten as the LAST edit of every supervisor move. Times come from `date`. -->
- **As of:** 2026-10-11 17:12 CDT
- **In flight:** M3 G3: lld.md revised per the 8 diagram findings (G3 Revise 17:10); 8 diagram jobs re-dispatched (D04-D07, D09, D10, D12, D13); D08 and D11 accepted
- **Next:** verify redraws → squash all 10 → embed → G3 (D19, D20, closing) → log-decision (D19, D20, then the local-first decision) → push → /define-acceptance; local-run instructions land in README + AGENTS.md Commands once M4 code exists (REQ-25)
- **Waiting on:** 8 diagram redraws
- **Verified:** design-lld Done-when checks all pass; openapi.yaml parses (ruby YAML); not machine-validated as OpenAPI (no lint command in AGENTS.md)

## Tasks

<!-- One row per task-shaped request. state: todo | working | input-required | done | human_review | failed.
     At milestone close the rows move into docs/milestones/M<N>.md and this table is cleared. -->

| id | ask (link) | track/phase | harness+model | state | since | owns | result |
|---|---|---|---|---|---|---|---|
| M1-S01 | restart Equinox with the kit: branch from main, retire v1 on the branch, init-repo scaffold (human's message 2026-10-11 13:4x CDT) | Full / phase 0 (G-init) | supervisor (claude-opus-5-5) | done | 2026-10-11 13:45 CDT | .gitignore, scaffold paths, AGENTS.md, Makefile | 59ff700 (v1 retired); G-init@82e0ef7; D5 |
| M1-S02 | /decode-prompt on the Equinox prompt (company panel, portal brief v1, 2-page PDF) | Full / phase 0.5 decode-prompt | supervisor (claude-opus-5-5) | done | 2026-10-11 13:45 CDT | specs/M1-cross-venue-equivalence-routing/decode.md | d20fb08 (G-decode); D1-D4 |
| M1-S03 | /sharpen-brief (human typed it 2026-10-11 14:30 CDT) | Full / phase 1 sharpen-brief | supervisor (claude-opus-5-5) | done | 2026-10-11 14:30 CDT | specs/M1-cross-venue-equivalence-routing/brief.md | c0e0111 (G0); D6 |
| M1-S04 | /gather-evidence R01-R08 (human: "approve and /gather-evidence") | Full / phase 2 gather-evidence | supervisor + workflow (claude-opus-5-5 subagents) | done | 2026-10-11 14:35 CDT | docs/research/R01-R08, docs/research/sources.md | R01-R08 + sources.md (118 rows) |
| M2-S01 | /write-prd (human: "approve and /write-prd") | Full / phase 3 write-prd | supervisor (claude-opus-5-5) | done | 2026-10-11 16:01 CDT | docs/product/prd.md | 40eb5d5 (G1); D7-D12 |
| M3-S01 | /design-hld (human: "approve, also push and /design-hld") | Full / phase 5 design-hld | supervisor (claude-opus-5-5) | done | 2026-10-11 16:14 CDT | docs/architecture/hld.md | — (awaiting G2) |
| M3-D01 | draw-diagrams hld-context | Full / phase 6b | claude-subagent | done | 2026-10-11 16:14 CDT | docs/architecture/diagrams/hld-context.mmd | cb8c797 → squashed cb87c3c |
| M3-D02 | draw-diagrams hld-container | Full / phase 6b | claude-subagent | done | 2026-10-11 16:14 CDT | docs/architecture/diagrams/hld-container.mmd | 90cbf0e → squashed 153a8af (attempt 1 archived) |
| M3-D03 | draw-diagrams deployment | Full / phase 6b | claude-subagent | done | 2026-10-11 16:14 CDT | docs/architecture/diagrams/deployment.mmd | 5328a88 → squashed 7652384 |
| M3-S02 | /design-lld (human: "approve, also push and /design-lld"; scope M4-M6 per "b, all three in one pass") | Full / phase 6 design-lld | supervisor (claude-opus-5-5) | done | 2026-10-11 16:38 CDT | docs/architecture/lld.md, docs/architecture/contracts/openapi.yaml | — (awaiting G3) |
| M3-D04 | draw-diagrams erd | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/erd.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D05 | draw-diagrams be-class | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/be-class.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D06 | draw-diagrams fe-components | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/fe-components.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D07 | draw-diagrams sequence-refresh-and-match | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/sequence-refresh-and-match.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D08 | draw-diagrams sequence-route-order | Full / phase 6b | claude-subagent | done | 2026-10-11 16:38 CDT | docs/architecture/diagrams/sequence-route-order.mmd | 168a16d (complete; only the kit step-counter false positive) |
| M3-D09 | draw-diagrams sequence-venue-down | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/sequence-venue-down.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D10 | draw-diagrams sequence-measure-precision | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/sequence-measure-precision.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D11 | draw-diagrams sequence-adjudicate-offline | Full / phase 6b | claude-subagent | done | 2026-10-11 16:38 CDT | docs/architecture/diagrams/sequence-adjudicate-offline.mmd | ce81839 |
| M3-D12 | draw-diagrams state-pair | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/state-pair.mmd | re-dispatched after G3 Revise (attempt 1 archived) |
| M3-D13 | draw-diagrams state-venue-health | Full / phase 6b | claude-subagent | working | 2026-10-11 16:38 CDT | docs/architecture/diagrams/state-venue-health.mmd | re-dispatched after G3 Revise (attempt 1 archived) |

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
| 2026-10-11 15:27 CDT | research review (no gate; gather-evidence STOP) | docs/research/R01-R08@c1c023e | "approve and /write-prd" |
| 2026-10-11 16:06 CDT | G1 | docs/product/prd.md@40eb5d5 (both open ⚠ taken as recommended per the STOP's recommendation; agent assumptions A-1..A-3, A-5..A-8 confirmed; also covers: push milestone/m2-product + draft MR !2) | "approve, also push and /design-hld" |
| 2026-10-11 16:17 CDT | G2 D13 | docs/architecture/hld.md@5d35114 ### D13 (journal id D13) | "yes, use go" (quoting option b "Go, standard library only (brief P-01).") |
| 2026-10-11 16:19 CDT | G2 D14 | docs/architecture/hld.md@5d35114 ### D14 (journal id D14) | "approved for the recc," |
| 2026-10-11 16:20 CDT | G2 D15 | docs/architecture/hld.md@5d35114 ### D15 (journal id D15) | "approved for the recc" |
| 2026-10-11 16:22 CDT | G2 D16 | docs/architecture/hld.md@5d35114 ### D16 (journal id D16) | "yes lets do it" (quoting the original recommendation: "b) exported files plus an agent harness"; the earlier venue-mock question stays unanswered, so the design's recorded-responses-plus-live path is unchanged) |
| 2026-10-11 16:22 CDT | decision: venue mocks (chat options card a/b/c, after the human asked for mock venue APIs to save time) | docs/architecture/hld.md@5d35114 (CN-3 recording, §11 CI row: unchanged) | "yes i agree" (quoting option a "Recorded real responses as the mock, plus a live switch (recommended).") |
| 2026-10-11 16:23 CDT | G2 Revise | docs/architecture/hld.md@5d35114 §5 (M3-D02's three gaps; proposal in the supervisor's message: host zone CN-1+CN-2, developer zone CN-3, repository zone CN-4, TB-1..TB-4 restated) | "/design-hld revise §5 trust zones as proposed" |
| 2026-10-11 16:30 CDT | G2 (closing) | docs/architecture/hld.md@e74350c (diagrams embedded; binds D13-D16 and the §5 revision; also covers: push milestone/m3-design + draft MR !3) | "approve, also push and /design-lld" |
| 2026-10-11 16:31 CDT | decision: LLD scope (design-lld Input 6 question, options a M4 only / b M4-M6 in one pass / do nothing) | docs/product/prd.md@40eb5d5 ## Milestone cut | "b, all three in one pass" |
| 2026-10-11 17:10 CDT | G3 Revise | docs/architecture/lld.md@5f07d6d (the 8 findings from M3-D04..D13 listed in the supervisor's message) | "/design-lld revise per the 8 diagram findings, also we can deploy things later, lets just run things locally for now, and also give instructions in the repo for anyone that wants to run locally" |
| 2026-10-11 17:10 CDT | decision: run locally first, deploy to Railway later; local-run instructions in the repo (same words as the G3 Revise row) | PRD C-2 / US-7 sequencing; REQ-25 setup instructions | "/design-lld revise per the 8 diagram findings, also we can deploy things later, lets just run things locally for now, and also give instructions in the repo for anyone that wants to run locally" |

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
2026-10-11 16:01 CDT ROUTE M2-S01 harness=supervisor why=human approved the research and typed /write-prd; new milestone branch milestone/m2-product from milestone/m1-research (M2 Product, WORKFLOW §8), MRs stack until v2 is green (D4) evidence=Approvals 2026-10-11 15:27 CDT
2026-10-11 16:14 CDT DISPATCH M3-D01 harness=claude-subagent why=draw-diagrams is one skill, isolated (one .mmd), parallel with G2 decisions; Codex not on PATH evidence=.handoff/runs/M3-D01/task.md from milestone/m3-design
2026-10-11 16:14 CDT DISPATCH M3-D02 harness=claude-subagent why=same as M3-D01 evidence=.handoff/runs/M3-D02/task.md from milestone/m3-design
2026-10-11 16:14 CDT DISPATCH M3-D03 harness=claude-subagent why=same as M3-D01 evidence=.handoff/runs/M3-D03/task.md from milestone/m3-design
2026-10-11 16:16 CDT RESULT M3-D03 harness=claude-subagent status=done evidence=result.md parses; head 5328a88
2026-10-11 16:16 CDT VERIFY M3-D03 harness=supervisor why=scope-guard OK (1 path in Owns); diff = deployment.mmd only, every node traces to §11; make check red in the worktree only because the other two .mmd files are not back and nothing is embedded yet (expected until embed) evidence=scope-guard exit 0
2026-10-11 16:16 CDT FAIL kit harness=supervisor why=committing hld.md with diagram placeholders turns make check red (embed-diagrams --check: .mmd missing) until every draw-diagrams job is back and embedded, so the milestone branch cannot be pushed mid-G2 evidence=make check on milestone/m3-design@5d35114; route-failure candidate
2026-10-11 16:16 CDT RESULT M3-D01 harness=claude-subagent status=done evidence=result.md parses; head cb8c797
2026-10-11 16:16 CDT VERIFY M3-D01 harness=supervisor why=scope-guard OK (1 path in Owns); diff = hld-context.mmd only; 10 nodes = the 10 §2 rows, 9 edges labelled with each row's interaction evidence=scope-guard exit 0
2026-10-11 16:18 CDT RESULT M3-D02 harness=claude-subagent status=human_review evidence=result.md: 3 gaps in hld.md §5 (CN-2's trust zone unstated; TB-2 and TB-3 overlap at CN-3 without nesting; TB-4's trusted side vs its crossing connections disagree)
2026-10-11 16:18 CDT VERIFY M3-D02 harness=supervisor why=scope-guard OK; the three gaps are real defects in hld.md §5 (design-hld's section), not the drawing; fix = /design-hld revision mode restating §5 as nested zones, then re-dispatch M3-D02 evidence=hld.md@5d35114 §5 rows TB-1..TB-4
2026-10-11 16:23 CDT ROUTE M3-D02 harness=supervisor why=G2 Revise of §5 (Approvals 16:23 CDT) fixed the three gaps M3-D02 reported; attempt 1 archived to .handoff/runs/M3-D02.attempt-20261011T1623/ evidence=hld.md Change log row 16:24 CDT
2026-10-11 16:24 CDT DISPATCH M3-D02 harness=claude-subagent why=redraw after the §5 revision evidence=.handoff/runs/M3-D02/task.md from milestone/m3-design@5842e20
2026-10-11 16:29 CDT RESULT M3-D02 harness=claude-subagent status=done evidence=result.md parses; head 90cbf0e; 0 gaps
2026-10-11 16:29 CDT VERIFY M3 harness=supervisor why=scope-guard OK for M3-D02; squashed D01 cb87c3c, D02 153a8af, D03 7652384; embed-diagrams embedded 3; make check OK; all three .mmd parsed and rendered to SVG by mermaid 11.4.1 in the browser pane (Docker mermaid-cli failed: the Colima VM disk is full, left alone per the human's earlier choice) evidence=browser render check 2026-10-11 16:29 CDT
2026-10-11 16:31 CDT ROUTE M3-S02 harness=supervisor why=design-lld first pass scoped to M4-M6 (FR-001..FR-043) by the human's choice; deviates from design-lld's one-milestone rule — route-failure candidate (allow a multi-milestone first pass when the HLD already fixes every box) evidence=Approvals 2026-10-11 16:31 CDT
2026-10-11 16:38 CDT DISPATCH M3-D04 harness=claude-subagent why=draw-diagrams erd: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D04/task.md
2026-10-11 16:38 CDT DISPATCH M3-D05 harness=claude-subagent why=draw-diagrams be-class: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D05/task.md
2026-10-11 16:38 CDT DISPATCH M3-D06 harness=claude-subagent why=draw-diagrams fe-components: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D06/task.md
2026-10-11 16:38 CDT DISPATCH M3-D07 harness=claude-subagent why=draw-diagrams sequence-refresh-and-match: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D07/task.md
2026-10-11 16:38 CDT DISPATCH M3-D08 harness=claude-subagent why=draw-diagrams sequence-route-order: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D08/task.md
2026-10-11 16:38 CDT DISPATCH M3-D09 harness=claude-subagent why=draw-diagrams sequence-venue-down: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D09/task.md
2026-10-11 16:38 CDT DISPATCH M3-D10 harness=claude-subagent why=draw-diagrams sequence-measure-precision: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D10/task.md
2026-10-11 16:38 CDT DISPATCH M3-D11 harness=claude-subagent why=draw-diagrams sequence-adjudicate-offline: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D11/task.md
2026-10-11 16:38 CDT DISPATCH M3-D12 harness=claude-subagent why=draw-diagrams state-pair: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D12/task.md
2026-10-11 16:38 CDT DISPATCH M3-D13 harness=claude-subagent why=draw-diagrams state-venue-health: one skill, isolated, parallel with G3 prep; Codex not on PATH evidence=.handoff/runs/M3-D13/task.md
2026-10-11 16:41 CDT RESULT M3-D06 harness=claude-subagent status=human_review evidence=1 gap: lld.md §3 FE-2 'Owns state' says ST-1 while the State ownership row says BE-6 Store owns it (real inconsistency; fix = FE-2 owns none, renders ST-1 passed from the server)
2026-10-11 16:41 CDT RESULT M3-D13 harness=claude-subagent status=human_review evidence=SM-2 'Terminal: none' leaves no end arrow (intended: health has no final state); kit check counts 5 rows vs 8 arrows because two rows list several From states (fix = one From per row)
2026-10-11 16:41 CDT RESULT M3-D12 harness=claude-subagent status=human_review evidence=SM-1 no terminal (intended within a snapshot); rows 3-4 list several From/To states (6 rows vs 10 arrows); human-override self-transitions (equivalent→equivalent, related→related) need stating
2026-10-11 16:41 CDT RESULT M3-D08 harness=claude-subagent status=human_review evidence=0 gaps, all 17 steps drawn; only the kit's step counter failed (it counts aN steps only, missing SEQ-2's b1/c1/c2)
2026-10-11 16:41 CDT FAIL kit harness=supervisor why=draw-diagrams' step counter counts only 'aN.' alt steps, so a journey with several alt branches (b1, c1) fails its own check though the diagram is complete evidence=.handoff/runs/M3-D08/result.md; route-failure candidate
2026-10-11 16:42 CDT RESULT M3-D05 harness=claude-subagent status=human_review evidence=4 Depends-on contradictions in lld.md §2 (BE-9, BE-12, BE-13, BE-14)
2026-10-11 16:43 CDT RESULT M3-D10 harness=claude-subagent status=human_review evidence=2 gaps in SEQ-4: step 5 names no receiver for the label files; the below-0.90 alt says 'after step 7' instead of what it replaces; kit counter misses b1 (same kit bug as M3-D08)
2026-10-11 16:43 CDT RESULT M3-D09 harness=claude-subagent status=human_review evidence=SEQ-3: retry owner unstated (BE-5 retries); alt a replaces only step 6 so step 7 would answer twice (should replace 6-7); steps 1 and 7 hold two messages each; BE-6 names both Refresher and Store
2026-10-11 16:44 CDT RESULT M3-D07 harness=claude-subagent status=human_review evidence=SEQ-1: the refusal alt names no replaced range; steps 3, 4, 8 hold two arrows; a1/b1 target 'BE-3 or BE-4'; BE-6 names both Refresher and Store
2026-10-11 17:12 CDT VERIFY M3-D08 harness=supervisor why=diagram complete (17 steps, 0 gaps); human_review only from the kit step counter (b/c alt steps) — accepted as a kit false positive evidence=.handoff/runs/M3-D08/result.md
2026-10-11 17:12 CDT ROUTE M3 harness=supervisor why=G3 Revise 17:10 fixed the 8 findings; attempts archived to .handoff/runs/M3-D{04,05,06,07,09,10,12,13}.attempt-20261011T1712/ evidence=lld.md Change log row 17:12 CDT
