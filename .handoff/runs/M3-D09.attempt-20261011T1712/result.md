---
task: M3-D09
harness: claude
model: claude-opus-5-5
status: human_review
reason: 2 ⚠ missing in docs/architecture/lld.md#seq-3-decide-while-a-venue-is-down-sequence-venue-down-covers-us-3-fr-020fr-023-fr-025-fr-026-ae-015ae-021
base_sha: 5f07d6d
head_sha: b3fb67c
changes: [docs/architecture/diagrams/sequence-venue-down.mmd]
tests: "dd_trace -> 1 line (steps in source 9; messages + notes drawn 11); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard.sh -> exit 0"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/sequence-venue-down.mmd` — sequenceDiagram; 8 participants, 10 messages; 2 ⚠ missing;
  source docs/architecture/lld.md#seq-3-decide-while-a-venue-is-down-sequence-venue-down-covers-us-3-fr-020fr-023-fr-025-fr-026-ae-015ae-021 @ 5f07d6d

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.ptcZv5/kit.sh && dd_trace
    steps in source (n. and aN.): 9; messages + notes drawn: 11
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.ptcZv5/kit.sh && dd_done
    steps in source (n. and aN.): 9; messages + notes drawn: 11
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T/draw-diagrams.ptcZv5/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D09/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D09/task.md)
    (exit 0)

The one dd_trace line is not a dropped step: SEQ-3 steps 1 and 7 each hold two messages (step 1 chains
BE-6 → BE-5 → Offshore venue; step 7 sends `Append(decision)` to BE-9 and then the 200 to Labs engineer), so the
9 source steps draw as 11 lines (10 messages + 1 Note for step 2). Every word and id traces; no id dropped or invented.

## Questions

- ⚠ design-lld: In step 1, which participant retries the timed-out book refresh request, BE-6 Refresher or BE-5 fetch.Client? (drawn as a `loop retried up to 3 times` around both hops)
- ⚠ design-lld: When every venue is stale or unhealthy, what does the Labs engineer get? The alt "replaces step 6" only, so step 7 (`Append(decision)` and a second 200 "naming both venues' reasons") still follows a2's 200 "not filled". Should the alt replace steps 6-7, and is the not-filled decision appended to BE-9?
- design-lld (form): split step 1 (BE-6 → BE-5, BE-5 → Offshore venue) and step 7 (BE-11 → BE-9, BE-11 → Labs engineer) into one `From → To: message` per step, so the step-count check passes on re-run.
- design-lld (form): BE-6 names two participants (Refresher and Store); drawn as `BE6` and `Store`. Confirm Store is part of BE-6 rather than its own BE id.
