# Test Results

Run on 2026-10-06 with Go 1.27.1 (darwin/arm64). The raw outputs referenced below are in
[`test-results/`](../test-results/). To reproduce everything offline:

```bash
go test -race -count=1 -cover ./...
```

## 1. Summary

| Area | Result |
|---|---|
| Unit, integration and end-to-end tests | **38/38 pass**, with the race detector on ([`go-test-race-cover.txt`](../test-results/go-test-race-cover.txt)) |
| Statement coverage | **75.5% total**: match 98.2%, route 96.9%, kalshi 86.1%, ingest 85.9%, market 85.4%, polymarket 84.6%, fetch 78.1%, cmd 13.4% (CLI glue; exercised end to end by the pipeline test) |
| Matcher on 55 hand-labelled live pairs | **precision 1.000, recall 0.844** (27/32 found, 0/23 hard negatives accepted) |
| Matcher in context (60,234 markets) | 26 labelled equivalents found, **0 false positives** |
| Live audit, out of sample | `equivalent` tier **precision 0.872** (82/94; 95% Wilson CI 0.79–0.93); `review` tier 0.167 |
| Determinism | 3 replays → identical pair hashes; identical routing decision ids ([`determinism.txt`](../test-results/determinism.txt)) |
| Live end-to-end run | Kalshi 56,776 + Polymarket 3,396 markets → 323 pairs; order books for 320/323 and 323/323 matched markets ([`live-scan-2026-10-06.txt`](../test-results/live-scan-2026-10-06.txt)) |

## 2. Test inventory

| Package | Tests | What they prove |
|---|---|---|
| `internal/market` | `TestParseAmount`, `TestParseQtyFloors`, `TestFeeCurve`, `TestBookNormalizeAndAsksFor` | Exact decimal parsing (up to 6 dp, rejecting more); sizes floored; **both venues' published fee examples reproduced to the micro-dollar** (Kalshi $1.75 / $0.00363825 → $0.01 / $1.7157 → $1.72; Polymarket $1.00, $0.84, $0.0937 at 5 dp, exponent 2); books sorted, merged, cleaned; NO asks derived from YES bids; crossed books detected |
| `internal/fetch` | retries then succeeds; no retry on 404 or malformed JSON; deadline bounds a hung venue; rate limit spaces requests; record → replay with the network gone | The resilience contract every adapter relies on |
| `internal/ingest` | failing venue keeps last-known-good and doesn't affect others; slow venue cut off **while 1,000 snapshot reads complete in < 50 ms**; invalid books rejected and previous kept | NFR2 (no blocking) and NFR3 (graceful failure) |
| `internal/venues/kalshi` | normalization + skip counts (scalar, MVE combo); empty-side price sentinels ignored; one bad timestamp doesn't fail a page; event fee override beats series multiplier; `flat` fee → untradable; venue 500 and malformed JSON surface as errors; YES asks derived from NO bids, malformed levels dropped | Adapter correctness on real payload shapes |
| `internal/venues/polymarket` | Yes/No and named-outcome decomposition (`["Ravens","Falcons"]` → 2 propositions with mirrored quotes); placeholders, inactive, 3-outcome and token-less markets skipped and counted; past-end and disputed markets untradable; feeSchedule → curve; books keyed by asset id, worst-first ordering normalized, ms timestamps; list fields accepted in both encodings | Adapter correctness on real payload shapes |
| `internal/route` | cheapest venue; **fees flip the decision**; every exclusion (stale, unhealthy, closed, crossed, empty, below minimum); full fill preferred, then split; split drops venues below minimum; limit price and partial fills; NO side; tie-break by venue id; invalid orders and no venues rejected; **determinism under 20 permutations**; **architecture test: route imports only `market` and never names a venue** | FR4, FR5, NFR1, NFR4 |
| `internal/match` | comparator-shape table; numbers vs dates vs years vs district ids; teams and leagues (Baltimore ≠ Falcons, Yankees ≠ Mets); one-to-one assignment and input-order independence; **12 regression cases from live false positives**; reviews promote and block; **labelled-pair evaluation (pairwise and in context)** | FR3 and the precision gate |
| `cmd/equinox` | `TestPipelineOnRecordedSnapshot`: the full system on the committed live recording (ingest both venues → match → books → route), healthy venues, ≥100 pairs, top pairs route deterministically on real books | End-to-end integration |

## 3. Matcher evaluation on hand-labelled pairs

[`matcher-labelled-eval.txt`](../test-results/matcher-labelled-eval.txt) has the per-pair table. The
fixture is [`internal/match/testdata/labelled_pairs.json`](../internal/match/testdata/labelled_pairs.json):
55 pairs, each with the raw venue JSON and its label. They were labelled from a live census on
2026-10-06 ([`research/overlap_census.md`](../research/overlap_census.md)).

| | Labelled equivalent (32) | Labelled not equivalent (23) |
|---|---|---|
| Judged equivalent | **27** (TP) | **0** (FP) |
| Judged not equivalent / review | 5 (FN) | 23 (TN) |

Each of the 23 hard negatives was rejected by the veto that targets its failure type:

| Negative type | n | Rejected by |
|---|---|---|
| threshold vs bucket ("Above 2.4%" vs "2.4%") | 6 | comparator / threshold |
| different date window | 4 | month, year, date |
| different resolution source (BRTI vs Binance) | 4 | comparator, threshold (the venues' different comparators are the visible symptom), outcome |
| different threshold | 3 | threshold, comparator |
| different proposition | 3 | outcome, threshold |
| related but different (NL pennant vs World Series) | 1 | scope (stage) |
| inverse polarity (Atlanta vs Ravens token) | 1 | team |
| boundary inclusivity (> 50 vs ≥ 50) | 1 | comparator |

The 5 missed equivalents:
- 2 sports series markets scored too low and landed in `review`.
- Netanyahu: deadline dates one day apart with trading cut-offs 22 h apart. The conservative deadline
  rule rejects it.
- Gemini vs Google: deliberately not aliased.
- "Wins ALDS" vs "advances to ALCS": a stage veto.

All five are listed in [`EQUIVALENCE.md`](EQUIVALENCE.md) §4.

## 4. Live audits

Hand labels cover templated markets well but miss the long tail. So live matches were audited.

### Method

1. Draw a stratified random sample of live pairs: `equivalent` and `review` tiers, fixed seed.
2. Six independent judges each label about 22 pairs against the strict definition in `EQUIVALENCE.md`,
   from both venues' question, outcome and rules text, **without seeing the matcher's tier or score**.
3. Two skeptical reviewers re-judge every "not equivalent" or "uncertain" label, looking for misreadings
   (e.g. Kalshi city names vs Polymarket nicknames). A false positive counts only if both reviewers
   agree.
4. Judges are LLM agents (see [`AI_USAGE_LOG.md`](../AI_USAGE_LOG.md)). Their labels, with reasons, are
   committed in [`research/audit/labels.json`](../research/audit/labels.json) and
   [`research/audit2/labels.json`](../research/audit2/labels.json) for human spot-checking.

### Results

| Audit | Matcher | Pairs | `equivalent` precision | `review` precision |
|---|---|---|---|---|
| 1, exploratory | first version | 130 | 0.626 (62 eq / 37 not / 1 uncertain) | 0.233 (7 / 30) |
| 1, re-scored | final version, **same pairs (in-sample)** | 55 still auto-tiered | 0.945 (52 / 3) | — |
| **2, out of sample** | **final version** | **120 never-audited pairs** | **0.872** (82 eq / 12 not / 2 uncertain) | **0.167** (4 / 24) |

Notes:
- The 95% Wilson interval for out-of-sample precision is **0.79–0.93**. Counting the 2 uncertain labels
  as wrong gives 0.854.
- The in-sample figure (0.945) is optimistic by construction, because the fixes were designed against
  those pairs. **0.872 is the number to quote.**
- The gap between the tiers (0.87 vs 0.17) shows the tier boundary is meaningful.
- The remaining false positives are mostly rules-level (announce vs complete, first round vs runoff, Fed
  meeting window vs calendar date). This is why routing in production should require a review
  (`-require-review`, `reviews/pairs.json`).

## 5. Determinism

[`determinism.txt`](../test-results/determinism.txt):

```
run 1: pairs=323 sha256(pairs)=f73127d27c626506
run 2: pairs=323 sha256(pairs)=f73127d27c626506
run 3: pairs=323 sha256(pairs)=f73127d27c626506
Decision e90dca203225521a: FILLED   (x3)
```

During development, an earlier version produced different pair lists across runs. The cause was float
sums in map-iteration order, which flipped near-tied scores. It was fixed and is now guarded by these
checks plus `TestDeterministic` and `TestOneToOneAndDeterminism`.

## 6. Performance (measured)

| Step | Time |
|---|---|
| Kalshi ingest, 30 pages / 56,776 markets (live) | 7.1 s |
| Polymarket ingest, 30 pages / 3,000 markets (live, in parallel) | 7.8 s |
| Matching, 60k markets / 97,658 candidates | about 7 s (10-core laptop); about 13 s (4-CPU container) |
| Books for 323 pairs (2 batched calls per venue) | 0.36 s Kalshi, 0.50 s Polymarket |
| One routing decision | microseconds (pure CPU) |

## 7. What is not tested

- **Live venues inside CI.** Tests run against recorded responses so they are deterministic. Live runs
  are manual (`-record`) and their outputs are committed.
- **Cloud Run deployment.** The guide is written and the image builds; no GCP project was provisioned.
- **Load and soak behaviour of `serve`** over hours: rate limiting under sustained polling.
