---
status: accepted
date: 2026-10-11
decision-makers: human (exact words in journal D13)
journal: D13
---

# 0013. Build Equinox in Go, standard library only

## Context and Problem Statement

The partner prefers Go and accepts Python (REQ-13). The router must replay exactly (FR-013, NFR-001), money must never
be held in floating point (INV-3), and a background refresher must run beside routing (FR-020). The stack was left
open until the design gate (AGENTS.md:57). Which language and runtime builds Equinox?

## Decision Drivers

* REQ-13 — the partner prefers Go; Python is acceptable
* FR-013, NFR-001 — the same order on the same data gives the same decision in 100% of runs
* INV-3 — money is never held in floating point; fees round as each venue publishes
* FR-020 — routing never contacts a venue, so refresh must run in the background beside it
* C-2 — the service runs on Railway

## Considered Options

* Do nothing / simplest: Python 3 scripts with the standard library
* Go 1.27, standard library only (brief P-01) (agent's first proposal) (agent's recommendation)
* TypeScript on Bun or Node

## Decision Outcome

Chosen option: "Go 1.27, standard library only", because it gives one static binary, safe concurrency between
refresh and routing, and exact integer money in the partner's preferred language.

### Consequences

* Good, because one binary serves the service and the command line, with no third-party dependencies to vet
* Good, because goroutines run the refresher while routing reads immutable snapshots, and integer math with explicit ordering makes determinism easy to prove
* Bad, because text matching is written by hand, and live runs on this Mac must go through a container (AGENTS.md:58)

### Confirmation

`make check` runs `gofmt`, `go vet` and `go test` once design-lld fills the Makefile; `go.mod` lists no third-party
modules. Revisit if matching quality needs a text library Go lacks (SC-001 below its gate with rules-text misses
as the cause).

## Pros and Cons of the Options

### Do nothing / simplest: Python 3 scripts with the standard library

* Good, because it is fastest to write and has strong text handling for matching
* Bad, because it is not the partner's preferred language, uses floats by default, and needs extra structure for a long-running service with a background refresher

### Go 1.27, standard library only (brief P-01)

* Good, because it is the partner's preferred language, ships as one static binary, runs the refresher in goroutines beside immutable snapshots, and keeps money in integers with explicit ordering
* Bad, because text matching is hand-written and locally built binaries are blocked by this Mac's firewall

### TypeScript on Bun or Node

* Good, because it is in the partner's stack list and quick to write
* Bad, because numbers are floating point by default, which puts INV-3 at risk, and it is not the stated preference

## More Information

Journal: D13 in `docs/decisions/DECISION_JOURNAL.md` · Approval: `.handoff/TRACKER.md ## Approvals`, G2 D13
2026-10-11 16:17 CDT, "yes, use go" · Artifact: `docs/architecture/hld.md@5d35114` · Revisit if: matching quality
needs a text library Go lacks
