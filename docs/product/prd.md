# Equinox — Product Requirements

| Field | Value |
|---|---|
| Source brief | `specs/M1-cross-venue-equivalence-routing/brief.md` @ c0e0111 |
| Track | Full |
| Last updated | 2026-10-11 15:58 CDT |
| Context names | PEAK6, PEAK6 Labs, Apex Fintech Solutions |
| Approval | G1 — recorded in `.handoff/TRACKER.md ## Approvals` (never in this file) |

## Problem

PEAK6 Labs is deciding whether to invest in infrastructure that treats prediction markets on different venues as one
tradable set of bets, and wants an evidence-backed answer to three questions first: can listings be normalized
across venues, what are the architectural tradeoffs, and how complex does choosing a venue become (REQ-16). The same
real-world event is listed on several venues under different names and rules, and today nothing tells an engineer
whether two listings are the same bet. On 2026-10-11 about 64% of one major venue's events concerned an event the
other major venue also listed, but only about 30% asked the same question (R02). Listings that look alike have
settled on opposite sides, for example over what counts as the event or a carve-out written on one venue only (R03),
so treating them as one would send a customer's order into a contract that pays out differently. Why now: in 2026
Apex Fintech Solutions, a PEAK6 company, began offering brokerages one connection to several regulated
prediction-market exchanges, with more venues planned (R06), and at least four products already pair listings
across venues without publishing how often their pairings are wrong (R07). How much the confusion costs today in
money: Not in the sources. Evidence: REQ-16, R02, R03, R06, R07.

### Job stories

- **JS-01** When the same real-world event is listed on two venues under different names and rules, I want to know whether the two listings would settle the same way in every outcome, so I can compare their prices as one bet, and never mix up two different bets. — Source: brief JS-01 (REQ-03, REQ-04)
- **JS-02** When I have a hypothetical order for an outcome, I want to see which venue would get it, at what all-in cost, and the reason it won, so I can trust and audit the choice. — Source: brief JS-02 (REQ-05, REQ-06)
- **JS-03** When a venue is slow, unreachable or sends contradictory data, I want the decision to come out on time from the last good data, or be declined with a stated reason, so I can rely on the decision step even when a venue misbehaves. — Source: brief JS-03 (REQ-08, REQ-09)
- **JS-04** When I give the same inputs twice, I want exactly the same decision both times, so I can reproduce and explain any past decision. — Source: brief JS-04 (REQ-07)
- **JS-05** When we are deciding whether to invest further in cross-venue infrastructure, I want measured evidence of how often matching is wrong and where choosing a venue gets complicated, so I can make the investment call on evidence rather than a demo. — Source: brief JS-05 (REQ-16, REQ-04)
- **JS-06** When a third venue needs adding later, I want to add it without touching the rules that choose a venue, so I can reuse the foundation instead of repeating the integration work. — Source: brief JS-06 (REQ-10, REQ-11, REQ-17)

## Users and personas

| Persona | Who they are | Context of use | What they need | How often |
|---|---|---|---|---|
| P-1 Labs engineer (primary) | A PEAK6 Labs engineer evaluating non-traditional assets | At a desk, running the system against both live venues and reading its pairs and decisions | Pairs with the evidence behind them, a venue decision with its reason, and a measured error rate | Throughout the evaluation |
| P-2 Labs lead | The person at PEAK6 Labs who decides whether to invest further | Reads the verdict and the measured numbers, and tries one order on the running system | One verdict per question, each backed by a number and its method | Once per evaluation |
| P-3 Broker integration engineer | An engineer who would route broker customers' orders across venues, as on Apex Fintech Solutions' multi-venue platform | Checks how customer region and venue eligibility shape a decision | Decisions that leave out venues a customer may not use, with the reason | Occasionally |

Not for: retail traders placing real orders — because real-money trading is out of scope (REQ-29); traders looking for
price gaps to profit from — because that is not the feasibility question (NOGO-3).

## Goals

- **GOAL-1** The partner's lab gets a measured answer to "can equivalent listings be detected across venues reliably enough to route orders on?". Measured by: SC-001, SC-002, SC-003.
- **GOAL-2** The partner's lab sees how complex choosing a venue becomes once listings are matched, from explained decisions over live data. Measured by: SC-004.

## Non-goals and no-gos

Non-goals:
- **NG-1** Placing real orders, holding funds or connecting wallets — because the assignment rules them out (REQ-29); revisit never in this project.
- **NG-2** Implementing regulatory checks (identity, suitability, reporting) — because regulatory implementation is out of scope (REQ-29); which customer regions may use a venue is declared data with its source, not legal advice (Q2, A-4); revisit when a real broker integration starts.
- **NG-3** A polished interface — because the assignment gives UI polish no weight (REQ-29); a plain view of pairs, health and one order form is enough (US-7); revisit never in this project.
- **NG-4** More than two venues — because the appetite is one day (Q2); the next venue named is a US-regulated one; revisit after the verdict.
- **NG-5** Markets with no trade in the last 24 hours, and markets lasting under 1 hour — because a market nobody trades has no price worth routing to, and very short markets close before a match is useful (Q5, R02, R08); the left-out counts are always reported (FR-003); revisit if the verdict needs the long tail of quiet markets.
- **NG-6** Optimizing execution quality (fill strategies, market-impact models) — because the assignment cares about the reasoning behind a decision more than execution quality (REQ-05); revisit never in this project.
- **NG-7** Splitting one order across venues — because it adds a second decision layer the verdict does not need today; FR-044 sits in Later; revisit after the verdict.
- **NG-8** A market-size estimate in the verdict — because the only estimate found rests on one day of trading (R07); revisit if the partner asks.

No-gos:
- **NOGO-1** Consulting an AI model while an order is being routed — because routing must give the same answer every time and never wait on an outside call (Q4, REQ-07, REQ-08).
- **NOGO-2** Treating "same event" as "same bet" — because look-alike listings have paid out on opposite sides (Q1, R03).
- **NOGO-3** Price-gap or profit alerts — because they invite the regulatory questions the assignment excludes and do not answer the feasibility question (REQ-29).
- **NOGO-4** Quoting the partner's documents word for word or committing copies of them — because they are not cleared for this public repository (D6).

## Appetite

- Time box: one day, 2026-10-11 13:45 CDT → 23:30 CDT, scope bends and the date does not; hard date: 2026-10-11 23:30 CDT. Source: D4, brief Appetite and deadlines.
- If time runs short, cut in this order: US-6 → US-5 → US-7. Never cut: US-1, US-2, US-3, US-4 and every [Unwanted] FR they own.

## User stories

### US-1 See which listings are the same bet, and why (P1)

As a Labs engineer, I want the system to collect both venues' active markets and propose the pairs that would pay the same side in every outcome, each with its evidence and caveats, so that I can see which listings are the same bet.

- Why this priority: matching is the core of the feasibility question (REQ-03, REQ-04) and everything else rests on it.
- Independent test: run one refresh against both live venues; it lists equivalent pairs with the evidence for each part of the definition, look-alikes it rejected with the differing part, and how many markets it left out and why.
- Requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007 · Examples: AE-001, AE-002, AE-003, AE-004, AE-005

### US-2 Route a hypothetical order and say why (P1)

As a Labs engineer, I want to submit a hypothetical order for one side of a matched pair and get one venue decision with its all-in cost and the rule that decided it, so that I can trust and audit the choice.

- Why this priority: the assignment's second half is a routing decision that explains itself (REQ-05, REQ-06).
- Independent test: submit a 100-contract order for a known pair twice; read the chosen venue, the all-in cost, the deciding rule and why the other venue lost; both answers are identical.
- Requirements: FR-008, FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018, FR-019 · Examples: AE-006, AE-007, AE-008, AE-009, AE-010, AE-011, AE-012, AE-013, AE-014

### US-3 Keep deciding when a venue misbehaves (P1)

As a Labs engineer, I want decisions to come from the last good data, or be declined with a reason, when a venue is slow, down or inconsistent, so that I can rely on the decision step.

- Why this priority: graceful handling of failure and bad data is a stated benchmark (REQ-08, REQ-09).
- Independent test: cut one venue off mid-refresh; orders are still decided from data already held, the venue shows as unhealthy with its reason, and stale or broken order books are left out with the defect named.
- Requirements: FR-020, FR-021, FR-022, FR-023, FR-024, FR-025, FR-026, FR-027 · Examples: AE-015, AE-016, AE-017, AE-018, AE-019, AE-020, AE-021, AE-022

### US-4 Measure how often matching is right, honestly (P1)

As a Labs lead, I want a blind-labelled measurement of how often the system's "equivalent" verdicts are right and how many true pairs it misses, so that I can trust the verdict.

- Why this priority: the North Star is this number, and the interview will ask how it could be gamed (Q3, R05).
- Independent test: run a measurement on a fixed seed; it reports precision with its count and interval, recall, and human-versus-AI agreement, and a rerun with the same seed draws the same sample.
- Requirements: FR-028, FR-029, FR-030, FR-031, FR-032, FR-033, FR-034 · Examples: AE-023, AE-024, AE-025, AE-026, AE-027, AE-028

### US-5 Leave out venues a customer may not use (P2)

As a broker integration engineer, I want each order to state its customer's region and the router to leave out venues that region may not use, with the reason, so that a decision never sends a customer somewhere they cannot trade.

- Why this priority: for a US broker one major venue takes no new US orders (R06); the partner's own platform is a multi-venue broker connection (R06).
- Independent test: route the same order for a US customer and a non-US customer; see different decisions and the stated reason.
- Requirements: FR-035, FR-036, FR-037 · Examples: AE-029, AE-030, AE-031

### US-6 Let AI read the fine print offline (P2)

As a Labs engineer, I want pairs the deterministic rules cannot settle to be read offline by an AI adjudicator, with stored and reviewable verdicts, so that differences in the rules text are caught without making routing depend on a model.

- Why this priority: the decisive differences sit in free-text rules (R03); AI use must stay out of the routing path (Q4).
- Independent test: run matching with AI off and see the count of undecided pairs; run it with AI on and see each AI verdict with its model, prompt version and reason, then accept or reject one by hand.
- Requirements: FR-038, FR-039, FR-040, FR-041 · Examples: AE-032, AE-033, AE-034, AE-035

### US-7 Let a reviewer try it live (P2)

As a Labs lead, I want to open the running system at a public address and try a hypothetical order myself, so that I can check the claims without setting anything up.

- Why this priority: the human chose a live instance for review (D6); graders open what is linked (brief Q1).
- Independent test: open the public address; see current pairs and venue health; submit an order and read the decision; see which version is running.
- Requirements: FR-042, FR-043 · Examples: AE-036, AE-037

## Functional requirements

- **FR-001** [Event] (US-1) When a refresh runs, the system shall collect from each venue every open market that traded at least once in the last 24 hours and lasts at least 1 hour.
- **FR-002** [Ubiquitous] (US-1) The system shall describe every collected market in one venue-neutral form holding its question, outcome, resolution source and statistic, time window, threshold, rules text, YES and NO prices and sizes, fee terms and tradability.
- **FR-003** [Event] (US-1) When a refresh completes, the system shall report for each venue how many open markets it listed, how many were considered, and how many were left out for each reason.
- **FR-004** [Event] (US-1) When matching runs, the system shall propose as equivalent only pairs whose question, resolution source and statistic, time window, threshold and edge-case rules all agree, showing the evidence for each part.
- **FR-005** [Event] (US-1) When two listings differ only in when they pay out, the system shall propose them as equivalent with the payout-timing difference shown as a caveat.
- **FR-006** [Unwanted] (US-1) If two look-alike listings differ in question, resolution source, statistic, time window, threshold or an edge-case rule, then the system shall mark them "related, not routable" and name the part that differs.
- **FR-007** [Event] (US-1) When a refresh completes, the system shall list the closest rejected look-alikes with the reason each was rejected.
- **FR-008** [Event] (US-2) When a hypothetical order naming a pair, a side, a quantity, an optional price limit and a customer region is submitted, the system shall return one decision naming the chosen venue, the quantity filled there, the average price, the fees and the all-in cost.
- **FR-009** [Ubiquitous] (US-2) The system shall compute each venue's all-in cost for an order from that venue's visible order-book depth and its published taker fee for that market.
- **FR-010** [Event] (US-2) When a decision is made, the system shall state the one rule that decided it (most contracts filled, then lowest all-in cost, then a fixed tie-break order) and the saving against the next-best venue.
- **FR-011** [Event] (US-2) When a decision is made, the system shall state for every venue not chosen why it lost or was left out.
- **FR-012** [Event] (US-2) When an order is decided or refused, the system shall append the decision with its inputs and reasons to a decision log that can be read back later.
- **FR-013** [Ubiquitous] (US-2) The system shall give an identical decision for the same order and the same frozen market data, whatever order that data arrives in.
- **FR-014** [Ubiquitous] (US-2) The system shall decide a route from venue-neutral facts supplied as data (prices, sizes, fees, eligibility, freshness, health), never from a venue's identity.
- **FR-015** [Unwanted] (US-2) If an order names a pair that is unknown or marked "related, not routable", then the system shall refuse it with the reason.
- **FR-016** [Unwanted] (US-2) If an order is malformed (a quantity below 1, an unknown side, or a price limit outside 0 to 1), then the system shall reject it with the reason and add nothing to the decision log.
- **FR-017** [Unwanted] (US-2) If no venue can fill any part of an order within its price limit, then the system shall return "not filled" with each venue's reason.
- **FR-018** [Event] (US-2) When an order is larger than the chosen venue can fill, the system shall report the filled quantity and the unfilled remainder.
- **FR-019** [Event] (US-2) When a decision is made, the system shall show the match evidence and caveats of the pair it routed across.
- **FR-020** [Ubiquitous] (US-3) The system shall decide every route from market data it already holds, without contacting any venue while deciding.
- **FR-021** [Unwanted] (US-3) If a venue fails, times out or returns unreadable data during a refresh, then the system shall keep that venue's last good data and mark the venue unhealthy with the reason.
- **FR-022** [Unwanted] (US-3) If a venue's data for a pair is older than the freshness limit when an order is decided, then the system shall leave that venue out of the decision with the data's age stated.
- **FR-023** [Unwanted] (US-3) If a venue's market for a pair is closed, paused, or has an empty, crossed or undated order book, then the system shall leave that venue out of the decision with the defect named.
- **FR-024** [Unwanted] (US-3) If one record from a venue cannot be read, then the system shall skip that record, count it in the refresh report, and keep the rest of the refresh.
- **FR-025** [Unwanted] (US-3) If a venue refuses requests for exceeding its limits, then the system shall stop requesting from that venue until its stated wait has passed while keeping its last good data.
- **FR-026** [Event] (US-3) When a person asks for system health, the system shall show for each venue the time of its last good data, its market counts and its last error.
- **FR-027** [Unwanted] (US-3) If a venue changes the rules of a matched listing after the pair was matched, then the system shall stop routing across that pair until it is matched again.
- **FR-028** [Event] (US-4) When a measurement run starts, the system shall draw a random sample of up to 100 pairs it calls equivalent, using a seed it records.
- **FR-029** [Ubiquitous] (US-4) The system shall show a rater only both venues' questions and rules for a pair, never the system's verdict or score.
- **FR-030** [Event] (US-4) When a sample's labels are complete, the system shall report precision with the counts behind it and a 95% confidence interval.
- **FR-031** [Event] (US-4) When a sample's labels are complete, the system shall report the agreement between the human rater and the AI rater as Cohen's kappa over the pairs both labelled.
- **FR-032** [Event] (US-4) When a measurement run starts, the system shall draw a recall pool from rejected near misses plus random same-topic candidate pairs and report the share of true pairs it found.
- **FR-033** [Ubiquitous] (US-4) The system shall store each label with its rater, its time and the version of the matching rules the sample was drawn from.
- **FR-034** [Unwanted] (US-4) If measured precision is below 0.90, then the system shall state in its measurement report that routing needs a human-reviewed pair list.
- **FR-035** [Ubiquitous] (US-5) The system shall hold for each venue the customer regions that may use it, as declared data with its source and date.
- **FR-036** [Unwanted] (US-5) If an order's customer region may not use a venue, then the system shall leave that venue out of the decision with the reason.
- **FR-037** [Unwanted] (US-5) If an order names no customer region, then the system shall reject it with the reason.
- **FR-038** [Optional] (US-6) Where AI adjudication is switched on, the system shall send each pair the deterministic rules cannot settle to the AI adjudicator with both venues' rules text before routing reads the pairs.
- **FR-039** [Optional] (US-6) Where AI adjudication is switched on, the system shall store each AI verdict with the model, the prompt version, the time and the stated reason.
- **FR-040** [Unwanted] (US-6) If AI adjudication is switched off or unavailable, then the system shall keep matching with rule-decided pairs only and report how many pairs stayed undecided.
- **FR-041** [Event] (US-6) When a person accepts or rejects an AI verdict, the system shall use the person's verdict in place of the AI's for that pair.
- **FR-042** [Event] (US-7) When a reviewer opens the running system's public address, the system shall show the current pairs, venue health and a form to try a hypothetical order.
- **FR-043** [Event] (US-7) When a reviewer asks which version is running, the system shall show the version identifier of the running build.
- **FR-044** [Optional] (US-2) Where order splitting is switched on, the system shall split an order across venues when that lowers the all-in cost, stating each part.

## Non-functional requirements

- **NFR-001** [Reliability] Replaying the same orders against the same frozen market data gives identical decisions in 100% of runs, including runs where the data arrives in a shuffled order. Source: REQ-07; brief SM-02.
- **NFR-002** [Performance] Deciding a route makes 0 requests to any venue. Source: REQ-08.
- **NFR-003** [Capacity] The system sends each venue at most 2 requests per second, half the rate at which a repeat full listing sweep was refused on 2026-10-11. Source: R08 F16 (refused at about 4 pages per second).
- **NFR-004** [Capacity] A refresh of both venues' active markets completes within 15 minutes. Source: R08 F4 (a full open-catalog sweep of the larger venue took 751 s on 2026-10-11).
- **NFR-005** [Accuracy] Fees and all-in costs match each venue's published rounding to its smallest unit in 100% of the worked examples in the venues' own fee documents. Source: R04.
- **NFR-006** [Auditability] 100% of decided or refused orders can be read back from the decision log with their inputs and reasons. Source: REQ-06.
- **NFR-007** [Security] The system holds 0 items of secret material for the venues (keys, passwords, session tokens), because it reads public data only. Source: REQ-01.

## Success criteria

- **SC-001** North Star: of the listing pairs the system would let an order be routed across, the share that truly pay the same side in every outcome (matching precision), so that no customer order lands in a different bet. Target ≥ 0.90 (hard gate), grounded in v1's one-shot out-of-sample precision of 0.879 (a prior experiment, D3), and ≥ 0.95 (aspirational), grounded in the only published cross-venue figure, a preprint reporting under 2% false positives (R05); lagging; if bad we fix the matching rules or route only on human-reviewed pairs; measured on a fixed-seed random sample of up to 100 equivalent pairs, labelled blind by an AI rater with 30 also labelled blind by the human (Q3); reviewed by the human at each implementation milestone report. (GOAL-1)
- **SC-002** Supporting (guards SC-001 against matching less): the share of truly equivalent pairs in a labelled pool of near misses and random same-topic candidates that the system finds (recall). Target ≥ 0.84 (aspirational), grounded in v1's one-shot labelled recall of 0.844 (a prior experiment, D3); lagging; if bad we widen the candidate search or relax the rule that rejects true pairs; measured on the recall pool drawn in each measurement run. (GOAL-1)
- **SC-003** Supporting (guards SC-001 against a model grading a model): agreement between the human rater and the AI rater, as Cohen's kappa over the pairs both labelled. Target ≥ 0.60 (hard gate), below which McHugh treats agreement as inadequate, and ≥ 0.80 (aspirational), McHugh's strong band (R05 F5); leading; if bad the human labels the whole sample and the AI rater is dropped; measured in each measurement run. (GOAL-1)
- **SC-004** Supporting (explains GOAL-2): the share of routed hypothetical orders on which fees change which venue is cheapest (fee-flip rate). Target: none — it explains how much normalizing fees matters to routing, it does not judge; if it is near zero, price-only comparison would do and routing is simpler than feared; leading; measured over the decision log for one order per matched pair at a fixed size. (GOAL-2)

Rejected as vanity: number of pairs found (no denominator; matching everything maximizes it) · markets ingested (scale,
not correctness) · statement coverage % (says nothing about whether matches are right) · precision judged by a model
that saw the system's own score (anchored and gameable, R05).

## Acceptance examples

Fictional venues: **Ashby Exchange** (US-regulated, open to US customers) and **Birch Markets** (offshore, not open to
US customers). Fictional people: Dana Ortiz (Labs engineer), Sam Reyes (Labs lead).

- **AE-001** (FR-001, FR-003) Given Ashby lists 900 open markets of which 300 traded in the last 24 hours and 20 of those last 15 minutes, When Dana runs a refresh, Then 280 Ashby markets are considered and the report says 600 inactive and 20 too short.
- **AE-002** (FR-002, FR-004) Given Ashby's "Will the Fed cut rates by 25 bps at its December 2026 meeting?" and Birch's "Fed decreases rates by 25 bps after December 2026 meeting?" name the same source, meeting and threshold with no carve-outs, When matching runs, Then they are proposed as equivalent with evidence for question, source, window, threshold and edge-case rules.
- **AE-003** (FR-005) Given an Ashby and a Birch listing on "Candidate Lee wins the 2026 Iowa Senate race" agree on everything except that Ashby pays at swearing-in and Birch pays when the race is called, When matching runs, Then the pair is proposed as equivalent with the caveat "pays out about 60 days apart".
- **AE-004** (FR-006) Given Ashby's "BTC above $120,000 at 5 PM ET" uses a 60-second average price and Birch's uses a single 1-minute close, When matching runs, Then the pair is marked "related, not routable" with "statistic differs" named.
- **AE-005** (FR-007) Given Ashby's "Fed cuts by more than 25 bps in December" and Birch's "Fed cuts by 25 bps in December", When a refresh completes, Then the pair appears among the closest rejected look-alikes with "threshold differs".
- **AE-006** (FR-008, FR-009, FR-010, FR-019) Given the Iowa pair from AE-003 with Ashby asking $0.58 for 2,000 YES and Birch asking $0.585 for 2,000 YES, When Dana submits buy 100 YES for a non-US customer, Then the decision names the venue with the lower all-in cost after fees, the fill of 100, the price, the fees, the rule "lowest all-in cost" with the saving, and the pair's caveat.
- **AE-007** (FR-011) Given the decision in AE-006, When Dana reads it, Then the venue not chosen shows why it lost with its own all-in cost.
- **AE-008** (FR-012, FR-013) Given the market data of AE-006 frozen, When the same order is submitted twice and once more with the data in shuffled order, Then all three decisions are identical and all three are in the decision log.
- **AE-009** (FR-014) Given the two venues of AE-006 are renamed "Venue 1" and "Venue 2" with every price, size, fee and eligibility fact unchanged, When the same order is decided, Then the decision is the same apart from the names.
- **AE-010** (FR-015) Given the BTC pair from AE-004 is "related, not routable", When Dana submits buy 50 YES on it, Then the order is refused with "pair is not routable: statistic differs".
- **AE-011** (FR-016) Given any matched pair, When Dana submits buy 0 YES, Then the order is rejected with "quantity must be at least 1" and the decision log gains no entry.
- **AE-012** (FR-017) Given both venues ask at least $0.70 for YES on a pair, When Dana submits buy 100 YES with a limit of $0.60, Then the decision is "not filled" and each venue shows "best ask above limit".
- **AE-013** (FR-018) Given the chosen venue shows only 400 YES contracts for sale within the limit, When Dana submits buy 1,000 YES, Then the decision reports 400 filled and 600 unfilled.
- **AE-014** (FR-010) Given both venues show the same all-in cost of $58.90 for 100 YES, When Dana submits that order, Then the decision names the venue first in the fixed tie-break order and says "tie on all-in cost, broken by fixed order".
- **AE-015** (FR-020) Given a refresh has completed and both venues are then unreachable, When Dana submits buy 100 YES on a matched pair, Then a decision returns and 0 requests reach either venue while deciding.
- **AE-016** (FR-021, FR-026) Given Birch times out during a refresh at 16:05, When Sam asks for system health, Then Birch shows as unhealthy with "timed out", its last good data time of 16:00 and its market counts.
- **AE-017** (FR-022) Given Birch's last good data for a pair is older than the freshness limit, When Dana submits buy 100 YES on that pair, Then Birch is left out with its data's age stated.
- **AE-018** (FR-023) Given Ashby's order book for a pair has a best bid of $0.62 above its best ask of $0.60, When Dana submits buy 100 YES on that pair, Then Ashby is left out with "crossed order book".
- **AE-019** (FR-023) Given Ashby is in its weekly maintenance pause, When Dana submits buy 100 YES on a pair, Then Ashby is left out with "market paused".
- **AE-020** (FR-024) Given one of 280 Ashby records has an unreadable close time, When a refresh runs, Then 279 records are kept and the refresh report counts 1 unreadable record.
- **AE-021** (FR-025) Given Birch refuses a request and asks the system to wait 30 seconds, When the next request to Birch would be due 5 seconds later, Then no request is sent to Birch until the 30 seconds have passed and its last good data stays in use.
- **AE-022** (FR-027) Given the Iowa pair was matched at 15:00 and Birch rewrites that listing's rules at 15:30, When Dana submits an order on the pair at 15:40, Then the order is refused with "rules changed since matching" until the pair is matched again.
- **AE-023** (FR-028, FR-033) Given 240 equivalent pairs and seed 20261011, When Sam starts two measurement runs with that seed, Then both draw the same 100 pairs, and each stored label carries its rater, time and matching-rules version.
- **AE-024** (FR-029) Given a sampled pair the system scored 0.93 and called equivalent, When the human rater opens it to label, Then they see both questions and rules texts and neither the verdict nor the 0.93.
- **AE-025** (FR-030) Given 92 of 100 sampled pairs are labelled truly equivalent, When labelling completes, Then the report shows precision 0.92 from 92 of 100 with its 95% confidence interval.
- **AE-026** (FR-031) Given the human and the AI rater both labelled the same 30 pairs and agreed on 28, When labelling completes, Then the report shows Cohen's kappa over those 30 pairs.
- **AE-027** (FR-032) Given a recall pool of 60 pairs of which 20 are labelled truly equivalent and the system had proposed 16 of them, When the run reports, Then recall shows 0.80 from 16 of 20.
- **AE-028** (FR-034) Given measured precision of 0.86, When the measurement report is written, Then it states that routing needs a human-reviewed pair list.
- **AE-029** (FR-035) Given the declared eligibility data, When Sam reads it, Then Ashby lists US and non-US customers and Birch lists non-US only, each with a source and date.
- **AE-030** (FR-036) Given the Iowa pair where Birch is cheaper after fees, When Dana submits buy 100 YES for a US customer, Then Birch is left out with "not open to US customers" and Ashby is chosen.
- **AE-031** (FR-037) Given any matched pair, When Dana submits an order with no customer region, Then the order is rejected with "customer region required".
- **AE-032** (FR-038, FR-039) Given AI adjudication is on and the rules cannot settle a pair that differs only in a death carve-out on one side, When matching runs, Then the pair goes to the AI adjudicator before routing reads pairs, and its verdict is stored with the model, prompt version, time and reason.
- **AE-033** (FR-040) Given AI adjudication is off and the rules leave 12 pairs undecided, When matching completes, Then those 12 pairs are not routable and the refresh report says "12 pairs undecided: AI adjudication off".
- **AE-034** (FR-041) Given the AI called a pair equivalent, When Sam rejects that verdict, Then the pair is treated as not equivalent from then on.
- **AE-035** (FR-020, FR-038) Given AI adjudication is on, When Dana submits an order, Then no AI model is contacted while deciding.
- **AE-036** (FR-042) Given the system is running at its public address, When Sam opens it, Then he sees current pairs, both venues' health and an order form, and the form returns a decision for buy 100 YES.
- **AE-037** (FR-043) Given the running build has version identifier 7924a39, When Sam asks which version is running, Then he sees 7924a39.
- **AE-038** (FR-044) Given splitting is on and buying 1,000 YES costs less as 400 on one venue and 600 on the other, When Dana submits that order, Then the decision states both parts and their costs.

## Edge cases

- **EC-1** A venue lists the same question twice → expected: the better-evidenced listing is paired and the other appears among the closest rejected look-alikes with its reason (FR-007)
- **EC-2** A venue returns zero active markets in a refresh → expected: the venue keeps its last good data and shows as unhealthy (FR-021)
- **EC-3** An order exceeds the depth on every venue → expected: the filled quantity and the remainder are reported (FR-018)
- **EC-4** Both venues tie on all-in cost → expected: the fixed tie-break order decides and the decision says so (FR-010)
- **EC-5** A market closes or pauses between a refresh and an order → expected: that venue is left out with the reason (FR-023)
- **EC-6** A venue's weekly maintenance pause → expected: that venue is left out with "market paused" (FR-023)
- **EC-7** A venue blocks the system's requests outright → expected: requests stop until its stated wait passes and last good data stays in use (FR-025)
- **EC-8** Prices quoted in different units (cents on one venue, fractions of a dollar on the other) → expected: both are held in one venue-neutral form (FR-002)
- **EC-9** The same order submitted twice → expected: identical decisions, both logged (FR-012, FR-013)
- **EC-10** A matched listing's rules change after matching → expected: the pair stops being routable until re-matched (FR-027)
- **EC-11** An order whose customer region may use neither venue → expected: both venues are left out with their reasons and the decision is "not filled" (FR-036, FR-017)

## Assumptions and open questions

Assumptions:
- **A-1** Simulated orders take resting liquidity, so only taker fees apply; maker fees and rebates are left out. — Basis: assumed by agent; confirm at G1 (R04 handoff).
- **A-2** One winning contract pays $1 on either venue, and a dollar paid in a dollar-backed stablecoin is treated as equal to a US dollar; every decision discloses this. — Basis: assumed by agent; confirm at G1 (R04 handoff).
- **A-3** Fees follow each venue's published schedule for an ordinary direct customer, rounded the way that schedule says. — Basis: assumed by agent; confirm at G1 (R04).
- **A-4** The customer-region data reflects the venues' and the regulator's published status read on 2026-10-11, and is a declared input, not legal advice. — Basis: (Q2), R06.
- **A-5** Labels take two values, equivalent or not equivalent, and only "equivalent" counts as a true positive. — Basis: assumed by agent; confirm at G1 (R05 handoff).
- **A-6** The measurement sample is stratified by topic, and results are reported per topic using the system's own topic labels, because the venues' native categories do not line up. — Basis: assumed by agent; confirm at G1 (R02, R07 handoffs).
- **A-7** The verdict is framed for a multi-venue broker connection like the one the partner's brokerage-infrastructure company launched in 2026 (R06), and states how Equinox compares with existing cross-venue pairing products, none of which publishes its error rate. — Basis: assumed by agent; confirm at G1 (R06, R07 handoffs).
- **A-8** The freshness limit used to leave out stale data is set at design from the venues' measured price-change rates and shown in every decision. — Basis: assumed by agent; confirm at G1 (R08 handoff).

Open questions:
- ⚠ May the prototype read both venues' public data before their data terms of use are read? (→ R01) — Recommended: yes, for a non-commercial prototype reading public data; the terms are read before any commercial use — Blocks: FR-001
- ⚠ What is the US-regulated venue's published taker-fee formula and rounding? (→ R04) — Recommended: you download its fee-schedule document by hand (it refused automated reads with HTTP 429) and design-hld records the formula — Blocks: FR-009

### Interview log

Answered from sources: Problem → REQ-16, R02, R03, R06, R07 · Job stories → brief JS-01..JS-06 · bank #1 primary user → brief Users (Labs engineer), JS-01 · bank #2 success measure → brief NS-01, SM-01..SM-03 · bank #3 non-goals → REQ-29, brief Parking lot · bank #4 appetite → D4, brief Appetite and deadlines · bank #5 must-never-happen → decode QB-2, R03 · bank #6 secondary users → brief Hiring partner, R06 · bank #7 mandated constraints → REQ-01, D4, D6 · bank #8 sensitive data → REQ-01 (public data only), REQ-29 · bank #9 volume → R08 · bank #10 companion files → brief REQ table (none named)

| Q# | Question | Recommended (and evidence) | Reply (verbatim) | Landed in |
|---|---|---|---|---|
| Q1 | What does "equivalent" mean for Equinox? | Same payout side in every outcome the rules name, edge cases included; only payout timing may differ, shown as a caveat; anything else that differs is "related, not routable" — because R02 (65 vs 38 of 150 drawn events) and R03 (look-alikes settled on opposite sides) | "ok" | JS-01, FR-004, FR-005, FR-006, FR-015, NOGO-2, A-5 |
| Q2 | Which two venues, and how should the router treat a venue a US customer may not use? | The US-regulated and the offshore venue with public depth confirmed live; customer-region eligibility held as declared data, ineligible venues left out with the reason; a US-regulated third venue named next — because R01, R02, R06 | "ok" | C-1, US-5, FR-035, FR-036, FR-037, NG-2, NG-4, A-4 |
| Q3 | How is the North Star measured, and what counts as success? | Fixed-seed random sample of up to 100 equivalent pairs, labelled blind by an AI rater, 30 also labelled blind by the human, kappa reported, labels frozen before tuning; hard gate ≥ 0.90, aspirational ≥ 0.95; recall as the guard — because R05, D3 | "ok" | SC-001, SC-002, SC-003, US-4, FR-028 to FR-034 |
| Q4 | Where is AI allowed inside the product? | Offline only, as an adjudicator for pairs the rules cannot settle; verdicts stored, reviewable and frozen before routing; routing never calls a model; the system still works with AI off — because REQ-07, REQ-08, R03 | "ok" | US-6, FR-038 to FR-041, NOGO-1, AE-035 |
| Q5 | Which markets does Equinox consider? | Open markets with at least one trade in the last 24 hours and a life of at least 1 hour, with what was left out reported — because R08 (17.8% and 7.9% traded in 24 hours; a repeat sweep was refused) and R02 (5- and 15-minute windows) | "ok" | FR-001, FR-003, NG-5, NFR-004 |

## Risks

| Id | Risk | Type | Likelihood | Impact | Mitigation in this PRD |
|---|---|---|---|---|---|
| RISK-1 | Measured precision lands below the 0.90 hard gate | feasibility | M | H | FR-034 states that routing needs a reviewed pair list; US-6 catches rules-text differences; the verdict reports the number either way |
| RISK-2 | A venue blocks the system's requests during a refresh | feasibility | M | M | NFR-003 caps the request rate; FR-025 and FR-021 keep last good data |
| RISK-3 | The AI rater and the AI adjudicator share a blind spot, so precision looks better than it is | value | M | H | SC-003 checks the AI rater against the human's blind labels; A-5 keeps labels binary |
| RISK-4 | The customer-region data is wrong or goes stale | viability | L | M | A-4 dates it and names its sources; NG-2 makes clear it is not legal advice |
| RISK-5 | The one-day appetite runs out before the P2 stories | feasibility | H | M | the Appetite cut order (US-6, then US-5, then US-7) |
| RISK-6 | Fee figures for the US-regulated venue are wrong because its schedule is unread | feasibility | M | M | the open question blocking FR-009; NFR-005 checks fees against the venues' own worked examples |
| RISK-7 | The verdict does not speak to the partner's broker business | value | L | H | A-7 frames it for a multi-venue broker connection; US-5 shows eligibility |

## Milestone cut

| Milestone | Goal (one line) | Scope (FR ids) | Proves (SC / AE ids) | Exit demo (one sentence) |
|---|---|---|---|---|
| M1 Research | Approved brief + cited research | none (docs only) | G0 approval | A reviewer reads the decode, the brief and every R01-R08 note |
| M2 Product | Approved PRD (+ clarifications) | none (docs only) | G1 approval | A reviewer reads docs/product/prd.md |
| M3 Design | Approved HLD, LLD, contracts, diagrams | none (docs only) | G2-G3 approvals | A reviewer can read every design doc from docs/README.md |
| M4 Tracer bullet | Thinnest live journey: collect both venues, match, route one order, explain, log | FR-001, FR-002, FR-004, FR-008, FR-009, FR-010, FR-012, FR-014, FR-020 | AE-001, AE-002, AE-006, AE-008, AE-009, AE-015 | Dana runs one refresh against both live venues, sees an equivalent pair with its evidence, submits buy 100 YES and reads the chosen venue, its all-in cost and the deciding rule |
| M5 Trust | The rest of matching, routing and resilience, plus the honest measurement | FR-003, FR-005, FR-006, FR-007, FR-011, FR-013, FR-015, FR-016, FR-017, FR-018, FR-019, FR-021, FR-022, FR-023, FR-024, FR-025, FR-026, FR-027, FR-028, FR-029, FR-030, FR-031, FR-032, FR-033, FR-034 | AE-003 to AE-005, AE-007, AE-010 to AE-014, AE-016 to AE-028, SC-001, SC-002, SC-003 | Sam cuts a venue off and still gets explained decisions, then reads a blind-labelled precision, recall and kappa report |
| M6 Reach | Customer eligibility, offline AI adjudication and a live instance | FR-035, FR-036, FR-037, FR-038, FR-039, FR-040, FR-041, FR-042, FR-043 | AE-029 to AE-037, SC-004 | Sam opens the public address, routes the same order for a US and a non-US customer, and reviews one AI verdict |
| Later | Not in this appetite | FR-044 | — | — |

## Constraints

| Id | Constraint | Names | Mandated by | Source |
|---|---|---|---|---|
| C-1 | Exactly two venues, read only through their public, unauthenticated data APIs, with no venue accounts, keys or orders: Kalshi (the US-regulated venue) and Polymarket's international site polymarket.com (the offshore venue); Polymarket US is the next venue named | Kalshi, Polymarket, API | assignment (public data, two venues) and human (venue choice) | REQ-01 (brief Spec is law; Project Equinox PDF p1 §Scope, paraphrased), (Q2) |
| C-2 | A live instance of the system runs on Railway for reviewers | Railway | human | D6 (brief Q2, approved at G0) |
| C-3 | Partner documents are paraphrased with page or section references, never quoted word for word or committed | none | human | D6 (brief U-01 to U-03 left at the default) |
| C-4 | Hard date 2026-10-11 23:30 CDT; main shows v2 only once v2 passes all its checks and covers every REQ | none | human | D4 |
| C-5 | Real-money trading, wallet integration, regulatory implementation and production UI are out of scope | none | assignment | REQ-29 (brief Spec is law; Project Equinox PDF p2 §Scope, paraphrased) |

### Assignment trace

| REQ | What it asks (short) | Covered by |
|---|---|---|
| REQ-01 | Live metadata and prices from at least two venues' public data | FR-001, FR-002, C-1, NFR-007 |
| REQ-03 | Identify listings that may be the same event | FR-004, FR-006, FR-007 |
| REQ-04 | Define "equivalent" and justify the method | JS-01, FR-004, FR-005, FR-006, A-5 (definition here; the method is design-lld's) |
| REQ-05 | Simulate routing, decide and explain why | FR-008, FR-009, FR-010, FR-011, NG-6 |
| REQ-06 | Log and explain matching and routing decisions | FR-007, FR-011, FR-012, FR-019, NFR-006 |
| REQ-09 | Degrade gracefully; document assumptions where data is incomplete | FR-021 to FR-025, ## Assumptions and open questions |
| REQ-12 | Documented assumptions, reasonable tests, readable code | ## Assumptions and open questions, ## Acceptance examples (tests and code quality are later phases') |
| REQ-16 | Verdicts on normalization, tradeoffs and routing complexity | GOAL-1, GOAL-2, SC-001, SC-004, A-7 |
| REQ-29 | Out of scope: real money, wallets, regulation, production UI | NG-1, NG-2, NG-3, C-5 |

## Change log

| Date | Change | Ids affected | Gate |
|---|---|---|---|
| 2026-10-11 15:58 CDT | First draft | all | G1 |

## Clarifications

None yet.
