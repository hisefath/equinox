---
task: M3-D09
harness: claude
model: claude-opus-5-5
status: done
reason: drawn, no gaps; render unverified
base_sha: d7cc8b2
head_sha: 1030e7e
changes: [docs/architecture/diagrams/sequence-venue-down.mmd]
tests: "dd_trace -> no output; dd_done -> no output; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render (mmdc not on PATH)]
---

## Changes

- `docs/architecture/diagrams/sequence-venue-down.mmd` — sequenceDiagram; 7 participants, 12 messages; 0 ⚠ missing;
  source docs/architecture/lld.md#seq-3-decide-while-a-venue-is-down-sequence-venue-down-covers-us-3-fr-020fr-023-fr-025-fr-026-ae-015ae-021 @ d7cc8b2

## Tests run

```text
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.aUqFje/kit.sh && dd_trace
(no output)
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.aUqFje/kit.sh && dd_done
(no output)
$ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.aUqFje/kit.sh && dd_render
unverified|mmdc not on PATH
$ bash scripts/scope-guard.sh .handoff/runs/M3-D09/task.md milestone/m3-design
scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D09/task.md)
exit 0
$ make check
embed-diagrams: docs/architecture/hld.md: 3 diagram(s) in sync
embed-diagrams: missing or empty .../docs/architecture/diagrams/erd.mmd (and 8 other sibling lld diagrams owned by parallel jobs)
embed-diagrams: docs/architecture/lld.md: not changed (fix the error above)
make: *** [test] Error 1   (exit 2; expected in this isolated worktree: the other lld diagrams land from their own jobs, then the supervisor embeds)
```

## Questions

- none

Notes for the reviewer: step 2's stated retry ("each attempt times out at 30 s and BE-5 retries up to 3 times") is
drawn as `loop BE-5 retries up to 3 times` around the one step-2 message, per the skill's loop notation. Steps 3 and
9/a3 use `-->>` as replies to steps 1 and 5. Read it like a reviewer: at each step that can fail, what does the
person see, and which step is safe to retry without charging or writing twice?
