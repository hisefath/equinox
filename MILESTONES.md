# Milestones

One row per milestone = one merge request (WORKFLOW.md §8). The planning rows M1-M3 (Full track) ship
pre-seeded here and the supervisor keeps them (it adds the `M4` row when the human splits Design into M3 HLD / M4
LLD); `slice-tasks` adds each implementation milestone's row at G4; the supervisor updates the **status** column; `report-milestone` and the human fill the SHAs and the tag.

- **status:** planned → in-progress → in-review (MR open, G5 pending) → merged → done
- **done** means merged **and** `make check` re-run on main, with that SHA in *main-check SHA* and in TRACKER Resume.
- **gate SHAs:** the artifact SHA each gate approved, e.g. `G3@abc1234 G4@def5678` (matches the Approvals rows).
- **Planning milestones** (Full track, WORKFLOW.md §8) are docs-only MRs: M1 Research, M2 Product, M3 Design.
  Implementation milestones (tracer bullet first) follow, each with its own acceptance + tasks. Standard track
  (an existing product) deletes no rows and starts at its design-delta milestone.

| id | goal | tasks | D-ids | MR | status | gate SHAs | merge SHA | main-check SHA | tag |
|---|---|---|---|---|---|---|---|---|---|
| M1 | Research: brief + cited research (docs only) | — | — | — | planned | — | — | — | — |
| M2 | Product: PRD + clarifications (docs only) | — | — | — | planned | — | — | — | — |
| M3 | Design: HLD, LLD, DB design, API contract, diagrams (docs only) | — | — | — | planned | — | — | — | — |
