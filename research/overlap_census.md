# Live overlap census: Kalshi x Polymarket (snapshot 2026-10-06 03:53-03:55 UTC)

**Bottom line.** Overlap is common and matching works on real data. On a single snapshot I hand-labelled **55 cross-venue pairs**: **32 equivalent** and **23 hard negatives**. They cover 8 topic groups: Fed, US midterms, geopolitics, AI/tech, macro prints, crypto, MLB/NFL and entertainment. For equivalent pairs the two venues' mid prices sit within 0.075 of each other (mean gap 0.015).

Text similarity and price agreement cannot separate the two classes on their own. Token-Jaccard on titles averages 0.42 for equivalent pairs and 0.38 for negatives, and 13 of 23 negatives score at or above the equivalent median. 8 of 23 negatives have mid prices within 0.03 of each other. The matcher has to work on structured fields: strike type and bounds, date or window, resolution source, and outcome-to-token polarity.

Files:
- `labelled_pairs.json`: the labelled set. Required fields plus extras: topic, negative_type, Kalshi strike fields and bid/ask/mid, PM event id/slug, outcomes, `polymarket_yes_outcome_index`, CLOB token ids, conditionId, bid/ask/mid, fee flags, and rules excerpts from both venues.
- `raw/census/labelled_pairs_raw.json`: the full, unmodified Kalshi event+market and PM event+market objects for every pair. Use this as the unit-test fixture.
- `raw/census/kalshi/*.json.gz`: all 60 Kalshi pages. `raw/census/polymarket/*.json.gz`: the 15 PM keyset pages that contain a labelled event. `raw/census/MANIFEST.json` lists the exact request URL (with cursor) and the event/market counts for every page, including PM pages that were fetched but not kept. Total size is 23 MB, gzip -9, readable with Go `compress/gzip`.
- `raw/series_*.json`: Kalshi `/series/{ticker}` responses (settlement sources, fee_type). `raw/kalshi_orderbook_KXFEDDECISION-26OCT-H0.json` and `raw/polymarket_book_2589812_yes.json` are live books for equivalent pair #1, used to check that snapshot bid/ask match the books.

## 1. How the census was pulled (all unauthenticated, verified live)

**Kalshi**: `GET https://api.elections.kalshi.com/trade-api/v2/events?status=open&with_nested_markets=true&limit=200&cursor=<cursor>`
- Response is `{cursor, events[], milestones[]}`. Pass the previous page's `cursor` to get the next page; an empty cursor means the end.
- 60 pages = 12,000 events and 111,813 nested markets. The cursor was **not** exhausted after 60 pages, so Kalshi has more than 12k open events.
- **0** event tickers began with `KXMVE`: `/events` does not return the multivariate (MVE) events, so no filter is needed on this endpoint. The skip rule stays as a guard.
- `status=open` filters **events**, not markets. Nested markets included 2,549 `finalized`, 36 `inactive`, 13 `closed` and 1 `initialized` (109,214 `active`). Filter on `market.status == "active"`.

**Polymarket Gamma**: `GET https://gamma-api.polymarket.com/events?active=true&closed=false&limit=100&offset=N` **fails beyond offset 2000**. HTTP 422 returns `{"type":"validation error","error":"offset too large, use /events/keyset for deeper pagination"}`. Use instead:
`GET https://gamma-api.polymarket.com/events/keyset?active=true&closed=false&limit=100&order=volume24hr&ascending=false&after_cursor=<next_cursor>`
- Response is `{"$schema", "events": [...], "next_cursor"}`. The cursor param is **`after_cursor`**: `cursor`, `next_cursor` and `after` are silently ignored and return page 1 again.
- 60 pages = 6,000 events (5,998 unique, because volume24hr reordering during paging produced 2 duplicates; dedupe by event `id`) and 102,968 nested markets.
- An active event does **not** mean active markets. Only 80,374 markets had `closed=false, active=true`; 18,392 had `active=false` and 4,202 had `closed=true`. Filter at market level (`closed==false && active==true && acceptingOrders`).
- Keyset events carry more fields than offset events (sports: `gameId`, `score`, `teams`, `eventWeek`; markets: `sportsMarketType`, `line`, `gameStartTime`, `bestBid`, `positionIds`). The first keyset page was 13 MB raw: 405 MB for 60 pages versus 233 MB for Kalshi.
- Sorting by volume24hr pulls in heavy sports props (300+ markets per NFL game). Some long-dated futures (Super Bowl / NBA champion) were **not** in the top 6,000. Census recall is bounded by the sort order; to get a known event, fetch it by slug.

Live-book check, pair #1: Kalshi `GET /markets/KXFEDDECISION-26OCT-H0/orderbook` returns `{"orderbook_fp":{"yes_dollars":[[price,size]...],"no_dollars":[...]}}`. These are bids only, sorted ascending, so best YES ask = 1 - best NO bid = 0.80. Best YES bid is 0.79, which matches the snapshot. PM `GET https://clob.polymarket.com/book?token_id=<clobTokenIds[0]>` lists bids ascending and asks descending (best price is the **last** element) and gave 0.80/0.81, matching Gamma `bestBid/bestAsk`. Gotcha: that `/book` response's `last_trade_price` was "0.190", while `/last-trade-price` returned 0.81 and `/midpoint` returned 0.805. Do not trust `/book.last_trade_price`.

## 2. Labelled set: counts and coverage

| topic | equivalent | not_equivalent |
|---|---|---|
| fed | 5 | 3 |
| elections | 8 | 3 |
| geopolitics | 2 | 1 |
| tech | 2 | 2 |
| economics | 3 | 4 |
| sports | 10 | 4 |
| entertainment | 2 | 3 |
| crypto | 0 | 3 |

| negative_type | count |
|---|---|
| threshold_vs_bucket | 6 |
| different_date_window | 4 |
| different_resolution_source | 4 |
| different_threshold | 3 |
| different_proposition | 3 |
| related_but_different | 1 |
| inverse_polarity | 1 |
| boundary_inclusivity | 1 |

Label policy (strict, for routing): **equivalent** means the same proposition, the same threshold/bucket, the same date/window, and a compatible resolution source. Edge-case-only wording differences (tie-breaks, postponement handling, "sworn in" vs "declared winner") are recorded as caveats in `reason`. Different reference prices (CF BRTI vs Binance) are labelled **not_equivalent** even when economically close, because they carry basis risk. Change that policy explicitly if the router should accept it.

### Equivalent pairs (32)
| topic | Kalshi ticker (yes_sub_title) | PM market id (groupItemTitle) | mid K / PM | close K / end PM (UTC) | label |
|---|---|---|---|---|---|
| fed | `KXFEDDECISION-26OCT-H0` (Fed maintains rate) | `2589812` (No change) | 0.795 / 0.805 | 2026-10-28T17:59 / 2026-10-29T03:59 | EQ |
| fed | `KXFEDDECISION-26OCT-C25` (Cut 25bps) | `2589811` (25 bps decrease) | 0.005 / 0.0035 | 2026-10-28T17:59 / 2026-10-29T03:59 | EQ |
| fed | `KXFEDDECISION-26OCT-H25` (Hike 25bps) | `2589813` (25 bps increase) | 0.205 / 0.195 | 2026-10-28T17:59 / 2026-10-29T03:59 | EQ |
| fed | `KXFEDDECISION-26DEC-H0` (Fed maintains rate) | `3215007` (No change) | 0.26 / 0.225 | 2026-12-09T18:59 / 2026-12-09T23:59 | EQ |
| fed | `KXFOMCDISSENTCOUNT-26OCT-0` (0) | `4386085` (0) | 0.33 / 0.255 | 2026-10-28T17:59 / 2026-10-28T15:59 | EQ |
| elections | `CONTROLS-2026-D` (Democratic Party) | `562793` (Democratic Party) | 0.645 / 0.655 | 2027-02-01T15:00 / 2026-11-04T04:59 | EQ |
| elections | `CONTROLH-2026-D` (Democratic Party) | `562802` (Democratic Party) | 0.9145 / 0.925 | 2027-02-01T15:00 / 2026-11-04T04:59 | EQ |
| elections | `KXBALANCEPOWERCOMBO-27FEB-DD` (Democrats sweep) | `562828` (Democrats Sweep) | 0.645 / 0.655 | 2027-02-01T15:00 / 2026-11-04T04:59 | EQ |
| elections | `SENATETX-26-D` (James Talarico) | `630963` (James Talarico (D)) | 0.655 / 0.645 | 2027-11-03T15:00 / 2026-11-04T04:59 | EQ |
| elections | `SENATEME-26-R` (Susan Collins) | `630773` (Susan Collins (R)) | 0.445 / 0.415 | 2027-11-03T15:00 / 2026-11-04T04:59 | EQ |
| elections | `KXSENATELA-26NOV-R` (Julia Letlow) | `634879` (Julia Letlow (R)) | 0.946 / 0.93 | 2027-11-03T15:00 / 2026-11-04T04:59 | EQ |
| elections | `HOUSENY17-26-D` (Cait Conley) | `943441` (Cait Conley (D)) | 0.615 / 0.635 | 2027-11-03T15:00 / 2026-11-04T04:59 | EQ |
| elections | `HOUSEMI7-26-D` (William Lawrence) | `942722` (William Lawrence (D)) | 0.595 / 0.615 | 2027-11-03T15:00 / 2026-11-04T04:59 | EQ |
| geopolitics | `KXLEAVEMERZ-29JAN01-26NOV01` (Before Nov 1, 2026) | `4597026` (October 31, 2026) | 0.025 / 0.0155 | 2026-11-01T03:59 / 2026-11-01T03:59 | EQ |
| geopolitics | `KXLEADERSOUT-27JAN01-BNETISR` (Benjamin Netanyahu) | `567688` (December 31) | 0.425 / 0.39 | 2027-01-02T03:17 / 2027-01-01T04:59 | EQ |
| tech | `KXGEMINI-GEM4-26OCT16` (Before Oct 16, 2026) | `5084763` (October 15) | 0.685 / 0.71 | 2026-10-16T03:59 / 2026-10-16T03:59 | EQ |
| tech | `KXLLM1-26OCT31-GOOG` (Gemini) | `3554659` (Google) | 0.68 / 0.666 | 2026-10-31T14:00 / 2026-10-31T16:00 | EQ |
| economics | `KXECONSTATCORECPIYOY-26SEP-T2.5` (Exactly 2.5%) | `4469651` (2.5%) | 0.405 / 0.405 | 2026-10-14T12:29 / 2026-10-15T03:59 | EQ |
| economics | `KXECONSTATCPICORE-26SEP-T0.2` (Exactly 0.2%) | `4469522` (0.2%) | 0.355 / 0.36 | 2026-10-14T12:29 / 2026-10-15T03:59 | EQ |
| economics | `KXECONSTATU3-26OCT-T4.2` (Exactly 4.2%) | `5211635` (4.2%) | 0.28 / 0.32 | 2026-11-06T13:29 / 2026-11-06T13:30 | EQ |
| sports | `KXMLBSERIES-26SDMILNLDS-MIL` (Milwaukee) | `5215617` (MLB Playoffs: Who Will Win Series? - Brewers vs. Padres) | 0.895 / 0.885 | 2026-11-01T00:30 / 2026-10-11T03:59 | EQ |
| sports | `KXMLBSERIESSPREAD-26SDMILNLDS-MIL2` (Milwaukee -1.5 games) | `5215611` (Brewers (-1.5)) | 0.715 / 0.715 | 2026-11-01T00:30 / 2026-10-11T03:59 | EQ |
| sports | `KXMLBALCSQUAL-26-TB` (Tampa Bay) | `4175303` (Tampa Bay Rays) | 0.845 / 0.85 | 2026-11-08T15:00 / 2026-10-15T23:59 | EQ |
| sports | `KXMLBSERIES-26NYYTBALDS-TB` (Tampa Bay) | `4175303` (Tampa Bay Rays) | 0.825 / 0.85 | 2026-10-31T22:30 / 2026-10-15T23:59 | EQ |
| sports | `KXMLBAL-26-TB` (Tampa Bay) | `1395703` (Tampa Bay Rays) | 0.51 / 0.491 | 2028-10-31T04:00 / 2026-12-01T04:59 | EQ |
| sports | `KXMLB-26-MIL` (Milwaukee) | `1235568` (Milwaukee Brewers) | 0.2615 / 0.25 | 2028-10-31T04:00 / 2026-10-31T23:55 | EQ |
| sports | `KXNFLGAME-26OCT11BALATL-BAL` (Baltimore) | `4024681` (Ravens vs. Falcons) | 0.555 / 0.54 | 2026-10-14T00:20 / 2026-10-12T00:20 | EQ |
| sports | `KXNFLSPREAD-26OCT11BALATL-BAL8` (BAL Ravens wins by over 7.5 points) | `4061292` (Spread -7.5) | 0.29 / 0.29 | 2026-10-14T00:20 / 2026-10-12T00:20 | EQ |
| sports | `KXNFLTOTAL-26OCT11BALATL-44` (Over 43.5 points) | `4061313` (O/U 43.5) | 0.56 / 0.55 | 2026-10-14T00:20 / 2026-10-12T00:20 | EQ |
| sports | `KXNFLGAME-26OCT08TBDAL-TB` (Tampa Bay) | `3951230` (Buccaneers vs. Cowboys) | 0.195 / 0.195 | 2026-10-11T00:15 / 2026-10-09T00:15 | EQ |
| entertainment | `KXTOPSONG-26OCT17-CHO` (Choosin' Texas) | `5220225` (Choosin' Texas - Ella Langley) | 0.165 / 0.155 | 2026-10-12T03:59 / 2026-10-13T23:59 | EQ |
| entertainment | `KXNETFLIXRANKSHOW-26OCT05-ADI` (A Different World: Season 1) | `5178963` (A Different World: Season 1) | 0.995 / 0.9935 | 2026-10-06T03:59 / 2026-10-06T23:59 | EQ |

### Hard negatives (23)
| topic | Kalshi ticker (yes_sub_title) | PM market id (groupItemTitle) | mid K / PM | close K / end PM (UTC) | label |
|---|---|---|---|---|---|
| fed | `KXFEDDECISION-26OCT-H25` (Hike 25bps) | `3215008` (25 bps increase) | 0.205 / 0.745 | 2026-10-28T17:59 / 2026-12-09T23:59 | NEG: different_date_window |
| fed | `KXFOMCDISSENTCOUNT-26OCT-4` (4) | `4386089` (4+) | 0.05 / 0.0495 | 2026-10-28T17:59 / 2026-10-28T15:59 | NEG: different_threshold |
| fed | `KXFED-26DEC-T3.50` (Above 3.50%) | `1168138` (3.5%) | 0.98 / 0.016 | 2026-12-09T18:55 / 2027-01-01T04:59 | NEG: threshold_vs_bucket |
| elections | `CONTROLS-2028-D` (Democratic party) | `562793` (Democratic Party) | 0.55 / 0.655 | 2029-02-01T15:00 / 2026-11-04T04:59 | NEG: different_date_window |
| elections | `SENATELA-26-R` (Andy Barr) | `634879` (Julia Letlow (R)) | 0.964 / 0.93 | 2027-11-03T15:00 / 2026-11-04T04:59 | NEG: different_proposition |
| elections | `KXCAELECTION-2614-AWAH` (Aisha Wahab) | `1281029` (Democratic Party) | 0.825 / 0.9985 | 2027-11-03T15:00 / 2026-11-04T04:59 | NEG: different_proposition |
| geopolitics | `KXLEAVEMERZ-29JAN01-26NOV01` (Before Nov 1, 2026) | `5180553` (November 30, 2026) | 0.025 / 0.044 | 2026-11-01T03:59 / 2026-12-01T04:59 | NEG: different_date_window |
| tech | `KXGEMINI-GEM4-26NOV15` (Before Nov 15, 2026) | `3669876` (November 30) | 0.865 / 0.977 | 2026-11-15T04:59 / 2026-12-01T04:59 | NEG: different_date_window |
| tech | `KXLLM1-26OCT31-GOOG` (Gemini) | `3554764` (Google) | 0.68 / 0.6775 | 2026-10-31T14:00 / 2026-11-01T03:59 | NEG: different_resolution_source |
| economics | `KXCPICOREYOY-26SEP-T2.4` (Above 2.4%) | `4469649` (2.4%) | 0.59 / 0.325 | 2026-10-14T12:25 / 2026-10-15T03:59 | NEG: threshold_vs_bucket |
| economics | `KXCPICORE-26SEP-T0.2` (Above 0.2%) | `4469522` (0.2%) | 0.36 / 0.36 | 2026-10-14T12:25 / 2026-10-15T03:59 | NEG: threshold_vs_bucket |
| economics | `KXU3-26OCT-T4.2` (Above 4.2%) | `5211635` (4.2%) | 0.305 / 0.32 | 2026-11-06T13:29 / 2026-11-06T13:30 | NEG: threshold_vs_bucket |
| economics | `KXGDP-26OCT30-T2.5` (Above 2.5%) | `3247817` (2.5–3.0%) | 0.825 / 0.125 | 2026-10-30T12:29 / 2026-10-30T03:59 | NEG: threshold_vs_bucket |
| crypto | `KXBTCD-26OCT0917-T83999.99` ($84,000 or above) | `5207028` (84,000) | 0.705 / 0.72 | 2026-10-09T21:00 / 2026-10-09T16:00 | NEG: different_resolution_source |
| crypto | `KXBTCMAXMON-BTC-26OCT31-9500000` (Above $95,000.00) | `5170732` (↑ 95,000) | 0.235 / 0.255 | 2026-11-01T03:59 / 2026-11-01T04:00 | NEG: different_resolution_source |
| crypto | `KXBTCMINMON-BTC-26OCT31-8000000` (Below $80,000.00) | `5170738` (↓ 80,000) | 0.48 / 0.445 | 2026-11-01T03:59 / 2026-11-01T04:00 | NEG: different_resolution_source |
| sports | `KXMLBSERIESSPREAD-26SDMILNLDS-MIL3` (Milwaukee -2.5 games) | `5215611` (Brewers (-1.5)) | 0.46 / 0.715 | 2026-11-01T00:30 / 2026-10-11T03:59 | NEG: different_threshold |
| sports | `KXMLBNL-26-MIL` (Milwaukee) | `1235568` (Milwaukee Brewers) | 0.405 / 0.25 | 2028-10-31T04:00 / 2026-10-31T23:55 | NEG: related_but_different |
| sports | `KXNFLGAME-26OCT11BALATL-ATL` (Atlanta) | `4024681` (Ravens vs. Falcons) | 0.455 / 0.54 | 2026-10-14T00:20 / 2026-10-12T00:20 | NEG: inverse_polarity |
| sports | `KXNFLSPREAD-26OCT11BALATL-BAL8` (BAL Ravens wins by over 7.5 points) | `4842245` (Spread -8.5) | 0.29 / 0.275 | 2026-10-14T00:20 / 2026-10-12T00:20 | NEG: different_threshold |
| entertainment | `KXRT-OTH-50` (Above 50) | `5144164` (50+) | 0.44 / 0.515 | 2026-10-12T14:00 / 2026-10-12T23:59 | NEG: boundary_inclusivity |
| entertainment | `KXBILLBOARDRUNNERUPSONG-26OCT17-CHO` (Choosin' Texas) | `5220225` (Choosin' Texas - Ella Langley) | 0.83 / 0.155 | 2026-10-12T03:59 / 2026-10-13T23:59 | NEG: different_proposition |
| entertainment | `KXNETFLIXTOPVIEWSTV-26OCT05-6` (At least 6 million) | `5219400` (6-9M) | 0.93 / 0.981 | 2026-10-06T03:59 / 2026-10-06T23:59 | NEG: threshold_vs_bucket |

Notes on the less obvious labels:
- `KXMLBSERIES-26NYYTBALDS-TB` and `4175303` are equivalent across structures ("wins ALDS" means "advances to ALCS"). This is a deliberate false-negative probe.
- `KXNFLGAME-26OCT11BALATL-ATL` paired with PM `4024681` is the right game but the wrong token. It is equivalent only to PM outcome index 1 ("Falcons").
- `SENATELA-26-R` is **Kentucky** (title "Kentucky Senate winner?", rules "Senator of Kentucky"). The Louisiana race is `KXSENATELA-26NOV-R`.

## 3. Structural differences the matcher must handle

1. **Outcome → token polarity (PM).** `outcomes` and `clobTokenIds` are JSON-encoded *strings* inside the JSON. Index i of `outcomes` matches index i of `clobTokenIds`. Most markets are `["Yes","No"]`, but sports moneylines are `["Ravens","Falcons"]`, series spreads `["Brewers -1.5","Padres +1.5"]` and totals `["Over","Under"]`. `bestBid/bestAsk/lastTradePrice` refer to outcome index 0. Kalshi instead lists **one market per team** (`-BAL`, `-ATL`). The canonical model needs a `(market_id, outcome_index)` → YES mapping, not just a market id. `polymarket_yes_outcome_index` records it per pair.
2. **Multi-outcome decomposition.** Kalshi event to markets: the bucket label is in `yes_sub_title` and ticker suffixes (`-H0`, `-C25`, `-T2.5`, `-BNETISR`). PM event to markets: the bucket label is in `groupItemTitle`, and `negRisk` events are mutually exclusive sets that can include an "Other" leg (e.g. Balance of Power has 5 legs vs Kalshi's 4). Kalshi sometimes packs unrelated people into one event (`KXLEADERSOUT-27JAN01` has 34 leaders) where PM has one event per person (`34051` Netanyahu, `73223` Merz...). Match at **market** granularity and use event titles only as context. PM `groupItemTitle` is missing or empty on 1,766 markets (single-market events) and has leading/trailing whitespace on 36 (e.g. `" ↑ 5.0%"`).
3. **Threshold vs exact bucket vs one-touch.** Kalshi encodes semantics in `strike_type` ∈ {greater, greater_or_equal, less, less_or_equal, between, custom, structured} plus `floor_strike`/`cap_strike`.
   - The same topic exists in both shapes: `KXCPICOREYOY-26SEP-T2.4` (greater 2.4) vs `KXECONSTATCORECPIYOY-26SEP-T2.5` (custom, "Exactly 2.5%"). `KXU3-26OCT` vs `KXECONSTATU3-26OCT`.
   - PM puts the shape in titles: `"2.4%"` (exact), `"≤2.9%"`, `"2.5–3.0%"` (range, with "boundary → higher bucket"), `"40+"` (at least), `"↑ 95,000"` / `"↓ 80,000"` (one-touch high/low within the window), `"Spread -7.5"`, `"O/U 43.5"`. Parse groupItemTitle with unicode arrows and en-dashes.
4. **Boundary inclusivity.** Kalshi "Above X" is strict, for example RT rules: *"a score of 75 would resolve 'Above 75' as No"*. PM "X+" means at least X. These match only for half-point or continuous strikes. Kalshi encodes "$84,000 or above" as `greater` with `floor_strike: 83999.99`; normalise by rounding to the tick, then use `strike_type` to decide `>` vs `>=`.
5. **Dates and windows.**
   - Kalshi uses exclusive "before Nov 1, 2026"; PM uses inclusive "by October 31, 2026, 11:59 PM ET".
   - Kalshi `close_time` is often far after the event: `CONTROLS-2026` closes 2027-02-01, `KXMLB-26` 2028-10-31, House races 2027-11-03.
   - PM `endDate` is nominal. It can fall **before** the event (dissent count `endDate` 15:59Z vs the 18:00Z FOMC announcement; World Series `endDate` 2026-10-31T23:55Z vs a possible November finish).
   - Neither field is safe as the "event time" key. Use a date parsed from the rules or title and allow a tolerance window.
   - Ticker date tokens are in ET (`KXNFLGAME-26OCT08TBDAL`), while PM slugs use the UTC date (`nfl-tb-dal-2026-10-09`, `gameStartTime 2026-10-09 00:15:00+00`).
   - Kalshi Netflix `-26OCT05` is neither the chart publish date (Oct 6) nor the viewing week. Kalshi `KXSB-27` "2027 Pro Football Champion" is the 2026 season.
6. **Same day, different time or index.** BTC "above $84k on Oct 9": Kalshi uses the 60-second BRTI average before **5pm EDT**; PM uses the Binance BTC/USDT 1m close at **12:00 ET**. Monthly one-touch: Kalshi uses a trimmed-mean BRTI; PM uses any Binance 1m High/Low. AI leaderboard: Kalshi `KXLLM1` series `settlement_sources` = arena.ai `overall-no-style-control`. PM runs two near-identical events, one with style control off (`841230`, equivalent) and one with style control on (`841241`, not). Kalshi exposes the source machine-readably via `GET /series/{series_ticker}` → `settlement_sources[].url`. PM's `resolutionSource` is usually empty, so the source has to be extracted from `description`.
7. **Naming normalisation.**
   - Kalshi avoids league trademarks: "Pro Baseball", "Pro Football Championship", cities with disambiguating letters ("Los Angeles D", "New York Y", "Chicago WS", "New York J").
   - PM uses team nicknames ("Dodgers", "Rays"), appends party "(D)/(R)" and "- Artist", and uses company where Kalshi uses product ("Google" vs "Gemini").
   - District formatting differs ("MI-7" in the Kalshi title vs "MI-07"). Kalshi copy has typos ("Democratics").
   - Kalshi party contracts show the nominee in `yes_sub_title` ("James Talarico") although the rules are party-based.
   - Kalshi `yes_sub_title` sometimes carries invisible U+3164 HANGUL FILLER (e.g. `KXCANCUTS-26DEC31-E0` "Exactly 0ㅤ"). NFKC alone does not remove it; strip it explicitly.
8. **Ticker semantics are unreliable.** Legacy non-`KX` series still exist (166 series, e.g. `CONTROLS`, `SENATE*`, `HOUSE*`, `GOVPARTY*`) alongside `KX*` ones. `SENATELA-26` is the Kentucky race. Use ticker parsing only as a hint and confirm against the title and rules.
9. **Sibling-cycle and sibling-rank traps.** `CONTROLS-2028` vs `CONTROLS-2026` share near-identical copy. Billboard `KXTOPSONG` (#1) vs `KXBILLBOARDRUNNERUPSONG` (#2) differ only by rank, for the same song and week. `KXMLBNL-26` (pennant) vs `KXMLB-26` (title).
10. **Prices and fees as matcher features.** Kalshi prices are 4-dp dollar strings (`yes_bid_dollars`); PM uses floats, and `bestBid` is **omitted** when there is no bid (present on 80,281 of 102,968 markets). Equivalent pairs agree within 0.075; use a large price disagreement only as a red flag, never as positive evidence. Fees per venue: PM markets carry `feesEnabled`, `feeType` (e.g. `finance_prices_fees`) and `feeSchedule {exponent, rate, takerOnly, rebateRate}`; Kalshi `/series/{t}` carries `fee_type` (`quadratic`, `quadratic_with_maker_fees`) and `fee_multiplier`.

## 4. Coverage and gaps

All topics requested in the brief were found on both venues:
- Fed (Oct/Dec decision, dissents, rate level)
- 2026 midterms (Senate/House control, sweep, TX/ME/LA Senate, NY-17/MI-07/CA-14)
- MLB postseason (NLDS/ALDS, pennants, World Series) and NFL week 5/6 (moneyline, spreads, totals)
- BTC daily and monthly
- CPI, U-3 and GDP prints
- Geopolitical leaders (Merz, Netanyahu)
- AI/tech (Gemini 4, LMArena)
- Entertainment (Billboard, Netflix, Rotten Tomatoes)

NBA and Super Bowl futures exist on Kalshi (`KXNBA-27`, `KXSB-27`) but their PM events fell outside the volume-sorted 6,000-event sample, so they are not labelled. Other overlapping families seen but not labelled include Fed cuts/hikes counts, end-of-year Fed rate, Zelenskyy/Putin/Macron/Meloni/Modi/Takaichi "out by", Netflix #2 rankings, NFL games across the whole week, and ETH thresholds. Zelenskyy was left out on purpose: Kalshi counts an "announced intention to leave", while PM requires ceasing office or an announced resignation, and the mids diverge (0.083 vs 0.045). It is a good candidate for a third label, `ambiguous`.
