---
id: R03
question: "How does each candidate venue define settlement (resolution source, time windows, cut-off times, ties, cancellations, carve-outs), and where do look-alike listings differ?"
asked_in: specs/M1-cross-venue-equivalence-routing/brief.md#research-questions
unblocks: 'the definition of "equivalent" (REQ-04)'
answered_when: "each venue's rulebook or contract terms, plus documented cases where look-alikes settled differently"
status: answered
confidence: high
supersedes: none
date: 2026-10-11
budget: 15/15 tool calls
researched_by: Claude Code, claude-opus-5-5
---

# R03 How each venue defines settlement, and where look-alike listings differ

## Answer

Kalshi settles against contract terms it self-certifies with the CFTC (one filing read in full; each of the three
series read links one): a named Source Agency, a payout criterion, a fixed expiration time (10:00 AM ET in the
contract read), and written carve-outs, with exchange review and fair-price fallbacks. Polymarket settles its
prediction markets against free-text rules through the UMA Optimistic Oracle: anyone proposes an outcome, there is a
2-hour challenge window, two disputes go to a token-holder vote, and a 50/50 outcome is possible; its up-down
markets instead settle mechanically on a Chainlink TWAP. Listings that look the same have diverged on four
dimensions, each shown from both venues' own records:
- **What counts as the event:** the 2024 shutdown settled No on Kalshi and Yes on Polymarket.
- **Carve-outs:** Khamenei 2026 settled as a $0.02 scalar on Kalshi and Yes on Polymarket.
- **Timing:** the 2024 election market closed on Polymarket the day after the vote (its rules resolve on network
  calls) and settled on Kalshi at inauguration, about 75 days later.
- **Source and statistic:** today's BTC "above" markets use a 60-second BRTI average on Kalshi and a single Binance
  1-minute close on Polymarket. NFL games also differ on postponement and cancellation.

Confidence: high, because every load-bearing claim rests on the venues' own API records or filed terms (Tier A).
(S03.1, S03.2, S03.3, S03.8, S03.10, S03.11, S03.12, S03.13, S03.14, S03.15, S03.16)

## Takeaway

Two prediction markets with the same headline can still pay out differently. One may count a missed deadline as a
shutdown while the other checks only at 10 AM, and one may read a different price feed or have a special rule if
someone dies. So "same bet" has to be judged on the settlement rules, not the title.

## Findings

<!-- One claim per line, worded as the source states it. The LB tag marks load-bearing claims. Tier in parentheses after every S-id, e.g. S03.1 (A). -->

- **F1** [LB] [S03.15 (A), S03.4 (A), S03.5 (A), S03.6 (A)] Kalshi defines a contract's settlement in terms it
  self-certifies with the CFTC. The WLEADEROUT terms (the one filing read) name an Underlying, a Source Agency list,
  a Payout Criterion, a Settlement Value of $1.00 and an Expiration time of 10:00 AM ET. They add that "Revisions to
  the Underlying made after Expiration will not be accounted for". Instructions for accessing the Underlying are
  non-binding and may be clarified at any time. Each of the three series read links its own certification PDF in
  `contract_url`. — S03.15 (A), WLEADEROUT certification dated 2025-05-15, concise explanation and Appendix A;
  S03.4 (A), S03.5 (A), S03.6 (A), `contract_url`.
- **F2** [LB] [S03.4 (A), S03.5 (A), S03.6 (A)] Kalshi's API exposes settlement terms as structured fields. Each market has `rules_primary` and
  `rules_secondary`; each series has `settlement_sources`, `contract_url` and `contract_terms_url`. Today the
  sources were 14 named news outlets for KXKHAMENEIOUT, "the Governing League" (nfl.com) for KXNFLGAME, and CF
  Benchmarks BRTI for KXBTCD. — S03.4 (A), S03.6 (A), S03.5 (A), live `GET /series/{ticker}` 2026-10-11.
- **F3** [LB] [S03.13 (A), S03.14 (A)] Polymarket resolves its prediction markets through the UMA Optimistic Oracle against each market's own rules:
  - **Proposal:** anyone may propose an outcome by posting a bond (given as $750). The proposal stands after a
    2-hour challenge window unless disputed.
  - **Disputes:** a first dispute starts a new proposal round. A second dispute goes to a 24-48 h debate, then a
    UMA token-holder (DVM) vote of about 48 h, 4-6 days in total.
  - **Other outcomes:** an unknown outcome resolves 50/50 ($0.50 per token), and a proposal made too early
    forfeits its bond.
  - **Clarifications:** "Additional context" can be published onchain after launch, but may not change the
    question's fundamental intent.
  - **Not UMA:** up-down markets resolve on a Chainlink TWAP without the proposal-and-dispute process, Up when the
    final price is equal to or greater than the price to beat.

  — S03.13 (A), help-center article (dated 2026-01-11), Overview and the propose and dispute sections; S03.14 (A),
  developer docs Resolution page (the UMA URL redirects to /concepts/resolution), sections UMA-Resolved Markets,
  Clarifications, Resolution Timeline, Chainlink TWAP-Resolved Up-Down Markets.
- **F4** [LB] [S03.15 (A), S03.1 (A), S03.2 (A), S03.7 (A)] The WLEADEROUT contract terms carry exchange discretion and timing contingencies:
  - Expiration moves earlier if the payout event occurs (Rule 7.2).
  - Settlement is no later than the day after expiration, unless the outcome is under review (Rule 7.1).
  - Kalshi may, at its sole discretion, start the Market Outcome Review Process (Rule 6.3(d)).
  - If no Expiration Value can be determined, Kalshi may determine payouts (Rule 6.3(b)).
  - Kalshi market records also carry an `early_close_condition` and a `settlement_timer_seconds` (3600, 1800 or 60
    in the markets read).

  — S03.15 (A), Appendix A §Settlement Date, §Expiration Date, §Contingencies; S03.1 (A), S03.2 (A), S03.7 (A).
- **F5** [LB] [S03.1 (A), S03.10 (A)] On the 2024 US government shutdown, the two venues settled in opposite directions:
  - **Kalshi** `SHUTDOWNBY-24` resolves Yes "If the government is shutdown at any 10:00 AM ET by 2024". Its result
    was **no**, settled 2025-01-01T07:28Z.
  - **Polymarket** `us-government-shutdown-before-2025` resolves Yes for a shutdown between Aug 30 and Dec 31, 2024
    11:59 PM ET. It also resolves Yes if the President fails to sign funding by the deadline, "even if no
    government shutdown is explicitly announced". Its sources are official US government information or a
    consensus of credible reporting. It resolved **Yes** (outcomePrices 1/0) and closed 2024-12-25.

  — S03.1 (A), S03.10 (A), live `historical/markets` and Gamma `markets?slug=...&closed=true` 2026-10-11.
- **F6** [LB] [S03.2 (A), S03.11 (A), S03.15 (A)] On Khamenei leaving office in 2026, the two venues settled differently:
  - **Kalshi** `KXKHAMENEIOUT-AKHA-26MAR01` (leaves office before March 1, 2026) has a death carve-out: if he
    leaves solely by dying, payouts are "based upon the last traded price (prior to the death)", with the Outcome
    Review Committee as fallback. Its result was **scalar**, settlement value **$0.02**, settled
    2026-03-01T03:19Z.
  - **Polymarket** `khamenei-out-as-supreme-leader-of-iran-by-march-31` resolves Yes if he is "removed from power
    for any length of time", including otherwise losing his position. Its source is a consensus of credible
    reporting. It resolved **Yes** after a status history of proposed, disputed, proposed, disputed, and closed
    2026-03-04.
  - The windows differ (before 2026-03-01 vs by 2026-03-31 11:59 PM ET).

  — S03.2 (A), S03.11 (A), S03.15 (A) Appendix A death clause.
- **F7** [LB] [S03.3 (A), S03.8 (A)] On the 2024 US presidential election, both venues paid Yes but on different dates:
  - **Kalshi** `PRES-2024-DJT` resolves on the candidate being "inaugurated as President for the term beginning
    January 20, 2025". It settled 2025-01-20T18:04Z.
  - **Polymarket** resolves when AP, Fox News and NBC all call the race for the same candidate, with inauguration
    only as the fallback. It closed 2024-11-06 15:17Z.

  — S03.3 (A), S03.8 (A).
- **F8** [LB] [S03.16 (A), S03.12 (A), S03.5 (A)] Today's BTC "above" markets for noon ET on Oct 12, 2026 differ in source and statistic, and state the strike differently:
  - **Kalshi** `KXBTCD-26OCT1212-T82499.99` uses the "simple average of the sixty seconds of CF Benchmarks' Bitcoin
    Real-Time Index" before 12 PM EDT. It resolves Yes if that average is above 82499.99 (labelled $82,500 or
    above, strike_type `greater`).
  - **Polymarket** `bitcoin-above-on-october-12-2026` uses the final Close of the Binance BTC/USDT 1-minute candle
    closing at 12:00 ET. It resolves Yes if that Close is higher than the price in the title, and the rules state the
    price is "not according to other exchanges or trading pairs".
  - **Strike grids:** Kalshi's event lists 188 `greater` markets with strikes ending in .99 at $100 steps
    (T81999.99 is labelled $82,000 or above); Polymarket's lists 11 strikes, 76,000 to 96,000 in steps of 2,000,
    so there is no Polymarket 82,500.

  — S03.16 (A), S03.12 (A), S03.5 (A).
- **F9** [LB] [S03.7 (A), S03.9 (A)] The live NFL game CIN @ MIA on 2026-10-11 shows how the two venues handle ties, postponements and
  cancellations:
  - **Kalshi** `KXNFLGAME-26OCT11CINMIA` lists one binary market per team. A tie pays $0.50 each. A postponed game
    that starts within 48 h stays open; otherwise "the market will resolve to a fair price".
  - **Polymarket** `nfl-cin-mia-2026-10-11` is one market with two outcomes and includes overtime. A tie resolves
    50-50. A postponed game will "remain open until the game has been completed". A game canceled with no make-up
    resolves 50-50. Its `resolutionSource` is nfl.com/scores.

  — S03.7 (A), S03.9 (A).
- **F10** Polymarket's `endDate` is not when a market resolves:
  - shutdown: endDate 2024-12-31, closed 2024-12-25;
  - election: endDate 2024-11-05, closed 2024-11-06;
  - Khamenei: endDate 2026-03-31, closed 2026-03-04.

  — S03.10 (A), S03.8 (A), S03.11 (A).
- **F11** Of the five Polymarket records read, the three closed markets (shutdown, election, Khamenei) had an empty
  `resolutionSource` and name their source only inside the free-text `description`. The two live ones had a
  market-level URL (NFL: nfl.com/scores; BTC: the Binance BTC_USDT page; the BTC event-level field was null).
  — S03.8 (A), S03.9 (A), S03.10 (A), S03.11 (A), S03.12 (A).
- **F12** Kalshi's API and its filed terms list different sources for the same contract: `settlement_sources` for
  KXKHAMENEIOUT names 14 outlets, while the WLEADEROUT Source Agency names the leader plus 16 outlets (adding
  Bloomberg News and NBC). The filing says a new Source Agency can be added via a Part 40 amendment. — S03.4 (A),
  S03.15 (A) Appendix A §Source Agency and concise explanation.

## Inferences

<!-- Reasoning across findings. Says what follows and how sure; never what to choose. -->

- **I1** from F5 + F6 + F7 + F8 + F9: every look-alike pair read differed on at least one of four settlement
  dimensions:
  - what counts as the event (F5, F6);
  - when or how often it is checked, or the window (F5, F6, F7);
  - which source and statistic (F8);
  - what happens in tail cases: death, ties, postponement, cancellation (F6, F9).

  The headlines would have paired every one of them. High confidence for these cases. How often such differences
  occur across today's whole overlap is not established here (see Gaps).
- **I2** from F5 + F1 + F2: the shutdown divergence follows from the rules texts. Polymarket counts a missed signing
  deadline as Yes even without an announced shutdown, while Kalshi only counts a shutdown in effect at a 10:00 AM ET
  check. A funding lapse that is short or overnight would satisfy the first and not the second. Medium confidence:
  the actual signing time was not read (Gaps).
- **I3** from F6 + F1 + F4: the Kalshi result `scalar` at $0.02 fits its death carve-out (last-traded-price payout);
  the only other non-binary path in the terms is an exchange payout under Rule 6.3(b). Polymarket's rule (removal
  from power for any length of time, by any means) has no such carve-out.
  The divergence is therefore a carve-out difference, not a window difference: Kalshi closed 2026-03-01T03:06Z,
  inside both windows. Medium confidence: the event itself was not read from a news source.
- **I4** from F3 + F4 + F9: both venues keep discretion after trading stops:
  - Kalshi through Market Outcome Review, fair-price and Outcome Review Committee determinations;
  - Polymarket through UMA disputes, onchain clarifications and a possible 50/50 outcome.

  So two listings with matching rules texts can still settle differently through adjudication. A definition of
  "equivalent" written over rules texts leaves this as residual risk. High confidence that the mechanisms exist;
  how often they fire is unknown.
- **I5** from F2 + F11 + F12: Kalshi exposes settlement source, strike type and bounds as structured fields, though
  its API source list need not match the filed terms (F12). Polymarket has a `resolutionSource` URL on some markets
  but states comparator, window and tail cases only in free text, so comparing settlement terms field by field
  needs text extraction at least on the Polymarket side. Medium confidence: 5 Polymarket records and 8 Kalshi
  records were sampled.
- **I6** from F8: at exactly 82,000.00, Kalshi T81999.99 ($82,000 or above) pays Yes, while Polymarket's 82,000
  market (higher than 82,000) pays No. Tick offsets like this are a settlement difference even with the same
  source; here the source and statistic also differ. High confidence on the texts.
- **I7** from F7 + F10: a pair can pay an identical outcome on dates about 75 days apart. Whether payout timing is part
  of "the same bet" is a definitional choice, not a fact.

## Gaps

- ⚠ What exactly do Kalshi Rulebook Rules 6.3(b), 6.3(d), 7.1 and 7.2 say: who runs Market Outcome Review, what
  triggers it, and how long it may take? — why: unread: https://cdn.robinhood.com/assets/robinhood/legal/KalshiEX_LLC_Rulebook.pdf
  (the fetch tool could not decode the PDF), and the CFTC filings surfaced by search were not opened; cap
  reached — would close it: read the current rulebook from kalshi.com/regulatory/filings with a PDF text
  extractor. Not load-bearing: F1 and F4 already show the rules exist and are cited by the contract terms.
- ⚠ Was the 2024 funding bill signed after the funding deadline, the mechanism I2 infers? — why: unread:
  https://www.congress.gov/bill/118th-congress/house-bill/10545/all-actions (HTTP 403) — would close it: read
  congress.gov or a White House statement for H.R.10545 (P.L. 118-158) signing time vs the prior continuing
  resolution's expiry.
- ⚠ Did Khamenei leave office by death, the condition I3 infers from Kalshi's scalar result, and what was argued
  in Polymarket's two disputes? — why: no news or dispute record read; cap reached — would close it: read
  Kalshi's settlement notice or a major-outlet report (v1 named a Bloomberg Law article on a class action, an
  unverified lead), and the UMA dispute record.
- ⚠ How often do look-alike pairs in today's live overlap differ in settlement terms? — why: not measured in this
  pass; v1 labelled live pairs with reasons such as different_resolution_source (v1, one-shot (main@9c3bf48), not
  re-measured) — would close it: the R02 overlap count plus the R05 labelled set, with a settlement-difference
  label per pair.
- ⚠ Do these differences also hold for other candidate venues? — why: only Kalshi and Polymarket were read; R01
  decides which two venues (REQ-01) — would close it: read the chosen venues' contract terms if they differ.
- ⚠ Are v1's other divergence leads real? — why: unverified leads from v1 notes
  (main@9c3bf48:research/prior_art.md §2), which name: a 2025 US National Bitcoin Reserve pair (prices 37% vs 51%
  per a personal blog, Tier C), 2025 shutdown-length buckets, a Polymarket "invasion" ruling on Venezuela,
  weather-station differences (LaGuardia vs Central Park), and 1°F bracket offsets. None was re-read — would close
  it: read each pair's records on both venues' APIs; option: run anthropic-skills:deep-research for a broader
  catalogue of divergent settlements.
- ⚠ What do Kalshi's `expected_expiration_time` and `latest_expiration_time` mean for cut-offs? On the Khamenei
  market the expected time (2027-01-01) was later than the latest (2026-03-01). — why: no Kalshi field doc read —
  would close it: read Kalshi's API docs on market lifecycle fields.
- ⚠ Which Kalshi source list binds when the API's `settlement_sources` (14 outlets) differs from the filed Source
  Agency (16 outlets plus the leader), F12? — why: the series' `contract_terms_url`
  (https://assets.kalshi.com/contract_terms/WLEADEROUT.pdf) and any Part 40 amendments were not read — would close
  it: read that PDF and the amendments listed at kalshi.com/regulatory/filings.
- ⚠ Page text addressed agents and was ignored: the header of https://docs.polymarket.com/developers/resolution/UMA
  told AI agents to fetch an llms.txt index first; not acted on.

## Handoffs

- write-prd: Does "equivalent" require the same payout in every outcome, including the tail cases (death carve-outs, ties, postponement over 48 h, cancellation, 50/50 outcomes), or only the main outcome, with tail differences recorded as caveats? — rests on F6, F9, I1, I4
- write-prd: Is payout timing part of "the same bet", so that a pair paying the same outcome about 75 days apart counts as equivalent or not? — rests on F7, F10, I7
- write-prd: Does a different source or statistic for the same quantity (a 60-second BRTI average vs a Binance 1-minute close) make a pair not equivalent, or equivalent with basis risk? — rests on F8, I6
- write-prd: How does the PRD treat residual adjudication risk (Kalshi outcome review, UMA disputes and clarifications) that no rules-text comparison removes, and is it stated to the user? — rests on F3, F4, I4
- design-hld: Where in the system do settlement terms become comparable fields, given that one venue returns them structured and the other mostly as free text? — rests on F2, F11, I5
- design-hld: How does the design notice that a matched pair's rules changed after matching (Polymarket onchain clarifications, Kalshi clarifications at any time)? — rests on F1, F3
