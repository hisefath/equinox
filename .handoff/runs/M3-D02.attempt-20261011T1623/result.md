---
task: M3-D02
harness: claude
model: claude-opus-5-5
status: human_review
reason: 3 ⚠ missing in docs/architecture/hld.md#5-trust-boundaries-threats
base_sha: 5d35114
head_sha: adcc25b
changes: [docs/architecture/diagrams/hld-container.mmd]
tests: "dd_trace -> no output; dd_done -> no output; dd_render -> unverified: mmdc not on PATH"
not_verified: [mermaid render]
---

## Changes

- `docs/architecture/diagrams/hld-container.mmd` — flowchart LR; 13 nodes, 14 edges; 3 ⚠ missing;
  source docs/architecture/hld.md#3-containers-c4-level-2 @ 5d35114,
  docs/architecture/hld.md#4-connections @ 5d35114, docs/architecture/hld.md#5-trust-boundaries-threats @ 5d35114

## Tests run

    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.SamWri/kit.sh && dd_trace
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.SamWri/kit.sh && dd_done
    (no output)
    $ . /var/folders/hy/jdyfzw1d74j_zxvpgy4bysm80000gp/T//draw-diagrams.SamWri/kit.sh && dd_render
    unverified|mmdc not on PATH
    $ bash scripts/scope-guard.sh .handoff/runs/M3-D02/task.md milestone/m3-design
    scope-guard: OK — 1 changed path(s) within Owns (milestone/m3-design:.handoff/runs/M3-D02/task.md)
    (exit 0)

## Questions

- ⚠ design-hld: Which trust zone holds CN-2 (decision log)? No TB row names it on its more trusted side, so it is
  drawn outside every zone and L-4 appears to cross TB-2, TB-1 and TB-4.
- ⚠ design-hld: TB-2 (CN-1, CN-3) and TB-3 (CN-3, CN-4) overlap at CN-3 without nesting; Mermaid can place CN-3
  in only one zone (drawn in TB-2, so L-8 does not visibly cross TB-3). Should the boundaries be restated so they
  nest, or is the overlap intended?
- ⚠ design-hld: TB-4's more trusted side is "the running CN-1", but its crossing connections are L-9 (CN-3 → CN-4)
  and L-11 (GitLab → Railway), neither of which reaches CN-1. Should TB-4's trusted side include CN-4 / Railway, or
  should its crossing connections change?
