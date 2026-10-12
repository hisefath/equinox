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
| M1 | Research: brief + cited research (docs only) | — | D1-D6 | [!1](https://labs.gauntletai.com/sefathchowdhury/equinox/-/merge_requests/1) | merged | — | dde58ac | 6428ca9 | — |
| M2 | Product: PRD + clarifications (docs only) | — | D7-D12 | [!2](https://labs.gauntletai.com/sefathchowdhury/equinox/-/merge_requests/2) | merged | — | f75fc70 | 6428ca9 | — |
| M3 | Design: HLD, LLD, DB design, API contract, diagrams (docs only) | — | D13-D23 | [!3](https://labs.gauntletai.com/sefathchowdhury/equinox/-/merge_requests/3) | merged | — | 18b351e | 6428ca9 | — |
| M4 | Build: matching, routing, measurement (M4-M6 in one milestone, D22) | T001-T005 | D22, D24, D25 | [!4](https://labs.gauntletai.com/sefathchowdhury/equinox/-/merge_requests/4) | merged | — | 6428ca9 | 6428ca9 | — |

## Diagrams by milestone

M3 (Design) produced every diagram; each is embedded as Mermaid in its design doc and also saved as a picture. The
short tour with all of them is [`ARCHITECTURE.md`](ARCHITECTURE.md).

| Milestone | Doc | Pictures |
|---|---|---|
| M3 HLD (G2) | [`hld.md`](docs/architecture/hld.md) | [context](docs/architecture/diagrams/png/hld-context.png) · [containers](docs/architecture/diagrams/png/hld-container.png) · [deployment](docs/architecture/diagrams/png/deployment.png) |
| M3 LLD (G3) | [`lld.md`](docs/architecture/lld.md) | [data model](docs/architecture/diagrams/png/erd.png) · [backend components](docs/architecture/diagrams/png/be-class.png) · [page components](docs/architecture/diagrams/png/fe-components.png) · [refresh and match](docs/architecture/diagrams/png/sequence-refresh-and-match.png) · [route an order](docs/architecture/diagrams/png/sequence-route-order.png) · [venue down](docs/architecture/diagrams/png/sequence-venue-down.png) · [measure precision](docs/architecture/diagrams/png/sequence-measure-precision.png) · [adjudicate offline](docs/architecture/diagrams/png/sequence-adjudicate-offline.png) · [pair verdict states](docs/architecture/diagrams/png/state-pair.png) · [venue health states](docs/architecture/diagrams/png/state-venue-health.png) |
| M4 Build | [`specs/M4-build/README.md`](specs/M4-build/README.md) | built to the M3 pictures; [route an order](docs/architecture/diagrams/png/sequence-route-order.png) is the core path |
