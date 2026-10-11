---
task: M3-D12
harness: claude
model: claude-opus-5-5
status: human_review
reason: 1 ⚠ missing in docs/architecture/lld.md#sm-1-pair-verdict-lifecycle-pairsverdict-state-pair
base_sha: 5f07d6d
head_sha: bf7523c
changes: [docs/architecture/diagrams/state-pair.mmd]
tests: "dd_trace -> 1 line (transitions in source: 6; drawn: 10); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render (mmdc not on PATH)]
---

## Changes

- `docs/architecture/diagrams/state-pair.mmd` — stateDiagram-v2; 4 states, 10 transitions; 1 ⚠ missing;
  source docs/architecture/lld.md#sm-1-pair-verdict-lifecycle-pairsverdict-state-pair @ 5f07d6d

## Tests run

```text
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.LPVCTl/kit.sh && dd_trace
transitions in source: 6; drawn: 10
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.LPVCTl/kit.sh && dd_done
transitions in source: 6; drawn: 10
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.LPVCTl/kit.sh && dd_render
unverified|mmdc not on PATH
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.LPVCTl/kit.sh && dd_secrets
(no output)
$ bash scripts/scope-guard.sh .handoff/runs/M3-D12/task.md milestone/m3-design
scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D12/task.md)
exit=0
```

The one dd_trace / dd_done line is the kit's row-vs-arrow count, not an untraced or dropped item: SM-1's
transition table has 6 rows, but row 3 (`undecided` → `equivalent` or `related`, AI verdict applied) is 2 arrows
and row 4 (`equivalent`, `related` → `equivalent` or `related`, human override applied) is 4 arrows, drawn
literally. No faithful drawing has 6 arrows; see Questions.

## Questions

- ⚠ design-lld: SM-1 says Terminal is "none within a snapshot; a new snapshot rebuilds every pair". Is having no
  end state intended (then say how the diagram should show the per-snapshot rebuild), or is there a terminal state
  for a pair?
- design-lld: SM-1 transition rows 3 and 4 each list several From / To states. Split them into one row per
  From → To so the table and the picture count the same (draw-diagrams checks 1 row = 1 arrow)?
- design-lld: row 4 read literally includes `equivalent` → `equivalent` and `related` → `related` (a human
  override that keeps the verdict and only sets `decided_by = human`). Are those self-transitions legal? They are
  drawn because the row allows them.
