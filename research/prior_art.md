# Project Equinox: prior art and methodology for cross-venue market equivalence and smart order routing

Compiled 2026-10-05 (US/Eastern). Snapshots taken 2026-10-06 ~03:00–04:05 UTC. All market facts below come from unauthenticated public endpoints (Kalshi `api.elections.kalshi.com/trade-api/v2`, Polymarket `gamma-api.polymarket.com` and `clob.polymarket.com`). Raw JSON is under `research/raw/`, and each claim names the file it came from. Web sources are cited inline and listed at the end.

---

## 0. Summary for Equinox design

1. **Nobody has solved general cross-venue matching deterministically.** Production systems use one of two approaches:
   - **Structured-identifier joins for templated products**, mainly sports. Dome's matcher was sports-only and keyed on `nfl-ari-den-2025-08-16` ↔ `KXNFLGAME-25AUG16ARIDEN`.
   - **Embeddings + LLM verification for everything else** (pmxt Router; Gebele & Matthes 2026; Saguillo et al. 2025).
   
   The open-source bots are much weaker. They use fuzzy string thresholds (Jaccard+Levenshtein ≥0.7, `difflib` ≥0.5) or build slugs from the clock, and none of them check resolution rules.
2. **"Same title" does not mean "same contract".** Live markets today that look identical differ in oracle (Binance BTCUSDT vs CF Benchmarks BRTI), weather station (LaGuardia vs Central Park), bracket edges (offset by 1°F), comparator (`>` vs `≥`), and postponement clauses. Historical markets have resolved in opposite directions: the 2024 shutdown (Kalshi **No**, Polymarket **Yes**) and Khamenei 2026 (Kalshi **$0.02 scalar** under a death carve-out, Polymarket **Yes**). This supports a **strict, veto-first** equivalence definition.
3. **Recommended matcher (deterministic, Go stdlib only):**
   1. normalize the text
   2. extract structured fields (entity, threshold, comparator, date/time window, timezone, oracle/source, bracket bounds)
   3. block with an inverted index on rare tokens, plus a date bucket
   4. score with TF-IDF cosine and Fellegi–Sunter-style per-field log-weights
   5. apply **hard vetoes**
   6. assign one-to-one with greedy mutual-best and deterministic tie-breaks
   7. accept above a precision-tuned threshold
   8. emit an explanation record for every pair
   
   Evaluate on a hand-labelled pair set that includes hard negatives. Report precision and recall separately (not F1) and pick the threshold for a target precision.
4. **Routing:** prediction markets have no Reg NMS. There is no order-protection rule, no consolidated tape, and no access-fee cap, so Equinox has to build its own **virtual consolidated book** and route on the **all-in price**: price + venue fee, with the fee depending on price as `rate·p·(1−p)` on both venues. Add penalties for staleness and partial fills. Log every considered and rejected venue. This is the same "best execution" approach the SEC now favors as it proposes rescinding Rule 611 (June 2026).

---

## 1. Prior art: cross-venue aggregators, arbitrage scanners and research

### 1.1 Commercial and hosted products

| Product | What it is | How it matches markets | Evidence / notes |
|---|---|---|---|
| **Dome** (YC; acquired by Polymarket; APIs end-of-life 2026-04-28) | Unified API for Polymarket and Kalshi | **Sports only.** `GET https://api.domeapi.io/v1/matching-markets/sports` takes `polymarket_market_slug` (e.g. `nfl-ari-den-2025-08-16`) or `kalshi_event_ticker` (e.g. `KXNFLGAME-25AUG16ARIDEN`). There is also a by-sport+date variant. The response maps each anchor to `[{platform:"KALSHI", event_ticker, market_tickers[]}, {platform:"POLYMARKET", market_slug, token_ids[]}]`. Matching is effectively a join on league + team codes + date. | `raw/dome_docs_matching_markets_sports_openapi.md`; [docs.domeapi.io](https://docs.domeapi.io/api-reference/endpoint/get-matching-markets-sports.md); EOL notice via [YC/Sacra](https://sacra.com/research/kurush-dubash-dome-unified-api-for-prediction-markets/) |
| **pmxt** (MIT SDK "CCXT for prediction markets", 2.1k GitHub stars; hosted "Router") | Unified Event → Market → Outcome schema across about 16 venues. Positions itself as a drop-in replacement for Dome (`npx dome-to-pmxt`). | `GET https://api.pmxt.dev/v0/matched-market-clusters` (also `matched-event-clusters`) returns "semantically matched" clusters. Each edge has a **relation** (`identity`, `subset`, `superset`, `overlap`, `disjoint`) and a **confidence** in [0,1]. Parameters include `minConfidence`, `minVenues`, `withOrderbook`, `includeRawMatches`, and `edgeLimit`. Pairwise edges are clustered into connected components. | `raw/pmxt_docs_router_matching.md`; [pmxt.dev/docs/router/matching](https://pmxt.dev/docs/router/matching); [github.com/pmxt-dev/pmxt](https://github.com/pmxt-dev/pmxt). **Caveat:** the doc's own `identity` example, "BTC > $100k by Dec 31 on Polymarket and Kalshi", is questionable because the two venues settle crypto on different oracles (see §2.2). |
| **Oddpool** (YC Spring 2026) | Cross-venue dashboards; arbitrage scanner across Kalshi, Polymarket and Opinion, net of fees | Not disclosed. Its "normalizes probability distributions" positioning suggests mapping at the event level. | [QuickNode listing](https://www.quicknode.com/builders-guide/tools/oddpool-by-oddpool-inc) |
| **Prediction Hunt / prediction.com** | Arbitrage API and blog | Describes doing it yourself as "fuzzy title matching, resolution-criteria comparison, ongoing maintenance"; their API reconciles server-side. | Ranks **resolution mismatch** as the top risk, then fees (gaps under 3¢ often vanish), leg slippage, and capital lock-up from USD vs USDC settlement. [prediction.com blog, 2026-07-20](https://prediction.com/blog/kalshi-vs-polymarket-arbitrage) |
| **PolyRouter** | Unified data API across 7 venues (Polymarket, Kalshi, Manifold, Limitless, ProphetX, Novig, SX.bet) with standardized sports game/league IDs | No general market-equivalence endpoint is documented. Sports normalization uses league and game identifiers. | [docs.polyrouter.io](https://docs.polyrouter.io/index.md) |
| **Verso** | "Bloomberg terminal" analytics over Kalshi + Polymarket | Analytics layer; no public matching method | [pm.wiki/projects/verso](https://pm.wiki/zh/projects/verso) |
| **Apify "Polymarket–Kalshi Arbitrage Verifier"** | Verifies a user-supplied or discovered pair | Walks both books to equal size for a VWAP, applies current fee formulas, and re-tests one tick worse. Raises **semantic conflict flags** for close-time gaps (`maxCloseDifferenceHours` default 48), threshold-boundary direction, observation time, source, and cancellation clauses. It labels profit `net_pnl_if_rules_equivalent_usd` so it is not read as risk-free. | [apify.com/redfoxxie/polymarket-kalshi-arbitrage-verifier](https://apify.com/redfoxxie/polymarket-kalshi-arbitrage-verifier.md). This is the closest public example of a strict-equivalence mindset. |

### 1.2 Open-source arbitrage bots (code read via the GitHub API)

| Repository | Matching method | Failure modes (stated in the repo or observed in code) |
|---|---|---|
| [realfishsam/prediction-market-arbitrage-bot](https://github.com/realfishsam/prediction-market-arbitrage-bot) (built on pmxt) | "fuzzy matching (Jaccard + Levenshtein distance)", `matchingThreshold` default **0.7** | README admits it "ignores gas fees, trading fees, and slippage". Resolution-rule differences are not addressed. |
| [ImMike/polymarket-arbitrage](https://github.com/ImMike/polymarket-arbitrage) `core/cross_platform_arb.py` | `difflib.SequenceMatcher` text similarity (`min_similarity` 0.5; config default 0.6), regex entity extraction (numbers, capitalized names), sports team + date heuristic | `dates_match()` returns **True when either date is missing** ("don't penalize"). Year defaults to `'2024'` when absent. Numbers are extracted but never used as a veto. These are textbook false-positive sources. |
| [CarlosIbCu/polymarket-kalshi-btc-arbitrage-bot](https://github.com/CarlosIbCu/polymarket-kalshi-btc-arbitrage-bot) | **Clock-derived identifiers.** It builds the Polymarket slug `bitcoin-up-or-down-{month}-{day}-{year}-{h}{am/pm}-et` and the Kalshi ticker for hour+1 ("Kalshi seems to use the *next* hour"). It then pairs Poly `Up`/`Down` with Kalshi strikes, assuming "Up means Price >= Poly_Strike" and "Kalshi Yes means Price >= Kalshi_Strike". | Treats as risk-free two contracts that settle on **different oracles and different statistics**: a Binance BTC/USDT 1h candle close ≥ open, versus the CF Benchmarks BRTI 60-second average > strike (verified live in §2.2). The "arbitrage" is basis risk. |
| ArbiBot (sports/esports), CraftyGeezer "Kalshi-Polymarket-Ai-bot" | "Intelligent name matching" and "AI validation" respectively | READMEs give no detail. |

### 1.3 Academic work

- **Saguillo, Ghafouri, Kiffer, Suarez-Tangil, "Unravelling the Probabilistic Forest: Arbitrage in Prediction Markets"** (arXiv [2508.03474](https://arxiv.org/abs/2508.03474); AFT 2025, [LIPIcs.AFT.2025.27](https://drops.dagstuhl.de/storage/00lipics/lipics-vol354-aft2025/LIPIcs.AFT.2025.27/LIPIcs.AFT.2025.27.pdf)).
  - Scope: Polymarket only. They separate **market-rebalancing arbitrage** (YES prices within one market don't sum to $1) from **combinatorial arbitrage** (dependent conditions across markets).
  - Dependency detection: heuristic reduction (same end date, topic clusters from **Linq-Embed-Mistral** embeddings), then **DeepSeek-R1-Distill-Qwen-32B** extracts logical dependencies. Markets with more than 4 conditions are truncated to the top 4 by volume plus a catch-all.
  - LLM quality: 97% valid JSON and **81.45% correct** on 128 NegRisk markets. Their stated limitation: the "LLM cannot handle too many conditions at a time".
  - Of **374** candidate dependent pairs, manual checks confirmed only **13** true combinatorial-arbitrage pairs. LLM candidate generation is high-recall and low-precision.
  - Realized arbitrage profit was about **$39.6M** (Apr 2024 – Apr 2025, 10,237 markets).
- **Gebele & Matthes, "Semantic Non-Fungibility and Violations of the Law of One Price in Prediction Markets"** (arXiv [2601.01706](https://arxiv.org/abs/2601.01706), Jan 2026). This is the most relevant matching methodology.
  - Data: more than 100k events across 10 venues (Polymarket, Kalshi, PredictIt, Futuur, Augur v1, Myriad, Limitless, Truemarkets, Seer, Omen); about 6% are listed on more than one venue.
  - **Blocking:** cross-platform only; same category from an LLM classifier over 20 categories; **non-empty overlap of validity windows**.
  - **Retrieval:** OpenAI `text-embedding-3-large` over title, description, outcome labels and resolution metadata, top-**k=20** neighbors. This captured >99.9% of verified equivalent or subset relations.
  - **Verification:** two LLM passes, first a plausibility check, then a structured comparison of the **YES-regions** (equivalent vs subset). This brought false positives below 2%. Human validation gave κ=0.94.
  - Output: 1,501 equivalence classes (6,709 relations), 1,645 subset sets, 1,123 neg-risk constructions.
  - **Non-fungibility taxonomy:** (i) oracle/source, (ii) temporal scope/cutoff (their example is Kalshi's Central Park NOAA station vs Polymarket's LaGuardia, still true today, §2.2), (iii) exception rules (invalidation, disputes, multi-stage finality, amendments).
  - Equivalent markets show persistent **execution-aware deviations of 2–4%**, up to $0.07 in the 2024 Polymarket–Kalshi election markets.
- **Ng, Peng, Tao, Zhou, "Price Discovery and Trading in Modern Prediction Markets"** ([SSRN 5331995](https://papers.ssrn.com/abstract=5331995), Apr 2026).
  - Compares Polymarket, Kalshi, PredictIt and Robinhood for the 2024 election. Polymarket **leads** Kalshi in price discovery, and disparities persist even in the most liquid pair.
  - They note a **strict subset relation**: Polymarket and Limitless resolve on media network calls, while Kalshi and Myriad resolve on inauguration.

### 1.4 Matching approaches compared

| Approach | Where used | Strengths | Failure modes reported or observed |
|---|---|---|---|
| Manual mapping table | Most human arbitrageurs; "maintenance" in Prediction Hunt's DIY description | Highest precision | Doesn't scale; goes stale when venues relist (Polymarket slugs get numeric suffixes such as `fed-decision-in-october-20260617190323537`) |
| Structured-identifier join (slug/ticker parsing) | Dome sports; CarlosIbCu BTC | Deterministic, cheap | Date conventions differ. Polymarket `nfl-tb-dal-2026-10-09` uses the **UTC** date of an 8:15 PM ET Oct 8 game, while Kalshi uses `KXNFLGAME-26OCT08TBDAL`. Hour naming also differs: Poly hourly slugs name the candle *start*, Kalshi names the settlement time. A join also says nothing about the rules. |
| Fuzzy string (Jaccard / Levenshtein / difflib) | OSS bots | Easy | Numbers, dates and negation are invisible to edit distance ("above 85,000" vs "above 86,000" scores ~0.95). Thresholds such as 0.5–0.7 are arbitrary. |
| Embeddings + kNN | pmxt (implied), Gebele, Saguillo | High recall | Embeddings are insensitive to thresholds, dates and oracles, so they need a verifier |
| LLM verification | Gebele (FP <2%), Saguillo (81% correct), CraftyGeezer | Reads rules | Non-deterministic, costly, and weak on many-condition markets. Even then, Saguillo kept only 13 of 374 candidates. |

**Implication:** Equinox's deterministic matcher should copy the *structure* of the LLM pipelines (block → retrieve → compare YES-regions) but replace the LLM verifier with **explicit field extraction + vetoes**. It should accept lower recall in exchange for auditable precision.

---

## 2. Resolution-criteria divergence: why equivalence must be strict

### 2.1 Historical incidents where "the same" market resolved differently (verified against live APIs)

| Event | Kalshi | Polymarket | Divergence axis |
|---|---|---|---|
| **US government shutdown in 2024** | `SHUTDOWNBY-24`, "Government shutdown in 2024?". `rules_primary`: "If the government is shutdown at any 10:00 AM ET by 2024, then the market resolves to Yes." **`result: "no"`**, `settlement_ts 2025-01-01T07:28Z`, `volume_fp 4184398`. (`raw/kalshi_historical_markets_SHUTDOWNBY-24.json`) | `us-government-shutdown-before-2025`, "Will there be a US Government shutdown?". The description says it resolves Yes "if the acting President fails to sign the relevant bill(s) extending government funding by the applicable deadline(s), even if no government shutdown is explicitly announced." **`outcomePrices ["1","0"]` (Yes)**, closed 2024-12-25, $53.5M volume. (`raw/poly_gamma_event_us_government_shutdown_before_2025.json`) A late clarification banner on Dec 20 set a midnight deadline, and Biden signed on Dec 21 ([michaellwy, 2025-01-09](https://michaellwy.substack.com/p/wisdom-of-crowds-or-collective-delusion); [Monad blog](https://monad.xyz/blog/prediction-market-disputes)). | **Definition** (an observed OPM status vs a missed deadline counting as a shutdown) plus a **mid-life rule amendment**. Opposite outcomes. |
| **Khamenei out (Feb–Mar 2026)** | `KXKHAMENEIOUT-AKHA-26MAR01`, "Will Ali Khamenei leave office before 2026-03-01…". `rules_secondary` death carve-out: "If Ali Khamenei leaves solely because they have died… payouts… based upon the last traded price (prior to the death)." **`result: "scalar"`, `settlement_value_dollars: "0.0200"`**, volume 27.96M contracts. (`raw/kalshi_historical_markets_KXKHAMENEIOUT.json`) Followed by a class action and a fee/loss reimbursement ([Bloomberg Law](https://news.bloomberglaw.com/daily-tax-report-state/kalshi-sued-over-death-carveout-in-iran-leader-prediction-market); [Next Event Horizon, 2026-03-01](https://nexteventhorizon.substack.com/p/the-chaos-of-khamenei-prediction)). | `khamenei-out-as-supreme-leader-of-iran-by-february-28`: "removed from power… resigns, is detained, or otherwise loses his position…". **Yes** (`["1","0"]`) after `umaResolutionStatuses ["proposed","disputed","proposed","disputed"]`, $131M volume. The "by March 31" market also resolved Yes ($63M). (`raw/poly_gamma_event_khamenei_out_by_*.json`) | **Exception rule** (death carve-out; a Kalshi regulatory constraint) plus **oracle process** (UMA disputes). A Kalshi-YES / Poly-NO "hedge" lost almost everything. |
| **2024 US presidential election** | `PRES-2024-DJT`: "…or another representative of their party is **inaugurated**…". Settled `2025-01-20T18:04Z`. (`raw/kalshi_historical_markets_PRES-2024.json`) | `presidential-election-winner-2024`: resolved on AP, Fox and NBC calls; closed **2024-11-06**. ([Fortune](https://fortune.com/2024/11/04/polymarket-presidential-election-prediction-market-bets-payout-inauguration); [Axios](https://www.axios.com/2024/11/06/kalshi-election-presidential-prediction-markets-polymarket-payout)) | **Timing/source** (network call vs inauguration, about 75 days of capital lock) and **subject** (Kalshi pays on the *party's* representative; Polymarket on the *person*). This is a strict subset relation, not identity (Ng et al.). |
| **US National Bitcoin Reserve in 2025** | Required a designated reserve "comparable to the SPR", confirmed by the White House or NYT; a signature by Dec 31 counts | "Hold any amount of bitcoin" during 2025; "consensus of credible reporting" plus UMA vote | **Definition, source and timing** all differed. Prices were 37% vs 51%. ([michaellwy, 2025-05-27](https://michaellwy.substack.com/p/prediction-markets-cant-agree-on)) |
| **2025 shutdown length / end date** | `KXGOVSHUTLENGTH-26JAN01-*`: OPM notices "checked at 10:00 AM ET each day", first shutdown only. "More than 42 days" = yes, "More than 43 days" = no, settled 2025-11-13. (`raw/kalshi_historical_markets_KXGOVSHUTLENGTH-26JAN01.json`) | `how-many-days-will-the-federal-government-be-shut-down-in-2025` copies the OPM 10:00 AM rule almost word for word: buckets `1–3`, `4-9`, `10–29`, `30+` (30+ = Yes). `when-will-the-government-shutdown-end-545` ("first day… OPM announces… not shut down") resolved to `November 12-15`. A separate Nov-12 date market burned traders who equated the bill signing (Nov 12) with the OPM update (Nov 13) ([The Defiant](https://thedefiant.io/newsletter/defi-daily/end-of-gov-t-shutdown-brings-more-polymarket-drama)). | **Converged**, because both used the same oracle and checkpoint rule. However, the **bucket structures differ** (Kalshi uses a "more than N" ladder; Poly uses ranges), so equivalence exists only at the YES-region level (`Poly 30+` ≡ `Kalshi >29 days`), not market by market. |
| Venezuela "invasion" (Jan 2026) | n/a | Polymarket ruled that the Maduro capture raid was **not** an "invasion" (~$10.5M) ([DeFi Rate](https://defirate.com/?p=3283); [Quartz](https://qz.com/polymarket-says-us-raid-venezuela-not-invasion)) | Shows that **definitional words** ("invasion", "out", "shutdown") carry the risk, independent of any cross-venue issue. |

### 2.2 Look-alike pairs live today that are *not* equivalent (snapshots 2026-10-06)

| Pair | Kalshi | Polymarket | Why a strict matcher must reject (or downgrade to `overlap`) |
|---|---|---|---|
| **BTC "above $85,000" at 12 AM ET Oct 6** | `KXBTCD-26OCT0600-T84999.99`: `strike_type "greater"`, `floor_strike 84999.99`, `yes_sub_title "$85,000 or above"`. "simple average of the sixty seconds of CF Benchmarks' Bitcoin Real-Time Index (BRTI) before 12 AM EDT". Event `settlement_sources: CF Benchmarks`. (`raw/kalshi_events_KXBTCD_open_nested.json`) | `bitcoin-above-on-october-6-2026-12am-et`, `groupItemTitle "85,000"`: "'Close' price for the BTC/USDT 1 hour candle… **higher than** the price specified", source **Binance** BTC/USDT. (`raw/poly_gamma_event_bitcoin_above_oct6_12am.json`) | Different **oracle** (a 60-second average of a USD composite index vs a single-venue USDT candle close) and different **comparator** (≥ 85,000.00 vs > 85,000). |
| **BTC hourly Up/Down** (the CarlosIbCu pairing) | Kalshi uses the strike ladder above | `bitcoin-up-or-down-october-6-2026-12am-et`: "Up if the close price is **greater than or equal to the open price** for the BTC/USDT 1 hour candle that **begins** on the time…" `eventStartTime 04:00Z`, `endDate 05:00Z`, outcomes `["Up","Down"]`. The 15-minute variant `btc-updown-15m-1791258300` uses a **Chainlink BTC/USD TWAP** stream. | The underlying statistic (a relative move vs a fixed strike) and the oracle both differ, and so does the time naming convention (candle start vs settlement time). Outcome labels are not Yes/No. |
| **NYC daily high, Oct 6** | `KXHIGHNY-26OCT06`: "maximum temperature recorded at New York City (**CLINYC**)… according to **The Weather Company**". Brackets: `62° or below`, `63° to 64°`, `65° to 66°`… (the `T63` ticker means "less than 63°"). (`raw/kalshi_event_KXHIGHNY-26OCT06_nested.json`) | `highest-temperature-in-nyc-on-october-6-2026`: "recorded by NOAA at the **LaGuardia Airport Station**" (`weather.gov/wrh/timeseries?site=klga`), Weather Underground as fallback; if no data, the lowest bracket wins. Brackets: `60-61°F`, `62-63°F`, `64-65°F`… (`raw/poly_gamma_event_highest_temperature_nyc_oct6.json`) | Different **station**, **bracket edges offset by 1°F**, and different **missing-data fallback**. Live prices disagree: Kalshi `62° or below` bid/ask 0.63/0.65 vs Poly `62-63°F` 0.57/0.61. |
| **Fed Oct 2026: level vs change** | `KXFED-*`: "upper bound of the target federal funds rate… **greater than** X% following the… meeting" (a level ladder). (`raw/kalshi_markets_KXFED_open.json`) | `fed-decision-in-october-20260617190323537`: "amount of basis points the upper bound… is **changed by**" (a change bracket) | These are not the same contract. They map onto each other only by conditioning on the pre-meeting level, which is a derived relation rather than identity. |
| **Fed Oct 2026: decision vs decision** (the *good* candidate) | `KXFEDDECISION-26OCT-H0` "Fed maintains rate" (`rules_primary` "Hike of 0bps on October 28, 2026"). If the meeting is cancelled, "maintains" = Yes. Yes ask 0.80. (`raw/kalshi_events_KXFEDDECISION_open_nested.json`) | "No change" market (`negRisk true`, `feeType economics_fees`, `feeSchedule {rate:0.05, exponent:1, takerOnly:true, rebateRate:0.25}`). If no statement is released by the next meeting, "No change". Moves that are not multiples of 25bp are **rounded up** to the next bracket. Best ask 0.81. (`raw/poly_gamma_event_fed_decision_october_2026.json`) | `H0` ↔ "No change" is a strong identity candidate. The `Cut 25bps` ↔ `25 bps decrease` pair carries edge-case risk (a 12.5bp move: Poly rounds up, Kalshi doesn't say). **Data gotcha:** Kalshi's `rules_primary` for `C26`/`H26` reads "Cut of   25bps", with the `>` lost from the template, so parse `yes_sub_title` ("Cut >25bps"), not the rules text. |
| **NFL TB @ DAL** | `KXNFLGAME-26OCT08TBDAL-DAL` (one binary market per team). Tie → $0.50. Postponed: stays open if the game starts within **48h**, otherwise "**resolve to a fair price**". (`raw/kalshi_events_KXNFLGAME_open_nested.json`) | `nfl-tb-dal-2026-10-09` (the date is the **UTC** kickoff date). One market with outcomes `["Buccaneers","Cowboys"]`. Tie → 50-50. **Postponed → "remain open until the game has been completed"**; cancelled → 50-50. (`raw/poly_gamma_event_nfl-tb-dal-2026-10-09.json`) | Identity holds except for the postponement tail (a "fair price" mark vs waiting). Identifier dates are off by one day. Outcomes need mapping from team labels to YES-regions. |

### 2.3 A strict-equivalence definition for Equinox

Two binary contracts A (venue X) and B (venue Y) are **equivalent** only if **all** of the following hold. Otherwise they are classified `subset`, `superset`, `overlap` or `disjoint` (pmxt/Gebele taxonomy), and the router never treats them as fungible.

1. **Same subject entity**: person, team, index or asset, after alias normalization. Party vs candidate counts as a mismatch.
2. **Same proposition polarity and statistic**: level vs change, max vs close vs average, win vs advance. Up/Down and team-name outcomes are mapped to explicit YES-regions.
3. **Same threshold and comparator**: a numeric strike, compared exactly after normalizing units. `>` vs `≥` is a veto unless the venue tick makes them identical; for example Kalshi `84999.99 greater` ≡ "≥ 85,000.00" at cent precision, which still differs from Poly "> 85,000".
4. **Same bracket bounds** for range markets, with inclusive/exclusive edges.
5. **Same observation window and timezone**: the deadline or observation instant in UTC must agree within a tolerance of 0 for intraday products. A small tolerance (hours) may apply to "by date X" political markets, with a flag.
6. **Same resolution source class**: the same oracle, station or index (e.g. BRTI ≠ Binance ≠ Chainlink; KLGA ≠ CLINYC). An unknown source on either side means *not equivalent*.
7. **Compatible exception clauses**: death carve-outs, postponement and cancellation handling, ties, rounding rules, and "announcement counts" clauses. Unparseable means a flag plus a confidence haircut; contradictory means a veto.
8. Both markets are **open and tradable** (`status: active` / `acceptingOrders: true`).

---

## 3. Record linkage / entity resolution methods for a deterministic Go (stdlib) matcher

### 3.1 Pipeline (classic ER, adapted)

```
ingest → normalize (lowercase, unicode fold, strip punctuation, number canonicalization "85,000"→85000, month names→ISO dates, ET→UTC)
       → extract structured fields (entity tokens, numbers+units, comparator, date window, source/oracle, bracket bounds, outcome labels)
       → BLOCK (candidate pairs)
       → SCORE (TF-IDF cosine + per-field agreement weights)
       → VETO (hard constraints from §2.3)
       → ASSIGN one-to-one
       → THRESHOLD (precision-tuned) → emit pair + explanation
```

### 3.2 Blocking (making the comparison sub-quadratic)

- **Inverted index on rare tokens.** Build `map[token][]marketID` over both venues. Generate candidate pairs only from tokens whose document frequency is below a cutoff, for example the top-k rarest tokens per market. Rare tokens such as proper names, tickers and numbers carry most of the identity signal, and stop-words like "will" or "2026" would otherwise create O(n²) blocks. This is standard token blocking and meta-blocking (Papadakis et al., "Blocking and Filtering Techniques for Entity Resolution: A Survey", ACM CSUR 2020, [arXiv 1905.06167](https://arxiv.org/abs/1905.06167)).
- **Compound keys** that mirror Gebele's structural filters: `(category, date-bucket)` and a non-empty intersection of the `[open, close]` windows. Use Kalshi `close_time`/`expected_expiration_time` and Polymarket `endDate`.
- **Measure blocking separately** with pair completeness (the recall of true pairs kept) and reduction ratio (Christen, "A Survey of Indexing Techniques for Scalable Record Linkage and Deduplication", IEEE TKDE 2012, doi:10.1109/TKDE.2011.127). Blocking should be close to lossless (Gebele: 100% of validated relations survived their structural filters).

### 3.3 Similarity features

- **TF-IDF cosine on word tokens** (and optionally character 3-grams for names). Cohen, Ravikumar & Fienberg (IJCAI-03 IIWeb, ["A Comparison of String Distance Metrics for Name-Matching Tasks"](https://www.cs.utexas.edu/~ai-lab/pubs/kdd03.pdf)) found that TF-IDF-based token metrics, with the TF-IDF + Jaro-Winkler hybrid best overall, beat pure edit distance for entity names. TF-IDF is roughly 40 lines of Go using `map[string]float64`.
- **Jaccard** on entity-token sets is useful as a cheap second opinion and is easy to explain.
- **Fellegi–Sunter weights.** Treat each extracted field (entity, threshold, date, source, comparator) as a comparison with agreement levels. Add `log2(m/u)` on agreement and `log2((1−m)/(1−u))` on disagreement, where *m* = P(agree | match) and *u* = P(agree | non-match). Fellegi & Sunter, JASA 1969, doi:10.1080/01621459.1969.10501049. See the explanation and "partial match weights" in [Splink's docs](https://moj-analytical-services.github.io/splink/topic_guides/theory/fellegi_sunter.html). In a deterministic prototype, set m/u by hand from the labelled set (no EM), so every score breaks down field by field, which gives the logged reasoning.
- **Hard-constraint vetoes** are applied *after* scoring and override any score. This is how dates, numbers and negation stop being invisible to text similarity.
  - **Number/threshold**: extracted strikes differ (after unit normalization), the comparator differs, or the bracket edges differ.
  - **Date/time**: the resolution instants differ (in UTC) beyond tolerance, or the observation window differs (a 1h candle vs a 60s average at the close).
  - **Negation/direction**: above vs below, Up vs Down, "not", "fail to", hike vs cut, "out" vs "remain".
  - **Source**: different oracle/station identifiers (`settlement_sources[].name` on the Kalshi event/series; Polymarket `resolutionSource` plus description URLs).
  - **Subject**: different team or person after alias mapping.
  - **Exception clause conflict**: a death carve-out on one side only; postponement policies that differ.

### 3.4 One-to-one assignment

Kalshi often splits one Polymarket multi-outcome event into N binary markets, and both venues relist. The assignment problem therefore applies at the **YES-region (binary outcome) level**, after vetoes:

- **Greedy mutual-best** with a deterministic order: sort candidate pairs by (score desc, kalshi_ticker asc, poly_token_id asc), then accept a pair only if neither side has already been taken. This is simple, deterministic and known to work well at scale (SiGMa: Lacoste-Julien et al., KDD 2013, [arXiv 1207.4525](https://arxiv.org/abs/1207.4525)).
- The **Hungarian algorithm** (Kuhn 1955) gives a globally optimal 1:1 assignment. It costs O(n³) and is only worth it inside a block if greedy shows conflicts.
- Enforcing 1:1 is itself a precision device: it removes the "one Poly market matched to three Kalshi strikes" false positives.

### 3.5 Why precision over recall

- The losses are asymmetric. A **false match** makes the router treat non-fungible contracts as one book. That can send a "best execution" fill into the wrong contract, or create an unhedged position, as in Khamenei: up to ~100% loss on a leg. A **false non-match** only costs a missed price improvement.
- Hand & Christen ("A note on using the F-measure for evaluating record linkage algorithms", *Statistics and Computing* 28(3), 2018) show that F1 implicitly weights precision vs recall in a way that depends on the *method*. Equinox should therefore **report precision and recall separately** and pick the operating threshold for a target precision, for example 1.0 on the labelled set with a stated recall, not maximal F1.
- Precedent: LLM pipelines reach high recall and still need heavy verification (Saguillo: 13 of 374; Gebele: FP <2% only after two LLM passes). A deterministic stdlib matcher should aim at a smaller, near-certain set.

### 3.6 Evaluation protocol

- **Labelled pair set** (target 100–300 pairs, hand-labelled from saved snapshots, frozen as a Go test fixture):
  - **Positives**: e.g. Fed `H0` ↔ "No change", NFL moneylines, shutdown-length YES-regions.
  - **Hard negatives** that differ in exactly one field: same-day BTC Kalshi BRTI vs Poly Binance; NYC CLINYC vs KLGA; adjacent strikes and brackets; level-Fed vs change-Fed; date ±1 day.
  - **Easy negatives** sampled from within blocks.
- **Label schema**: `{kalshi_ticker, poly_condition_id/token_id, relation ∈ {identity, subset, superset, overlap, disjoint}, veto_reason?, note}`, reusing the pmxt/Gebele relation vocabulary.
- **Metrics**:
  - blocking pair completeness and reduction ratio
  - matcher precision, recall, and the confusion breakdown **by veto reason**
  - a precision-vs-threshold curve
  - the identical output on repeated runs (a determinism check)
- **Regression:** every new false positive found in the wild becomes a fixture row.

### 3.7 Where LLMs fit (optionally, offline)

Peeters, Steiner & Bizer ("Entity Matching using Large Language Models", [arXiv 2310.11244](https://arxiv.org/abs/2310.11244)) show that generative LLMs are robust zero-shot matchers. Equinox's spec requires determinism, so the most an LLM should do is **propose labels for human review** while the fixture set is being built. It must never sit in the routing path.

---

## 4. Smart order routing: equities concepts and their prediction-market analogs

### 4.1 Equities background

- **Reg NMS Rule 611 (Order Protection / trade-through rule)**, 17 CFR 242.611 ([Cornell LII](https://www.law.cornell.edu/cfr/text/17/242.611)). Trading centers must have policies to prevent trade-throughs of **protected quotations**: automated, immediately accessible top-of-book quotes disseminated via the SIP. The **intermarket sweep order** (ISO) exception lets a router take several venues at once. On **2026-06-11 the SEC proposed rescinding Rule 611 and Rule 610(e)** (locked/crossed quotes), arguing that modern routing technology and best-execution duties make mandated linkage unnecessary. Comments were due 2026-08-17 ([Sidley](https://www.sidley.com/en/insights/newsupdates/2026/06/sec-proposes-rescission-of-the-order-protection-rule); [WilmerHale](https://www.wilmerhale.com/en/insights/client-alerts/20260617-the-sec-takes-aim-at-the-trade-through-rule)).
- **Rule 610(c) access-fee cap.** The September 2024 amendments cut the cap from $0.003 to **$0.001/share** (stocks ≥ $1) and require fees to be **determinable at the time of execution**. Rule 612 introduces a $0.005 tick. Compliance was deferred to Nov 2, 2026 and then to the **first business day of November 2027** ([SEC 2024-137](https://sec.gov/newsroom/press-releases/2024-137); extension reported by [Trade Informer](https://tradeinformer.com/regulations/sec-extends-nms-relief-rule-611-repeal-2027); SIFMA extension requests [letter](https://www.sifma.org/advocacy/letters/supplemental-request-for-immediate-extension-of-tick-size-and-access-fee-compliance-dates)).
- **Consolidated book / NBBO.** The SIP aggregates protected top-of-book quotes, and routers build a "virtual consolidated book" from direct feeds.
- **Best execution.** FINRA Rule 5310 requires "reasonable diligence to ascertain the best market… so that the resultant price to the customer is as favorable as possible under prevailing market conditions". Its factors are the character of the market (price, volatility, relative liquidity), the size and type of transaction, **the number of markets checked**, the **accessibility of the quotation**, and the order's terms. It also requires a "regular and rigorous review" ([FINRA 5310](https://www.finra.org/rules-guidance/rulebooks/finra-rules/5310)). MiFID II Art. 27 makes **"total consideration"** (price plus all execution costs, including venue, clearing and settlement fees) the retail best-execution yardstick ([ESMA](https://www.esma.europa.eu/publications-and-data/interactive-single-rulebook/mifid-ii/article-27-obligation-execute-orders)).
- **Academic SOR work:**
  - Foucault & Menkveld (J. Finance 2008, doi:10.1111/j.1540-6261.2008.01312.x): fragmentation can deepen the consolidated book, but trade-throughs (in their setting, routers ignoring the entrant venue) reduce liquidity supply there.
  - Cont & Kukanov ("Optimal order placement in limit order markets", [arXiv 1210.1625](https://arxiv.org/abs/1210.1625)): splitting across venues is a convex problem driven by queue state, **fee structure**, fill probability and execution-risk aversion.
  - Ganchev, Kearns, Nevmyvaka, Vaughan ("Censored exploration and the dark pool problem", UAI 2009, [arXiv 1205.2646](https://arxiv.org/abs/1205.2646)): learning **fill rates** per venue from censored fill data.

### 4.2 Translation table: equities concept → prediction markets → Equinox

| Equities concept | Prediction-market reality (verified) | Equinox implementation |
|---|---|---|
| NBBO / SIP consolidated tape | None. Kalshi (a CFTC DCM) and Polymarket are separate books with no linkage or trade-through obligation. The CFTC's June 2026 event-contract NPRM covers listing standards, not cross-venue linkage ([Mayer Brown](https://www.mayerbrown.com/ja/insights/publications/2026/06/the-odds-are-in-cftc-proposes-framework-for-event-contracts-and-prediction-markets)). | Build a **virtual consolidated book**, but only across markets the matcher accepted as `identity`. |
| Quote normalization | **Kalshi** `GET /markets/{ticker}/orderbook` returns `orderbook_fp: {yes_dollars:[[price,count]...], no_dollars:[...]}`. These are **bids only**, ascending, with the best bid last. **YES ask = 1 − best NO bid** ([Kalshi docs](https://docs.kalshi.com/getting_started/orderbook_responses.md)). Live: best YES bid 0.79; best NO bid 0.20 × 49,649.18, giving a YES ask of 0.80 (`raw/kalshi_orderbook_KXFEDDECISION-26OCT-H0.json`). **Polymarket** `GET clob.polymarket.com/book?token_id=…` returns `bids` ascending (best last = 0.80) and `asks` **descending** (best last = 0.81 × 20,600.6), plus `tick_size "0.01"`, `min_order_size "5"`, `neg_risk true` and `timestamp` in ms (`raw/poly_clob_book_fed_oct2026_no_change_yes_token.json`). | Normalize both into `[]Level{PriceMicros int64, Qty int64}` sorted best-first for YES-buy and YES-sell, per matched YES-region. Never rely on array order implicitly; sort explicitly. |
| Access fees (capped, flat per share) | **Not capped, and price-dependent on both venues.** **Kalshi** charges `round_up(0.07 × C × P × (1−P))` for takers and `0.0175 × …` for makers on series with maker fees ([fee schedule](https://kalshi.com/docs/kalshi-fee-schedule.pdf)). Each series carries `fee_type` (`quadratic`, `quadratic_with_maker_fees`, `flat`, …) and `fee_multiplier` (live `KXFEDDECISION`: `quadratic_with_maker_fees`, `1`, see `raw/kalshi_series_KXFEDDECISION.json`), with event-level overrides via fee-change endpoints. The current rounding doc says trade fee = `ceil_6dp(model_fee)` plus a rounding fee to the account's balance precision ($0.01 for non-direct members) ([Fee Rounding](https://docs.kalshi.com/getting_started/fee_rounding.md)). **Polymarket** charges `fee = C × feeRate × p × (1−p)`, takers only, rounded to 5 dp. Rates are per category: Crypto 0.07, Sports/Economics/Weather/Culture 0.05, Politics/Finance/Tech/Mentions 0.04, Geopolitics 0 ([Polymarket fees](https://docs.polymarket.com/trading/fees)). Per market, read Gamma `feesEnabled`, `feeType`, and `feeSchedule {rate, exponent, takerOnly, rebateRate}` (live: Fed `economics_fees` rate 0.05; BTC `crypto_fees_v2` rate 0.07; NYC `weather_fees`; NFL `sports_fees_nfl_cfb_oct26`). | Compute the **all-in cost per contract** for each consumed level: `price + fee(price, qty)`. This is MiFID's "total consideration". Use **integer micro-dollars**: in float64, `0.07*100*0.8*0.2 = 1.1200000000000003`, so a naive ceil-to-cent gives $1.13 instead of $1.12. |
| Order protection / ISO sweep | No obligation. Routing is purely economic. | Sweep in all-in price order across venues, level by level. A deterministic tie-break (e.g. venue name, then ticker) keeps runs reproducible. |
| Fill probability / queue position | Displayed depth is firm only at snapshot time. Polymarket has a minimum order size (5 shares) and a tick (0.01, with some markets at 0.001); Kalshi `price_ranges[].step` gives its tick (0.01 here). Thin books are common (e.g. Poly BTC 85,400 bid/ask 0.02/0.98). | Score a venue on executable quantity at or under the limit, with a configurable haircut on displayed size. Reject levels below the minimum size. Report a partial-fill residual instead of assuming a full fill. |
| Latency / staleness | The Kalshi REST orderbook has **no timestamp** (`orderbook_fp` only), so staleness must use the client receive time. Polymarket books carry `timestamp` (e.g. `1791259305517` = 04:01:45Z). Gamma's `bestBid`/`bestAsk` are cached summaries, not the book. | Simulate on a **frozen snapshot** with `fetched_at` per venue. Exclude or penalize books older than a TTL, and log the age. |
| Best-ex "factors" and "regular and rigorous review" (5310; Rule 605/606 disclosure) | Equinox is a simulator, so the analog is **logged reasoning**. | For every route decision, emit: the candidate venues checked, each venue's book age, the levels consumed, fee per level, all-in VWAP, rejected venues with reasons (stale, not matched as identity, insufficient size, fees), and the match confidence / relation of the pair. |
| Settlement and counterparty | Kalshi settles in USD at a CFTC-regulated DCM; Polymarket settles in USDC on Polygon with UMA dispute risk (see the Khamenei dispute history). Capital is locked until resolution, and resolution times can differ (2024 election: Nov 6 vs Jan 20). | Out of scope for price ranking. Surface it as a per-venue annotation, and optionally as a configurable per-venue penalty in basis points (time-value of locked capital). Do not net positions across venues. |

### 4.3 Worked example (live snapshot, Fed holds at the Oct 28, 2026 FOMC; buy 100 YES)

| Venue | Market | Best ask × size | Taker fee for 100 | All-in per contract |
|---|---|---|---|---|
| Kalshi | `KXFEDDECISION-26OCT-H0` | 0.80 × 49,649 (= 1 − NO bid 0.20) | 0.07×100×0.80×0.20 = **$1.12** | **0.8112** |
| Polymarket | "No change" YES token (`negRisk`) | 0.81 × 20,600 | 0.05×100×0.81×0.19 = **$0.7695** | **0.8177** |

The deterministic routing decision is **Kalshi**, which saves 0.65¢ per contract. A one-tick gap survives fees here because both fee curves are small near 0.8. At 50¢ the fee gap between venues (0.07 vs 0.05 × 0.25 = 1.75¢ vs 1.25¢) would shift the break-even by about 0.5¢.

---

## 5. Open questions and risks

- **Kalshi fee rounding for simulation.** Model it as legacy cent round-up per order, or as `ceil_6dp` plus a balance-precision rounding fee per the current docs? For non-direct members both converge to cent precision per order, but per-fill accumulation differs.
- **Polymarket `feeSchedule.exponent`** is 1 on every market observed. The docs formula has no exponent term, so its meaning for other values is unverified.
- **Gamma `bestBid`/`bestAsk` vs CLOB `/book`.** Gamma values are cached. The router should use only CLOB books, which costs one call per token (POST `/books` batches).
- **Tolerance for "by date X" political markets** (e.g. Kalshi "before Mar 1 15:00Z" vs Poly "by Feb 28 11:59 PM ET"). Strict identity rejects them; `subset`/`overlap` classification needs window arithmetic.
- **Kalshi rules text is lossy** (`>` dropped in `KXFEDDECISION` `rules_primary`). Which structured fields (`strike_type`, `floor_strike`, `cap_strike`, `yes_sub_title`, `custom_strike`) are reliable for every market type, and what should happen for markets that lack them?
- **Polymarket relisting.** Slugs gain numeric suffixes (`…-20260617190323537`). Stable identity should be `conditionId` / `clobTokenIds`, not the slug.

---

## Sources

Primary (live APIs and official docs)
- Kalshi API: `https://api.elections.kalshi.com/trade-api/v2/{markets,events,series,historical/markets,markets/{t}/orderbook}`. Docs: [orderbook responses](https://docs.kalshi.com/getting_started/orderbook_responses.md), [fee rounding](https://docs.kalshi.com/getting_started/fee_rounding.md), [series fee changes](https://docs.kalshi.com/api-reference/exchange/get-series-fee-changes.md), [fee schedule PDF](https://kalshi.com/docs/kalshi-fee-schedule.pdf)
- Polymarket: `https://gamma-api.polymarket.com/{events,markets,public-search}`, `https://clob.polymarket.com/book`. Docs: [fees](https://docs.polymarket.com/trading/fees)
- Dome matching API: [docs.domeapi.io](https://docs.domeapi.io/api-reference/endpoint/get-matching-markets-sports.md). pmxt Router: [pmxt.dev/docs/router/matching](https://pmxt.dev/docs/router/matching)

Research
- Saguillo et al., arXiv [2508.03474](https://arxiv.org/abs/2508.03474) (AFT 2025)
- Gebele & Matthes, arXiv [2601.01706](https://arxiv.org/abs/2601.01706)
- Ng, Peng, Tao, Zhou, [SSRN 5331995](https://papers.ssrn.com/abstract=5331995)
- Fellegi & Sunter (1969) JASA 64(328), doi:10.1080/01621459.1969.10501049. [Splink F-S guide](https://moj-analytical-services.github.io/splink/topic_guides/theory/fellegi_sunter.html)
- Papadakis et al. (2020) ACM CSUR, doi:10.1145/3377455 ([arXiv](https://arxiv.org/abs/1905.06167))
- Christen (2012) IEEE TKDE, doi:10.1109/TKDE.2011.127
- Cohen, Ravikumar, Fienberg (2003) [PDF](https://www.cs.utexas.edu/~ai-lab/pubs/kdd03.pdf)
- Hand & Christen (2018) Stat. Comput. 28(3):539–547 ([ANU](https://openresearch-repository.anu.edu.au/items/e3bc0e58-14c1-43a5-b1fa-78b3fdbdcd10))
- Lacoste-Julien et al., SiGMa, [arXiv 1207.4525](https://arxiv.org/abs/1207.4525)
- Peeters, Steiner, Bizer, [arXiv 2310.11244](https://arxiv.org/abs/2310.11244)
- Foucault & Menkveld (2008) J. Finance 63(1):119–158
- Cont & Kukanov, [arXiv 1210.1625](https://arxiv.org/abs/1210.1625)
- Ganchev et al., [arXiv 1205.2646](https://arxiv.org/abs/1205.2646)

Regulation
- [17 CFR 242.611](https://www.law.cornell.edu/cfr/text/17/242.611)
- [SEC 2024-137](https://sec.gov/newsroom/press-releases/2024-137)
- [Trade Informer: NMS relief extended to Nov 2027](https://tradeinformer.com/regulations/sec-extends-nms-relief-rule-611-repeal-2027)
- [Sidley on the 611 rescission proposal](https://www.sidley.com/en/insights/newsupdates/2026/06/sec-proposes-rescission-of-the-order-protection-rule)
- [FINRA 5310](https://www.finra.org/rules-guidance/rulebooks/finra-rules/5310)
- [MiFID II Art. 27 (ESMA)](https://www.esma.europa.eu/publications-and-data/interactive-single-rulebook/mifid-ii/article-27-obligation-execute-orders)
- [CFTC event-contract NPRM summary (Mayer Brown)](https://www.mayerbrown.com/ja/insights/publications/2026/06/the-odds-are-in-cftc-proposes-framework-for-event-contracts-and-prediction-markets)

Incidents and commentary
- [michaellwy: Prediction markets can't agree on the truth](https://michaellwy.substack.com/p/prediction-markets-cant-agree-on)
- [michaellwy: Wisdom of crowds or collective delusion](https://michaellwy.substack.com/p/wisdom-of-crowds-or-collective-delusion)
- [Monad: prediction market disputes](https://monad.xyz/blog/prediction-market-disputes)
- [The Defiant: shutdown end](https://thedefiant.io/newsletter/defi-daily/end-of-gov-t-shutdown-brings-more-polymarket-drama)
- [Bloomberg Law: Kalshi death-carveout suit](https://news.bloomberglaw.com/daily-tax-report-state/kalshi-sued-over-death-carveout-in-iran-leader-prediction-market)
- [Next Event Horizon: Khamenei](https://nexteventhorizon.substack.com/p/the-chaos-of-khamenei-prediction)
- [Fortune: 2024 payout timing](https://fortune.com/2024/11/04/polymarket-presidential-election-prediction-market-bets-payout-inauguration)
- [Axios: 2024 payout timing](https://www.axios.com/2024/11/06/kalshi-election-presidential-prediction-markets-polymarket-payout)
- [DeFi Rate: Venezuela invasion](https://defirate.com/?p=3283)
- [prediction.com arbitrage guide](https://prediction.com/blog/kalshi-vs-polymarket-arbitrage)
- [Apify verifier](https://apify.com/redfoxxie/polymarket-kalshi-arbitrage-verifier.md)
