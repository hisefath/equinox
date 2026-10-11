---
task: M3-D05
harness: claude
model: claude-opus-5-5
status: human_review
reason: 2 ⚠ missing in docs/architecture/lld.md#2-backend-components
base_sha: d7cc8b2
head_sha: 0d67651
changes: [docs/architecture/diagrams/be-class.mmd]
tests: "dd_trace -> no output; dd_done -> no output; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render (mmdc not on PATH)]
---

## Changes

- `docs/architecture/diagrams/be-class.mmd` — classDiagram; 14 classes, 41 relationships; 2 ⚠ missing;
  source docs/architecture/lld.md#2-backend-components @ d7cc8b2
- Notes: stereotypes are written as member lines (`BE1 : <<package internal/market>>`) because every kind in §2
  is several words, which the `<<kind>> BE<n>` form may not parse. Go receivers are kept as written
  (`+(f FeeTerms) TakerFee(...)`). `BE2 <|.. BE3/BE4` comes from BE-2's "two implementations: BE-3, BE-4".

## Tests run

    $ . $tmp/kit.sh && dd_trace
    (no output)
    $ . $tmp/kit.sh && dd_done
    (no output)
    $ . $tmp/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D05/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D05/task.md)
    (exit 0)

## Questions

- ⚠ design-lld: BE-3 kalshi.Adapter names `Catalog` and `Books` without a signature (Returns: "as BE-2"). Do they have exactly BE-2's signatures? If so, write them in the row so the diagram can draw them.
- ⚠ design-lld: same for BE-4 polymarket.Adapter `Catalog` and `Books`.
