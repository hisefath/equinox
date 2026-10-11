---
task: M3-D07
harness: claude
model: claude-opus-5-5
status: human_review
reason: 1 ⚠ missing in docs/architecture/lld.md#seq-1-refresh-match-and-publish-a-snapshot-sequence-refresh-and-match-covers-us-1-fr-001fr-007-fr-024-fr-040-ae-001ae-005
base_sha: 5f07d6d
head_sha: ec1be4f
changes: [docs/architecture/diagrams/sequence-refresh-and-match.mmd]
tests: "dd_trace -> 1 line (step count 12 vs 17, see Questions); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/sequence-refresh-and-match.mmd` — sequenceDiagram; 9 participants, 14 messages; 1 ⚠ missing;
  source docs/architecture/lld.md#seq-1-refresh-match-and-publish-a-snapshot-sequence-refresh-and-match-covers-us-1-fr-001fr-007-fr-024-fr-040-ae-001ae-005 @ 5f07d6d

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.qqoL2i/kit.sh && dd_trace
    steps in source (n. and aN.): 12; messages + notes drawn: 17
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.qqoL2i/kit.sh && dd_done
    steps in source (n. and aN.): 12; messages + notes drawn: 17
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.qqoL2i/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D07/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D07/task.md)
    (exit 0)

Every drawn word and id traces to the section (no `untraced word`, `invented id` or `dropped id` lines). The one
`dd_trace` line is the step count: 17 drawn = the kit's 12 (steps 1-9, a1-a3) + 2 (b1-b2, which the kit's `aN.`
counter does not count) + 3 (steps 3 and 4 name two hops each, step 8 names two receivers; each hop/receiver is
its own arrow so the picture says exactly what the step says).

## Questions

- ⚠ design-lld: Which main-path steps does **alt a venue refuses requests (rate limit or blocked)** replace? It has
  no "(replaces steps a-b)", so it is drawn around the last step (step 9) with a gap line and a visible ⚠ note.
- design-lld (form): the second alt numbers its steps `b1.`/`b2.`; design-lld form (and the kit's step counter) use
  `aN.`. Renumbering them (with a stated range) would make the count check pass.
- design-lld (form): steps 3 and 4 (`BE-3 → BE-5 → US-regulated venue`, `BE-4 → BE-5 → Offshore venue`) and step 8
  (`BE-6 → BE-3, BE-4`) each hold more than one From → To; split into one step per arrow if the count check
  should match one line per step.
- design-lld (form): a1 and b1 name the receiver "BE-3 or BE-4"; a message has one receiver, so each is drawn to
  BE-3 with "(to BE-3 or BE-4)" kept in the label.
- design-lld (form): the Participants line names BE-6 twice (BE-6 Refresher, BE-6 Store); drawn as two lifelines,
  the Store with fallback id N3. a3's "Store" is drawn as that same lifeline.
