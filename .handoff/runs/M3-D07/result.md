---
task: M3-D07
harness: claude
model: claude-opus-5-5
status: human_review
reason: 3 ⚠ missing in docs/architecture/lld.md#seq-1-refresh-match-and-publish-a-snapshot-sequence-refresh-and-match-covers-us-1-fr-001fr-007-fr-024-fr-040-ae-001ae-005
base_sha: d7cc8b2
head_sha: 4a97f7f
changes: [docs/architecture/diagrams/sequence-refresh-and-match.mmd]
tests: "dd_trace -> 1 line (step count 16 vs 19: kit counts only n. and aN. steps, source's second alt uses b1-b3); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard.sh -> exit 0"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/sequence-refresh-and-match.mmd` — sequenceDiagram; 8 participants, 19 messages; 3 ⚠ missing;
  source docs/architecture/lld.md#seq-1-refresh-match-and-publish-a-snapshot-sequence-refresh-and-match-covers-us-1-fr-001fr-007-fr-024-fr-040-ae-001ae-005 @ d7cc8b2

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.zBHlV4/kit.sh && dd_trace
    steps in source (n. and aN.): 16; messages + notes drawn: 19
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.zBHlV4/kit.sh && dd_done
    steps in source (n. and aN.): 16; messages + notes drawn: 19
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.zBHlV4/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D07/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D07/task.md)
    (exit 0)

The one dd_trace / dd_done line is a counter mismatch, not a dropped or invented step: the kit counts source steps
with `a?[0-9]+.`, so SEQ-1's second alt (`b1.`–`b3.`) is not counted. Counting `[a-z]?[0-9]+.` gives 19 source
steps (13 + a1–a3 + b1–b3), equal to the 19 messages drawn. Dropping b1–b3 to make the counter pass would drop an
alt branch the job's "Must show" requires. No other trace, type, kind, header or scope check printed anything;
dd_secrets printed nothing.

## Questions

- ⚠ design-lld: The offshore alts replace steps 4, 6 and 8 and the US-regulated ones replace steps 2, 5 and 7, so
  the replaced ranges interleave. A sequence alt holds one contiguous range, so the diagram's alt spans steps 4-8 and
  its else branches leave out steps 5 and 7 even though they still happen. Can SEQ-1 order each venue's steps
  together so each alt replaces one contiguous range?
- ⚠ design-lld: What are the messages of the US-regulated venue's failure branch? SEQ-1 says only that it "replaces
  steps 2, 5 and 7 the same way" (drawn as a ⚠ Note before step 2).
- ⚠ design-lld: What are the messages of the US-regulated venue's refusal branch? SEQ-1 says only "the same for the
  US-regulated venue in steps 2, 5 and 7" (drawn as a ⚠ Note before step 2).
- ⚠ design-lld: The offshore alts say they replace step 4 (BE-4's page requests), yet a1/b1 are BE-5's replies to
  those requests after its retries. Do the alts replace steps 4, 6 and 8, or only step 8 after 4 and 6 fail?
  (Drawn as written: a1/b1 are plain arrows because their request is in a replaced step.)
- Kit/source mismatch: SEQ-1's second alt numbers its steps b1-b3 where references/types.md's Source form says aN.
  Either design-lld renames them a1-a3 or the kit's step counter accepts bN; until then dd_trace reports 16 vs 19.
