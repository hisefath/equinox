# Demo Video Script (about 4–5 minutes)

Everything below runs **offline** from the committed live recording, so the demo is reproducible and
can't be derailed by a venue outage. Segment 6 optionally shows a live run.

Preparation:

```bash
go build -o bin/equinox ./cmd/equinox
clear
```

---

## 1. The problem (0:00 to 0:30)

Show [`docs/PRD.md`](PRD.md) §1, or just say:

> "The same real-world bet trades on Kalshi and Polymarket under different names, structures, prices
> and fees. Equinox asks three questions. Can we tell programmatically that two markets are the same
> bet? Can we compare them on a like-for-like, fee-inclusive basis? And can a venue-agnostic router
> choose between them deterministically and explain why?"

## 2. The architecture (0:30 to 1:15)

Show the diagram in [`docs/SYSTEM_DESIGN.md`](SYSTEM_DESIGN.md) §1. Points to make:

- Adapters are the only venue-specific code. Everything maps into one canonical binary-market model, with
  integer micro-dollars and fees expressed as a curve.
- Ingestion publishes immutable snapshots. The router is a pure function that never touches the network.
- `route` imports only the canonical model. A test fails the build if it ever mentions a venue.

## 3. Matching on real data (1:15 to 2:30)

```bash
./bin/equinox scan -replay testdata/snapshot -near-misses 8 | less
```

Talk through:

- **Venues:** 56,776 Kalshi markets and 3,000 Polymarket markets. Polymarket's became 3,396 binary
  propositions, because a market like `["Ravens","Falcons"]` becomes two.
- **Pairs:** French election candidates, Fed decisions ("Hike 25bps" ↔ "25 bps increase"), Senate races.
  Point out that the two venues' asks for the same bet are usually within a cent of each other. That is
  the market telling you the matches are real.
- **Near misses:** pairs that read almost identically but were rejected, and why. Show one comparator
  veto and one timing veto.

Then explain the evaluation:

> "On 55 hand-labelled live pairs, including 23 hard negatives built to fool text matching, the matcher
> makes zero false matches and finds 84% of the true ones. On live data, independent audits measured 87%
> precision on pairs it had never been tuned on. The remaining errors live in the rules prose: announce
> vs complete, first round vs runoff. So production routing should require a reviewed mapping table,
> and the system supports that with `-require-review`."

Optional: show `docs/EQUIVALENCE.md` §3.3 (the audit table).

## 4. A routing decision (2:30 to 3:30)

```bash
./bin/equinox route -replay testdata/snapshot -pair 27 -side yes -qty 2000 -split
```

Talk through the output:

- The **match evidence** the decision relies on: shared terms, office, outcome, predicate. Also the
  **caveat** that the venues settle 60 days apart, which is a capital-lock-up difference.
- **Per-venue evaluation:** Kalshi's fee ($34.32) is nearly double Polymarket's ($19.49), but its price is
  a cent lower. On all-in cost Kalshi wins: 0.58716 vs 0.589744 per contract.
- **Split considered** and rejected: one child order is simpler when splitting doesn't save money.

Then the NO side with a limit, which goes the other way:

```bash
./bin/equinox route -replay testdata/snapshot -pair 7 -side no -qty 300 -limit 0.70
```

Show the audit trail:

```bash
tail -1 data/decisions.jsonl | python3 -m json.tool | head -40
```

## 5. Determinism and resilience (3:30 to 4:15)

```bash
for i in 1 2 3; do ./bin/equinox route -replay testdata/snapshot -pair 27 -qty 2000 -split 2>/dev/null | grep '^Decision'; done
go test ./internal/route/ -run 'Deterministic|VenueAgnostic|Exclusions' -v 2>&1 | grep -E '^(---|ok)'
go test ./internal/ingest/ -run SlowVenue -v 2>&1 | grep -E '^(---|ok)'
```

> "The same inputs give the same decision id every time, whatever order the quotes arrive in. Money is
> integer micro-dollars, because Go may fuse float operations on ARM and not on x86. A slow venue is cut
> off at its deadline while a thousand snapshot reads still complete in under 50 milliseconds. A failed
> venue keeps its last good data, is marked unhealthy, and the router excludes it and says why."

## 6. Optional: live (4:15 to 4:45)

```bash
docker build -t equinox . && docker run --rm -p 8080:8080 equinox
# in another terminal, after about 30 s:
curl -s localhost:8080/healthz | head -20
curl -s "localhost:8080/route?pair=1&side=yes&qty=100&split=true" | python3 -m json.tool | head -40
```

## 7. Close (4:45 to 5:00)

> "Answering the three questions. Normalization across venues works: one canonical model holds both, and
> a third venue is one adapter. Equivalence can be detected, at high precision for templated markets and
> about 87% across the live long tail, and the gap is rules-level, so it belongs to a reviewed mapping
> table fed by offline adjudication. Smart routing is tractable once markets are normalized. The real
> complexity is fee curves, depth and data freshness, and all of it is handled deterministically and
> explained."
