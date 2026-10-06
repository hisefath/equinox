# Kalshi Trade API v2: public market data for Project Equinox

Checked against the live API on 2026-10-05 US / 2026-10-06T03:52–04:05Z, with no credentials.
Base URL: `https://api.elections.kalshi.com/trade-api/v2`. The OpenAPI spec v3.32.0 lists it as "Production shared API server, also supported". The primary host is `https://external-api.kalshi.com/trade-api/v2`.
Primary sources:
- Spec: https://docs.kalshi.com/openapi.yaml (saved as `raw/kalshi_openapi_2026-10-05.yaml`)
- Docs pages: https://docs.kalshi.com/llms.txt plus the `.md` pages saved in `raw/kalshi_docs/`
- Live responses: saved in `raw/` (file names are given inline below)

Confidence tags: **[live]** = I saw it in a live response. **[doc]** = it is stated in the official docs or spec. **[inferred]** = my conclusion from the above.

---

## 0. TL;DR for the Go implementation

| Concern | What to do |
|---|---|
| List open single-outcome markets | `GET /markets?status=open&mve_filter=exclude&limit=1000&cursor=…`. Keep going until `cursor == ""`. |
| Series metadata (category, tags, fees) | `GET /series` once. It is **not paginated**: all 14,681 series came back in one 2.3 MB response in 0.22 s. Join to markets with `event_ticker` → `GET /events` (`series_ticker`), or split the ticker prefix before the first `-`. |
| Order book | `GET /markets/{ticker}/orderbook[?depth=N]` returns `orderbook_fp.yes_dollars` / `no_dollars`. These are **bids only, sorted ascending, best = last element**. Derive asks as `yes_ask = 1 − best_no_bid` and `no_ask = 1 − best_yes_bid`. |
| Batch books | `GET /markets/orderbooks?tickers=A&tickers=B` takes up to 100 tickers and needs **repeated** params. A comma list gets treated as one bogus ticker and you get back an empty book with HTTP 200. |
| Prices | Strings like `"0.4300"`. Use int64 micro-dollars, because the spec says responses can emit up to 6 dp. Never use float. |
| Sizes | `*_fp` strings with 2 dp, e.g. `"72612.49"`. **Fractional contracts are real.** Use int64 centi-contracts. |
| Legacy integer-cent fields | **Gone** from market and order-book responses. Only the `*_dollars` and `*_fp` fields remain. |
| Fees | Read `fee_type` and `fee_multiplier` from the series. The event's `fee_type_override` / `fee_multiplier_override` wins when present. Taker = `ceil(fee_multiplier × 0.07 × C × P × (1−P))`. Maker = 0, 0.25× or 0.5× of that, depending on `fee_type`. |
| Rate limit (no auth) | Not documented for anonymous use. What I saw matches the Basic tier: 200 read tokens/s, a 600-token bucket, 10 tokens per GET → about 20 req/s sustained with a burst of about 60. Back off on 429. Treat it as shared per IP. |
| Empty-side detection | Use `yes_bid_size_fp == "0.00"` and `yes_ask_size_fp == "0.00"`. **Do not trust the price sentinels**: they are inconsistent (`0.0000` and `1.0000` both show up). |

---

## 1. Endpoints, pagination and filters

All of these are `security: []` in the spec **[doc]** and returned 200 with no auth **[live]**.

| Endpoint | Purpose | Key params | Notes |
|---|---|---|---|
| `GET /markets` | List markets | `limit` (0–1000, default 100), `cursor`, `status` (`unopened` / `open` / `paused` / `closed` / `settled`; one value only), `series_ticker`, `event_ticker` (one only), `tickers` (comma-separated), `mve_filter` (`only` / `exclude`), `min/max_created_ts`, `min/max_close_ts`, `min/max_settled_ts`, `min/max_updated_ts` | `limit=1001` → HTTP 400 `"…'Limit' failed on the 'lte' tag"` **[live]**. Response is `{"cursor": "...", "markets": [...]}`. |
| `GET /markets/{ticker}` | Single market | | 404 `{"error":{"code":"not_found",...}}` for an unknown ticker **[live]** |
| `GET /events` | List events | `limit` (1–**200**, default 200), `cursor`, `status`, `series_ticker`, `tickers`, `with_nested_markets` (bool), `with_milestones`, `min_close_ts`, `min_updated_ts` | **Excludes multivariate events by design** **[doc]**. `status=open` matches an event if *any* child market is open **[doc]**. Page 1 (`status=open&limit=200&with_nested_markets=true`) gave 200 events / 1,782 markets / 310 KB / 0.16 s **[live]** (`raw/events_open_limit200_nested_page1.json`). |
| `GET /events/{event_ticker}?with_nested_markets=true` | Event + markets | | Markets come back under `event.markets`. The top-level `markets` key is present but **empty** **[live]**. |
| `GET /events/multivariate` | Combo events | `series_ticker`, `collection_ticker` | Not needed. Ignore. |
| `GET /series` | All series | `category`, `tags`, `include_product_metadata`, `include_volume`, `min_updated_ts` | **No cursor**: one response held all 14,681 series **[live]** (`raw/series_list_all.json`). |
| `GET /series/{series_ticker}` | One series | `include_volume` | Has `fee_type`, `fee_multiplier`, `category`, `categories`, `tags`, `frequency`, `settlement_sources`, `contract_url`, `contract_terms_url`, `exchange_index`, `last_updated_ts` **[live]** |
| `GET /markets/{ticker}/orderbook` | Book | `depth` (0 = all; 1–100) | `depth=101` → 400. **Unknown ticker → 200 with empty arrays, not 404** **[live]**. No `cache-control` header (not CDN-cached). |
| `GET /markets/orderbooks` | Up to 100 books | `tickers` (repeat the param; 1–100) | Returns `{"orderbooks":[{"ticker":…,"orderbook_fp":{…}}]}` **[live]** |
| `GET /series/fee_changes` | Scheduled series fee changes | `series_ticker`, `show_historical` | |
| `GET /events/fee_changes` | Scheduled event fee overrides | `event_ticker`, `limit` (≤1000), `cursor` | |
| `GET /search/tags_by_categories` | Category → tags taxonomy | | |
| `GET /structured_targets/{id}`, `GET /structured_targets?ids=…` | Resolve IDs inside `custom_strike` (teams, players) | `type`, `competition`, `page_size` ≤ 2000 | |
| `GET /exchange/status` | Exchange and shard status | | Shards: 0 Default, 1 Combos, 2 Crypto & Commodities, 3 Tennis/Baseball/Basketball **[live]** |
| `GET /account/endpoint_costs` | Token cost table | | Works unauthenticated. `default_cost: 10`. Every market-data GET uses the default **[live]** (`raw/account_endpoint_costs.json`). |

### Pagination [live]
- The cursor is opaque, base64url-encoded protobuf of `(created_time, ticker)`. `/markets` is ordered by **`created_time` descending**: I decoded page cursors and got 2026-10-06T03:52 → 2026-10-01 → 2026-09-10.
- **End of results = `"cursor": ""`** (empty string, not null), e.g. `markets?series_ticker=KXMLBGAME&status=open` → 10 markets, `cursor=''`. The docs page says "until the cursor is null", but the spec says "Empty if there are no more results". Handle both.
- `/markets` and `/series/{t}` responses carry `cache-control: public, max-age=15` (CloudFront). So data can be up to about 15 s stale. Order books are not cached.
- Gzip works with `Accept-Encoding: gzip`. A 1000-market page is about 2.1 MB uncompressed.

### Excluding multivariate / combo ("KXMVE…") markets
- `mve_filter=exclude` works **[live]**. Without it, page 1 of `status=open` was full of `KXMVECROSSCATEGORY-…` markets (`raw/markets_open_limit5.json`). With it, 0 of the 22,000 markets I scanned in the tail run had a `KXMVE` prefix or `mve_collection_ticker`.
- MVE markets identify themselves by `mve_collection_ticker`, `mve_selected_legs[]`, `custom_strike["Multivariate Event Ticker"]`, `price_level_structure = center_deci_edge_centi_cent` and `is_provisional: true`. Their series carries `fee_type: quadratic_with_combo_maker_fees` and category `Exotics`.
- `/events` excludes MVE automatically **[doc]**.

### Status semantics [doc + live]
- The filter `status=open` matches market `status: "active"`. Other mappings: `unopened` → `initialized`, `paused` → `inactive`, `closed` → past `close_time` and not finalized, `settled` → `finalized` (docs: Market Lifecycle).
- Timestamp filters are documented as mutually exclusive with some status values: `min/max_close_ts` only pairs with `closed` or no status. Without a status filter, a close-window query gave 968 `initialized` and 32 `active` markets. **Always pass `status=open`.**
- `status=open&min_close_ts=…&max_close_ts=…` was *not* rejected with a 400 **[live]**, but the docs say it's unsupported. Don't rely on it.

### How many open non-MVE markets, and how long a crawl takes [live]
- Run 1: `status=open&mve_filter=exclude&limit=1000`, sequential, no sleep. **33 pages in about 5.6 s** (0.14–0.21 s/page), then **HTTP 429**. Other research agents were hitting Kalshi from the same IP at the same time, which probably contributed.
- Run 2: resumed from the page-33 cursor with a 0.5 s sleep. **22 pages in 15.2 s**, mean 0.165 s/page, max 0.186 s, mean 2.11 MB/page, **zero 429s** (`raw/crawl_stats_open_mve_exclude_tail_from_page34.json`).
- **Total ≥ 55,000 open non-MVE markets, and the crawl was still not finished after 55 pages.** I stopped at the ~60-page budget. The last cursor had reached markets created 2026-09-10T18:51Z.
- Probes by `created_ts` window, 1 page each (`raw/created_ts_window_probe.json`):
  - 1,000+ open markets were created in one *minute* on 2026-09-10 (a `KXVOTEGENERAL` bulk listing).
  - 1,000+ were created on Jul 30–31 2026.
  - 1,000+ were created Dec 5–31 2025.
  - So there is a long tail of older open markets. **[inferred] Total is probably 70k–120k+.** Treat it as unknown above 55k.
- Shape of the 22k-market tail sample:
  - All `market_type: binary`.
  - `strike_type`: greater_or_equal 6,477; structured 5,192; greater 4,612; custom 4,498; less 239; between 184; *missing* 798.
  - `price_level_structure`: linear_cent 21,458; tapered_deci_cent 482; center_half_edge_half_cent 60.
  - 21,326 had a quote on at least one side. 14,768 were two-sided.
  - 951 distinct series. `KXVOTEGENERAL` alone had 5,478.
- **Crawl time estimate [inferred]:** about 0.17 s/page of server time. A full crawl of roughly 100 pages takes about 20–30 s if paced at ~4–5 req/s with 429 backoff.
- **Recommendation:** Equinox only needs categories that can match Polymarket. Prefer `GET /series` (1 call) → filter by category/tags → `GET /events?series_ticker=X&status=open&with_nested_markets=true`, or `GET /markets?series_ticker=X&status=open&mve_filter=exclude`. That is cheaper than crawling everything.

---

## 2. Rate limits

**Documented** (https://docs.kalshi.com/getting_started/rate_limits.md, `raw/kalshi_docs/getting_started_rate_limits.md`) **[doc]**:
- It is a token bucket. "Most requests cost the default of **10 tokens**". Read and Write buckets are separate.
- Per-second Read budgets by tier: **Basic 200**, Advanced 300, Expert 600, Premier 1,200, Paragon 2,400, Prime 4,800, Prestige 12,000 (Write budgets: 100 / 300 / 600 / 1,200 / 2,400 / 4,800 / 9,600).
- Basic and Advanced Read buckets hold **3 s of budget**, so up to 3× burst. Higher read tiers hold 1 s.
- When limited you get a 429. The docs say there is no `Retry-After` and no `X-RateLimit-*`, no penalty or cooldown, and you should use exponential backoff.
- **The docs do not give a tier for unauthenticated requests.** The text only says "Every authenticated request costs tokens".

**Observed for unauthenticated reads** **[live]** (`raw/ratelimit_probe_*.json`):
- 60 and then 150 sequential `GET …/orderbook?depth=1` calls on one keep-alive connection: about 19–20 req/s, **0 429s**.
- 4 threads × 50 requests (about 71 req/s offered): **132 OK, 68 × 429**. The first 429 came at 1.196 s after about 79 OKs, and after that about 20–40 OK/s got through.
  - [inferred] That matches the Basic tier exactly: a 600-token burst (60 requests) plus about 20 req/s refill. It looks keyed per IP.
- The 429 body is `{"error":{"code":"too_many_requests","message":"too many requests"}}`. The docs show `{"error": "too many requests"}`, so parse both. No `Retry-After` header.
- Practical budget for Equinox: **≤ 10 req/s steady**, with jittered exponential backoff on 429. Batch books via `/markets/orderbooks` (100 tickers per call; [inferred] each call is probably billed as one GET).

---

## 3. Price and size representation [doc + live]

### Fixed-point
Source: https://docs.kalshi.com/getting_started/fixed_point_migration.md, last updated Aug 20 2026.

- **Prices**: `*_dollars` are strings, e.g. `"0.4300"`. Docs say "up to 4 decimal places". The spec's `FixedPointDollars` says "responses emit up to 6" dp.
  - **Use int64 micro-dollars (1e-6)** to be safe.
  - Integer-cent fields "cannot represent sub-cent prices".
- **Sizes/counts**: `*_fp` strings with exactly 2 dp in responses, e.g. `"103.00"` or `"72612.49"`. Minimum granularity is 0.01 contracts.
  - Fractional sizes are common: 3,226 of 22k markets had a fractional top-of-book size, and order-book levels like `["0.5700","72612.49"]`.
  - Store as int64 centi-contracts.

### Legacy integer fields are gone [live]
I took the union of keys across about 2,800 live market objects from `/markets`, `/events?with_nested_markets` and `/markets/{t}`. None of these appear: `yes_bid`, `yes_ask`, `no_bid`, `no_ask`, `last_price`, `previous_*`, `volume`, `volume_24h`, `open_interest`, `liquidity`, `tick_size`, `notional_value`, `risk_limit_cents`.
- The only legacy field left is a deprecated `subtitle` on some markets.
- The order-book response contains **only** `orderbook_fp`. No legacy `orderbook` key.

### Market fields that are present now [live]
`ticker, event_ticker, market_type, title (deprecated), subtitle (deprecated, sometimes), yes_sub_title, no_sub_title, created_time, updated_time, open_time, close_time, expected_expiration_time, expiration_time (deprecated), latest_expiration_time, occurrence_datetime, settlement_timer_seconds, status, yes_bid_dollars, yes_bid_size_fp, yes_ask_dollars, yes_ask_size_fp, no_bid_dollars, no_ask_dollars, last_price_dollars, previous_yes_bid_dollars, previous_yes_ask_dollars, previous_price_dollars, volume_fp, volume_24h_fp, open_interest_fp, notional_value_dollars ("1.0000"), result, can_close_early, early_close_condition, expiration_value, settlement_value_dollars, settlement_ts, settlement_bounds_type ("default"|"floor"), strike_type, floor_strike, cap_strike, functional_strike, custom_strike, rules_primary, rules_secondary, price_level_structure, price_ranges, is_provisional, exchange_index, primary_participant_key, mve_collection_ticker, mve_selected_legs`.

- There is **no `category` or `tags` on markets.** They live on the series (see §6).
- There are no `no_bid_size` / `no_ask_size` fields:
  - `yes_ask_size_fp` is the size at the best NO bid. Verified: it equalled the last `no_dollars` level size.
  - `yes_bid_size_fp` is the size at the best YES bid.

### Empty-quote sentinels are inconsistent [live]
- With no YES bid, the sentinel was `yes_bid=0.0000`, `no_ask=1.0000` (6,570 cases).
- With no YES ask, most cases showed `yes_ask=1.0000`, `no_bid=0.0000` (1,295 cases). But 41 cases showed `yes_ask=0.0000`, `no_bid=1.0000`.
- The fully empty book on `KXSILVERH-26OCT0601-T60.499` showed `yes_ask 0.0000`, `no_bid 1.0000`, `no_ask 1.0000`.
- **Rule:** a side exists iff its `*_size_fp` is greater than 0. Better still, build top-of-book from the order book.

### Price grid: `price_ranges` / `price_level_structure`
- `price_ranges` is "the source of truth for valid prices". It looks like `[{start,end,step}]` with dollar strings. **Don't key logic off the `price_level_structure` name** **[doc]**.
- Structures seen live:
  - `linear_cent`: one band, step 0.0100.
  - `tapered_deci_cent`: 0–0.10 step 0.001, 0.10–0.90 step 0.01, 0.90–1.00 step 0.001.
  - `deci_cent`: 0–1, step 0.001.
  - `center_half_edge_half_cent`.
  - `center_deci_edge_centi_cent` (MVE): 0–0.01 step 0.0001, 0.01–0.99 step 0.001, 0.99–1.00 step 0.0001.
- The docs list 13 named structures. Whole-cent prices are valid in every one.
- Live sub-penny book, `KXWC-30-ESP` (`deci_cent`): YES bids `0.1340…0.1380`, NO bids `…0.8560`. The market showed `yes_bid 0.1380`, `yes_ask 0.1440` = 1 − 0.8560 ✓ (`raw/orderbook_KXWC-30-ESP_depth5.json`).

### `market_type`
- The enum is `binary | scalar` **[doc]**.
- All 22,000+ open non-MVE markets I sampled are `binary` **[live]**.
- Scalar markets settle with `result: "scalar"` and `settlement_value_dollars`. Filter `market_type == "binary"`.

### Strike semantics
`floor_strike` is "Minimum expiration value that leads to a YES settlement". `cap_strike` is "Maximum expiration value that leads to a YES settlement" **[doc]**. Live examples **[live]**:

| strike_type | Example | Fields |
|---|---|---|
| `greater` | `KXSILVERH-26OCT0601-T60.499` "above 60.499" | `floor_strike: 60.499` |
| `greater_or_equal` | `KXNKY-26OCT0600-70320.00` "at least ¥70,320" | `floor_strike: 70320` |
| `less` | `KXGDPYEAR-36-T0.1` "below 0.1%" | `cap_strike: 0.1` |
| `less_or_equal` | `USCLIMATE-2025` "4909.9 … or fewer" | `cap_strike: 4909.9` |
| `between` | `KXGDPYEAR-36-B0.3` "between 0.1% to 0.5%" | `floor_strike: 0.1, cap_strike: 0.5` (inclusivity is only stated in `rules_primary`) |
| `custom` | `KXOSCARSUPACTO-27-AND` | `custom_strike: {"Nominee":"Andrew Garfield"}` |
| `structured` | `KXCS2MAP-26OCT070930RFEVITA-2-VITA` | `custom_strike: {"esports_competitor":"1aae9260-…"}` → `GET /structured_targets/{id}` → `{"name":"Vitality Academy","type":"esports_competitor","details":{"league":"CS2","abbreviation":"VITA"}}` |
| `functional` | (none seen) | `functional_strike` string |
| *(absent)* | `KXINTLFRIENDLY1HBTTS-…-BTTS` | yes/no question with no strike |

- `custom_strike` is also used for free-form metadata, e.g. `{"front_month_contract":"NA"}` or `{"Index":"Nikkei 225"}`.

### Sub-titles and rules
- `yes_sub_title` is the short label of the YES outcome, e.g. "Tampa Bay", "Above $60.499", "0.1% to 0.5%".
- **`no_sub_title` is usually identical to `yes_sub_title`; it is NOT a negated label.** The old `USCLIMATE-2025` is an exception ("By 2025" / "Not By 2025").
- `title` on Market is deprecated. For full context use event `title` + `sub_title` + market `yes_sub_title`.
- `rules_primary` is the plain-language resolution criterion. It is the best text for semantic matching, e.g. "If the close price of the 1-minute candlestick for silver on October 06, 2026 at 1:00 AM EDT is above 60.499 USD/ounce, then the market resolves to Yes."
- `rules_secondary` holds edge cases and disclaimers. It is empty for MVEs.

### Time fields
Source: https://docs.kalshi.com/getting_started/market_lifecycle.md.

| Field | Meaning |
|---|---|
| `open_time` | Trading opens. |
| `close_time` | Trading stops. Can move earlier if `can_close_early`. Often set well past the real event (sports: days later, to allow rescheduling). |
| `expected_expiration_time` | When the outcome is expected to be known. **Can be before `close_time`** (sports). Nullable. |
| `latest_expiration_time` | Latest possible expiry. |
| `expiration_time` | **Deprecated.** Same as `latest_expiration_time` in practice. |
| `occurrence_datetime` | When the underlying event occurs, if known. |

- **Matching recommendation [inferred]:** use `occurrence_datetime`, falling back to `expected_expiration_time`, as the "event time" key. Use `close_time` only as a trading-availability bound.

---

## 4. Order book semantics [doc + live]

- `GET /markets/{ticker}/orderbook` → `{"orderbook_fp":{"yes_dollars":[[price,count],…],"no_dollars":[[price,count],…]}}`.
- **Bids only**, on both sides. "A YES BID at price X is equivalent to a NO ASK at price ($1.00 − X)" **[doc]**.
- Arrays are **sorted ascending by price; the best bid is the LAST element** **[doc]**. I verified this live: `yes_dollars` ran 0.0100 → 0.4200 (40 levels) and `no_dollars` ran 0.0100 → 0.5700 (49 levels).
- `depth=N` returns the **N best levels per side, still ascending**. Example with `depth=3`: `no: [0.55,0.56,0.57]`, `yes: [0.40,0.41,0.42]` **[live]**.
- Derivation, verified live on `KXMLBGAME-26OCT072000TBNYY-TB`. I fetched the book and the market at the same moment (`raw/orderbook_…TB.json`, `raw/market_…TB.json`):
  - best `no_dollars` = `["0.5700","72612.49"]` → derived YES ask = **0.4300 × 72612.49**.
  - Market: `yes_ask_dollars "0.4300"`, `yes_ask_size_fp "72612.49"` ✓.
  - best `yes_dollars` = `["0.4200","103.00"]` → market `yes_bid_dollars "0.4200"`, `yes_bid_size_fp "103.00"` ✓, and `no_ask_dollars "0.5800"` ✓.
- **Buying YES walks the NO-bid ladder from last to first**: YES fill price = 1 − no_price at each level, size = no count. Buying NO walks the YES-bid ladder the same way.
- Levels can be fractional (`"1273684.83"`). No per-level order count is returned.
- **Gotchas [live]:**
  - An unknown ticker returns 200 with an empty book.
  - The batch endpoint needs repeated `tickers=` params.
  - Order books are not CDN-cached; `/markets` quotes can be up to 15 s stale.

---

## 5. Fees

### Where the parameters live [doc + live]
- On the series: `fee_type` and `fee_multiplier`.
  - Spec enum `FeeType`: `quadratic`, `quadratic_with_maker_fees`, `quadratic_with_combo_maker_fees`, `flat`.
  - Spec text: "'quadratic' is described by the General Trading Fees Table, 'quadratic_with_maker_fees' is described by the General Trading Fees Table with maker fees described in the Maker Fees section, 'quadratic_with_combo_maker_fees' is the same maker-fee structure with a 0.5 maker multiplier instead of 0.25, 'flat' is described by the Specific Trading Fees Table." It cites https://kalshi.com/docs/kalshi-fee-schedule.pdf.
  - `fee_multiplier`: "a floating point multiplier applied to the fee calculations."
- On the event: **`fee_type_override` and `fee_multiplier_override`**, omitted when null. "When present, takes precedence over the series-level fee for this event's markets" **[doc]**.
- Scheduled changes:
  - `GET /series/fee_changes[?show_historical=true]`: 154 historical rows (`raw/series_fee_changes_historical.json`).
  - `GET /events/fee_changes`: 73 pending rows, all MLB playoff events. Example: `KXMLBGAME-26OCT072000TBNYY` switches from series `quadratic_with_maker_fees` ×0.5 to override ×1 at `2026-10-08T00:00:00Z` (`raw/event_fee_changes_limit200.json`).
  - The `market_lifecycle_v2` WS channel emits `event_fee_update`.
- Some fee types sit outside the enum: the history includes `margin_market_maker_program_fees`, for perps. **Treat `fee_type` as an open string. If it's unknown, mark the market non-routable.**
- Markets may carry `fee_waiver_expiration_time` (spec). None of the 22k markets in my sample had it.

### Live distribution across all 14,681 series [live]
| fee_type | multiplier | # series | Examples |
|---|---|---|---|
| quadratic | 1 | 14,486 | KXINX, KXNASDAQ100, KXBTCD, KXHIGHNY, KXWC |
| quadratic_with_maker_fees | 1 | 159 | KXFED, KXNFLGAME, KXNHLGAME, KXNBASPREAD, KXGDP, KXPAYROLLS, KXU3 |
| quadratic | 0.5 | 18 | MLB props (KXMLBTOTAL, KXMLBSPREAD, KXMLBHR, KXMLBKS, …) |
| quadratic | 0 | 14 | fee-free (KXBTCY, KXETHY, KXTRUMPOUT, KXGREENLAND, …) |
| quadratic_with_combo_maker_fees | 1 | 3 | KXMVECROSSCATEGORY, KXMVESPORTSMULTIGAMEEXTENDED, KXMVECROSSCATEGORY-SHARD1 |
| quadratic_with_maker_fees | 0.5 | 1 | KXMLBGAME |
| flat | – | **0** | none live |

- Note: KXINX, KXINXU and KXNASDAQ100 were moved to `quadratic` ×1 on 2026-07-03 (fee-change history). Older blog posts quote a 0.035 S&P/Nasdaq rate; that no longer applies.

### Formula
- **Taker** (all `quadratic*` types): `fee = roundup( fee_multiplier × 0.07 × C × P × (1 − P) )`.
  - C = contracts, P = price in dollars.
  - Per the fee schedule PDF: "fees = round up(0.07 x C x P x (1-P))… round up rounds to the next cent". I read this through the search-engine index of https://kalshi.com/docs/kalshi-fee-schedule.pdf, because direct fetches got a Vercel bot challenge (HTTP 429 `x-vercel-mitigated: challenge`). I did not bypass it.
  - **The 0.07 coefficient is confirmed in primary docs:** the Fee Rounding worked example uses a model fee of `$0.00363825` for a `$0.055` buy, and 0.07 × 1 × 0.055 × 0.945 = 0.00363825 exactly.
- **Maker:**
  - `quadratic` → 0.
  - `quadratic_with_maker_fees` → `roundup(fee_multiplier × 0.0175 × C × P × (1−P))`. 0.0175 = 0.25 × 0.07, consistent with the spec's "0.25" and the PDF's Maker Fees section (via the search index).
  - `quadratic_with_combo_maker_fees` → maker coefficient 0.5 × 0.07 = 0.035.
  - Maker fees are charged only on execution. Cancels are free (help.kalshi.com/trading/fees).
- **[inferred]** `fee_multiplier` scales both the taker and the maker coefficient. This matches Allium's documented reading (`M_taker × 0.07…`, `M_maker × 0.0175…`). Kalshi's own text only says "applied to the fee calculations".
- **`flat`**: the "Specific Trading Fees Table" in the PDF, which I could not retrieve. No live series uses it. **Mark `flat` as unsupported (non-routable).**

### Rounding
Source: https://docs.kalshi.com/getting_started/fee_rounding.md **[doc]**.

- Trade fee = `ceil` to `$0.000001`.
- A "rounding fee" then aligns the balance change to the user's precision: **$0.0001 for direct members, $0.01 for non-direct (FCM-cleared)**.
- A per-order **fee accumulator** rebates over-payment across fills, so "the total fee converges to what a single equivalent fill would cost".
- **Equinox recommendation [inferred]:**
  - Compute `raw = mult × coeff × C × P × (1−P)` exactly in integer micro-dollars.
  - Charge `ceil` to $0.01 **per order (per venue leg)**, as the published PDF says. This is conservative: it over-estimates by under 1¢ compared with direct-member precision.

### Worked examples (C = 100, P = 0.43)
| Case | Raw | Charged |
|---|---|---|
| Standard taker | 0.07 × 100 × 0.43 × 0.57 = 1.7157 | **$1.72** |
| KXMLBGAME taker (×0.5) | 0.85785 | **$0.86** |
| `quadratic_with_maker_fees` maker | 0.428925 | **$0.43** |
| Combo maker | 0.85785 | $0.86 |

---

## 6. Category and tag fields for blocking during matching

- **Series** (`GET /series`, one call) **[live]**:
  - `category`: primary category. Counts: Sports 3,850; Entertainment 2,548; Politics 2,417; Elections 1,895; Financials 1,000; Economics 838; Mentions 454; Climate and Weather 415; Science and Technology 360; Crypto 275; Companies 179; World 143; Commodities 99; Health 96; Social 52; Transportation 38; Exotics 14; AI 6; Business 1; Education 1.
  - `categories[]`: discovery categories. The `?category=` filter matches any entry, exact and case-sensitive.
  - `tags[]`: nullable; **2,790 of 14,681 series have none**. Top tags: Soccer 1,414; Music 801; US Elections 739; Basketball 630; Football 595; Awards 487; Congress 401; Trump 369; …; Baseball 231; BTC; Fed; Indices; Daily temperature.
  - Also `frequency` (`daily`, `hourly`, `weekly`, `one_off`, `custom`, …), `settlement_sources[{name,url}]`, `contract_terms_url`, and `product_metadata` (`include_product_metadata=true`).
- **Taxonomy:** `GET /search/tags_by_categories` → `{"tags_by_categories":{"Crypto":["BTC","15 min","Hourly","ETH",…],"Sports":["Soccer","Football","Basketball","Baseball","Hockey","Tennis","Golf","Esports"],…}}` (`raw/search_tags_by_categories.json`). `GET /search/filters_by_sport` also exists; I did not fetch it.
- **Event:**
  - `category` is deprecated ("use series-level category") but still populated.
  - `product_metadata`, e.g. `{"competition":"Pro Baseball","competition_scope":"Game"}`.
  - `strike_date` / `strike_period`, `sub_title` (e.g. "TB vs NYY (Oct 7)").
  - `mutually_exclusive` + `collateral_return_type` (`MECNET` = at most one YES).
  - `settlement_sources`.
- **Market:**
  - No category.
  - Use `custom_strike` structured-target IDs. Resolve them with `/structured_targets?ids=…` (repeat the param, up to 2000/page) to get canonical team/player names and `details.league`.
  - Also use `occurrence_datetime`, `strike_type`, `floor_strike` / `cap_strike`, `yes_sub_title` and `rules_primary`.
- **Blocking recommendation [inferred]:** use the coarse key (series.category, normalized event date from `occurrence_datetime` or `expected_expiration_time`), then the tag, then the structured target, competitor or strike type. Never compare across categories.

---

## Notes and side effects
- Rate-limit probes: about 410 small requests to `/orderbook?depth=1` in total. They deliberately triggered 68 × 429.
- I used about 55 `/markets?limit=1000` pages plus about 25 other requests.
- My scratchpad shared a filename (`crawl.py`) with a sibling Polymarket agent. One of my runs executed **their** script once: 60 pages of `gamma-api.polymarket.com/markets/keyset`, which re-wrote `raw/polymarket_keyset_census/page_*.json` with fresh data. I reverted a harmless one-line import edit I had made to their file. My own scripts now live in `scratchpad/kalshi_agent/`.
