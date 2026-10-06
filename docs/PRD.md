# Project Equinox — Product Requirements Document

| | |
|---|---|
| **Partner** | PEAK6 Labs (Non-Traditional Assets Platform, Apex Fintech Solutions) |
| **Technical contact** | Bryce Harris |
| **Author** | Sefath Chowdhury |
| **Status** | v1.0, implemented (see [`README.md`](../README.md)) |
| **Type** | Infrastructure feasibility prototype, not a trading product |

## 1. Problem

Prediction markets list the same real-world event on several venues, but each venue names,
structures, prices and resolves its contracts differently. Kalshi might list *"Will the Federal Reserve cut
rates by 25bps at their October 2026 meeting?"* as one leg of a multi-outcome event priced in dollars with
sub-penny ticks. Polymarket might list *"Fed decreases interest rates by 25 bps after October 2026
meeting?"* as a separate binary market with its own token IDs and a decimal order book. No shared layer
says that these two contracts are the same bet, so nothing can compare their prices on a like-for-like
basis or route an order between them.

PEAK6 Labs is exploring non-traditional-asset infrastructure (prediction markets, crypto, tokenization) for
Apex. Before investing further, the business wants evidence on three questions:

1. **Is cross-venue normalization realistically achievable?**
2. **What architectural tradeoffs does it involve?**
3. **How complex does smart routing become in practice?**

| | Today | Desired |
|---|---|---|
| Integration | Venue-specific integrations, duplicated effort | One normalization layer reused by every venue |
| Comparison | Impossible without manual mapping | Equivalent markets detected programmatically, with evidence |
| Routing | None | A venue-agnostic engine that explains every decision |

## 2. Goals and non-goals

### Goals
- **G1** Ingest live market metadata and pricing (order books) from **two** public venues: **Kalshi** and
  **Polymarket**.
- **G2** Define a **canonical, venue-independent market model** that every adapter maps into.
- **G3** **Detect equivalent markets** across venues, with a written definition of "equivalent", a justified
  methodology, a confidence tier, and human-readable evidence for every match.
- **G4** **Simulate routing** of a hypothetical order across the venues that list an equivalent market, and
  **log the reasoning** behind each decision.
- **G5** Produce a written assessment of feasibility, tradeoffs and routing complexity, which is the actual
  product of this project.

### Non-goals (explicitly out of scope per the brief)
Real-money trading, order placement, wallets and custody, authentication against venues, regulatory
implementation, production UI, and execution-quality optimisation beyond a reasoned simulation.

## 3. Users and use cases

| Persona | Need | Use case |
|---|---|---|
| **PEAK6 Labs engineer / architect** (primary) | Decide whether to invest in an NTAP aggregation layer | Run the prototype, read the matches and routing decisions, and read the tradeoff docs |
| **Trader / product owner** | Know where the same bet is cheaper | "Buy 500 YES on *Fed cuts 25bp in October*": which venue, at what all-in price, and why? |
| **Integration engineer** | Add a third venue cheaply | Implement one adapter interface; matching and routing need no changes |
| **Risk / compliance reviewer** | Trust that "equivalent" means equivalent | Audit the evidence and conflicts behind each match; see why near-matches were rejected |

## 4. Functional requirements

| ID | Requirement | Source | Acceptance criterion |
|---|---|---|---|
| FR1 | Fetch live market data from two venues | PRD | `equinox scan` (live by default; `-replay DIR` runs offline) pulls open markets **and** order books from Kalshi and Polymarket public APIs |
| FR2 | Define an internal market representation | PRD | `internal/market` types; no venue schema crosses the adapter boundary |
| FR3 | Identify potentially equivalent markets | PRD | `internal/match` emits pairs with tier, score, evidence and conflicts; precision and recall measured on a hand-labelled set |
| FR4 | Simulate a routing decision between venues | PRD | `internal/route` returns a decision (venue/allocation, all-in cost) for a hypothetical order |
| FR5 | Log reasoning for routing decisions | PRD | Every decision is written as a structured JSON record with per-venue evaluation, exclusions and explanation |
| FR6 | Define "equivalent" and justify the method | PDF | [`docs/EQUIVALENCE.md`](EQUIVALENCE.md) |
| FR7 | Explain why a venue was selected | PDF | Decision record's `explanation` plus [`docs/ROUTING.md`](ROUTING.md) |
| FR8 | Document assumptions where data is incomplete or ambiguous | PDF | Assumptions register in [`docs/SYSTEM_DESIGN.md`](SYSTEM_DESIGN.md) §3 |

## 5. Non-functional requirements

| ID | Requirement | Design response |
|---|---|---|
| NFR1 | **Routing is deterministic** | Pure function of `(order, snapshot, policy, now)`; integer fixed-point money (no floats); canonical input ordering; total-order tie-breaks; decision ID = hash of inputs. Tested by permutation and repetition. |
| NFR2 | **Routing never blocks on external calls** | Ingestion runs separately and writes immutable snapshots; the router reads only in-memory snapshots and does no I/O. |
| NFR3 | **Graceful handling of API failure and inconsistent data** | Per-venue timeouts, bounded retries with backoff, last-known-good snapshots, venue health and staleness tracking, validation that drops or flags bad records and counts them instead of crashing. |
| NFR4 | **No hardcoded venue logic in the routing layer** | `route` imports only `market`. Fees reach it as data (a fee curve), never as `if venue == …`. An architecture test enforces the import rule. |
| NFR5 | Clear separation of concerns | Packages: `venues/*` (integration), `market` (canonical model), `match` (equivalence), `route` (decisions), `ingest` (I/O orchestration). |
| NFR6 | Reasonable test coverage | Unit, fixture-replay (real recorded API data), failure-injection, determinism and architecture tests; results in [`docs/TEST_RESULTS.md`](TEST_RESULTS.md). |
| NFR7 | Readable, documented code | Go stdlib only, small packages, documented assumptions. |
| NFR8 | Runs locally; GCP optional | Single static binary and Docker image; Cloud Run guide in [`docs/DEPLOYMENT.md`](DEPLOYMENT.md). |

## 6. Success criteria

1. **Feasibility shown on live data:** the prototype finds real equivalent markets listed on both venues today.
2. **Precision over recall:** on the hand-labelled pair set, auto-routable matches have **precision ≥ 0.95**.
   A false match routes money into a different bet; a missed match only loses an opportunity.
   *Result:* met on the labelled set (1.000). On audited live matches the out-of-sample figure is 0.879, so
   production routing runs on the reviewed mapping table (`-require-review`). See
   [`EQUIVALENCE.md`](EQUIVALENCE.md) §3.
3. **Determinism:** identical inputs give byte-identical decision records, whatever the input order.
4. **Resilience:** with one venue down, slow or returning garbage, the system still matches and routes on the
   other data and says why the venue was excluded.
5. **Extensibility:** adding a venue means one adapter package (`internal/venues/<new>`) plus registering it
   in `setup()`. Matching compares every pair of venues with no changes. Routing works per matched pair;
   routing across three or more venues at once would also need pairs clustered into groups (not built).

## 7. Deliverables (hard requirements)

| Deliverable | Location |
|---|---|
| Source code | this repository |
| Technical documentation | `README.md`, `docs/` (PRD, system design, architecture, equivalence, routing) |
| Demo video | [`docs/demo/equinox-demo.mp4`](demo/equinox-demo.mp4) (terminal recording); script for a narrated version: [`docs/DEMO_SCRIPT.md`](DEMO_SCRIPT.md) |
| Test results | [`docs/TEST_RESULTS.md`](TEST_RESULTS.md) and raw output in `test-results/` |
| AI usage log | [`AI_USAGE_LOG.md`](../AI_USAGE_LOG.md) |
| Deployment guide | [`docs/DEPLOYMENT.md`](DEPLOYMENT.md) |

## 8. Risks and open questions

| Risk | Mitigation |
|---|---|
| Resolution rules differ subtly between "equivalent-looking" markets | Strict definition; conflicts and vetoes; a `review` tier that is not auto-routable; human override table |
| Venue APIs change shape (both have during 2025–26, e.g. Kalshi moved to `*_dollars` / `*_fp` fields) | Adapters isolate the schemas; fixture tests break loudly; unknown fields ignored |
| Rate limits during full crawls | Per-venue rate limiter, books fetched only for matched markets |
| Fee schedules change | Fees are read from venue data (Kalshi series `fee_type`/`fee_multiplier`, Polymarket `feeSchedule`) where available |
| Text matching does not generalise across categories | Precision and recall measured per topic on the labelled set (TEST_RESULTS §3) and audited on live data. The LLM/human adjudication path is documented as the next step |
