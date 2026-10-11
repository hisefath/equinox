---
task: M3-D13
harness: claude
model: claude-opus-5-5
status: human_review
reason: "dd_done prints 'no end': SM-2 states Terminal: none by design, and the state-<entity> Check has no form for a stated no-terminal; diagram drawn faithfully, no gaps"
base_sha: d7cc8b2
head_sha: 94cb6c0
changes: [docs/architecture/diagrams/state-venue-health.mmd]
tests: "dd_trace -> no output; dd_done -> 1 line (no end); dd_render -> unverified: mmdc not on PATH; scope-guard -> exit 0; make check -> exit 2 (sibling diagrams of other M3 jobs absent in this worktree)"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/state-venue-health.mmd` — stateDiagram-v2; 4 states, 8 transitions; 0 ⚠ missing;
  source docs/architecture/lld.md#sm-2-venue-health-venue_healthstate-state-venue-health @ d7cc8b2

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.W2A5He/kit.sh && dd_trace
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.W2A5He/kit.sh && dd_done
    no end
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.W2A5He/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D13/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D13/task.md)
    (exit 0)
    $ make check
    embed-diagrams: docs/architecture/hld.md: 3 diagram(s) in sync
    embed-diagrams: missing or empty .../docs/architecture/diagrams/erd.mmd
    (… same for be-class, fe-components, sequence-refresh-and-match, sequence-route-order, sequence-venue-down,
    sequence-measure-precision, sequence-adjudicate-offline, state-pair: other M3 diagram jobs' outputs)
    embed-diagrams: docs/architecture/lld.md: not changed (fix the error above)
    make: *** [test] Error 1
    (exit 2)

## Questions

- Supervisor / skill owner: SM-2 says `**Terminal:** none by design (draw no end state)`, so the diagram has
  `[*] --> warming` and no `--> [*]` arrow. draw-diagrams' state Check (`dd_typecheck`) only accepts an end arrow
  or a `%% ⚠ missing:` line naming the end, so `dd_done` reports `no end`. Drawing an end would invent a terminal
  state, and a gap line would re-ask design-lld a question SM-2 already answers. Accept the diagram as is (the
  check is a false positive here), or should the kit learn a "Terminal: none" form?
- `make check` fails in this worktree only because the other M3 diagram jobs' `.mmd` files are not on this
  branch; expected to pass once the supervisor merges all diagram jobs and runs `scripts/embed-diagrams.sh`.
- No ⚠ gaps for design-lld.
