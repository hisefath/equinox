---
task: M3-D13
harness: claude
model: claude-opus-5-5
status: human_review
reason: 1 ⚠ missing in docs/architecture/lld.md#sm-2-venue-health-venue_healthstate-state-venue-health
base_sha: 5f07d6d
head_sha: 9cb84f7
changes: [docs/architecture/diagrams/state-venue-health.mmd]
tests: "dd_trace -> 1 line (transitions in source: 5; drawn: 8 — multi-From rows, see Questions); dd_done -> same 1 line; dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0"
not_verified: [mermaid render (mmdc not on PATH), make check (fails on the 8 sibling LLD diagrams not yet in this worktree; embedding is the supervisor's step)]
---

## Changes

- `docs/architecture/diagrams/state-venue-health.mmd` — stateDiagram-v2; 4 states, 8 transitions; 1 ⚠ missing;
  source docs/architecture/lld.md#sm-2-venue-health-venue_healthstate-state-venue-health @ 5f07d6d

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.lvCC84/kit.sh && dd_trace
    transitions in source: 5; drawn: 8
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.lvCC84/kit.sh && dd_done
    transitions in source: 5; drawn: 8
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.lvCC84/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.lvCC84/kit.sh && dd_secrets
    (no output)
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D13/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D13/task.md)
    $ make check
    embed-diagrams: docs/architecture/hld.md: 3 diagram(s) in sync
    embed-diagrams: missing or empty .../docs/architecture/diagrams/erd.mmd
    (… same for be-class, fe-components, 5 sequence-*, state-pair …)
    embed-diagrams: docs/architecture/lld.md: not changed (fix the error above)
    make: *** [test] Error 1

The one `dd_trace` / `dd_done` line is the kit's row counter, not a dropped or invented transition: SM-2's
transition table has 5 rows, but row 2 lists From `warming`, `healthy` and row 3 lists From `healthy`,
`unhealthy`, `warming`, so the rows name 8 From → To pairs (1 + 2 + 3 + 1 + 1). stateDiagram-v2 has no
multi-source arrow, so each pair is one arrow, all with the row's own event text and no guard (every Guard is `—`).
Every word and id traces; every State row is drawn; the start (`[*] --> warming`) is drawn.

## Questions

- ⚠ design-lld: SM-2 says `Terminal: none`. Is venue health meant to have no end state (it cycles for as long as
  the process runs), or is there a terminal state (e.g. when a venue is removed) that the block should name?
  The diagram draws no end arrow and carries one gap line until this is answered.
- design-lld / supervisor (format, not a gap): transition rows 2 and 3 each list several From states. The
  draw-diagrams kit counts one arrow per row, so its trace check will keep printing `5 vs 8` for this diagram.
  Either split those rows into one From per row in lld.md §6 SM-2, or accept the line as a known kit limitation.
