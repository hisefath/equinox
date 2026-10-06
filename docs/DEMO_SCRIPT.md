# Demo Video Script (about 4–5 minutes)

A terminal recording of this demo, without narration, is committed as
[`docs/demo/equinox-demo.mp4`](demo/equinox-demo.mp4) (about 3 minutes). The script below is for a
narrated version.

Everything runs **offline** from the committed live recording (2026-10-06 05:33 UTC), so the demo is
reproducible and can't be derailed by a venue outage. Segment 6 optionally shows a live run.

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
- The yellow notes on the diagram are the assumptions; each sits next to the component that relies on it.

## 3. Matching on real data (1:15 to 2:30)

```bash
./bin/equinox scan -replay testdata/snapshot -near-misses 0 | head -30
```

Talk through:

- **Venues:** 56,776 Kalshi markets and 3,000 Polymarket markets. Polymarket's became 3,390 binary
  propositions, because a market like `["Ravens","Falcons"]` becomes two.
- **Pairs:** 374, from about 1.2 million scored candidates. They include French election candidates,
  Fed decisions ("Hike 25bps" ↔ "25 bps increase") and Senate races. The two venues' asks for the same
  bet are usually within a cent of each other, which is the market telling you the matches are real.

Then the near misses: pairs that read almost the same but were rejected, and why. This filter shows a
variety of veto kinds:

```bash
./bin/equinox scan -replay testdata/snapshot -near-misses 200 2>/dev/null | grep 'veto:' | grep -v 'veto: timing' | awk -F'veto: ' '{split($2,a,":"); if (c[a[1]]++ < 2) print}' | head -14
```

Examples: "Cut >25bps" vs "25 bps" (comparator), Fed December vs January (month), "appear on the
ballot" vs "win" (predicate), "nominee" vs "president" (predicate), "qualify for the Nations League
Final" vs "win the Nations League" (modifier).

Then the evaluation:

> "On 55 hand-labelled live pairs, including 23 hard negatives built to fool text matching, the matcher
> makes zero false matches and finds 84% of the true ones. On live data, independent audits measured 88%
> precision on pairs it was never tuned on. The remaining errors live in the rules prose: announce vs
> complete, first round vs runoff. So production routing should run on a reviewed mapping table, and the
> system supports that with `-require-review`."

Optional: show the audit table in `docs/EQUIVALENCE.md` §3.3.

## 4. Routing decisions (2:30 to 3:30)

Fees flip the decision. On the Mississippi Senate race Kalshi has the cheaper ask, but Polymarket wins
on all-in cost:

```bash
./bin/equinox route -replay testdata/snapshot -pair 23 -side yes -qty 100
```

Price beats fees, with splitting considered and rejected:

```bash
./bin/equinox route -replay testdata/snapshot -pair 26 -side yes -qty 2000 -split
```

Talk through the output:

- The **match evidence** the decision relies on, and the **caveat** that the venues settle 60 days
  apart, which is a capital-lock-up difference.
- **Per-venue evaluation**, and the **"why" line**: the rule that picked the venue, with the saving.

Buying NO with a limit:

```bash
./bin/equinox route -replay testdata/snapshot -pair 4 -side no -qty 300 -limit 0.70
```

The audit trail:

```bash
tail -1 data/decisions.jsonl | python3 -m json.tool | head -40
```

## 5. Determinism and resilience (3:30 to 4:15)

```bash
for i in 1 2 3; do ./bin/equinox route -replay testdata/snapshot -pair 26 -qty 2000 -split 2>/dev/null | grep '^Decision'; done
go test ./internal/route/ -run 'Deterministic|VenueAgnostic|Exclusions' -v 2>&1 | grep -E '^(---|ok)'
go test ./internal/ingest/ -run 'SlowVenue|FailingVenue' -v 2>&1 | grep -E '^(---|ok)'
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
> about 88% across the live long tail, and the gap is rules-level, so it belongs to a reviewed mapping
> table fed by offline adjudication. Smart routing is tractable once markets are normalized. The real
> complexity is fee curves, depth and data freshness, and all of it is handled deterministically and
> explained."
