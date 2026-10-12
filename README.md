# Equinox — can we tell when two prediction markets are the same bet, and route between them?

A feasibility prototype for **PEAK6 Labs**. It reads live markets from **Kalshi** (a US-regulated exchange) and
**Polymarket** (polymarket.com, offshore), puts them into one venue-neutral form, proposes pairs that would **pay the
same side in every outcome**, and simulates routing a hypothetical order between them, explaining every decision.
It places no orders and needs no accounts or keys.

> This is v2. v1 was built in one unattended "one-shot" AI session (kept in `main`'s history at `9c3bf48`). v2 was
> rebuilt through a gated, human-in-the-loop workflow: every decision below was made by a person at a gate and is
> recorded in [`docs/decisions/DECISION_JOURNAL.md`](docs/decisions/DECISION_JOURNAL.md).

## Run it locally

Needs Go 1.27 (`brew install go`). No other dependencies: the standard library only.

```bash
go build -o bin/equinox ./cmd/equinox
./bin/equinox serve                 # page + API on http://localhost:8080, using recorded real venue responses
./bin/equinox serve -live           # the same, reading both venues live (about 4 minutes for the first refresh)
```

Open http://localhost:8080: venue health, equivalent pairs with their evidence, the closest rejected look-alikes,
and a form to route a hypothetical order.

From the command line:

```bash
./bin/equinox refresh -top 10                                   # counts, equivalent pairs, near misses
./bin/equinox route -pair <pair_id> -side yes -qty 100 -region US     # one decision, logged to data/decisions.jsonl
./bin/equinox decisions -n 5                                    # read back the decision log
make check                                                      # gofmt, go vet, every test, diagrams in sync
```

- **Data source:** `-replay testdata/live-20261011` (default: a full live read of both venues on 2026-10-11, offline and deterministic; the small `testdata/recordings` is for tests), `-live`, or `-record DIR`
  (live, saving every response for later replay).
- **Live calls go through the system `curl`.** The development Mac's firewall blocks newly built binaries but
  allows curl. Elsewhere set `EQUINOX_NATIVE_HTTP=1` to use Go's own HTTP client.
- **API:** `GET /pairs`, `POST /route`, `GET /decisions`, `GET /health`, `GET /version`. The contract is
  [`docs/architecture/contracts/openapi.yaml`](docs/architecture/contracts/openapi.yaml).

Example:

```bash
curl -s localhost:8080/pairs | head -40
curl -s -X POST localhost:8080/route -H 'Content-Type: application/json' \
  -d '{"pair_id":"<pair_id>","side":"yes","quantity":100,"customer_region":"US"}'
```

## Demo deploy (Railway)

Full steps (Docker, Railway, live data, how to check a deploy and roll back): [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md).

Live: **https://equinox-production-92b0.up.railway.app** — a hosted demo for clicking around. The live app makes **no venue calls**: it replays the recorded
2026-10-11 responses in `testdata/live-20261011` (plain gzipped JSON files) exactly like the local default, so its
numbers match this README and `docs/TEST_RESULTS.md`. The decision log lives in the container and resets on redeploy.

```bash
railway init -n equinox            # once: create the project
railway up --detach                # build the Dockerfile and deploy (railway.json sets the /version health check)
railway variables --set EQUINOX_VERSION=$(git rev-parse --short HEAD)
railway domain                     # print or create the public URL
```

## How it works

```
venues (Kalshi, Polymarket) ──► adapters ──► one venue-neutral market per yes/no proposition
                                                │
              background refresher (every 15 min; books every 30 s; ≤ 2 requests/s per venue)
                                                │
                       matcher: rare-word candidates → compare settlement parts → one-to-one
                                                │
                         immutable snapshot (swapped atomically) ◄── routing reads only this
                                                │
         POST /route ──► router (pure function, venue-neutral facts only) ──► decision log (synced first)
```

- **"Equivalent" means the same payout in every outcome** the rules name, edge cases included. Only payout timing
  may differ, shown as a caveat. Anything else (a different price source, a carve-out on one side, a different
  threshold or window) is *related, not routable* with the differing part named (D7). A part the parser cannot read
  makes the pair *undecided*: never guessed, sent to an offline AI review instead (D10, D16, D20).
- **Routing never contacts a venue** and gives the same decision for the same order and snapshot (INV-1, INV-2).
  Money is integer micro-dollars; fees are per-market data. The decision names the rule that decided it (most
  filled, lowest all-in cost after fees, tie-break) and why every other venue lost or was left out.
- **Customer eligibility is data:** a US customer's order never goes to polymarket.com (D8).
- **Failure is visible:** a venue that is down keeps its last good data, is marked unhealthy, and ages out of
  decisions after 90 seconds.

### In pictures

Every diagram is a picture in the repo (click to open full size); the short tour is [`ARCHITECTURE.md`](ARCHITECTURE.md).

[![Containers and connections](docs/architecture/diagrams/png/hld-container.png)](docs/architecture/diagrams/png/hld-container.png)

[![Route an order](docs/architecture/diagrams/png/sequence-route-order.png)](docs/architecture/diagrams/png/sequence-route-order.png)

| Diagram | What it shows |
|---|---|
| [System context](docs/architecture/diagrams/png/hld-context.png) | who and what Equinox talks to |
| [Containers](docs/architecture/diagrams/png/hld-container.png) | what runs where, and every connection |
| [Deployment](docs/architecture/diagrams/png/deployment.png) | local, CI and Railway |
| [Refresh and match](docs/architecture/diagrams/png/sequence-refresh-and-match.png) | how pairs are found and checked |
| [Route an order](docs/architecture/diagrams/png/sequence-route-order.png) | how one order is decided and logged |
| [Venue down](docs/architecture/diagrams/png/sequence-venue-down.png) | graceful failure |
| [Measure precision](docs/architecture/diagrams/png/sequence-measure-precision.png) | how matching quality is measured |
| [Adjudicate offline](docs/architecture/diagrams/png/sequence-adjudicate-offline.png) | where AI reviews pairs, outside routing |
| [Pair verdict states](docs/architecture/diagrams/png/state-pair.png) | how a pair becomes equivalent, related or stale |
| [Venue health states](docs/architecture/diagrams/png/state-venue-health.png) | healthy, unhealthy, backing off |
| [Data model](docs/architecture/diagrams/png/erd.png) | every record and how they link |
| [Backend components](docs/architecture/diagrams/png/be-class.png) | Go packages and their dependencies |
| [Page components](docs/architecture/diagrams/png/fe-components.png) | the web page's parts |

## Documentation

| What | Where |
|---|---|
| Decode of the prompt (what is really being tested) | [`specs/M1-cross-venue-equivalence-routing/decode.md`](specs/M1-cross-venue-equivalence-routing/decode.md) |
| Brief | [`specs/M1-cross-venue-equivalence-routing/brief.md`](specs/M1-cross-venue-equivalence-routing/brief.md) |
| Research (8 questions, cited) | [`docs/research/`](docs/research/) |
| Product requirements | [`docs/product/prd.md`](docs/product/prd.md) |
| Architecture overview, in pictures | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| High-level design (HLD) | [`docs/architecture/hld.md`](docs/architecture/hld.md) |
| Equivalence and routing logic (LLD) | [`docs/architecture/lld.md`](docs/architecture/lld.md) — §1, §5 (SEQ-1, SEQ-2), §6, decisions D19-D20 |
| Decisions (every gate, with the options that lost) | [`docs/decisions/DECISION_JOURNAL.md`](docs/decisions/DECISION_JOURNAL.md), [`REVIEW.md`](docs/decisions/REVIEW.md) |
| AI usage log | [`docs/AI_USAGE.md`](docs/AI_USAGE.md) |
| Test results | [`docs/TEST_RESULTS.md`](docs/TEST_RESULTS.md) |
| Deployment guide | [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) |

## Deliverables

| Required | Where |
|---|---|
| Source code | `cmd/`, `internal/` (Go standard library only) |
| Setup instructions | this README, "Run it locally" |
| Technical documentation, architecture overview | `ARCHITECTURE.md` (short, with diagrams), `docs/architecture/hld.md`, `docs/architecture/lld.md` |
| Equivalence and routing logic, written | `docs/product/prd.md` (definition), `docs/architecture/lld.md` (method) |
| Test results | `docs/TEST_RESULTS.md` |
| AI usage log / disclosure | `docs/AI_USAGE.md` |
| Deployment guide | [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md): local, Docker, Railway, live data, checks, rollback |
| Demo video | linked from `docs/demo/` |

## Limits

A prototype: no real orders, wallets, accounts or regulatory checks (out of scope). The US-regulated venue's fee is
assumed at 0.07 × its series multiplier, rounded up to the cent, until its fee schedule is read by hand. Matching
covers markets that traded in the last 24 hours and last at least an hour (D11).

Repositories: GitLab [`labs.gauntletai.com/sefathchowdhury/equinox`](https://labs.gauntletai.com/sefathchowdhury/equinox)
(primary) · GitHub mirror [`github.com/hisefath/equinox`](https://github.com/hisefath/equinox).
