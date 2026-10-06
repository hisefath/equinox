# Polymarket public APIs: research for Project Equinox

Verified 2026-10-05 (US) / 2026-10-06 ~03:50–04:05 UTC against live, unauthenticated endpoints. Raw captures are in `research/raw/polymarket_*`. No credentials were used.

- **Gamma** (metadata): `https://gamma-api.polymarket.com`
- **CLOB** (books, fee params): `https://clob.polymarket.com`
- Primary docs: https://docs.polymarket.com (machine-readable index: https://docs.polymarket.com/llms.txt; OpenAPI: https://docs.polymarket.com/api-spec/gamma-openapi.yaml, https://docs.polymarket.com/api-spec/clob-openapi.yaml)

---

## TL;DR for the Go implementation

1. **List markets with `GET /markets/keyset?closed=false&limit=100&after_cursor=<next_cursor>`.** Use cursor pagination. The legacy `/markets` sends `deprecation: true`, a `sunset: Fri, 01 May 2026` header and `warning: 299 - "use /markets/keyset"`, and it caps `offset` at 2000. `limit` is silently clamped to 100. Passing `offset` to keyset returns 422.
2. **Decide what is tradable yourself.** Keep a market only when `active && !closed && enableOrderBook && acceptingOrders`, `endDate` is in the future, and `umaResolutionStatus` is empty. 12.6% of the oldest-6000 sample had a past `endDate` but were still `acceptingOrders=true`. Markets whose resolution is "proposed" have **no CLOB book**.
3. **Asset IDs:** if `version=="v2"`, use `positionIds` (a real JSON array). Otherwise parse `clobTokenIds`, which is a **JSON-encoded string**. Index 0 is `outcomes[0]` ("Yes" or the first team or label) and index 1 is `outcomes[1]`. Every market sampled live was `version:"v1"`.
4. **Books:** in `GET /book` and `POST /books`, **bids are ascending and asks are descending, so the best level is the LAST element.** The docs prose confirms this. The OpenAPI schema text says the opposite and is wrong. Prices and sizes are decimal **strings**. Sizes are shares. `timestamp` is a string of epoch **milliseconds**.
5. **The NO token has its own book, and it is an exact mirror of the YES book**: NO bids = {1−p : size} of YES asks, and NO asks = {1−p : size} of YES bids. So NO best ask = 1 − YES best bid exactly. Buying NO at q costs the same as selling YES at 1−q.
6. **Batch books:** `POST /books` takes `[{"token_id":"..."}]`, with **at most 500** items (501 returns `400 {"error":"Payload exceeds the limit"}`). Unknown tokens or tokens without a book are **silently dropped**, and the **response order does not match the request order**, so key results by `asset_id`. `GET /books?token_ids=` returns 400 live, even though the spec lists it.
7. **Fees (taker only):** `fee = C × rate × (p × (1 − p))^exponent`, where C = shares, p = trade price of the token traded, and `rate`/`exponent` come from the market's `feeSchedule` (Gamma) or `fd.r`/`fd.e` (CLOB `/clob-markets/{conditionId}`). Every live market seen except one had `exponent: 1`, which gives the documented `fee = C × feeRate × p × (1 − p)`. Fees are rounded to 5 decimals (minimum 0.00001). A market is fee-free when `feesEnabled` is false or `feeSchedule` is null (Geopolitics), or when `rate` is 0 (`feeType:"zero_fees"`). **Do not use `base_fee`/`takerBaseFee` (1000) in fee math.** It is a legacy V1 order-signing value in bps: 1000 when fees are on, 0 when off.
8. **Use the per-market `feeSchedule`, not the docs category table.** Older sports markets still carry `rate 0.03` (`sports_fees_v2`) while new ones carry `0.05` (`sports_fees_v3`).
9. **Gamma is CDN-cached** (`cache-control: public, max-age=300`). In 15% of top-volume YES books, Gamma `bestBid`/`bestAsk` differed from the live CLOB top. Always price from CLOB.
10. **Rate limits are documented per IP and enforced by Cloudflare throttling (delay), not rejection.** Gamma `/markets` allows 300 req/10s and `/events` 500 req/10s. CLOB `/book` allows 1,500 req/10s and `/books` 500 req/10s. No rate-limit headers are returned. Python's default `Python-urllib` User-Agent gets **403** from Cloudflare. Go's default `Go-http-client/1.1` works, but set an explicit UA anyway.

---

## 1. Listing active tradable markets and events

### 1.1 Endpoints

| Endpoint | Status | Pagination | Max limit (verified) |
|---|---|---|---|
| `GET /markets/keyset` | current, recommended | `after_cursor` ← `next_cursor` | 100 (larger values silently clamped to 100) |
| `GET /events/keyset` | current, recommended | `after_cursor` ← `next_cursor` | 100 (clamped) |
| `GET /markets` | deprecated (headers below) | `limit`/`offset` | 100; `offset` > 2000 → 422 `"offset too large, use /markets/keyset for deeper pagination"` |
| `GET /events` | deprecated (`warning: 299 - "use /events/keyset"`) | `limit`/`offset` | 100 |
| `GET /events/pagination` | `x-excluded` (undocumented) | offset | returns `{"data":[...],"pagination":{"hasMore":true,"totalResults":N}}`, useful for **counts** |

Legacy `/markets` response headers (live):
```
deprecation: true
sunset: Fri, 01 May 2026 00:00:00 GMT
warning: 299 - "use /markets/keyset"
```
It still answers 200 today, past its sunset date. Do not rely on it.

Changelog (https://docs.polymarket.com/changelog/predictions.md):
- Apr 10, 2026: keyset endpoints added. "Same filters, same response shape per item — the only differences are the wrapper response (`{ "markets": [...], "next_cursor": "..." }`) and the rejection of `offset`."
- May 14, 2026: "`GET /markets/keyset` limit: The maximum `limit` value is now `100`."
- Apr 9, 2026: `closed` now defaults to `false` on `GET /markets`. The keyset spec also defaults `closed` to false.

### 1.2 Keyset response shape (live)

```json
{ "$schema": "https://gamma-api.polymarket.com/schemas/MarketsKeysetListResponse.json",
  "markets": [ { ...market... } ],
  "next_cursor": "7nQOm0ypGyI0kjQ_3miyt6C6jRFewdLCmFFAu1KegPl7InYiOjEsImsiOiJtYXJrZXRzIi..." }
```
`/events/keyset` returns `{"$schema", "events": [...], "next_cursor"}`. The last page has no `next_cursor`. The cursor is opaque and signed. Its base64 tail shows the sort key, for example `{"keys":[{"t":"string","v":"559653"}]}`. The default order is by market `id` ascending.

### 1.3 Parameters (from gamma-openapi.yaml, behavior checked live)

`/markets/keyset`: `limit` (1–100, default 20), `order` (comma-separated **camelCase JSON field names**), `ascending` (default true, only used with `order`), `after_cursor`, `offset` (rejected with **422** `"offset is not allowed on keyset endpoints"`), `id[]`, `slug[]`, `closed` (default false), `decimalized`, `clob_token_ids[]`, `condition_ids[]`, `question_ids[]`, `liquidity_num_min/max`, `volume_num_min/max`, `start_date_min/max`, `end_date_min/max`, `tag_id[]`, `related_tags`, `tag_match`, `cyom`, `rfq_enabled`, `uma_resolution_status`, `game_id`, `sports_market_types[]`, `include_tag`, `locale`.

Live checks:
- `order=volume24hr&ascending=false` works and returns the top 24h volume first. `order=volumeNum`, `liquidityNum`, `endDate`, `startDate` and `id` work. **`order=volume_num` and `order=end_date` return 422 `"order fields are not valid"`**, even though the spec example says `volume_num,liquidity_num`. `order=volume`/`liquidity` are accepted but sort the string fields, so avoid them.
- `end_date_min=2026-10-06T00:00:00Z&end_date_max=2026-10-08T00:00:00Z` filters correctly (RFC3339).
- `tag_id=2` filters (politics).
- **`active` and `archived` are not keyset parameters.** They are accepted and ignored: the same cursor comes back as with `bogus=1`. Unknown params are ignored, not rejected. **The keyset endpoint only returns `active:true` markets.** `id=559705` (an inactive "Person X" placeholder) returns `[]` even with `active=false`.
- `closed=true` returns closed markets. Ids 12, 17 and up date back to 2020.

`/events/keyset` adds: `live`, `featured`, `title_search`, `liquidity_min/max`, `volume_min/max`, `start_time_min/max`, `tag_slug`, `exclude_tag_id`, `series_id`, `event_date`, `event_week`, `recurrence`, `parent_event_id`, `include_children`, and others. **Events nest ALL of their markets, including `closed:true` and inactive placeholders.** In `/events/keyset?closed=false`, the "Kraken IPO" event included closed child markets. Filter child markets by their own flags.

### 1.4 How many open markets, and how long a crawl takes (measured)

Crawl: `GET /markets/keyset?closed=false&limit=100`, followed by cursor, capped at **60 pages**:

| metric | value |
|---|---|
| pages / markets | 60 / 6,000 (**not exhausted**; the cursor was still present at id 2,208,077) |
| wall time | **10.7 s** sequential (avg 0.172 s/page, max 0.43 s) |
| payload | ~0.6 MB per 100-market page (36 MB for 60 pages) |
| `enableOrderBook && acceptingOrders` | 5,945 / 6,000 (99.1%) |
| …but `endDate` already past | 754 / 6,000 (12.6%), e.g. a Feb-24-2026 "XRP Up or Down" 5-minute market is still `acceptingOrders:true` |

The full open universe is too big to crawl in 60 pages, so it was estimated:
- `GET /events/pagination?limit=1&closed=false&active=true&archived=false` → `totalResults: 21971` open events. `closed=false` alone gives 21,989. `closed=false&end_date_min=2026-10-06T04:00:00Z` gives 18,661. Unfiltered gives 1,116,160.
- In 2×10 pages of `/events/keyset?closed=false` (oldest first and newest first), each event had on average **3.15** (oldest) to **3.90** (newest) child markets with `active && !closed && enableOrderBook && acceptingOrders`.
- **Estimate: about 70k–85k open, order-book-enabled markets.** A full keyset crawl would be about 700–850 pages: roughly **2–2.5 min sequential** and about 0.4–0.5 GB of JSON. **Recommendation:** scope ingest with `order=volume24hr&ascending=false` (top N pages), `tag_id`, and/or an `end_date_min/max` window instead of a full crawl. 10 pages ordered by `volume24hr` took 3.1 s.

---

## 2. Rate limits (documented)

Source: https://docs.polymarket.com/api-reference/rate-limits.md, which says the limits are "IP-based and enforced using Cloudflare's throttling system. When you exceed the limit for any endpoint, requests are throttled (delayed/queued) rather than immediately rejected. Limits reset on sliding time windows."

| API | Endpoint | Limit |
|---|---|---|
| all | general | 15,000 req / 10 s |
| Gamma | general | 4,000 / 10 s |
| Gamma | `/events` | 500 / 10 s |
| Gamma | `/markets` | 300 / 10 s |
| Gamma | `/markets` + `/events` listing | 900 / 10 s |
| Gamma | `/tags` 200, `/comments` 200, `/public-search` 350 | per 10 s |
| CLOB | general | 9,000 / 10 s |
| CLOB | `/book` | 1,500 / 10 s |
| CLOB | `/books` | 500 / 10 s |
| CLOB | `/price` 1,500, `/prices` 500, `/midpoint` 1,500, `/midpoints` 500, `/prices-history` 1,000 | per 10 s |
| CLOB | market tick size | 200 / 10 s |

Observed:
- No `x-ratelimit-*` headers on Gamma or CLOB responses.
- Gamma responses carry `cache-control: public, max-age=300` with `cf-cache-status: EXPIRED/HIT`. CLOB `/book` is `cf-cache-status: DYNAMIC` and not cached.
- **User-Agent:** `Python-urllib/3.14` gets **403** on both hosts. `Go-http-client/1.1`, `Go-http-client/2.0`, an empty UA and `equinox/0.1` all get 200.

---

## 3. Field semantics (Gamma market object)

Example: `research/raw/polymarket_gamma_markets_keyset_sample3.json`, market 559651, "Xi Jinping out before 2027?". The fields that matter for Equinox:

| Field | Type (live) | Example | Semantics / notes |
|---|---|---|---|
| `id` | string | `"559651"` | Gamma market id (numeric string) |
| `conditionId` | string (0x…32 bytes) | `0xa467b1…43b7` | CTF condition; the key for CLOB `/clob-markets/{conditionId}`; equals book `market` |
| `question` | string | | Binary question. For grouped markets, read it together with `groupItemTitle` |
| `description` | string | "This market will resolve to "Yes" if …" | **The resolution rules** (criteria, window, source). `resolutionSource` is usually `""` (present on only 59% of markets) |
| `outcomes` | **JSON-encoded string** | `"[\"Yes\", \"No\"]"` | Labels. Docs: "Outcome labels and prices are JSON-encoded strings. Parse them into arrays; entries at the same index describe the same outcome." |
| `outcomePrices` | **JSON-encoded string** of decimal strings | `"[\"0.0245\", \"0.9755\"]"` | Display prices (mid, or last trade if spread > 0.10). Absent on inactive placeholders |
| `clobTokenIds` | **JSON-encoded string** | `"[\"3233…3401\", \"2565…1962\"]"` | CTF (v1) ERC-1155 token ids, as uint256 **decimal strings (77–78 digits; do not parse to int64/float)** |
| `positionIds` | JSON **array** | `["8952…5792","8952…5793"]` | Protocol V2 ids. Docs: use them when `version=="v2"` (`clobTokenIds` is then `null`). "Both fields can be populated, so their presence does not select the protocol." |
| `version` | string | `"v1"` | `"v1"` = CTF, `"v2"` = Polymarket Protocol V2. 8,000/8,000 sampled were `v1` |
| `bestBid`, `bestAsk` | **number** (float, sometimes int, e.g. `1`) | `0.024`, `0.025` | Top of the **outcome[0]** book. Missing on about 20% (empty bid side). Cached up to 5 min, so stale versus CLOB in 15% of top-volume books |
| `lastTradePrice`, `spread` | number | `0.025`, `0.001` | |
| `endDate` | RFC3339 string | `"2027-01-01T04:59:00Z"` | Scheduled end, **not** a trading cutoff. It can be in the past while the market is still `active`/`acceptingOrders` (awaiting resolution). Missing on 0.4%. Also `endDateIso` (date only) |
| `gameStartTime` / `eventStartTime` | string | | Sports only |
| `active` / `closed` / `archived` | bool | | Docs: active = "deployed and not archived"; closed = "resolved or been closed, so no further trading"; archived = read-only |
| `enableOrderBook` | bool | `true` | Market has a CLOB book (false means AMM/legacy) |
| `acceptingOrders` | bool | `true` | Docs: "Order book is open for new limit and market orders." Docs' `isTradeReady = active && !closed && acceptingOrders` |
| `umaResolutionStatus` | string | `"proposed"` | When non-empty, resolution is underway. **All 16 tokens that `POST /books` dropped belonged to `proposed` markets**, and `GET /book` returns 404 `"No orderbook exists for the requested token id"` |
| `orderPriceMinTickSize` | number | `0.001` / `0.01` (also `0.0025` World Cup, `0.0001`, `0.005`, `0.1` documented) | Tick. Can change: watch `tick_size_change` on the WS, or re-read the book `tick_size` |
| `orderMinSize` | number | `5` | Minimum order size. Place-orders docs: "the minimum number of **shares**". The SDK type comment in market-details calls it "minimum USDC notional" (a doc inconsistency; treat it as shares). 5 for every sampled market |
| `negRisk` | bool | | Market belongs to a negative-risk (mutually exclusive) group. 58% of the oldest-6000, 50% of the top-1000 by volume |
| `negRiskMarketID` | string | `0x2c3d…4700` | Shared by every member of the group (equals the event's `negRiskMarketID`) |
| `negRiskOther` | bool | | `true` on the explicit "Other" catch-all of an augmented neg-risk event |
| `groupItemTitle` | string | `"Gavin Newsom"`, `"December 31"` | The outcome label inside a multi-market event: candidate, date bucket, threshold, etc. Non-empty on about 90% |
| `groupItemThreshold` | string | `"0"` | Ordering within the group |
| `feesEnabled`, `feeType`, `feeSchedule`, `takerBaseFee`, `makerBaseFee` | see §5 | | |
| `events[]` | array | | Parent event(s), with `slug`, `title`, `negRisk`, `enableNegRisk`, `negRiskAugmented`, `series[]` |
| `restricted` | bool | | Geo-restricted in some jurisdictions |

Numeric fields switch between int and float in JSON (for example `bestAsk: 1`, `volume24hr: 2000`). Decode them as `float64` or `json.Number`, never `int`. Several are omitted rather than null (`bestBid`, `volume24hr`, `clobRewards`, …).

### 3.1 Multi-outcome events built from binary markets (negRisk)

Example: `research/raw/polymarket_gamma_event_negrisk_democratic-presidential-nominee-2028.json` (`GET /events/slug/democratic-presidential-nominee-2028`):
- Event fields: `negRisk:true`, `enableNegRisk:true`, `negRiskAugmented:true`, `negRiskMarketID:"0x2c3d…4700"`.
- 128 child markets. Each is a **binary Yes/No market** ("Will Gavin Newsom win…?") with `groupItemTitle:"Gavin Newsom"`, `negRisk:true` and the same `negRiskMarketID`.
- 53 are active. 75 are inactive **placeholders** (`groupItemTitle:"Person X"` … `"Person CR"`, plus `"Other"` with `negRiskOther:true`). These have `active:false` and no `outcomePrices`, yet `enableOrderBook:true` and `acceptingOrders:true`. Docs (https://docs.polymarket.com/concepts/negative-risk.md) say to trade only named outcomes and ignore placeholders and "Other".
- Across the 53 active markets: Σ YES `outcomePrices` = 0.9255, Σ `bestAsk` = 0.966, Σ `bestBid` = 0.885.
- Mechanism (docs): "A No share in any market can be converted into 1 Yes share in every other market."

For Equinox, model each child market as its own canonical binary market (question + `groupItemTitle`), keep `negRiskMarketID` as the group key, and drop `active:false` and `negRiskOther:true` members.

### 3.2 Two-outcome markets whose outcomes are not `["Yes","No"]`

No market in any sample had a number of outcomes other than 2.

| Sample | Tradable markets | non-Yes/No | Breakdown |
|---|---|---|---|
| Oldest 6,000 open (id asc) | 5,945 | **520 (8.7%)** | Odd/Even 437, Up/Down 59, Over/Under 13, team/other names ~11 |
| Newest 1,000 (`order=id&ascending=false`) | 1,000 | **650 (65%)** | Over/Under 474, Up/Down 64, team names ~112 (sports moneylines) |
| Top 1,000 by `volume24hr` | 1,000 | **174 (17.4%)** | Over/Under 54, Up/Down 6, team/player names ~114 (e.g. `["Saints","Falcons"]`, `["Carlos Alcaraz","Jiri Lehecka"]`, `["Lakers","Kings"]`) |

The orientation rule: token[0] ↔ `outcomes[0]` is the canonical "YES" leg. Here "YES" means "outcomes[0] happens" (e.g. Saints win, Over, Up). Equivalence matching against Kalshi must map Kalshi's YES to the matching **label**, not assume "Yes". The same matchup can appear in both orders (`("Iwaki FC","Ventforet Kōfu")` and the reverse) as separate markets.

---

## 4. CLOB order books

### 4.1 `GET /book?token_id=<decimal token id>`

Live example: `research/raw/polymarket_clob_book_xi_yes.json`
```json
{"market":"0xa467b14d…43b7",
 "asset_id":"32338220190071351435772801779725302244575775216413325951443816017994629993401",
 "timestamp":"1791258810641",
 "hash":"6bd013d5f211e2eaecf60fbf357b1f936422049f",
 "bids":[{"price":"0.001","size":"16069385.26"}, …, {"price":"0.023","size":"104394.73"},{"price":"0.024","size":"7838.1"}],
 "asks":[{"price":"0.999","size":"2178920.26"}, …, {"price":"0.026","size":"10335.22"},{"price":"0.025","size":"3563.48"}],
 "min_order_size":"5","tick_size":"0.001","neg_risk":false,"last_trade_price":"0.024"}
```
- **Ordering:** bids are **ascending** and asks **descending**, so **best bid = `bids[len-1]`, best ask = `asks[len-1]`**. Checked on 484/484 books in a 500-token batch with zero violations. The docs prose agrees (https://docs.polymarket.com/market-data/prices-order-books.md): "Bids are ordered by ascending price and asks by descending price, so the best bid and ask are the last entries in their respective arrays." **The OpenAPI schema says "sorted by price descending" for bids, which is wrong.** Don't trust it.
- Prices and sizes are **decimal strings**. **Size is in shares** (outcome tokens, 2 decimals). Price is pUSD per share in [0,1].
- `timestamp`: string of **epoch ms** (1791258810641 = 2026-10-06T03:53:30.641Z).
- `hash`: a hash of the book state. Docs: "Compare it with the previous response's hash to determine whether the book changed between reads." It is 40 hex chars live; the docs example shows 64.
- `tick_size` and `min_order_size` are strings. `neg_risk` is bool.
- **`last_trade_price` is the same on the YES and NO books** (242/242 pairs). It is market-level, in outcome[0] terms: the NO book of Xi shows `0.024` even though the last fill was a BUY NO at 0.976. Don't read it as the NO price.
- 47/484 books had an empty bid side and 47 an empty ask side. Handle empty arrays.
- Errors: no `token_id` → 400 `{"error":"Invalid token id"}`. Unknown token or no book → 404 `{"error":"No orderbook exists for the requested token id"}`.

### 4.2 `POST /books` (batch)

```
POST https://clob.polymarket.com/books
Content-Type: application/json
[{"token_id":"<id1>"},{"token_id":"<id2>"}]        // optional "side":"BUY"|"SELL" accepted, ignored
```
- Returns `[OrderBookSummary, …]` with the same shape as `/book`.
- **Max 500 items.** The docs say "Maximum 500 items per request." Live: 500 → 200 (0.39 s, 1.6 MB); 501 → **400 `{"error":"Payload exceeds the limit"}`**.
- `[]` → `200 []`. Unknown or book-less tokens are **silently omitted**: requesting 500 returned 488, and the 16 missing tokens were from 8 markets with `umaResolutionStatus:"proposed"`.
- **The response order differs from the request order.** Index results by `asset_id`.
- The YES/NO pair in one batch shares one `timestamp`.
- `GET /books?token_ids=a,b` is in the OpenAPI but returns **400 `{"error":"Invalid payload"}`** live in every variant tried. Use POST.

### 4.3 Is buying NO the same as buying the second token?

Yes. The second token id (index 1, outcome "No") has **its own book**, and it is an **exact mirror** of the YES book:
- Xi: NO bids == {1−p: s for YES asks}, NO asks == {1−p: s for YES bids}. Both checked **True** with Decimal equality on every level. NO best ask 0.976 (size 7838.1) = 1 − YES best bid 0.024 (size 7838.1).
- Across 242 YES/NO pairs from one `POST /books` batch, 241 mirrored exactly. The one mismatch is consistent with the two snapshots being taken at slightly different moments.
- Mechanism (https://docs.polymarket.com/concepts/prices-orderbook.md): orders on complementary outcomes match when prices sum to $1.00. "1 pUSD of collateral creates a complete set."
- Routing implication: on Polymarket, "buy NO at q" and "sell YES at 1−q" draw on the **same liquidity**. Don't count it twice. To buy NO, walk the NO token's asks (or equivalently the YES bids at 1−p). The fee uses the price of the token actually traded (the fee curve is symmetric, so the result is the same).

### 4.4 Compact market info: `GET /clob-markets/{conditionId}` (the authoritative fee source)

Live (`research/raw/polymarket_clob_clob-markets_xi.json`):
```json
{"r":{"mi":200,"ma":3.5,"e":true,"moas":4},
 "t":[{"t":"3233…3401","o":"Yes"},{"t":"2565…1962","o":"No"}],
 "c":"0xa467…43b7","mos":5,"mts":0.001,"mbf":1000,"tbf":1000,"ao":true,
 "cbos":true,"aot":"2025-07-03T20:36:33Z","ibce":true,
 "fd":{"r":0.04,"e":1,"to":true},"v":"v1"}
```
`t` = tokens with outcome labels (same order as `clobTokenIds`), `mts` = min tick, `mos` = min order size, `ao` = accepting orders, `nr` = negRisk (omitted when false), `fd` = fee details `{r: rate, e: exponent, to: takerOnly}` (omitted on fee-free markets), `mbf`/`tbf` = maker/taker base fee in bps (legacy), `v` = version. The official SDKs read fees from here: `ts-sdk` `MarketInfoSchema` maps `fd` to `feeInfo: fd ?? { rate: 0, exponent: 0 }`, and the V2 migration guide says "Use `getClobMarketInfo(conditionID)` to query fee parameters (`fd.r`, `fd.e`, `fd.to`)."

Side note: `GET /markets-by-token/{tokenId}` returned `primary_token_id` = the **No** token for Xi. Don't infer YES/NO from it.

---

## 5. Fees (critical)

### 5.1 Official formula (quoted)

https://docs.polymarket.com/trading/fees.md:
> Fees are calculated using the following formula:
> `fee = C × feeRate × p × (1 - p)`
> Where **C** = number of shares traded and **p** = price of the shares.

Also from that page: "Makers are never charged fees. Only takers pay fees." "Geopolitical and world events markets are fee-free." "Fees are rounded to 5 decimal places. The smallest fee charged is 0.00001 USDC. Anything smaller rounds to zero." The collateral is now pUSD; the page still says USDC.

The general form with `exponent`, as implemented by the official SDKs:
- `ts-sdk/packages/client/src/actions/orders/market.ts`: `platformFeeRate = params.platformFeeRate * (params.price * (1 - params.price)) ** params.platformFeeExponent; platformFee = (params.amount / params.price) * platformFeeRate`, where amount/price = C shares.
- `clob-client-v2/src/fees/index.ts` and `py-clob-client-v2/fees.py`: `platform_fee_rate = fee_rate * (price * (1 - price)) ** fee_exponent`.
- Docs field definition (https://docs.polymarket.com/market-data/market-details.md#trading-fees): `feeSchedule.exponent` = "Exponent applied to the price component of the fee curve."

Implement:
```
fee_usd = C × rate × (p × (1 − p))^exponent        // exponent == 1 for ~all live markets
          rounded to 5 dp; 0 if !feesEnabled or feeSchedule == null; taker only (takerOnly:true everywhere)
BUY:  total cost  = C × p + fee        ("BUY fees add to collateral spend")
SELL: net proceeds = C × p − fee       ("SELL fees are deducted from proceeds")
```
The BUY/SELL wording comes from https://docs.polymarket.com/migrate/polymarket-v2/api-integrations.md. The 5-decimal rounding *direction* (half-up, banker's rounding or truncation) is not documented.

Changelog context:
- Mar 30, 2026, "Fee Structure V2": new categories and rates.
- Mar 31, 2026: "Fees should now be calculated using the `feeSchedule` object within a market."
- Apr 28, 2026, CLOB V2: "Fees are now set at match time — no more `feeRateBps` on orders."
- Jul 10, 2026: sports rate 0.03 → 0.05 and sports maker rebate 25% → 15%.

### 5.2 Worked examples

From the docs fee tables (100 shares). I recomputed rows across all three rate tables with the formula and got 0 mismatches at 2-dp display rounding:

| Category (rate) | p = 0.05 | p = 0.30 | p = 0.50 | p = 0.95 |
|---|---|---|---|---|
| Crypto (0.07) | $0.33 | $1.47 | **$1.75** | $0.33 |
| Sports / Econ / Culture / Weather / Other (0.05) | $0.24 | $1.05 | **$1.25** | $0.24 |
| Finance / Politics / Mentions / Tech (0.04) | $0.19 | $0.84 | **$1.00** | $0.19 |

For example, Politics at p = 0.50: 100 × 0.04 × 0.5 × 0.5 = **$1.00**. Docs: "The fee in USDC peaks at 50% probability ($1.00)."

Live example (Xi market, `feeSchedule {rate:0.04, exponent:1}`):
- Buy 100 YES at the best ask 0.025 → fee = 100 × 0.04 × 0.025 × 0.975 = **$0.09750**.
- Buy 100 NO at the best ask 0.976 → fee = 100 × 0.04 × 0.976 × 0.024 = 0.093696 → **$0.09370**.

Docs category table (current):

| Category | Taker rate | Maker rebate |
|---|---|---|
| Crypto | 0.07 | 20% |
| Sports | 0.05 | 15% |
| Finance, Politics, Mentions, Tech | 0.04 | 25% |
| Economics, Culture, Weather, Other/General | 0.05 | 25% |
| Geopolitics | 0 | none |

### 5.3 Live fee fields

| `feeType` (live values) | `feeSchedule` seen | Notes |
|---|---|---|
| `politics_fees`, `finance_prices_fees`, `tech_fees`, `mentions_fees` | `{exponent:1, rate:0.04, takerOnly:true, rebateRate:0.25}` | matches docs |
| `sports_fees_v3` | `{1, 0.05, true, 0.15}` | all newest sports markets; matches docs |
| `sports_fees_v2` | `{1, **0.03**, true, 0.25}` | **older sports markets keep the pre-July rate** (1,691 of the oldest 6,000; 62 of the top 1,000) |
| `sports_fees_nfl_cfb_oct26` | `{1, 0.03, true, 0.15}` | promotional / special |
| `crypto_fees_v2` | `{1, 0.07, true, 0.20}` | |
| `economics_fees`, `culture_fees`, `weather_fees`, `general_fees` | `{1, 0.05, true, 0.25}` | |
| `zero_fees` | `{1, 0, true, 0}` with `feesEnabled:true` | fee-free via rate 0 (29 in the top 1,000) |
| `crypto_15_min` | `{**exponent:2**, rate:0.25, true, 0.20}` | 1 stale legacy market (Feb 2026); `/clob-markets` `fd:{r:0.25,e:2}` |
| `null` | `feeSchedule: null`, `feesEnabled:false`, no `takerBaseFee` | Geopolitics (e.g. "Putin out…", "Sudan ceasefire…"); 11% of the top 1,000 |

### 5.4 What `base_fee` 1000 means

- `GET /fee-rate?token_id=<id>` (or `/fee-rate/{id}`) → `{"base_fee":1000}` on fee-enabled markets and **`{"base_fee":0}` on fee-free** ones (Putin market). Missing `token_id` gives 400 `Invalid token id`; an unknown token gives 404 `fee rate not found for market`.
- The spec's `FeeRate.base_fee` is "Base fee in basis points". `py-clob-client-v2` documents `taker_base_fee` as "Taker base fee in bps (from tbf field, e.g. 1000 = 10%)". Gamma `takerBaseFee`/`makerBaseFee` and CLOB `tbf`/`mbf` carry the same 1000.
- It is **not the effective fee**. The Xi market charges 4% × p(1−p), not 10%. In `clob-client-v2`, `getFeeRateBps()` is used only by `_resolveFeeRateBps()`, which checks a user-supplied `feeRateBps` (a field V2 removed from orders). Treat it as a **legacy V1 value and on/off indicator**, and possibly a protocol ceiling (unconfirmed). **Never use it in fee math.**

### 5.5 Rebates (not needed for taker routing)

The maker rebate is `rebateRate` × taker fees, paid daily and pro rata by "fee_equivalent = C × feeRate × p × (1 - p)" (https://docs.polymarket.com/programs/maker-rebates.md). There is also a volume-tiered taker rebate program (https://docs.polymarket.com/programs/taker-rebates.md). Both pay out after the fact, so ignore them in deterministic routing.

---

## 6. Polymarket US (CFTC-regulated): a separate API

- Polymarket US is QCX LLC, a CFTC Designated Contract Market. It has separate docs at https://docs.polymarket.us (index: https://docs.polymarket.us/llms.txt), a separate gateway `https://gateway.polymarket.us` (`/v1/markets`, `/v1/market/slug/{slug}`, `/v1/markets/{slug}/book`, `/v1/markets/{slug}/bbo`, …), and separate KYC'd API keys from polymarket.us/developer. Its quickstart says: "No authentication required for public endpoints."
- `GET https://gateway.polymarket.us/v1/markets?limit=2&active=true&closed=false` returned 200 unauthenticated with a different schema: `{"markets":[{id:"7900", slug:"tec-mlb-nlchamp-…", marketSides:[…], …}]}`. Sample: `research/raw/polymarket_us_gateway_markets_sample2.json`.
- Its fee formula has the same shape but a different coefficient and a maker rebate (https://docs.polymarket.us/fees.md): "Fee = Θ × C × p × (1 - p)", taker Θ = 0.0695, maker rebate Θ = −0.0125, effective 2026-10-01.
- **Equinox uses the public global Gamma + CLOB read APIs.** Polymarket US is a separate venue, with different markets, ids, books and fees. It is out of scope, but it could be added later as a third venue adapter.

---

## 7. Recommended ingest recipe (Go, stdlib)

1. Page `GET /markets/keyset?closed=false&limit=100&order=volume24hr&ascending=false[&tag_id=..][&end_date_min=..]` to the desired depth with `after_cursor`. Set a `User-Agent` and use a 30 s timeout. Stay under 30 req/s (the Gamma `/markets` limit is 300/10 s).
2. Filter: `active && !closed && enableOrderBook && acceptingOrders && umaResolutionStatus=="" && endDate > now`. Also drop `negRiskOther` and members of `negRiskAugmented` events that have `active:false`.
3. Decode `outcomes`, `outcomePrices` and `clobTokenIds` with a second `json.Unmarshal` from the string. Keep token ids as strings. Use `positionIds` if `version=="v2"`.
4. Read fees from `feesEnabled` + `feeSchedule` (or `GET /clob-markets/{conditionId}` → `fd`). The fee is 0 if either is absent or rate is 0.
5. Fetch books with `POST /books` in chunks of ≤500 token ids (both YES and NO, or YES only and derive NO by mirroring). Index by `asset_id`, reverse each side or read it from the end, and parse prices and sizes with exact decimal parsing (e.g. integer micro-units).
6. Respect `tick_size`/`min_order_size` from the book (they can differ from the Gamma snapshot).

---

## Sources

- Fees: https://docs.polymarket.com/trading/fees.md
- Market fields and fee fields: https://docs.polymarket.com/market-data/market-details.md
- Books (ordering, batch max 500): https://docs.polymarket.com/market-data/prices-order-books.md
- Listing: https://docs.polymarket.com/market-data/discover-markets.md
- Rate limits: https://docs.polymarket.com/api-reference/rate-limits.md
- Negative risk: https://docs.polymarket.com/concepts/negative-risk.md
- Prices and orderbook: https://docs.polymarket.com/concepts/prices-orderbook.md
- Place orders (tick and min size): https://docs.polymarket.com/trading/place-orders.md
- Changelog: https://docs.polymarket.com/changelog/predictions.md
- V2 migration (fee model, `getClobMarketInfo`): https://docs.polymarket.com/v2-migration.md and https://docs.polymarket.com/migrate/polymarket-v2/api-integrations.md
- Maker rebates: https://docs.polymarket.com/programs/maker-rebates.md
- OpenAPI: https://docs.polymarket.com/api-spec/gamma-openapi.yaml, https://docs.polymarket.com/api-spec/clob-openapi.yaml
- CLOB market info reference: https://docs.polymarket.com/api-reference/markets/get-clob-market-info.md
- SDK source (public GitHub): https://github.com/Polymarket/ts-sdk, https://github.com/Polymarket/clob-client-v2, https://github.com/Polymarket/py-clob-client-v2, https://github.com/Polymarket/py-sdk
- Polymarket US: https://docs.polymarket.us/llms.txt, https://docs.polymarket.us/fees.md, https://docs.polymarket.us/api-reference/market/overview.md

## Raw captures (`research/raw/`)

- `polymarket_gamma_markets_legacy_sample3.json` + `polymarket_gamma_markets_keyset_sample3.json` (+ `.headers`)
- `polymarket_keyset_census/page_00..59.json`: 6,000 open markets, id asc
- `polymarket_keyset_newest/page_00..09.json`, `polymarket_keyset_top_volume24hr/page_00..09.json`
- `polymarket_events_keyset_sample/{oldest,newest}_page_*.json`, `polymarket_gamma_events_keyset_sample2.json`
- `polymarket_gamma_event_negrisk_democratic-presidential-nominee-2028.json`
- `polymarket_clob_book_xi_{yes,no}.json`, `polymarket_clob_book.headers`, `polymarket_clob_books_post_xi.json`, `polymarket_clob_books_post_500_topvolume.json`
- `polymarket_clob_fee_rate_xi_yes.json` (1000), `polymarket_clob_fee_rate_feefree_putin.json` (0)
- `polymarket_clob_clob-markets_{xi,feefree_putin,exp2_crypto15}.json`
- `polymarket_us_gateway_markets_sample2.json`

## Open questions

- The 5-dp fee rounding direction (half-up, banker's rounding or truncation) is undocumented. Round half-up and log it.
- The meaning of `base_fee`/`tbf` = 1000 bps in V2 (cap? legacy only?) is not documented beyond "base fee in basis points".
- For `exponent != 1`, the docs formula omits the exponent. The SDKs apply `(p(1−p))^exponent`. Only 1 stale market has `exponent:2`, so this is low impact.
- The exact total of open, order-book-enabled markets: estimated at 70k–85k, not crawled to exhaustion (capped at 60 pages).
- `orderMinSize` is called shares in place-orders and "minimum USDC notional" in the market-details SDK comment.
