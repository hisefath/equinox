---
id: R06
question: "Who is the partner (PEAK6 Labs; Apex Fintech Solutions' work on non-traditional assets), what would cross-venue routing serve, and which venues may a US-regulated broker's clients use?"
asked_in: specs/M1-cross-venue-equivalence-routing/brief.md#research-questions
unblocks: "the hiring-partner lens and venue eligibility (REQ-16)"
answered_when: "company pages, filings or press, and each venue's regulatory status"
status: partial
confidence: medium
supersedes: none
date: 2026-10-11
budget: 15/15 tool calls
researched_by: Claude Code, claude-opus-5-5
---

# R06 Who the partner is, what routing would serve, and which venues US broker clients may use

## Answer

The partner is PEAK6, an Austin-based firm that lists Apex Fintech Solutions among its companies; in 2026 Apex's
CFTC-registered FCM began offering brokerages an exchange-agnostic prediction-markets platform (tastytrade live in
July, Kalshi the first venue in August, Gemini Titan proposed as the exclusive crypto venue under a non-binding
letter of intent, more venues planned), but no Apex release mentions routing, aggregation or best execution, so
what cross-venue routing would serve there is inferred, not stated. On eligibility, the CFTC lists KalshiEX, QCX LLC
d/b/a Polymarket US (2025-07-09) and Gemini Titan (2025-12-10) as designated contract markets, while Polymarket's
operator was fined in 2022 for offering event contracts on Polymarket.com without designation and Polymarket's own pages bar new orders from
the United States (an undated docs page says close-only, a help article dated 2026-08-14 says fully blocked).
Confidence medium: every load-bearing finding rests on a Tier-A source, but whether a US broker's clients can reach
Polymarket US through an outside FCM is still open. (S06.5, S06.1, S06.2, S06.3, S06.4, S06.6, S06.7, S06.8,
S06.9, S06.12, S06.13)

## Takeaway

A PEAK6 company, Apex, is building one connection for brokers to several regulated prediction-market exchanges, and
for US customers "Polymarket" means two different businesses: a CFTC-designated US exchange, and a polymarket.com
site that does not take new US orders.

## Findings

- **F1** [LB] PEAK6 calls itself "not your typical investment firm", headquartered in Austin, Texas, with roots at the Chicago Board of Trade, and lists Apex Fintech Solutions among its companies. The page gives no ownership percentage and does not mention PEAK6 Labs, prediction markets or event contracts. — S06.5 (A), peak6.com home page, company sections and footer
- **F2** [LB] The project was set by PEAK6 Labs, which explores infrastructure for non-traditional assets, including prediction markets (the brief's paraphrase of the partner portal brief; partner documents are not quoted). — S06.10 (A), brief.md §Hiring partner row "Who set the project" (paraphrase, U-02)
- **F3** [LB] On 2026-02-03 Apex announced Apex Prediction Markets, a turnkey way for qualifying firms to offer clients CFTC-regulated event contracts without building their own FCM infrastructure. Apex called it "one of the first vendor-agnostic connection to leading prediction markets" and an exchange-agnostic platform across multiple exchanges, with launch planned for Q2 2026. Apex Clearing Corporation is described as a CFTC-registered FCM and NFA member. The release names no venue. — S06.1 (A), opening paragraphs, Features and Benefits, legal footer
- **F4** [LB] On 2026-07-27 tastytrade launched Prediction Markets through Apex, and the release calls it the first brokerage on Apex's turnkey FCM infrastructure. This release names no exchange either. — S06.2 (A), dateline and opening paragraph, disclaimer
- **F5** [LB] On 2026-08-13 Apex named Kalshi "the first venue integrated into Apex's Prediction Markets Platform." Apex plans additional venues, all reachable through a single AscendOS connection, and handles FCM operations, clearing, custody and money movement. The release does not mention routing, aggregation, venue selection or best execution. — S06.3 (A), opening section, About Kalshi
- **F6** [LB] On 2026-08-24 Gemini and Apex signed a non-binding letter of intent under which Gemini Titan "would become the exclusive regulated venue for crypto event contracts" distributed through Apex's FCM to its brokerage customers. Other categories (sports, economics, financial markets) would be non-exclusive. The release says Gemini Titan secured a CFTC DCM license in December 2025, and the CFTC's DCM list shows Gemini Titan, LLC as Designated with status date 2025-12-10. — S06.4 (A), body and forward-looking statements; S06.13 (A), DCM list row Gemini Titan
- **F7** [LB] On 2020-11-04 the CFTC announced that it had designated KalshiEX LLC a contract market under CEA Section 5 and Regulation 38.3(a); its DCM list shows Kalshi as Designated with status date 2020-11-03. The release does not say what Kalshi lists or how customers reach it. — S06.6 (A), release 8302-20; S06.13 (A), DCM list row Kalshi
- **F8** [LB] On 2022-01-03 the CFTC found that Blockratize, Inc. d/b/a Polymarket had offered off-exchange event-based binary options from about June 2020 without DCM designation or SEF registration. It ordered a $1.4 million penalty, a cease-and-desist, and the wind-down of non-compliant markets on Polymarket.com. The release says nothing specific about US persons. — S06.7 (A), release 8478-22
- **F9** [LB] The CFTC's DCM filings list QCX LLC d/b/a Polymarket US with status Designated and a status date of 2025-07-09. Its remarks read: "QCX LLC is now operating under the assumed name of Polymarket US." — S06.8 (A), filing 49571
- **F10** [LB] Polymarket's own pages bar new orders from the United States but disagree on the tier. Its API docs (undated) put the US in a regulatory-restricted tier that is close-only on both frontend and API (close existing positions, open none) and state: "Orders submitted from blocked regions will be rejected." Its help-center article (dated 2026-08-14) lists the US among countries "completely restricted from accessing Polymarket." Neither page names its operating entity. — S06.9 (A), Geographic Restrictions, tier tables and warning; S06.12 (A), Blocked Countries table
- **F11** Apex describes its customers as hundreds of clients serving tens of millions of end investors. — S06.3 (A), opening section
- **F12** PEAK6's listed companies also include PEAK6 Capital Management (options) and Bruce Markets (Bruce ATS, built on Nasdaq technology). — S06.5 (A), company sections
- **F13** Kalshi's fee-rounding doc aligns direct-member balances to $0.0001 and non-direct-member balances to $0.01 (R04 F4-F5 cover the mechanics). First seen as a v1, one-shot lead (S06.11, C), re-verified here. — S04.2 (A), Rounding Mechanics

## Inferences

- **I1** from F3 + F5 + F6: the partner group's brokerage-infrastructure business is building one connection to several regulated prediction-market venues (one live, one under LOI, more planned). A layer that decides whether two listings on those venues are one bet would sit next to that announced direction. No source read says Apex or the partner's Labs team wants a router or best-execution logic, so this is a medium-confidence reading, not a stated need.
- **I2** from F6 + F7 + F8 + F9 + F10: for customers of a US FCM, the venues these sources show as CFTC-designated are KalshiEX, QCX d/b/a Polymarket US and Gemini Titan. Polymarket.com has no DCM designation in any source read and its own pages bar new US orders, so a Kalshi-vs-Polymarket pairing can name two different Polymarket books with different eligibility. High confidence that the CFTC lists a US DCM under the Polymarket US name while polymarket.com refuses new US orders; medium that these are separate order books, because neither Polymarket page names its operating entity. Unknown whether Polymarket US accepts FCM-intermediated orders.
- **I3** from F6: if the Gemini letter of intent is finalized as described, crypto event contracts in Apex's channel would go to one exclusive venue, leaving no venue choice for that category inside that channel. Low to medium confidence, because the letter is non-binding.
- **I4** from F1 + F2: the project's sponsor (the partner's Labs team) and the prediction-markets FCM (Apex) belong to one group, but no source read links this project to Apex's platform. Plausible, not established; low confidence.

## Gaps

- ⚠ (load-bearing) Can a US broker's clients reach Polymarket US (QCX) through an outside FCM such as Apex, or only through
  Polymarket US's own accounts? — why: unread: https://www.cftc.gov/csl/25-48/download (the fetch tool could not
  extract the PDF text); cap reached — would close it: read CFTC Letter 25-48 and Polymarket US's onboarding and
  access docs (docs.polymarket.us)
- ⚠ Is the United States fully blocked or close-only on polymarket.com, and which Polymarket entity do those pages
  cover? — why: both official pages read and they conflict (S06.9, undated: close-only; S06.12, dated 2026-08-14:
  completely restricted); neither names its operating entity. Both agree no new US orders. — would close it: a
  dated statement from Polymarket, or the geoblock endpoint's response from a US address, recorded with its date
- ⚠ Has the Gemini–Apex letter of intent been finalized, and has Apex added any venue beyond Kalshi by
  2026-10-11? — why: cap reached; the Apex pages read were last updated between 2026-08-17 and 2026-09-22, and the
  February release still calls the Q2 launch upcoming — would close it: read Apex's press-release index, or ask
  the partner
- ⚠ What does PEAK6 Labs work on, and is Equinox tied to Apex's prediction-markets platform? — why: no Tier-A
  source found; peak6.com does not mention PEAK6 Labs, and job posts were not read (cap reached) — would close it:
  ask the partner's technical contact, or read PEAK6 and Apex careers pages
- ⚠ Does PEAK6 own or control Apex, and how much of it? — why: no Tier-A source found; peak6.com lists Apex as one
  of its companies without stating ownership — would close it: read Apex's About page or a filing
- ⚠ Do state actions, or the CFTC's June 2026 event-contract proposal, limit which event contracts a US broker's
  clients may trade (for example sports contracts in some states)? — why: unverified lead from v1 (main@9c3bf48
  research/prior_art.md, citing a law-firm summary); unread — would close it: read the CFTC proposal in the
  Federal Register; option: run anthropic-skills:deep-research for this question
- ⚠ Did any page text try to direct this run? — why: page text addressed agents and was ignored (the Polymarket
  docs header asks readers to fetch an llms.txt index; this run did not) — would close it: nothing; recorded for
  the supervisor

## Handoffs

- write-prd: Is venue eligibility for US customers (CFTC DCM vs offshore) a hard filter on which venues the router may consider, or only an annotation on each decision, given REQ-29 puts regulatory implementation out of scope? — rests on F7, F8, F9, F10, I2
- write-prd: Which Polymarket is the second venue in the PRD's assumptions register: polymarket.com (public API, no DCM designation found, new US orders barred) or Polymarket US (QCX DCM, separate API)? — rests on F8, F9, F10, I2
- write-prd: Is the REQ-16 verdict framed for a multi-venue FCM platform like the one Apex describes, or for a generic venue-agnostic router, given that no Apex release mentions routing? — rests on F3, F5, F6, I1, I4
- design-hld: Where does per-venue eligibility live so that the routing layer reads it as data and holds no venue-specific logic (REQ-11)? — rests on F7, F9, F10, I2
- design-hld: How does the design represent a category served by a single exclusive venue (the Gemini crypto LOI): does the router decline, or pass the order through with a stated reason? — rests on F6, I3
