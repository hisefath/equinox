# Deployment Guide

Equinox is a single static Go binary with no third-party dependencies and no credentials. It runs in
three places:

| Mode | Needs network | Use for |
|---|---|---|
| **Replay** (offline, from the committed recording) | no | demos, tests, reviewing decisions, CI |
| **Live, local** (binary or Docker) | yes | real venue data on a laptop |
| **Live, Google Cloud Run** | yes | a shared, always-on instance |

## 1. Prerequisites

- **Go 1.27+** for building from source: `brew install go`, or <https://go.dev/dl/>. No other toolchain.
- **Docker** (optional), for the container image or for live runs on a machine with an outbound
  firewall (see Troubleshooting).
- **gcloud CLI** (optional), for Cloud Run.
- No API keys. Both venues' market-data endpoints are public. Equinox never places orders.

## 2. Build and test

```bash
git clone https://labs.gauntletai.com/sefathchowdhury/equinox.git   # or github.com/hisefath/equinox
cd equinox
go build -o bin/equinox ./cmd/equinox
go test -race ./...            # about 3 minutes with -race; `go test -short ./...` skips the 60k-market tests
```

## 3. Run offline (replay)

`testdata/snapshot/` is a live recording from 2026-10-06 05:33 UTC: 56,776 Kalshi markets, 3,000
Polymarket markets, and the order books of every matched market. Replay sets the clock to the recording
time, so books are "fresh" and decisions are reproducible.

```bash
./bin/equinox scan  -replay testdata/snapshot                    # venues, matched pairs, near misses
./bin/equinox route -replay testdata/snapshot -pair 26 -side yes -qty 2000 -split
./bin/equinox serve -replay testdata/snapshot -addr :8080        # same data over HTTP
```

A replay can only answer requests that were recorded. Keep `-kalshi-pages` and `-poly-pages` at the
recorded values (30 each). If you change the matcher, re-record, because the book batches depend on
which pairs matched.

## 4. Run live

```bash
./bin/equinox scan                                  # about 20 s: crawl both venues, match, fetch books
./bin/equinox scan -record testdata/snapshot        # same, and save every response (gzipped) for replay
./bin/equinox route -pair 3 -side no -qty 500 -limit 0.70 -split
./bin/equinox serve -addr :8080                     # background refresh + HTTP API
```

### Flags (all commands)

| Flag | Default | Meaning |
|---|---|---|
| `-replay DIR` | | run offline from a recording |
| `-record DIR` | | save raw responses while running live |
| `-kalshi-pages N` | 30 | Kalshi `/events` pages (200 events each, ≈1,900 markets per page) |
| `-poly-pages N` | 30 | Polymarket `/markets/keyset` pages (100 most-traded markets each) |
| `-timeout D` | 90s | deadline for one venue's refresh |
| `-reviews FILE` | `reviews/pairs.json` | reviewed mapping table (confirm or block pairs) |
| `-require-review` | false | route only pairs a reviewer confirmed (**recommended for anything beyond a demo**) |
| `-min-tier T` | equivalent | `review` also lets unconfirmed review-tier pairs route |
| `-v` | false | debug logs |

`route` adds `-pair N|ID`, `-side yes|no`, `-qty N`, `-limit 0.55`, `-split`, `-max-age 30s`, and
`-log data/decisions.jsonl`. `serve` adds `-addr`, `-markets-every 5m`, `-books-every 10s` and `-log`.
`scan` adds `-out FILE.json` and `-near-misses N`.

### HTTP API (`serve`)

| Endpoint | Returns |
|---|---|
| `GET /healthz` | `ready`, pair count, per-venue health (ok, last success, kept/seen/skipped, book errors) |
| `GET /pairs?tier=equivalent` | matched pairs with evidence, caveats, review status |
| `GET /near-misses` | similar-looking pairs that a veto rejected, with the veto |
| `GET /route?pair=N&side=yes&qty=100&limit=0.55&split=true&max_age=30s` | `{pair, decision}`. The decision is also appended to the log. `400` for an invalid side, quantity (1..1,000,000), limit (must be above 0) or max age (must be positive); `404` for an unknown pair; `409` if the pair isn't routable (rejected by review, or unconfirmed with `-require-review`); `503` while warming up. Invalid requests are not logged |

## 5. Docker

```bash
docker build -t equinox .
docker run --rm equinox scan -replay testdata/snapshot      # offline, inside the image
docker run --rm -p 8080:8080 equinox                        # live `serve` on :8080
curl -s localhost:8080/healthz
curl -s "localhost:8080/route?pair=1&side=yes&qty=100&split=true"
```

The image is about 39 MB: distroless `static-debian12:nonroot`, no shell, uid 65532. It contains the
binary, the reviewed mapping table and the replay snapshot. Decisions are written to `/app/data`. Mount a
volume there to keep them: `-v $PWD/data:/app/data`.

## 6. Google Cloud Run

```bash
PROJECT=your-project REGION=us-central1
gcloud config set project $PROJECT
gcloud services enable run.googleapis.com artifactregistry.googleapis.com cloudbuild.googleapis.com

gcloud run deploy equinox \
  --source . \
  --region $REGION \
  --no-allow-unauthenticated \
  --cpu 2 --memory 1Gi \
  --no-cpu-throttling \
  --min-instances 1 --max-instances 1 \
  --timeout 300
```

Why these flags:

- **`--no-cpu-throttling`** ("CPU always allocated"). `serve` refreshes in background goroutines. With
  default request-based CPU, they starve between requests, books go stale, and every route request is
  rejected as stale. This is the most important setting.
- **`--min-instances 1`**. State is in memory. Scaling to zero discards it, and the next request waits
  about 20 s for a full ingest (it gets `503 warming up`).
- **`--max-instances 1`**. Each instance holds its own snapshot. Two instances could return different
  decisions for the same request. Horizontal scale needs a shared snapshot store first (§7).
- **SIGTERM** is handled. When Cloud Run scales an instance down, in-flight requests finish and the
  background loops stop.
- **`--no-allow-unauthenticated`**. There's no reason to expose a routing simulator publicly. Call it with
  `gcloud run services proxy equinox --region $REGION` or an identity token.
- **Memory 1 GiB.** A full scan peaks at about 670 MB resident (measured: 60k markets plus features plus TF-IDF vectors). 1 GiB leaves headroom; 512 MiB would not.
- Logs go to Cloud Logging as structured `slog` lines. Each decision is also logged as a
  `routing decision` entry.
- Cloud Run egress reaches both venues directly. Both rate-limit per IP, and Equinox stays under both
  limits (Kalshi 10 req/s, Gamma 20 req/s).

Cost at these settings is roughly the price of one always-on 2 vCPU / 1 GiB instance. Lower `--cpu` to 1
if refreshes can be slower.

## 7. Going further (not built)

| Need | Change |
|---|---|
| Durable decisions and reviews | Write `decisions.jsonl` to Cloud Storage or BigQuery; keep `reviews/pairs.json` in Firestore with an approval UI |
| Horizontal scale | One ingester instance publishes snapshots (e.g. GCS object + Pub/Sub notification); stateless routers load them |
| Fresher books | Websocket market-data adapters (both venues offer them) behind the same `ingest.Venue` port |
| Scheduled re-matching and audits | Cloud Scheduler → `scan -out` → a reviewer queue |

## 8. Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Live run: every venue fails with `context deadline exceeded`, but `curl` to the same URL works | An outbound firewall (e.g. Little Snitch on macOS) silently drops connections from new, unsigned binaries | Allow `bin/equinox` (or `go`'s temporary binaries) in the firewall, or run live inside Docker (§5), whose traffic leaves through the already-allowed container VM |
| Replay: `HTTP 404: not recorded` | The request differs from what was recorded: different page counts, or a different matcher producing different book batches | Use `-kalshi-pages 30 -poly-pages 30`, or re-record with `-record` |
| `/route` returns `409 ... rejected by review` or `no confirming review` | The reviewed mapping table blocks the pair, or `-require-review` is set | Choose another pair, or review it in `reviews/pairs.json` |
| Every venue excluded as `stale book` | Books older than `-max-age`: on Cloud Run, almost always CPU throttling | `--no-cpu-throttling`, or raise `max_age` for experiments |
| Kalshi `HTTP 429` in logs | The per-IP read budget is shared with other clients on your IP | Retries back off automatically; lower request rates with fewer pages |
| Docker on macOS: `docker-credential-desktop not found` | `~/.docker/config.json` references a credential helper that isn't installed | `DOCKER_CONFIG=$(mktemp -d)` with `{}` in `config.json`, and point `DOCKER_HOST` at your Docker/Colima socket |
