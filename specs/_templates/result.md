---
task: M1-T001
harness: codex
model: <model id>
status: done
reason: <one line>
base_sha: <sha the branch started from>
head_sha: <sha after your last commit>
changes: [path/a, path/b]
tests: "make check -> 0 (42 passed)"
not_verified: []
---

<!--
Result contract (software-factory-kit WORKFLOW.md §7). The worker writes it to $HANDOFF_DIR/runs/<id>/result.md as its last
act; scripts/delegate.sh parses `task` and `status`. Delete this comment when filling.

status   done          Done-when passes, all work committed, tree clean
         human_review  blocked, unsure, or a question only the human can answer (put it under ## Questions)
         relaunch      the run failed for a reason a fresh attempt could fix (say which under ## Questions)
tests    the exact command and its exit code + counts, copied from the output below — never from memory
not_verified  everything you could not check (no network, no browser, untested edge case …)

The supervisor treats a missing or unparseable file, a timeout, or a dirty tree after "done" as human_review, and it
re-runs make check itself: this file is a claim, not proof.
-->

## Changes

- `path/a` — <what changed and why, one line>

## Tests run

```text
$ make check
<paste the real output tail, including counts and the exit code>
```

## Questions

- none
