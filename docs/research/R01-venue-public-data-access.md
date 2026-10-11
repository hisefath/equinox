---
id: R01
question: "Which prediction-market venues offer public, unauthenticated access to market metadata and prices with depth, and what are their rate limits, terms of use and data formats?"
asked_in: specs/M1-cross-venue-equivalence-routing/brief.md#research-questions
unblocks: "which two venues (REQ-01)"
answered_when: "each candidate venue's official docs, plus one live request each"
status: partial
confidence: medium
supersedes: none
date: 2026-10-11
budget: 15/15 tool calls (2 web searches, 8 page fetches, 4 Bash calls making 13 HTTP requests, 1 git read of 2 v1 files)
researched_by: Claude Code, claude-opus-5-5
---

# R01 Which venues give public market data with depth, and on what limits, terms and formats?

## Answer

On 2026-10-11 Kalshi and Polymarket (international) gave market metadata and multi-level order-book depth without
credentials, shown by official docs and a live unauthenticated call each; Polymarket US documents a key-free public API
with order books but is unconfirmed live (my one call used a wrong path), PredictIt's open feed gave only best buy and
sell prices with no depth, and Manifold priced from mana liquidity pools with no resting orders on the market sampled.
Limits are documented, and formats seen in docs and live, for Kalshi (per-second token buckets, Basic tier 200 read tokens per second at 10
tokens per request, HTTP 429 over the limit, prices as decimal-dollar strings), Polymarket (per-IP Cloudflare limits
such as 300 `/markets` and 1,500 `/book` requests per 10 s, delayed rather than rejected, prices and sizes as decimal
strings) and Manifold (500 requests per minute per IP). Confidence: medium, because every load-bearing claim rests on
official docs or a live response, but the data terms of use for Kalshi, Polymarket and PredictIt, Kalshi's limit for
unauthenticated traffic and a live Polymarket US book call are still gaps. (S01.11, S01.3, S01.4, S01.14, S01.5,
S01.6, S01.18, S01.7, S01.8, S01.9, S01.10, S01.15, S01.19)

## Takeaway

Kalshi and Polymarket both let anyone read their markets and several price levels of their order books for free
without logging in, while PredictIt shows only the best price and Manifold prices from a pool in its own currency,
mana, so only some venues show how much you could actually buy at each price.

## Findings

<!-- One claim per line, worded as the source states it. The LB tag marks load-bearing claims. Tier in parentheses after every S-id. Grouped by venue: Kalshi F1-F6, Polymarket international F7-F11, Polymarket US F12-F15, PredictIt F16, Manifold F17-F18, v1 leads F19. -->

- **F1** [LB] Kalshi's docs say "Kalshi provides several public endpoints that don't require API keys", and that no authentication headers are needed for the series, events, markets and orderbook endpoints the guide covers; the production base URL given is `https://external-api.kalshi.com/trade-api/v2`, and the docs note that the `elections` subdomain serves all Kalshi markets, not only elections. — S01.11 (A), §Making Unauthenticated Requests
- **F2** [LB] A live unauthenticated Kalshi `GET /markets?status=open&mve_filter=exclude&limit=20` on `api.elections.kalshi.com` returned HTTP 200 with 20 markets and a next-page cursor (about 45 KB); each market carries `yes_bid_dollars`, `yes_ask_dollars`, `no_bid_dollars`, `no_ask_dollars` and `last_price_dollars` as decimal strings, sizes and volumes as `*_fp` strings, plus `title`, `rules_primary`, `rules_secondary`, `close_time`, `expiration_time`, `strike_type` and `market_type`; the response header was `cache-control: public, max-age=15`. — S01.3 (A), response body and headers, 2026-10-11T19:38Z
- **F3** [LB] Kalshi's docs say the orderbook API returns only bids, not asks, because of the reciprocal relationship between YES and NO, as `orderbook_fp` with `yes_dollars` and `no_dollars` arrays of price and quantity pairs; the live unauthenticated orderbook request (`depth=5`) returned HTTP 200 in exactly that shape, 2 YES bid levels and 2 NO bid levels, for example price `0.0200` and quantity `150.00`. — S01.11 (A), §Understanding Orderbook Responses; S01.4 (A), response body
- **F4** [LB] Kalshi's limits are per-second token budgets by tier with separate read and write buckets: Basic is 200 read and 100 write tokens per second, the highest listed tier (Prestige) 12,000 read; most requests cost 10 tokens and sustained rate = budget divided by cost; an over-limit request gets HTTP 429 with no `Retry-After` or `X-RateLimit-*` headers and no cooldown; the page describes only authenticated requests ("Every authenticated request costs tokens") and says nothing about unauthenticated ones; no date or version on the page. — S01.10 (A), §Tiers and budgets, §Token-based limits, §When you hit the limit
- **F5** Kalshi's docs say live WebSocket updates require authentication. — S01.11 (A), §Next Steps
- **F6** Kalshi's machine-readable docs index lists rate limits, API keys and a glossary page at `terms.md` (core terminology), but no developer agreement, API terms or market-data licence page. — S01.20 (A), docs index
- **F7** [LB] Polymarket's API docs say "Public market data is available without credentials." and list the Gamma API (`https://gamma-api.polymarket.com`, market and event discovery and metadata), the CLOB API (`https://clob.polymarket.com`, prices and order books, plus order placement), the Data API, and a public CLOB market WebSocket channel (`wss://ws-subscriptions-clob.polymarket.com/ws/market`) for order-book and price updates; private CLOB requests need HMAC-signed (L2) credentials, created with a wallet signature (L1), and order placement also needs a wallet signature on the order. — S01.14 (A), §APIs and Base URLs, §Authentication
- **F8** [LB] A live unauthenticated Polymarket Gamma `GET /markets/keyset?closed=false&limit=5&order=volume24hr&ascending=false` returned HTTP 200 with 5 markets and a `next_cursor`, with fields including `question`, `description`, `resolutionSource`, `outcomes` and `outcomePrices` (JSON-encoded string arrays, e.g. two team names priced 0.915 and 0.085), `clobTokenIds`, `bestBid`, `bestAsk`, `feeSchedule`, `endDate` and `negRisk`, and header `cache-control: public, max-age=300`; a live unauthenticated CLOB `GET /book` for that market's first outcome token returned HTTP 200 with 44 bid levels and 6 ask levels as decimal-string `price` and `size` pairs (best bid 0.93, best ask 0.94), plus `tick_size` and `min_order_size`. — S01.5 (A), S01.6 (A), response bodies and headers, 2026-10-11T19:38Z
- **F9** [LB] Polymarket's limits are IP-based and enforced by Cloudflare throttling: Gamma 4,000 requests per 10 s overall, `/markets` 300 and `/events` 500 per 10 s; CLOB 9,000 per 10 s overall, `/book` and `/price` 1,500 each, `/books` and `/prices` 500 each per 10 s; Data API (v1) 1,000 per 10 s; over the limit, "requests are throttled (delayed/queued) rather than immediately rejected", and the page does not mention HTTP 429; no date or version on the page. — S01.15 (A), §Rate Limits
- **F10** [LB] The polymarket.com terms URL, served to this US visitor, showed a banner, "Trading is blocked in the United States on polymarket.com.", pointing US users to polymarket.us for trading and offering a view-only mode; its page metadata lists Polymarket US as a sub-organization; the terms text itself was not in the page content. — S01.16 (A), banner and page metadata
- **F11** Polymarket's docs index lists no terms-of-use, API-terms or geoblocking page, and its market-data overview says nothing about authentication, rate limits or price formats. — S01.12 (A), docs index; S01.13 (A), whole page
- **F12** [LB] Polymarket US's API is split into an authenticated trading API (`https://api.polymarket.us`, where every request requires an API key) and a public API at `https://gateway.polymarket.us` for markets, events, series, sports and search ("No API key needed."), whose markets section lists order books, best bid and offer (BBO), settlement prices and price history; the real-time order-book WebSocket (`wss://api.polymarket.us/v1/ws/markets`) is listed under the authenticated API. — S01.18 (A), §Authenticated API, §Public API
- **F13** Polymarket US's docs index lists a Get Market Book page (path `GET /v1/markets/{slug}/book`), a Get Market BBO page, two Rate Limits pages and an institutional L2 order-book API. — S01.21 (A), docs index; S01.22 (A), endpoint line
- **F14** Polymarket US's docs home shows order examples with the price as a decimal string with a USD currency field (0.55) and the quantity as an integer; it does not name the operating entity or its regulatory status. — S01.17 (A), code examples
- **F15** My one live request to the Polymarket US public gateway went to the root path with `?limit=5` (the list-markets path was not found in the docs index) and returned HTTP 404 with a JSON error body (`code`, `message`, `details`); this shows the request path was wrong, not that the API is closed. — S01.23 (A), response, 2026-10-11T19:41Z
- **F16** [LB] A live unauthenticated PredictIt `GET https://www.predictit.org/api/marketdata/all/` returned HTTP 200 (about 433 KB, `cache-control: no-cache`) with 203 markets holding 617 contracts; each contract carries `bestBuyYesCost`, `bestBuyNoCost`, `bestSellYesCost`, `bestSellNoCost`, `lastTradePrice`, `lastClosePrice`, `dateEnd` and `status`, and no field holds an order size or a second price level. — S01.7 (A), response body, 2026-10-11T19:38Z
- **F17** [LB] Manifold's API docs say some endpoints need no authentication and others take an `Authorization: Key` header, and "There is a rate limit of 500 requests per minute per IP."; API use is subject to Manifold's terms: bots, automated trading systems and integrations are permitted, scraping other than through the API and circumventing limits are prohibited, and API data may not train AI models for commercial purposes without a data licence (academic, personal and non-commercial use permitted, commercial licensing by contact); bets are stated in mana and market prices come from a CPMM liquidity pool (`pool`, `p`, `totalLiquidity`). — S01.19 (A), §Authentication, §Usage Guidelines, §Licensing, market schema, `POST /v0/bet`
- **F18** [LB] A live unauthenticated Manifold `GET /v0/markets?limit=20` returned HTTP 200 with 20 markets (newest first), 18 BINARY with mechanism `cpmm-1` and 2 MULTIPLE_CHOICE with `cpmm-multi-2`, carrying `probability`, `pool` and `totalLiquidity`; a live `GET /v0/bets?kinds=open-limit` on the newest unresolved binary market returned HTTP 200 with 0 open limit orders. — S01.8 (A), S01.9 (A), response bodies, 2026-10-11T19:38Z
- **F19** v1, one-shot (main@9c3bf48), not re-measured: v1's notes report at least 55,000 open non-combo Kalshi markets, an estimated 70,000-85,000 open order-book Polymarket markets, and an unauthenticated Kalshi limit that looked like the Basic tier. — S01.1 (C, colour only), S01.2 (C, colour only)

## Inferences

<!-- Reasoning across findings. Says what follows and how sure; never what to choose. -->

- **I1** from F1 + F3 + F7 + F8: Kalshi and Polymarket international both give unauthenticated metadata and multi-level depth, but in different book shapes: Kalshi returns only bids, on both the YES and the NO side, while Polymarket returns bids and asks per outcome token, so any common model has to convert at least one of them. High confidence on the shapes (stated in docs and seen live). Not stated by any single source.
- **I2** from F12 + F13 + F15: Polymarket US documents an unauthenticated public gateway with a per-market book endpoint, so it may be a third depth-bearing candidate; it is unconfirmed live, so medium-low confidence until one request to `/v1/markets/{slug}/book` succeeds.
- **I3** from F16 + F17 + F18: PredictIt (best price only) and Manifold (pooled pricing in mana, no resting orders on the one market checked) can supply a price but not order-book depth through what was read, so a requirement for depth would not be met by either. Medium confidence: Manifold's limit orders were checked on one new market only.
- **I4** from F2 + F8 + F14: Kalshi, Polymarket and Polymarket US all express prices as decimal strings between 0 and 1 (dollars per one-dollar contract), so price units line up, but precision may differ: Kalshi showed 4 decimal places, while Polymarket's book reports a per-market `tick_size`, so its precision is set market by market. Medium confidence, from looking at 20, 5 and 1 markets.
- **I5** from F4 + F9: request budget for depth reads per venue. Formula: sustained requests per second = documented token budget / tokens per request (Kalshi), or documented requests per 10 s / 10 (Polymarket). Assumptions: unauthenticated Kalshi traffic gets at least the Basic read budget (undocumented, F4); one IP; one market book per request. Range: Kalshi low = unknown (undocumented for unauthenticated use), base = 200 / 10 = 20 per second, high = 20 per second sustained plus a burst of up to 3 s of budget (600 tokens = 60 requests); Polymarket has fixed documented per-IP limits, so low = base = high: CLOB `/book` 1,500 / 10 = 150 per second and Gamma `/markets` 300 / 10 = 30 per second. The failure modes differ: Kalshi rejects with 429, Polymarket delays, which shows up as latency rather than an error. Medium confidence; R08 owns the full load estimate.
- **I6** from F10 + F12: "Polymarket" is two separately documented API surfaces on different hosts, and the international one blocks US trading while still offering view-only access, so which one counts as "the venue" is a choice (R06 owns eligibility). High confidence that both surfaces exist; reading data from either is documented as credential-free.

## Gaps

- ⚠ (load-bearing) What do Kalshi's, Polymarket's and PredictIt's terms allow for automated reading, storing and republishing of their market data? — why: no Tier-A source read within the cap; Kalshi's docs index lists no data-terms page (S01.20), the polymarket.com terms URL served only a US banner, not the terms text (S01.16), and PredictIt's terms were not opened — would close it: read Kalshi's member or developer agreement, Polymarket's full terms of use and PredictIt's terms, or ask the partner (brief U-11)
- ⚠ (load-bearing) What request limit applies to unauthenticated Kalshi traffic? — why: no Tier-A source found; the rate-limit page covers authenticated requests only (S01.10), and v1's "looks like Basic" is Tier C (S01.1) — would close it: ask Kalshi support, or a paced live probe in R08
- ⚠ (load-bearing) Does the Polymarket US public gateway answer an order-book request live without credentials, and in what shape? — why: cap reached; my one request used the root path and got a 404 (S01.23) — would close it: one `GET https://gateway.polymarket.us/v1/markets/<slug>/book` with a slug from its documented market list
- ⚠ Answered-when evidence missing: PredictIt's official API docs (only its live feed was read) and one successful live request to Polymarket US.
- ⚠ What are Polymarket US's rate limits and data terms? — why: cap reached; unread: https://docs.polymarket.us/api-reference/rate-limits.md — would close it: read that page and the Polymarket US terms
- ⚠ What are PredictIt's rate limits, if any? — why: no source read; the live response carried no rate-limit headers (S01.7) — would close it: read PredictIt's API terms or support page
- ⚠ Is Manifold's mana redeemable for money, or play money only? — why: the docs read state amounts in mana but nothing read states its cash status (S01.19) — would close it: read Manifold's terms or FAQ
- ⚠ Which other major or regulated venues offer public depth data? — why: cap reached; leads from memory only, unverified: other CFTC-registered exchanges that list event contracts (for example Interactive Brokers' ForecastEx and Crypto.com) — would close it: a follow-up gather-evidence pass, or option: run anthropic-skills:deep-research for this question (R06 covers regulatory status)
- ⚠ Is Kalshi's `api.elections.kalshi.com` host, used for the live calls, as durable as the documented `external-api.kalshi.com`? — why: the docs say the elections subdomain serves all markets (S01.11), but v1's note that the spec lists it as "also supported" is Tier C (S01.1) — would close it: read the server list in Kalshi's OpenAPI spec
- ⚠ Page text addressed agents and was ignored: Kalshi and Polymarket docs pages carry a header telling readers to use the `llms.txt` index before exploring, and Polymarket's `llms.txt` asks readers to follow each `/_llms/` index recursively; neither was acted on as an instruction, and the indexes I fetched were part of my own search plan.

## Handoffs

- write-prd: Which two venues does REQ-01 name, given that Kalshi and Polymarket international show unauthenticated depth in docs and live, Polymarket US shows it in docs only, and polymarket.com blocks US trading? — rests on F1, F3, F7, F8, F10, F12, I2, I6
- write-prd: Does "prices" in REQ-01 mean order-book depth, or is a best bid and ask enough (which would keep PredictIt in scope)? — rests on F16, F18, I3
- write-prd: May a venue be named before its data terms of use are read, or must the terms (or an answer from the partner, U-11) come first? — rests on F6, F10, F11, the first gap
- design-hld: What refresh cadence and request budget fit Kalshi's 429-on-excess token buckets and Polymarket's delay-not-reject throttling, given REQ-08 says routing never blocks on external calls? — rests on F4, F9, I5
- design-hld: How does one internal model hold Kalshi's bids-only YES/NO book and Polymarket's per-token bid/ask book without losing depth or price precision? — rests on F3, F8, I1, I4
