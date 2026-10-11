---
task: M3-D04
harness: claude
model: claude-opus-5-5
status: done
reason: drawn, no gaps; render unverified
base_sha: d7cc8b2
head_sha: fea24ed
changes: [docs/architecture/diagrams/erd.mmd]
tests: "dd_trace -> no output; dd_done -> no output; dd_render -> unverified: mmdc not on PATH"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/erd.mmd` — erDiagram; 11 entities, 7 relationships; 0 ⚠ missing;
  source docs/architecture/lld.md#1-data-model @ d7cc8b2
- Notation note (also a `%%` line in the file): Go types `pkg.Type` are drawn `pkg-Type` and `[]T` as `T[]`,
  because a Mermaid erDiagram attribute type cannot contain `.` or start with `[` (it would not render in GitLab).
  The words are unchanged; `dd_trace` passes.
- `decisions` is drawn with no PK because the section states `Primary key: none (append-only log; position in the
  file orders it)`; recorded as a `%%` comment above the entity, not as a gap.
- `markets` — `pairs` is drawn as two relationships (`a_key`, `b_key`), one per FK named in Via;
  `uq_pairs_one_to_one` is a partial unique, so it is a comment on `a_key` and `b_key`, not `UK`.
- Unique bullets that only restate the primary key (`pk_*`, "the primary key") are carried by `PK`; no separate `UK`.

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.qK3lOl/kit.sh && dd_trace
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.qK3lOl/kit.sh && dd_done
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.qK3lOl/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D04/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D04/task.md)  [exit 0]

## Questions

- none
