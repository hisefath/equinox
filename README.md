# Equinox: cross-venue prediction market equivalence and routing

A Go prototype for **PEAK6 Labs** that answers three feasibility questions:

1. Can markets from different prediction venues be normalized into one model?
2. Can we detect, programmatically, that two venues list **the same bet**?
3. Can a **venue-agnostic router** choose between them deterministically and explain why?

It ingests **Kalshi** and **Polymarket** live, normalizes both into a canonical binary-market model,
proposes equivalent markets with evidence, and simulates fee-aware routing of hypothetical orders. Every
decision is logged with its reasoning.

It is an infrastructure prototype, not a trading product: it places no orders and needs no credentials.

## Results at a glance

| | |
|---|---|
| Live data | 56,776 Kalshi + 3,000 Polymarket markets (→ 3,390 binary propositions) per refresh |
| Matching | **374** proposed pairs: about 1.22M candidate pairs scored, 90,726 vetted, in about 7 s |
| Precision on 55 hand-labelled live pairs | **1.000** (0 of 23 hard negatives accepted); recall **0.844** |
| Precision on live matches, independently audited **out of sample** | **0.879** (124/141; 95% CI 0.82–0.92). The `review` tier is far lower, so it is not routed |
| Routing | Deterministic (byte-identical decisions, input-order independent); fee-inclusive; names the rule that chose the venue and why every other venue was excluded |
| Tests | 46 tests pass with the race detector; 79.9% coverage (match 98%, route 98%) |

**Finding:** normalization and routing are very achievable. Equivalence is the hard part. A
deterministic matcher reaches high precision on templated markets (elections, Fed decisions, sports
winners). Across the live long tail the remaining errors are in the rules prose (announce vs complete,
first round vs runoff), so production routing should run on a **reviewed mapping table**. Equinox
supports that (`-require-review`). Details: [`docs/EQUIVALENCE.md`](docs/EQUIVALENCE.md).

## Quick start (offline, about 1 minute)

Requires Go 1.27+. No network or keys needed: the repo includes a live recording from 2026-10-06.

```bash
go build -o bin/equinox ./cmd/equinox
./bin/equinox scan  -replay testdata/snapshot -near-misses 8
./bin/equinox route -replay testdata/snapshot -pair 26 -side yes -qty 2000 -split
go test ./...
```

Live, against the real venues (about 20 s per refresh):

```bash
./bin/equinox scan                                 # or: docker build -t equinox . && docker run --rm equinox scan
./bin/equinox serve -addr :8080                    # GET /pairs, /route?pair=1&side=yes&qty=100, /healthz
```

Example output, a real decision on recorded live books:

```
Pair kalshi:SENATEIA-26-R~polymarket:630734 (equivalent, score 1.00, reviewed by llm-audit ...)
  evidence: office agrees: senate · outcomes agree: ashley, hinson · predicate agrees: win
  caveat:   venues expect resolution 60 days apart (settlement timing / capital lock-up differs)

Decision 5c7e7694477187a2: FILLED
  - kalshi/SENATEIA-26-R: eligible; alone fills 2000 at all-in 0.58716/contract (fees 34.32)
  - polymarket/630734: eligible; alone fills 2000 at all-in 0.589744/contract (fees 19.488)
  - split considered; no improvement over best single venue, keeping one child order
  - route 2000 to kalshi/SENATEIA-26-R: notional 1140.00 + fees 34.32 = 1174.32 (all-in 0.58716/contract)
  - why kalshi/SENATEIA-26-R: lowest all-in cost including fees: 0.58716 vs 0.589744 per contract, saving 5.168 on 2000 contracts
```

## How it works

```
venues (Kalshi, Polymarket)
   └─ adapters ── canonical markets + books ──► ingest.Store (immutable snapshots, atomic swap)
                                                   ├─► match: normalize → block → score → veto → 1:1 → tier → reviews
                                                   └─► route: pure function (order, quotes, policy, now) → explained decision
```

- **Canonical model** (`internal/market`): one binary proposition per market. Money is in integer
  micro-dollars, and fees are a curve (`q × rate × p^a(1−p)^b`) that fits both venues' published
  schedules.
- **Matching** (`internal/match`): TF-IDF candidates over normalized text, then 18 veto rules (thresholds,
  comparator shape, dates, oracles, teams, ranks, modifiers, and 9 groups of mutually exclusive scopes
  such as office and predicate) and one-to-one assignment. Every pair carries evidence; the
  highest-scoring rejections are reported as near misses.
- **Routing** (`internal/route`): eligibility (health, tradability, valid and fresh book, limit, minimum
  size), a fill simulation per venue, best single venue by all-in cost, and optional consolidated-book
  splitting. Every decision states the rule that won it. It imports only `internal/market`, and a test
  enforces that.
- **Ingestion** (`internal/ingest`, `internal/fetch`): venues run concurrently with deadlines, bounded
  retries, rate limiting and last-known-good data. Routing reads a snapshot and never waits on a venue.

## Documentation

| Document | Contents |
|---|---|
| [`docs/PRD.md`](docs/PRD.md) | Product requirements, goals, success criteria |
| [`docs/SYSTEM_DESIGN.md`](docs/SYSTEM_DESIGN.md) | The plan, why, the **assumptions register**, Mermaid diagrams, failure handling |
| [`docs/diagrams/`](docs/diagrams/) | Standalone Mermaid sources (`.mmd`) and rendered `.svg` files |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | System-design concepts, tradeoffs and decisions |
| [`docs/EQUIVALENCE.md`](docs/EQUIVALENCE.md) | Definition of "equivalent", methodology, vetoes, evaluation, limits |
| [`docs/ROUTING.md`](docs/ROUTING.md) | Routing logic, fees, determinism, explanations, decision log |
| [`docs/TEST_RESULTS.md`](docs/TEST_RESULTS.md) | Test inventory, coverage, labelled evaluation, live audits |
| [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) | Local, Docker, Google Cloud Run; flags; API; troubleshooting |
| [`docs/DEMO_SCRIPT.md`](docs/DEMO_SCRIPT.md) | The demo video walkthrough |
| [`AI_USAGE_LOG.md`](AI_USAGE_LOG.md) | How AI was used to build this (and why none runs inside it) |
| [`research/`](research/) | API research (Kalshi, Polymarket), prior art, live overlap census, labelled pairs, audits |

## Deliverables

| Required | Where |
|---|---|
| Source code | `cmd/`, `internal/` (Go, standard library only) |
| Technical documentation | `docs/` |
| Demo video | [`docs/demo/equinox-demo.mp4`](docs/demo/equinox-demo.mp4): a terminal recording of the offline demo, about 3 min, no narration. Script for a narrated version: [`docs/DEMO_SCRIPT.md`](docs/DEMO_SCRIPT.md) |
| Test results | [`docs/TEST_RESULTS.md`](docs/TEST_RESULTS.md), raw output in [`test-results/`](test-results/) |
| AI usage log | [`AI_USAGE_LOG.md`](AI_USAGE_LOG.md) |
| Deployment guide | [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) |

## Repository layout

```
cmd/equinox/            CLI (scan, route) and HTTP server (serve)
internal/market/        canonical model: Market, Book, Amount (µ$), FeeCurve
internal/fetch/         HTTP client: timeouts, rate limit, retries, record/replay
internal/ingest/        concurrent ingestion, immutable snapshots, venue health
internal/venues/kalshi/ Kalshi adapter
internal/venues/polymarket/  Polymarket adapter
internal/match/         equivalence detection, evaluation fixtures, reviewed mapping table
internal/route/         deterministic routing simulation
reviews/pairs.json      reviewed mapping table (297 audited verdicts)
testdata/snapshot/      gzipped live API recording for offline runs and tests
research/               research reports, labelled pairs, audit labels
```

## Status and limitations

This is a prototype. It does not do: real-money trading, authentication, SELL orders, market-impact or
fill-probability models, websocket feeds, or persistent storage. Each is noted in the docs, with how it
would be added. See also the assumptions register (`docs/SYSTEM_DESIGN.md` §3) and the known gaps
(`docs/EQUIVALENCE.md` §4).

Repositories: GitLab [`labs.gauntletai.com/sefathchowdhury/equinox`](https://labs.gauntletai.com/sefathchowdhury/equinox)
(primary) · GitHub mirror [`github.com/hisefath/equinox`](https://github.com/hisefath/equinox).
