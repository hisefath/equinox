# Deployment guide

How to run Equinox on your own machine, in a container, and on Railway, and how to check that a deploy works.
Equinox needs no accounts, keys or database: it reads public venue data and writes one log file.

Live demo: **https://equinox-production-92b0.up.railway.app**

[![Deployment](architecture/diagrams/png/deployment.png)](architecture/diagrams/png/deployment.png)

## 1. What you are deploying

| Part | What it is |
|---|---|
| One binary | `equinox` (Go 1.27, standard library only), built from `cmd/equinox` |
| Venue data | By default, a recorded full read of both venues from 2026-10-11 (`testdata/live-20261011`, gzipped JSON). Optional: live calls to the venues' public APIs |
| Reviewed data | `refdata/` (customer eligibility per venue, AI verdicts a person approved), read once at start |
| Decision log | `data/decisions.jsonl`, append-only, written before every answer |

The default is the recorded data, so every deploy shows the same pairs and the same routing numbers as
[`TEST_RESULTS.md`](TEST_RESULTS.md). Switch to live data with `-live` (section 5).

## 2. Run it locally (Go)

Needs Go 1.27 (`brew install go`).

```bash
go build -o bin/equinox ./cmd/equinox
./bin/equinox serve
```

Open http://localhost:8080. The first refresh over the recorded data takes a few seconds; until then the page says
it is warming up and `POST /route` answers 503.

Useful flags for `serve`: `-addr :8080` (listen address), `-data DIR` (decision log directory, default `data`),
`-refdata DIR` (default `refdata`), and the data-source flags in section 5.

## 3. Run it in a container (Docker)

The [`Dockerfile`](../Dockerfile) builds a small Alpine image that runs as a non-root user and listens on `$PORT`
(default 8080).

```bash
docker build -t equinox .
docker run --rm -p 8080:8080 equinox
```

The image contains the binary, `refdata/` and the recorded venue data. Its decision log lives at `/app/data`, so
mount a volume there if you want to keep it: `docker run --rm -p 8080:8080 -v equinox-data:/app/data equinox`.

## 4. Deploy to Railway

The repo is ready for Railway: [`railway.json`](../railway.json) tells it to build the `Dockerfile` and to treat
`GET /version` as the health check (up to 120 s), restarting the service if it fails.

First time, with the [Railway CLI](https://docs.railway.com/guides/cli) installed and `railway login` done, from the
repo root:

```bash
railway init -n equinox
railway up --detach
railway domain
```

`railway domain` prints (or creates) the public URL. To show the commit you deployed on `/version`:

```bash
railway variables --service equinox --set EQUINOX_VERSION=$(git rev-parse --short HEAD) --skip-deploys
```

Later deploys, from the commit you want live (usually `main` after a merge):

```bash
git checkout main && git pull
railway variables --service equinox --set EQUINOX_VERSION=$(git rev-parse --short HEAD) --skip-deploys
railway up --service equinox --detach
```

Pass `--service equinox` whenever the CLI says "Multiple services found".

**Keep the decision log across deploys (optional).** Without a volume, the log resets on every redeploy, which is
fine for a demo. To keep it, add a volume to the service in the Railway dashboard with the mount path `/app/data`.

## 5. Recorded or live venue data

| Flag | Data source | Use it for |
|---|---|---|
| (none) / `-replay testdata/live-20261011` | recorded responses, no network | demos, grading, tests: same numbers every run |
| `-live` | the venues' public APIs, paced at 2 requests per second per venue | real current markets (first refresh about 4 minutes) |
| `-record DIR` | live, saving every response under `DIR` | making a new recording to replay later |

To serve live data:

```bash
./bin/equinox serve -live
```

How live calls are made:
- Locally, live calls go through the system `curl` by default. The development Mac's firewall blocks newly built
  binaries but allows `curl`.
- **In a container or on Railway, set `EQUINOX_NATIVE_HTTP=1`**, so Go's own HTTP client is used; the image has no
  `curl`. On Railway, set that variable and change the service's start command to
  `sh -c 'exec ./equinox serve -live -addr :$PORT'`.

To add requests that are missing from a recording without touching what is already there (for example order books
for markets added later), replay with `-fill`: recorded requests are replayed, and only missing ones are fetched live
and saved. This is how the higher-payout pairs' order books were added to `testdata/live-20261011` (D26).

```bash
./bin/equinox refresh -fill
```

To make a fresh recording and use it:

```bash
./bin/equinox refresh -record testdata/live-$(date +%Y%m%d)
./bin/equinox serve -replay testdata/live-$(date +%Y%m%d)
```

## 6. Check that a deploy works

Replace `URL` with `http://localhost:8080` or the Railway address.

```bash
URL=https://equinox-production-92b0.up.railway.app
curl -s $URL/version
curl -s $URL/health
curl -s $URL/pairs | head -20
curl -s -X POST $URL/route -H 'Content-Type: application/json' \
  -d '{"pair_id":"96fd378f2b486591","side":"yes","quantity":500,"customer_region":"non-US"}'
```

On the recorded data, expect:

| Check | Expected |
|---|---|
| `/version` | the commit you set in `EQUINOX_VERSION` |
| `/health` | both venues `healthy`; Kalshi lists 118,696 markets and Polymarket 20,886 |
| `/pairs` | 12 equivalent pairs |
| `/route`, non-US customer | `"venue": "polymarket"`, `"all_in": "11.32274"`, rule `lowest_all_in` |
| `/route`, same order with `"customer_region":"US"` | `"venue": "kalshi"`, `"all_in": "21.35"`, rule `only_eligible` |
| `/decisions` | the orders you just routed, newest first |

Then open the page and route the Messi pair once from the order ticket.

## 7. Roll back

Every Railway deploy is kept. In the Railway dashboard, open the service's **Deployments** tab, pick the last good
deployment and choose **Redeploy**. Or check out the last good commit and run `railway up --service equinox --detach`
from it. Confirm with `/version`.

## 8. Settings reference

| Setting | Where | Default | Effect |
|---|---|---|---|
| `PORT` | environment (Railway sets it) | 8080 | port the container listens on |
| `EQUINOX_VERSION` | environment | the build's git commit, else `dev` | what `/version` and the page footer show |
| `EQUINOX_NATIVE_HTTP=1` | environment | unset (use `curl`) | use Go's HTTP client for live calls; required in containers |
| `-addr` | `serve` flag | `:8080` | listen address |
| `-replay` / `-live` / `-record` | any command | `-replay testdata/live-20261011` | venue data source (section 5) |
| `-fill` | any command, with `-replay` | off | fetch only requests missing from the recording, live, and add them |
| `-goal` | `route` | `best_price` | `best_price` or `higher_payout` (D26) |
| `-data` | any command | `data` | decision log directory |
| `-refdata` | any command | `refdata` | reviewed reference data |
| `-kalshi-pages`, `-poly-pages` | any command | 0 (all) | cap pages read, for quick live tests |

## 9. Good to know

- **No secrets anywhere.** The venues' data is public; there are no API keys, accounts or wallets (out of scope by
  design: PRD NG-1, NG-2).
- **Routing never calls a venue.** It reads the latest snapshot only, so a slow or broken venue never blocks an
  order; a venue that fails is marked unhealthy and its prices age out after 90 seconds.
- **CI.** Every push runs `make check` (format, vet, all tests, diagrams in sync) through `.githooks/pre-push`, and
  GitLab CI runs it too (`.gitlab-ci.yml`).
- **Railway config format.** Railway now prefers its newer infrastructure-as-code format; `railway.json` keeps
  working until 2026-12-01. Run `railway config migrate` before then.
