# Project Equinox: Architecture, Concepts and Tradeoffs

[`SYSTEM_DESIGN.md`](SYSTEM_DESIGN.md) shows *what* the system is. This document explains *why*: the
system-design ideas each part relies on, the alternatives considered, and what each choice costs.

## 1. Summary

Equinox is a **ports-and-adapters** system built around one **canonical data model**:

- Venue adapters translate each venue's API into that model.
- A **deterministic matcher** proposes which markets are the same bet. Text similarity generates
  candidates; structural vetoes enforce precision.
- A **pure-function router** prices a hypothetical order against every equivalent market on fully
  fee-inclusive terms and explains its choice.
- Ingestion publishes **immutable snapshots**, so routing never waits on a network call.

Everything is Go standard library: one static binary with no third-party dependencies.

## 2. Language and tooling

| Option | For | Against | Verdict |
|---|---|---|---|
| **Go, standard library only** | The partner's stated preference and in-house stack (Go + GCP). Goroutines and `context` make "fetch two venues concurrently, each with a deadline" trivial. `net/http`, `encoding/json`, `log/slog`, `testing` and `httptest` cover every need. One static binary deploys straight to Cloud Run. No dependencies means no supply-chain exposure and nothing for a reviewer to vet. | Text processing is more verbose than in Python. There is no stdlib fuzzy-matching or Unicode-normalization library. | **Chosen.** Nothing the matcher needs (TF-IDF, regexes, a small stemmer) is more than about 100 lines of stdlib code. |
| Python | Fastest to prototype. `rapidfuzz` / `scikit-learn` / `sentence-transformers` for matching | The partner's second choice. Concurrency and determinism are harder to guarantee. Dependency weight. | Rejected. Easier matching libraries would come at the expense of the stated preference and of determinism. |
| Java | PEAK6 uses it widely | Heavier runtime and boilerplate for a prototype. Not the stated preference | Rejected |

Tooling choices that follow from the above:
- **No web framework.** The Go 1.22+ `http.ServeMux` handles method-and-path patterns (`GET /route`).
- **`log/slog`** for structured logs, which Cloud Logging ingests natively.
- **Docker / distroless** for deployment. The image has no shell and runs as non-root.
- **Record/replay transport** (`fetch.Recorder`) instead of a mocking library. It records real venue
  responses (gzipped) and serves them back, so tests and demos run on real data with no network.

## 3. Core concepts

### 3.1 Ports and adapters (hexagonal architecture)

`ingest.Venue` is the port: `Name()`, `Markets(ctx)`, `Books(ctx, markets)`. Each venue is an adapter.
The dependency rule is that the core (`market`, `match`, `route`) never imports an adapter.
`route` imports *only* `market`, and `TestRouteIsVenueAgnostic` parses the package's imports and source
and fails the build if that ever changes. Writing architecture rules down as tests ("fitness
functions") is how they survive the first deadline.

**Why:** the PRD's desired state is "a reusable foundation capable of supporting multiple venues", and
both venue APIs changed shape during 2025–26: Kalshi moved to `*_dollars` and `*_fp` fixed-point
strings, and Polymarket moved to keyset pagination and a `feeSchedule` object. Only adapters should
have to change when that happens.

**Cost:** an extra layer of translation. Some venue nuance is lost in it, such as Kalshi's structured
strike fields; §5.4 discusses this.

### 3.2 A canonical data model (normalization layer)

Trading systems call this a *security master* or *symbology* layer. Every venue's instrument maps to
one internal representation. Here that representation is the **binary proposition**:

- `Market` holds a question, an outcome label, close and resolution times, the resolution rules text,
  a fee curve, a minimum size and an opaque `BookRef`.
- `Book` stores **only the YES side** (bids and asks). A NO contract is the complement of YES, so NO
  asks are YES bids at `$1 − p`.

This works for both venues:
- Kalshi publishes only bids: YES bids, plus NO bids, which are YES asks in disguise.
- Polymarket's two outcome tokens have mirror-image books.

Multi-outcome structures are decomposed:
- A Kalshi event with 34 leaders becomes 34 markets.
- A Polymarket `["Rays","Yankees"]` market becomes two propositions, "Rays" and "Yankees", each priced
  from its own token's book.

As a result, matching is always YES-to-YES, and no polarity logic exists anywhere downstream.

### 3.3 Fixed-point money and determinism

`market.Amount` is an `int64` of **micro-dollars**:
- Kalshi's fixed-point fields allow up to 6 decimal places, and Polymarket settles in 6-decimal
  USDC/pUSD.
- Fees use `math/big` and are exact to the micro-dollar. Venue-specific rounding (Kalshi to the cent per
  order, Polymarket to $0.00001) is applied once per venue leg.

Floats are avoided because **the Go spec permits fused multiply-add**, and the compiler emits it on
arm64 but not amd64. A float-based router could therefore make a different near-tie decision on a
developer's Mac than on an x86 server. For the same reason:
- the router compares costs per contract by exact cross-multiplication (`total_a × qty_b` vs
  `total_b × qty_a`) instead of dividing;
- inputs are sorted before processing;
- every tie-break is a total order (price, then venue id, then market id);
- the matcher sums floats in sorted key order, after a real run-to-run difference caught during
  development (§7).

### 3.4 Concurrency: immutable snapshots with an atomic swap

`ingest.Store` holds an `atomic.Pointer[Snapshot]`:
- **Writers** (refresh loops) take a writer-only mutex, copy the current snapshot's maps, apply their
  changes and swap the pointer.
- **Readers** (router, matcher, HTTP handlers) do one atomic load and get a consistent, immutable view.

This is read-copy-update. Readers never take a lock and never wait for a writer, which is what makes
"routing does not block on external calls" true by construction rather than by care. A test does 1,000
snapshot reads while a venue refresh is stuck and asserts they finish in under 50 ms.

| Alternative | Why not |
|---|---|
| `sync.RWMutex` around shared maps | Readers can be blocked by a writer, and readers could see a half-applied refresh |
| Channels / an actor goroutine owning state | More code, and the router would wait in a queue behind ingestion |
| An external store (Redis / Postgres) | Network I/O on the routing path, which is exactly what the PRD forbids. It belongs in a production evolution for *durability*, not on the hot path |

**Cost:** each refresh copies the map headers (O(markets)), which is negligible at about 60k markets.

### 3.5 Resilience patterns

| Pattern | Where | Notes |
|---|---|---|
| **Timeouts everywhere** | `context.WithTimeout` per venue refresh; `http.Client.Timeout` per request | A hung venue cannot hold up the system |
| **Bounded retries with exponential backoff** | `fetch.Client` | Retries only network errors, 429 and 5xx. Honours `Retry-After` (Kalshi doesn't send one; Polymarket throttles rather than rejecting). Never retries malformed JSON or 4xx |
| **Client-side rate limiting** | `fetch.Client.Interval` | Kalshi at 10 req/s (its anonymous budget behaves like the Basic tier of about 20 req/s). Gamma at 20 req/s (documented 300 per 10 s) |
| **Bulkheads** | One goroutine and one deadline per venue | Kalshi failing cannot fail Polymarket |
| **Last-known-good + health** | `ingest.Store` | A failed refresh keeps old data and marks the venue unhealthy, so the router excludes it with the reason |
| **Staleness bounds** | `route.Policy.MaxBookAge` | Old books are excluded rather than trusted |
| **Validate at the boundary** | Adapters parse each record defensively (times as strings, list fields in either encoding); `Book.Normalize` / `Validate` | One malformed record costs that record, not the page |

### 3.6 Batching and scoping

- **Books are fetched in batches:** Kalshi `GET /markets/orderbooks` takes 100 tickers per call and
  Polymarket `POST /books` takes 500 tokens. Books for all 646 matched markets take about 4 HTTP calls.
- **Books are fetched only for matched markets.** There is no reason to price a market that has no
  equivalent elsewhere.
- **Ingestion is scoped.** Polymarket lists about 257k open order-book markets (verified), so Equinox
  takes the 3,000 most-traded by 24-hour volume. Liquidity is where routing matters.

## 4. The matcher: record linkage under a precision constraint

Matching markets across venues is an **entity-resolution (record-linkage)** problem. Equinox uses the
classic pipeline:

| Stage | Technique | Why |
|---|---|---|
| Normalize | Lowercase, fold accents, venue-specific phrasing → canonical tokens ("Pro Baseball Championship" → "mlb championship", "25bps" → "25 bp"), a light stemmer, team-alias resolution (Kalshi writes cities, Polymarket nicknames) | The two venues name the same thing differently by policy. Kalshi avoids league trademarks |
| Block | Inverted index on *rare* tokens only (document frequency ≤ 5%) | Turns a 56,776 × 3,396 ≈ 193M comparison space into about 98k scored candidates |
| Score | TF-IDF cosine, with outcome-label tokens weighted double | A standard, explainable similarity; it outperforms edit distance on names (Cohen et al. 2003) |
| **Veto** | 20 kinds of structural contradiction: thresholds, comparator shape, dates, oracles, teams, scopes, ranks, modifiers… | Text similarity is blind to "above 2.4%" vs "exactly 2.4%". This is where precision comes from |
| Assign | Greedy one-to-one per venue pair with total-order tie-breaks | Stops one Polymarket bucket matching three Kalshi strikes |
| Tier | `equivalent` (routable) vs `review` (not) | A confidence boundary the router respects |
| Review | Reviewed mapping table (`reviews/pairs.json`) | Humans (or an offline LLM audit) confirm or block pairs. With `-require-review`, only confirmed pairs route |

**Why deterministic rather than embeddings or an LLM.** Per the prior-art review:
- Embedding retrieval has high recall but cannot tell 84,000 from 85,000, and needs a verifier.
- LLM verification works (Gebele & Matthes 2026 report <2% false positives after two LLM passes) but is
  non-deterministic, costs money per pair, and sends data to a third party.

A deterministic matcher can be audited line by line. Every rejection names its veto, and every match
lists its evidence. **LLMs are used offline only, to *evaluate* the matcher**; they are never in the
runtime path (see [`AI_USAGE_LOG.md`](../AI_USAGE_LOG.md)).

**The measured tradeoff.**
- On the hand-labelled set the matcher scores precision 1.000 and recall 0.844.
- On live data, independent audits measured **0.87 precision out of sample** for the `equivalent` tier.
  The remaining errors are mostly differences that live in the rules prose (announce vs complete,
  first round vs runoff, different deadline windows).

That is the central finding of this feasibility study: **deterministic normalization gets you
candidates and a high-precision core, but the long tail needs rules-level adjudication.** The
architecture accommodates that with the review tier and the reviewed mapping table, rather than
pretending the matcher is perfect.

## 5. The router: a smart order router in miniature

### 5.1 Concepts borrowed from equities

| Equities concept | Here |
|---|---|
| Consolidated book / NBBO | There is no consolidated tape and no Reg NMS linkage for prediction markets, so `split` mode builds a *virtual consolidated book* from every eligible venue's levels |
| Best execution, "total consideration" (FINRA 5310, MiFID II Art. 27) | Rank by **all-in cost**: price plus fee, where fees depend on price (`rate × p(1−p)` on both venues), not by headline price |
| Access fees | `FeeCurve` per market, read from venue data |
| Order protection / sweep | Greedy fill across the consolidated book in fee-inclusive price order |
| Best-ex review | Every decision is explained and logged with the evidence of the match it relied on |

### 5.2 Fees as data (strategy without polymorphism)

Both venues' taker fees fit one formula, `fee = q × rate × p^a × (1−p)^b`, with venue-specific
`(rate, a, b, rounding)`:
- **Kalshi:** `rate = 0.07 × multiplier`, `a = b = 1`, cent rounding.
- **Polymarket:** `rate = feeSchedule.rate`, `a = b = feeSchedule.exponent`, rounding to $0.00001.

So the router needs no fee strategy interface and no venue switch. It evaluates a curve. A venue whose
schedule can't be expressed this way (Kalshi `flat`) is marked untradable by its adapter.

### 5.3 Single venue vs split

- **Best single venue:** most contracts filled first (a partial fill is worse than a complete one),
  then lowest all-in cost per contract, then venue and market id.
- **Split:** greedy over the consolidated book. Levels are ranked by "price + fee" for a reference size
  of one million contracts, which ranks levels exactly enough even though fees round per order.
  - A venue whose share falls below its minimum order size (Polymarket: 5 shares) is dropped and the
    fill recomputed.
  - The split is used only if it fills more, or fills the same for strictly less money; one child order
    is simpler to execute.

### 5.4 Known simplifications

All of these are listed in the assumptions register.
- Taker BUY only.
- Displayed size is assumed fully fillable.
- No market impact, queue position or latency model.
- Per-order fee rounding is applied after allocation, so split mode can over-estimate fees by under 1¢
  per venue.
- Kalshi's structured strike fields (`strike_type`, `floor_strike`) are not used by the matcher, which
  reads `yes_sub_title` text instead. That keeps the matcher venue-agnostic, at the cost of re-deriving
  comparators from text.

## 6. Observability and auditability

- **Structured logs** (`log/slog`): per-venue refresh duration, kept/seen/skipped counts with reasons,
  book error counts, matching statistics with veto counts by kind.
- **Decision log** (`decisions.jsonl`): each line holds the full decision (order, policy, per-venue
  evaluations, allocations, explanation) plus the match ID, tier, score, evidence and caveats it relied
  on. The **decision ID is a SHA-256 of the canonical inputs**, so identical inputs produce identical IDs.
- **`/healthz`**: venue health, last success, stats. **`/near-misses`**: pairs that looked alike but were
  vetoed, the fastest way to see what the matcher refuses and why.

Production would add metrics (OpenTelemetry/Prometheus) and alerts on venue health and staleness.

## 7. Testing strategy

| Layer | Approach |
|---|---|
| Money and fees | Each venue's published worked examples reproduced exactly (Kalshi $1.75 at 100 @ 0.50 and the $0.00363825 rounding example; the Polymarket fee table) |
| Adapters | `httptest` servers serving payloads trimmed from real captured responses; paging, skips, empty-quote sentinels, fee overrides, book derivation, malformed JSON, 5xx |
| Fetch | Retry then success, no retry on permanent errors, deadline enforcement, rate-limit spacing, record → replay with no network |
| Ingest | A failing venue keeps last-known-good; a slow venue is cut off while readers stay non-blocking; invalid books are rejected |
| Router | Table tests for every exclusion, fee-flipped decisions, full-fill preference, split and minimum size, limits, NO side, tie-breaks; **permutation-and-repetition determinism**; **architecture fitness test** |
| Matcher | Feature-extraction unit tests; regression tests for every live false positive found; **evaluation on 55 hand-labelled live pairs against the full 60k-market corpus** (precision gate: zero false positives) |
| End to end | Full pipeline on the committed live recording: ingest → match → books → route, with determinism checked on real books |
| Live audits | Two independent audits of live matches (stratified samples, two judges per disputed pair), in-sample and out-of-sample. See [`TEST_RESULTS.md`](TEST_RESULTS.md) |

Two real defects were caught this way and are worth knowing about:
1. **Non-deterministic matching.** TF-IDF norms were summed in Go's randomized map order, and near-tied
   scores flipped between runs. Fixed by summing in sorted key order; three consecutive runs are now
   byte-identical.
2. **A regex that swallowed adjacent numbers.** `"1 (25 bps)"` parsed as `{1}` instead of `{1, 25}`.
   It was found while investigating an audited false positive.

## 8. Deployment model

There is one static binary in a distroless image. `serve` runs ingestion loops in the background and
answers from memory. On **Cloud Run**:
- **CPU must be always allocated** (`--no-cpu-throttling`). Otherwise background goroutines starve
  between requests and books go stale.
- **`--min-instances=1`**, so state survives without traffic.
- **`--max-instances=1`**, because each instance holds its own snapshot and decisions could otherwise
  differ between instances.

Moving the reviewed mapping table and decision log to durable storage (Cloud Storage / Firestore) is the
first production change. See [`DEPLOYMENT.md`](DEPLOYMENT.md).

## 9. Decision record (short form)

The full journal, with pros and cons for every decision, is kept locally in `DECISION_JOURNAL.md`.

| Decision | Alternatives | Why | Cost |
|---|---|---|---|
| Go, stdlib only | Python; Go + libraries | Partner stack, determinism, zero dependencies | More hand-written text processing |
| Binary proposition as the canonical unit | Event-level model; venue-native shapes | Both venues are binary underneath; removes polarity logic | A Polymarket named-outcome market becomes two markets |
| Micro-dollar `int64` | `float64`; `big.Rat` everywhere | Exact and platform-independent; fast | Explicit parsing; overflow bounds (`MaxOrderQty` = 1M) |
| Immutable snapshots, atomic swap | RWMutex; channels; external store | Lock-free reads, consistent views, no I/O on the routing path | O(n) copy per refresh |
| Deterministic matcher + vetoes | Embeddings; LLM; manual table only | Auditable, free, reproducible; precision by construction | Recall ceiling; vocabularies to maintain |
| Review tier + reviewed mapping table | Trust the matcher | Measured precision is 0.87, not 1.0 | A human or offline process in the loop |
| Router as a pure function | Router with access to clients/state | Deterministic and testable; cannot block | The caller must assemble quotes |
| Fees as a curve (data) | Fee interface per venue; venue switch | No venue knowledge in the router | Only fee shapes that fit the formula |
| Batch book fetch for matched markets only | Per-market GETs; all markets | 4 calls instead of about 650; rate-limit friendly | Replay recordings depend on the batch composition |
| Record/replay transport | Mocks; live-only tests | Real data, offline, deterministic tests and demo | About 11 MB of gzipped fixtures in the repo |
