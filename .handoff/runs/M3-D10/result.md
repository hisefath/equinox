---
task: M3-D10
harness: claude
model: claude-opus-5-5
status: human_review
reason: "section is not in design-lld form; run design-lld (alt 'precision is below 0.90' numbers its steps b1./b2., not aN.; dd_trace counts 11 of the 13 source steps)"
base_sha: d7cc8b2
head_sha: 824571a
changes: [docs/architecture/diagrams/sequence-measure-precision.mmd]
tests: "dd_trace -> 1 line (step count 11 vs 13 drawn, b-steps uncounted); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/sequence-measure-precision.mmd` — sequenceDiagram; 7 participants, 13 messages; 0 ⚠ missing;
  source docs/architecture/lld.md#seq-4-measure-matching-precision-sequence-measure-precision-covers-us-4-fr-028fr-034-ae-023ae-028-sc-001sc-003 @ d7cc8b2

## Tests run

```text
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.mRmHlK/kit.sh && dd_trace
steps in source (n. and aN.): 11; messages + notes drawn: 13
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.mRmHlK/kit.sh && dd_done
steps in source (n. and aN.): 11; messages + notes drawn: 13
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.mRmHlK/kit.sh && dd_render
unverified|mmdc not on PATH
$ bash scripts/scope-guard.sh .handoff/runs/M3-D10/task.md milestone/m3-design
scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D10/task.md)
(exit 0)
```

## Questions

- ⚠ design-lld: SEQ-4's second alt (`precision is below 0.90`, replaces step 9) numbers its steps `b1.`/`b2.`; design-lld's form is `- aN.` per alt. Renumber them `a1.`/`a2.` so the step count check passes? All 13 steps (1-9, a1-a2, b1-b2) are drawn; only the kit's counter misses the two b-steps, which is the one Done-when line that fails.
- design-lld (non-blocking): CN-4 reviewed files is on the Participants line but no step has it as From or To (steps 9 and b2 name it only inside the message text), so it is drawn as a participant with no arrows. Intended?
- design-lld (non-blocking): steps 7 and 8 put a BE-13 call (`ImportLabels`, `Score(...)`) inside a Labs lead → BE-12 message, and b1 / a1 then show BE-13 → BE-12 replies with no BE-12 → BE-13 call stated for them. Drawn as written; split into explicit BE-12 → BE-13 steps?
