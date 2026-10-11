---
task: M3-D05
harness: claude
model: claude-opus-5-5
status: human_review
reason: 4 ⚠ missing in docs/architecture/lld.md#2-backend-components
base_sha: 5f07d6d
head_sha: 42caa52
changes: [docs/architecture/diagrams/be-class.mmd]
tests: "dd_trace -> no output; dd_done -> no output; dd_render -> unverified: mmdc not on PATH"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/be-class.mmd` — classDiagram; 14 classes, 35 relationships; 4 ⚠ missing;
  source docs/architecture/lld.md#2-backend-components @ 5f07d6d. Edges come only from each `Depends on:` bullet
  (33 `depends on`) plus BE-2's stated "two implementations: BE-3, BE-4" (2 realizations); non-BE dependencies are
  notes. BE-3/BE-4 `Catalog`, `Books` are drawn as the note "Catalog, Books: as BE-2" (the row's own words) plus the
  realization arrow, not as copied signatures. Go receivers are kept as written, e.g. `+(f FeeTerms) TakerFee(...)`.

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.p35BJY/kit.sh && dd_trace
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.p35BJY/kit.sh && dd_done
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.p35BJY/kit.sh && dd_render
    unverified|mmdc not on PATH

## Questions

- ⚠ design-lld: BE-12 `Depends on: every BE above` covers BE-1..BE-11, but BE-13 and BE-14 methods are `Called by BE-12`. Should BE-12's Depends on include BE-13 and BE-14 (or read "every other BE")?
- ⚠ design-lld: BE-9 `Depends on: os` only, but `Append(d route.Decision)` and `Page(...) ([]route.Decision, ...)` use BE-8's type. Does BE-9 depend on BE-8 (route), or should the Decision type live elsewhere?
- ⚠ design-lld: BE-13 `Depends on: BE-1, BE-7, math/rand/v2`, but `Draw` and `Cards` take `*ingest.Snapshot` (BE-6). Should BE-13 list BE-6?
- ⚠ design-lld: BE-14 `Depends on: BE-1, BE-7, BE-13 Card`, but `Export` takes `*ingest.Snapshot` (BE-6) and `Import` returns `[]refdata.Verdict` (BE-10). Should BE-14 list BE-6 and BE-10?
