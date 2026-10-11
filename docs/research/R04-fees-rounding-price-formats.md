---
id: R04
question: "What are each venue's fee schedules, rounding rules and price formats, and how much do fees move the all-in cost of an order?"
asked_in: specs/M1-cross-venue-equivalence-routing/brief.md#research-questions
unblocks: "how routing compares venues; SM-03 (REQ-05)"
answered_when: "each venue's published fee schedule with worked examples"
status: partial
confidence: medium
supersedes: none
date: 2026-10-11
budget: 15/15 tool calls
researched_by: Claude Code (Claude Agent SDK subagent), claude-opus-5-5
---

# R04 Fees, rounding rules and price formats on Kalshi and Polymarket

## Answer

Both venues price a YES/NO contract in dollars between 0 and 1, with YES and NO complementary, and both set fees
per market (Polymarket) or per series with event overrides (Kalshi): Polymarket charges takers only
`fee = C × feeRate × p × (1 − p)`, rounded to 5 decimals, while Kalshi rounds each fee up to $0.000001 and then
re-aligns the balance to $0.0001 or $0.01. At 50¢, Polymarket's documented fee is $1.00 to $1.75 per 100 shares (2%
to 3.5% of notional, about one 1¢ tick per share), but Kalshi's fee formula and fee tables sit in a fee-schedule PDF
that neither the research pass nor the verifier could open (HTTP 429), and the fee-flip rate (SM-03) is not
measured. Confidence: medium, status partial, because Kalshi's published fee schedule, the "Answered when" evidence
for that venue, is unread (S04.1, S04.2, S02.4, S04.5, S04.8, S04.9, S04.11, S04.12).

## Takeaway

Polymarket's trading fee is largest near a 50/50 price, and both venues set fees market by market or series by
series, so the venue with the cheaper quoted price can turn out to cost more once its fee is added.

## Findings

- **F1** [LB] Kalshi sets fees per series with `fee_type` (`quadratic`, `quadratic_with_maker_fees`, `quadratic_with_combo_maker_fees`, `flat`) and a floating-point `fee_multiplier`. The structures are defined in the Kalshi fee schedule PDF: `quadratic` by its General Trading Fees Table, maker fees by its Maker Fees section, the combo variant with a 0.5 maker multiplier instead of 0.25, and `flat` by its Specific Trading Fees Table. Events can override both fields — S02.4 (A), OpenAPI v3.34.0 `Series.fee_type`, `fee_multiplier`, Get Event Fee Changes description
- **F2** [LB] Kalshi's help center says it charges "a transaction fee on the expected earnings on the contract". Maker fees apply on some markets, only when a resting order executes, and some markets carry different fees around special events. The page gives no formula and refers to the fee-schedule PDF — S04.4 (A), help.kalshi.com §Fees and §Maker Fees, dated April 19, 2026 (verifier re-read the page HTML on 2026-10-11)
- **F3** [LB] Live on 2026-10-11, `GET /series` returned 14,947 series: `fee_type` quadratic 14,784, quadratic_with_maker_fees 160, quadratic_with_combo_maker_fees 3, and no `flat`. `fee_multiplier` was 1 on 14,914, 0.5 on 19 and 0 on 14 — S04.7 (A), live response
- **F4** [LB] Kalshi rounding: the trade fee is the model fee rounded up to the nearest $0.000001. A rounding fee then brings the balance change onto the member's grid ($0.0001 for direct members, $0.01 for non-direct members). A per-order accumulator rebates overpayment across fills, and the net fee is always ≥ $0.00 — S04.2 (A), §Overview, §Rounding Mechanics, §Fee Accumulator
- **F5** [LB] Kalshi's worked example ("FCM-cleared fill", non-direct member) is a buy with −$0.055000 signed revenue and a model fee of $0.00363825. The trade fee is $0.003639, the rounding fee $0.001361, the balance change −$0.06, and trade fee plus rounding fee is exactly $0.005 — S04.2 (A), §Worked Examples
- **F6** Kalshi's docs changelog (entry labelled August 22, 2026) says that on combo trades where a quoter executes against an order resting under five seconds, the quoter pays the maker fee, at a multiplier of 0.5 instead of the standard 0.25. RFQs sized by target cost cap principal plus fees by default; with `target_cost_excludes_fees: true` the contract count is target cost ÷ price and the taker fee is charged on top — S04.6 (A), llms-full.txt lines ~1961–1980 and ~10241–10249 (read as excerpts, not whole pages)
- **F7** [LB] Polymarket: `fee = C × feeRate × p × (1 − p)`, where C = shares traded and p = share price. "Makers are never charged fees". Fees are in USDC and symmetric around 50%. The category table lists taker rates: Crypto 0.07, Sports 0.05, Finance 0.04, Politics 0.04, Economics 0.05, Culture 0.05, Weather 0.05, Other 0.05, Mentions 0.04, Tech 0.04, Geopolitics 0 (fee-free). The table's Maker Rebate column is 15% to 25% (Crypto 20%, Sports 15%, others 25%) — S04.1 (A), §Fee Structure; `rebateRate` is the fraction of taker fees rebated to the resting maker — S04.12 (A), §Trading Fees
- **F8** [LB] Polymarket's worked fee tables for 100 shares: at $0.50 (trade value $50) the taker fee is $1.75 for Crypto, $1.25 for Sports and Economics/Culture/Weather/Other, and $1.00 for Finance/Politics/Mentions/Tech. At $0.10 ($10) it is $0.63 / $0.45 / $0.36, at $0.90 ($90) it is the same $0.63 / $0.45 / $0.36, and at $0.99 ($99) it is $0.07 / $0.05 / $0.04 — S04.1 (A), §Fee Tables (100 Shares)
- **F9** [LB] Polymarket fees are rounded to 5 decimal places, and the smallest fee charged is 0.00001 USDC (smaller fees round to zero). Polymarket charges no deposit or withdrawal fees, though intermediaries may — S04.1 (A), §Fee Precision and intro
- **F10** [LB] Each Polymarket market carries `feesEnabled` and `feeSchedule {rate, exponent, takerOnly, rebateRate}`. `exponent` is "applied to the price component" — S04.12 (A), §Trading Fees
- **F11** [LB] Live on 2026-10-11, of the 100 open Polymarket markets with the most 24h volume, 86 had fees enabled and 14 had `feeSchedule` null. Rates were 0.03 (41), 0.05 (27) and 0.04 (18); `exponent` was 1 and `takerOnly` true on all 86. 33 markets of `feeType` `sports_fees_nfl_cfb_oct26` and 8 of `sports_fees_v2` carried rate 0.03, while the docs table lists Sports at 0.05 — S04.9 (A), live response
- **F12** [LB] Kalshi prices are fixed-point dollar strings (`*_dollars`), documented as up to 4 decimals. The spec says responses emit up to 6 decimals. Contract counts (`*_fp`) have 2 decimals and may be fractional, down to 0.01 contracts. Each market has a `price_ranges` tick grid (from $0.01 down to $0.0001 by structure), and whole-cent prices are valid in every structure — S04.5 (A), §Prices, §Price Level Structures, §Fractional Contracts; S02.4 (A), `FixedPointDollars`, `FixedPointCount`
- **F13** [LB] Kalshi's order book returns YES bids and NO bids only, with no asks — S01.20 (A), Get Market Orderbook description. Live, re-fetched by the verifier on 2026-10-11 19:44 UTC, the first page of 1,000 open non-combo markets was all `linear_cent` (step $0.0100), with prices as 4-decimal strings. Of 893 markets with a YES bid and a NO bid above zero, 0 broke `yes_ask = 1 − no_bid` or `yes_bid = 1 − no_ask` (the research pass's earlier fetch counted 925 two-sided, also 0 broken; that count was not reproduced) — S04.8 (A), live response
- **F14** [LB] Every Polymarket share is priced between $0.00 and $1.00, and the price is the implied probability. The displayed price is the bid-ask midpoint, or the last trade when the spread is wider than $0.10. A YES buy at $0.60 matches a NO buy at $0.40 because the two sum to $1.00, and 1 pUSD of collateral mints the complete set — S04.11 (A), §Prices Are Probabilities, §Price Discovery
- **F15** [LB] Each Polymarket market sets `orderPriceMinTickSize` (prices must be multiples of it) and `orderMinSize` (described as a minimum order size in USDC) — S04.12 (A). Live, re-fetched by the verifier on 2026-10-11 19:44 UTC, the top-100 markets had ticks of 0.01 (51) and 0.001 (49) (the research pass's earlier fetch gave 55 and 45; ticks and the top-100 set move), `orderMinSize` 5 on all 100, and `outcomePrices` as a JSON-encoded string of decimal strings whose two entries summed to 1.000 in 100/100 markets — S04.9 (A), live response
- **F16** In one live Polymarket book pair (nfl-min-no-2026-10-11), YES bid/ask was 0.95/0.96 and NO bid/ask was 0.03/0.05, with tick 0.01 and min order size 5. NO ask = 1 − YES bid held, but NO bid 0.03 ≠ 1 − YES ask 0.04; the two books were fetched one after the other, not atomically. Gamma's cached bestBid/bestAsk (0.93/0.94) differed from the CLOB — S04.10 (A), live response

## Inferences

- **I1** from F7 + F8: on Polymarket, the fee as a fraction of notional falls as the price rises, so cheap outcomes carry the heaviest relative fee. Formula: fee ÷ (C × p) = feeRate × (1 − p). Assumptions: taker side, exponent 1 (true for all 86 live fee markets, F11), rounding ignored. Range: low 0% (Geopolitics or fee-free markets) / base 2.0%–3.5% at p = 0.50 for the docs-table rates (1.5% for the live 0.03 rate, F11) / high 3.6%–6.3% at p = 0.10 (0.4%–0.7% at p = 0.90). Matches the worked tables exactly, so high certainty.
- **I2** from F5: Kalshi's worked-example model fee, $0.00363825, equals 0.07 × 1 × 0.055 × (1 − 0.055) exactly. This fits a quadratic fee of 0.07 × C × P × (1 − P), which is what the v1 notes said (v1, one-shot (main@9c3bf48), not re-measured). The fee-rounding page never states that formula, though, and the PDF that does is unread. So the Kalshi multiplier 0.07 is plausible but not established. Low-to-medium certainty.
- **I3** from F4 + F5: for a non-direct (FCM-cleared) Kalshi member, every fill's balance debit lands on the cent grid. In the worked example, a $0.058639 debit became $0.06, so fee plus rounding cost $0.005 on a $0.055 trade (about 9% of notional). Overpayment is rebated only across later fills of that one order. Small single-contract orders therefore bear rounding of up to almost one cent each, about the size of one `linear_cent` tick. Direct members round at $0.0001 and bear far less. High certainty for the mechanism; the 9% figure holds only for that example.
- **I4** from F1 + F3 + F7 + F11: on both venues the fee comes from per-market (Polymarket) or per-series and per-event (Kalshi) parameters, not from a category table. 41 of the 100 live Polymarket markets carried a rate (0.03) that differs from the docs table for their category. An all-in comparison built from category tables alone would misprice those markets. High certainty for the 100 live markets; the full population is not measured.
- **I5** from F7 + F8 + F11 + F13 + I2: fees alone can reverse which venue is cheaper by about one tick near 50¢. Formula: venue A's lower quoted price loses after fees when (p_B − p_A) < r_A × p_A(1 − p_A) − r_B × p_B(1 − p_B); at p ≈ 0.5 each side's fee per share or contract is r ÷ 4. Assumptions: taker on both sides, exponent 1, Kalshi rate 0.07 × multiplier 1 (unverified, I2), rounding (I3) ignored, prices in USD and USDC treated as equal (unread, see Gaps). Range: low 0¢ (equal effective rates, or a Kalshi multiplier-0 series against a fee-free Polymarket market) / base 1.0¢ (0.07 against 0.03 at p = 0.5, exactly one `linear_cent` tick) / high 1.75¢ (0.07 against a fee-free market at p = 0.5); near p = 0.10 or 0.90 the gap shrinks to at most 0.63¢. This bounds the size of a flip, not how often it happens (SM-03). Medium certainty.

## Gaps

- ⚠ [LB] What is Kalshi's general trading fee formula (multiplier, rounding of the fee to the cent, maker and special-market multipliers), and what are its worked fee tables? — why: unread: https://kalshi.com/docs/kalshi-fee-schedule.pdf (HTTP 429 bot challenge, once via WebFetch and once via curl; the verifier got 429 again on 2026-10-11 for both that URL and https://kalshi.com/fee-schedule; the help center (F2), API spec (F1) and docs changelog (F6) all defer to it, and the docs corpus llms-full.txt states no formula) — would close it: the human opens the PDF in a browser and records the General Trading Fees Table, the Maker Fees section and one worked row, or a later pass reads it when kalshi.com serves it
- ⚠ Answered-when evidence missing: Kalshi's published fee schedule with worked examples (only the fee-rounding worked example, F5, was read) — why: same 429 — would close it: same as above
- ⚠ Which direction does Polymarket round a fee to 5 decimals (up, down or half)? — why: no Tier-A source found; §Fee Precision gives the precision only — would close it: read Polymarket's exchange contract or SDK fee code
- ⚠ How does Polymarket's `exponent` enter the fee when it is not 1? — why: §Trading Fees says only that it applies to the price component, and the fees page shows the exponent-1 form; all 86 live fee markets had exponent 1 — would close it: read the SDK `MarketTrading` fee calculation
- ⚠ Is Polymarket's `orderMinSize` in USDC or in shares? — why: sources conflict (S04.12 says USDC; the v1 notes, Tier C, cite a place-orders page that says shares) — would close it: read docs.polymarket.com place-orders page
- ⚠ Is Polymarket's NO book an exact mirror of the YES book (NO bid = 1 − YES ask)? — why: one live pair broke it, but the books were fetched one after the other (F16) — would close it: one `POST /books` call for both tokens at once, repeated across many markets
- ⚠ What is the live fee-flip rate (SM-03)? — why: cap reached; it needs matched pairs (R02) and both venues' books at one instant, and v1's fee-flip finding was not read this pass (v1, one-shot (main@9c3bf48), not re-measured) — would close it: compute it from the v2 routing decision log over live matched pairs
- ⚠ Do a Kalshi USD contract and a Polymarket share in USDC or pUSD settle to the same $1? — why: no source read this pass states Kalshi's payout per contract or the USDC-to-pUSD-to-USD relationship — would close it: read Kalshi's contract terms (R03) and Polymarket's collateral docs
- ⚠ Do Kalshi's tiered percentage fee rates for margin-enabled accounts (llms-full.txt ~line 10153, Get Fee Tier Rates) apply to event contracts in scope? — why: cap reached; only grep hits read — would close it: read the margin-fee pages of docs.kalshi.com
- ⚠ Do sub-cent Kalshi tick structures (tapered, `center_*`) obey the same YES/NO complement in live data? — why: the live page sampled was 100% `linear_cent` — would close it: a live scan filtered to non-`linear_cent` markets
- ⚠ Does a separately regulated US Polymarket venue publish a different fee schedule, if R01/R06 make it a candidate? — why: cap reached; not searched — would close it: read that venue's official fee page
- ⚠ Docs pages from both venues carry a header telling LLM readers to fetch the documentation index first — why: page text addressed agents and was ignored; this run's own plan had already chosen the index pages — would close it: nothing needed

## Handoffs

- write-prd: Does "all-in cost" for routing count taker fees only, or also maker fees and rebates, and which Kalshi member type (direct at $0.0001 or FCM-cleared at $0.01 balance precision) does the simulation assume? — rests on F2, F4, F5, F7, I3
- write-prd: Should the assumptions register treat one Kalshi contract in USD and one Polymarket share in USDC or pUSD as the same $1 payout, and how is that assumption disclosed? — rests on F7, F14, Gaps
- write-prd: For SM-03, what counts as a "flip": any change in which venue is cheapest after fees, or a change bigger than one tick, and how is a tie after fees reported? — rests on I1, I5
- write-prd: Is Kalshi's fee formula an assumption the PRD may state before the fee-schedule PDF is read, or must the human read the PDF before G1? — rests on F1, F2, I2
- design-hld: Where do per-market fee parameters (Kalshi series `fee_type` and `fee_multiplier` with event overrides; Polymarket `feeSchedule`) enter the frozen inputs the router reads, given that both venues change them over time? — rests on F1, F3, F10, F11, I4
- design-hld: What money and price representation keeps Kalshi's 6-decimal fees, Polymarket's 5-decimal fees and each market's tick grid exact, so replays give identical decisions (SM-02)? — rests on F4, F9, F12, F15
