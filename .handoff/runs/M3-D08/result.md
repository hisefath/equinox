---
task: M3-D08
harness: claude
model: claude-opus-5-5
status: human_review
reason: "dd_trace step-count check misfires: kit counts only aN. alt steps, SEQ-2's 2nd/3rd alts use bN./cN. (14 counted, 17 in source, 17 drawn); no gaps, render unverified"
base_sha: 5f07d6d
head_sha: 168a16d
changes: [docs/architecture/diagrams/sequence-route-order.mmd]
tests: "dd_trace -> 1 line (step count 14 vs 17, kit regex); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render (mmdc not on PATH)]
---

## Changes

- `docs/architecture/diagrams/sequence-route-order.mmd` — sequenceDiagram; 9 participants, 16 messages (+1 Note for
  remark step 3); 0 ⚠ missing; source
  docs/architecture/lld.md#seq-2-route-a-hypothetical-order-sequence-route-order-covers-us-2-us-5-us-7-fr-008fr-019-fr-022-fr-023-fr-027-fr-035fr-037-ae-006ae-014-ae-017ae-019-ae-022-ae-029ae-031-ae-036 @ 5f07d6d
  - Three alts nest by their replaced ranges: steps 4-10 (malformed order, b1) ⊃ 6-10 (pair unknown / not
    routable / rules changed, a1-a4) ⊃ 9-10 (decision log cannot be written, c1-c2). No partial overlaps.

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.hZGuA2/kit.sh && dd_trace
    steps in source (n. and aN.): 14; messages + notes drawn: 17
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.hZGuA2/kit.sh && dd_done
    steps in source (n. and aN.): 14; messages + notes drawn: 17
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.hZGuA2/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ grep -cE '^[[:space:]]*(- )?[a-z]?[0-9]+\.[[:space:]]' /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.hZGuA2/src.md   # any alt letter, not just a
    17
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D08/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D08/task.md)
    (exit 0)
    $ make check
    embed-diagrams: docs/architecture/hld.md: 3 diagram(s) in sync
    embed-diagrams: missing or empty .../diagrams/erd.mmd   (and be-class, fe-components, 4 other sequence-*, 2 state-*: sibling jobs)
    embed-diagrams: docs/architecture/lld.md: not changed (fix the error above)
    make: *** [test] Error 1
    (expected until every M3 diagram job is back and the supervisor runs embed-diagrams.sh on lld.md)

## Questions

- Kit vs design-lld form: SEQ-2 numbers its 2nd and 3rd alt branches' steps `b1.` and `c1.`/`c2.`; draw-diagrams'
  `dd_complete` regex (`a?[0-9]+\.`) counts only `aN.`, so it reports 14 source steps while the section has 17 (all
  17 drawn, 16 messages + 1 Note). Either widen the kit regex to `[a-z]?[0-9]+\.` or have design-lld number every
  alt's steps `aN.`; the diagram itself traces fully (no untraced words, no invented or dropped ids).
- none for design-lld: 0 ⚠ missing, `<!-- diagram: sequence-route-order -->` placeholder present once.
