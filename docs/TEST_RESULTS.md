# Test Results

Run on 2026-10-06 with Go 1.27.1 (darwin/arm64), against the committed live recording
`testdata/snapshot` (recorded 2026-10-06 05:33:57 UTC). Raw outputs are in
[`test-results/`](../test-results/). To reproduce everything offline:

```bash
go test -race -count=1 -cover ./...
```

## 1. Summary

| Area | Result |
|---|---|
| Unit, integration and end-to-end tests | **46/46 pass** with the race detector on ([`go-test-race-cover.txt`](../test-results/go-test-race-cover.txt)) |
| Statement coverage | **79.9% total**: match 98.0%, route 97.8%, ingest 88.9%, kalshi 87.4%, polymarket 85.4%, market 84.0%, fetch 78.1%, cmd 32.2% (HTTP API, routing gate and decision log tested directly; the full pipeline end to end) |
| Matcher on 55 hand-labelled live pairs | **precision 1.000, recall 0.844** (27/32 found, 0/23 hard negatives accepted); precision 1.000 in every topic |
| Matcher in context (60,223 markets) | 26 labelled equivalents found, **0 false positives** |
| Live precision, final matcher, out of sample | **0.879** (124/141 audited `equivalent` pairs; 95% Wilson CI 0.82–0.92). Details in §4 |
| Determinism | 3 replays → identical pair hashes; identical routing decision ids ([`determinism.txt`](../test-results/determinism.txt)) |
| Live end-to-end run | Kalshi 56,776 + Polymarket 3,390 markets → 374 pairs; books for 372/374 and 374/374 matched markets ([`live-scan-2026-10-06.txt`](../test-results/live-scan-2026-10-06.txt)) |

## 2. Test inventory

| Package | Tests | What they prove |
|---|---|---|
| `internal/market` | `TestParseAmount`, `TestParseQtyFloors`, `TestFeeCurve`, `TestBookNormalizeAndAsksFor` | Exact decimal parsing (up to 6 dp). Overflow and stray signs rejected. Sizes floored. **Both venues' published fee examples reproduced to the micro-dollar** (Kalshi $1.75; $0.00363825 → $0.01; $1.7157 → $1.72; Polymarket $1.00, $0.84, $0.0937 at 5 dp, exponent 2). Books sorted, merged (saturating) and cleaned; NO asks derived from YES bids; crossed books detected |
| `internal/fetch` | retry then success; no retry on 404 or malformed JSON; a deadline bounds a hung venue; the rate limit spaces requests; record → replay with the network gone | The resilience contract every adapter relies on |
| `internal/ingest` | a failing venue keeps last-known-good and doesn't affect others; a slow venue is cut off **while 1,000 snapshot reads finish in < 50 ms**; invalid books rejected and the previous kept; partial book results still used | NFR2 (no blocking) and NFR3 (graceful failure) |
| `internal/venues/kalshi` | normalization and skip counts (scalar, combo); empty-side price sentinels ignored; one bad timestamp doesn't fail a page; event fee override beats the series multiplier; `flat` fee → untradable; a 500 or malformed JSON surfaces as an error; a cursor that never advances stops the crawl; one failed book chunk doesn't cost the others; YES asks derived from NO bids | Adapter correctness on real payload shapes |
| `internal/venues/polymarket` | Yes/No and named-outcome decomposition (`["Ravens","Falcons"]` → 2 propositions with mirrored quotes); placeholders, inactive, 3-outcome and token-less markets skipped and counted; past-end and disputed markets untradable; feeSchedule → curve; books keyed by asset id, worst-first ordering normalized, stamped with receipt time; list fields accepted in both encodings; cursor guard | Adapter correctness on real payload shapes |
| `internal/route` | cheapest venue; **fees flip the decision**; every exclusion (stale, unhealthy, closed, crossed, empty, below minimum); future-stamped books, a non-positive max age and conflicting duplicate quotes; full fill preferred, then split; split drops venues below minimum; limit and partial fills; NO side; tie-break by venue id; invalid orders; **every decision names its deciding rule**; **determinism under 20 permutations**; **architecture test: route imports only `market` and never names a venue** | FR4, FR5, NFR1, NFR4 |
| `internal/match` | comparator shapes; numbers vs dates vs years vs district ids; thousands separators, season labels, ISO dates; teams and leagues (Baltimore ≠ Falcons, Yankees ≠ Mets); one-to-one assignment and input-order independence; **12 regression cases from live false positives, each pinned to its veto**; reviews promote and block; **labelled-pair evaluation, pairwise and in context** | FR3 and the precision gate |
| `cmd/equinox` | `TestPipelineOnRecordedSnapshot` (the whole system on the live recording: ingest → match → books → route, deterministic on real books); `TestRouteEndpoint` (`/route` status codes for good and bad input, decisions logged and invalid requests not); `TestRoutableGate` (rejected / review / `-require-review`) | End-to-end integration and the API contract |

## 3. Matcher evaluation on hand-labelled pairs

The per-pair table is [`matcher-labelled-eval.txt`](../test-results/matcher-labelled-eval.txt). The
fixture is [`internal/match/testdata/labelled_pairs.json`](../internal/match/testdata/labelled_pairs.json):
55 pairs, each stored with the raw venue JSON and its label, hand-labelled from a live census on
2026-10-06 ([`research/overlap_census.md`](../research/overlap_census.md)). Each pair goes through the
**real adapters** and is judged against term statistics from the full recorded corpus.

| | Labelled equivalent (32) | Labelled not equivalent (23) |
|---|---|---|
| Judged equivalent | **27** (TP) | **0** (FP) |
| Judged not equivalent / review | 5 (FN) | 23 (TN) |

| Topic | TP | FP | FN | TN | Precision | Recall |
|---|---|---|---|---|---|---|
| economics | 3 | 0 | 0 | 4 | 1.000 | 1.000 |
| elections | 8 | 0 | 0 | 3 | 1.000 | 1.000 |
| entertainment | 2 | 0 | 0 | 3 | 1.000 | 1.000 |
| fed | 5 | 0 | 0 | 3 | 1.000 | 1.000 |
| geopolitics | 1 | 0 | 1 | 1 | 1.000 | 0.500 |
| sports | 7 | 0 | 3 | 4 | 1.000 | 0.700 |
| tech | 1 | 0 | 1 | 2 | 1.000 | 0.500 |
| crypto | 0 | 0 | 0 | 3 | — | — |

Each of the 23 hard negatives was rejected by a veto aimed at its failure type. The breakdown is
threshold vs bucket (6), different date window (4), different resolution source (4, rejected through the
comparator and threshold differences they also have), different threshold (3), different proposition (3),
related but different (1), inverse polarity (1) and boundary inclusivity (1).

The 5 missed equivalents:
- 2 MLB series markets scored too low and land in `review`.
- Netanyahu: deadline dates one day apart with trading cut-offs 22 h apart, so the conservative deadline
  rule rejects it.
- Gemini vs Google: deliberately not aliased.
- "Wins ALDS" vs "advances to ALCS": a predicate-scope veto (win vs advance).

All five are discussed in [`EQUIVALENCE.md`](EQUIVALENCE.md) §4.

## 4. Live audits

Hand labels cover templated markets well and miss the long tail, so live matches were audited three times.

### Method

1. Take a stratified random sample of live pairs with a fixed seed. Audit 3 took **all** pairs that a set
   of late bug fixes newly created.
2. Six LLM judges each label their share of pairs against the strict definition in `EQUIVALENCE.md`,
   from both venues' question, outcome, times and a 700-character rules excerpt.
   **In audits 1 and 2 the judges' input files also contained the matcher's tier and score**, with an
   instruction to ignore them. That carries an anchoring risk. **Audit 3 removed them entirely**, so its
   judges were blind.
3. One skeptic agent re-judges every "not equivalent" or "uncertain" label. A false positive is counted
   only if the judge and the skeptic agree. "Equivalent" labels come from one judge.
4. Every pair shown to a judge (`chunk*.json`) and every verdict with its reason (`labels.json`) is
   committed under `research/audit*/` for human spot-checking.

### Results

| Audit | Matcher | Pairs | `equivalent` precision | `review` precision |
|---|---|---|---|---|
| 1, exploratory | first version | 130 | 0.626 (62 / 37 / 1 uncertain) | 0.233 (7/30) |
| 2, out of sample | after audit-1 fixes | 120 never-audited | 0.872 (82 / 12 / 2 uncertain) | 0.167 (4/24) |
| 2, re-scored | **final** | the audit-2 pairs the final matcher still tiers `equivalent` | 0.863 (82 / 13) | — |
| 3, blind | **final** | all 50 pairs new under the final fixes | 0.913 (42 / 4) | 0.000 (0/4) |
| **Final matcher, out of sample** | **final** | **audit 2 re-scored + audit 3** | **0.879 (124 / 17; 95% CI 0.82–0.92)** | |

Notes:
- The re-scoring of audit-2 pairs and the selection of audit-3 pairs used the 04:46 UTC recording, the
  one the audits sampled from. The committed `testdata/snapshot` was re-recorded at 05:33 UTC with the
  final matcher, so that its order books cover the final pairs.
- Audit 1's pairs, re-scored with the final matcher on that recording, give 0.927 (51/55). That number is
  in-sample (the fixes were designed against them), so it is not quoted as a result.
- **The tiers mean something:** `equivalent` is right about 88% of the time, `review` rarely. Not routing
  `review` is correct.
- The remaining false positives are mostly rules-level (announce vs complete, first round vs runoff, Fed
  meeting window vs calendar date). A few are structural: cross-league nickname homonyms, such as the LA
  Kings (NHL) vs the Sacramento Kings (NBA). Production routing should therefore require a review
  (`-require-review`, `reviews/pairs.json`).

## 5. Determinism

[`determinism.txt`](../test-results/determinism.txt):

```
run 1: pairs=374 sha256(pairs)=29191859569da05e
run 2: pairs=374 sha256(pairs)=29191859569da05e
run 3: pairs=374 sha256(pairs)=29191859569da05e
Decision 5c7e7694477187a2: FILLED   (x3)
```

An earlier version produced different pair lists across runs, because float sums ran in map-iteration
order and flipped near-tied scores. That was fixed and is guarded by these checks plus
`TestDeterministic` and `TestOneToOneAndDeterminism`. The final review then found that duplicate quotes
for one market could still make a decision depend on input order. They are now excluded as conflicting,
and quotes are sorted into a total order that includes their content.

## 6. Performance (measured)

| Step | Time / size |
|---|---|
| Kalshi ingest: 30 pages, 56,776 markets (live) | 4.3 s |
| Polymarket ingest: 30 pages, 3,000 markets (live, in parallel) | 8.2 s |
| Matching: 60k markets, about 1.22M candidates scored, 90,726 vetted | about 7 s (10-core laptop); about 11 s (4-CPU container) |
| Books for 374 pairs | 4 Kalshi GETs (0.37 s) + 1 Polymarket POST (2.3 s) |
| One routing decision | microseconds (pure CPU) |
| Peak memory, full scan | about 670 MB resident |

## 7. Final review

Before submission, an independent review pass covered four dimensions: requirements compliance against
the PDF and PRD, money and routing correctness, pipeline correctness, and documentation accuracy. Each
finding then went to an adversarial verifier. Confirmed findings were fixed with regression tests. They
included:
- Polymarket books stamped with their last-change time, not receipt time;
- thousands separators and season labels misparsed by the matcher;
- the routing explanation not naming the rule that decided;
- duplicate quotes making a decision order-dependent;
- an explicit `limit=0` silently becoming a market order;
- unbounded pagination on a repeated cursor;
- one failed book chunk discarding the rest;
- several inaccurate statements in these documents.

One finding was refuted with Kalshi's own fee-rounding documentation: the claim that Kalshi rounds the
exact fee sum once. Kalshi rounds per fill.

## 8. What is not tested

- **Live venues inside CI.** Tests use recorded responses so they are deterministic. Live runs are
  manual (`-record`) and their outputs are committed.
- **Cloud Run deployment.** The image builds and live `serve` was verified in a local container. No GCP
  project was provisioned.
- **Long-running soak of `serve`:** rate limits under hours of polling.
- **Cross-architecture equality of matcher scores** (amd64 vs arm64). Routing is integer-only, but
  matcher scores are floats summed in a fixed order.
