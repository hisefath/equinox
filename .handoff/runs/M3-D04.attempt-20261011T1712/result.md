---
task: M3-D04
harness: claude
model: claude-opus-5-5
status: human_review
reason: "dd_trace/dd_done print 3 'invented column' lines that are kit false positives: lld.md#1-data-model lists two columns in one row (decisions evidence+caveats, verdicts and overrides fingerprint_a+fingerprint_b), and dd_complete reads only the first name in a row; 0 ⚠ missing"
base_sha: 5f07d6d
head_sha: ef6eabc
changes: [docs/architecture/diagrams/erd.mmd]
tests: "dd_trace -> 3 lines (invented column false positives); dd_done -> same 3 lines, nothing else; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render (mmdc not on PATH)]
---

## Changes

- `docs/architecture/diagrams/erd.mmd`: erDiagram; 11 entities, 7 relationships; 0 ⚠ missing;
  source docs/architecture/lld.md#1-data-model @ 5f07d6d
  - Every table, column, nullability, PK/FK and Relationships row from §1 is drawn. `decisions` has no PK
    because §1 says "Primary key: none". `labels`, `verdicts` and `overrides` have composite PKs.
  - `markets` → `pairs` is drawn as two lines, one each for `a_key` and `b_key` (one per Via column and FK).
    That is why there are 7 relationship lines for 6 Relationships rows.
  - `pairs` → `verdicts`, `overrides` and `decisions` are dashed (non-identifying) with no FK marker on the child,
    because §1 says "Foreign keys: none" for those tables. `markets` → `books` and `samples` → `labels` are solid
    because the FK is part of the child PK.
  - `uq_pairs_one_to_one` is partial (it only applies to `equivalent` pairs), so it is a comment on `a_key` and
    `b_key` with no `UK` marker. The other `Unique:` lines name the PK constraint, so `PK` covers them.
  - Notation adaptation: Mermaid erDiagram attribute types cannot contain `.` or start with `[]`. So `market.Key`
    is drawn as `Key`, `time.Time` as `Time`, `[]string` as `string[]`, and so on, and each attribute comment keeps
    the Type cell as written. `<`/`>` in copied text are escaped as `#60;`/`#62;`.

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.8jsuYP/kit.sh && dd_trace
    invented column: decisions.caveats
    invented column: overrides.fingerprint_b
    invented column: verdicts.fingerprint_b
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.8jsuYP/kit.sh && dd_done
    invented column: decisions.caveats
    invented column: overrides.fingerprint_b
    invented column: verdicts.fingerprint_b
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.8jsuYP/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.8jsuYP/kit.sh && dd_typecheck
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.8jsuYP/kit.sh && dd_placeholders
    1
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.8jsuYP/kit.sh && dd_secrets
    8:%% Attribute types are Mermaid-safe tokens (dots and leading [] cannot parse); each comment keeps the Type cell as written.
    (eyeballed: the word "tokens" in a comment, not a value)
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D04/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D04/task.md)
    exit 0

## Questions

- The 3 dd_trace/dd_done lines are kit false positives. `caveats`, `fingerprint_b` (verdicts) and `fingerprint_b`
  (overrides) are real §1 columns, each written in the same table row as another column. Dropping them would
  break the erd rule "every column". The supervisor needs to choose a fix:
  (a) design-lld splits those three rows into one row per column, then draw-diagrams erd re-runs (the diagram
  stays the same and the check goes clean), or (b) the draw-diagrams kit's `dd_complete` learns to read
  multi-column rows.
- The `pairs` → `verdicts`, `overrides` and `decisions` Relationships rows put `1` on the `pairs` end.
  But verdicts, overrides and decisions have no FK ("pairs are rebuilt"; "copied, not linked"). This is drawn as
  stated, not as a gap. For the G3 reviewer: can a verdict, override or decision outlive its pair, which would
  make that parent end 0..1?
- Render is unverified (no `mmdc`). The type-token adaptation above was chosen so GitLab's Mermaid can parse it,
  but nobody has checked that in a renderer yet.
