---
id: R02
question: "How much do the leading venues' open markets overlap today (the same real-world events), and in which topics?"
asked_in: specs/M1-cross-venue-equivalence-routing/brief.md#research-questions
unblocks: "whether matching matters at scale and where to measure it (JS-01, NS-01)"
answered_when: "a dated live count by topic"
status: partial
confidence: medium
supersedes: none
date: 2026-10-11
budget: 9/15 tool calls
researched_by: Claude Code (Claude Agent SDK subagent, gather-evidence worker), claude-opus-5-5
---

# R02 How much do Kalshi and Polymarket open catalogs overlap today, by topic?

## Answer

On 2026-10-11 (19:48-19:53 UTC), Kalshi listed 13,038 events with an active market and Polymarket 22,025 with an
open market; estimates from 15-event random draws per venue and topic put about 8,300 Kalshi events (64%, range
5,300-10,600) and 6,900 Polymarket events (31%, range 3,500-11,900) on a real-world event the other venue also lists,
but only about 4,000 (30%) and 3,500 (16%) asking the same question about it. By topic, elections have the highest
same-question rate (10 of 15 drawn Polymarket election events), sports the largest estimated absolute overlap,
economics/Fed little (0 of 15 drawn Kalshi economics events had a same-question Polymarket listing), and crypto runs
one way (6 of 15 drawn Kalshi crypto events matched, 0 of 15 drawn Polymarket crypto events, mostly 5- and 15-minute
windows). Confidence: medium, because the catalog counts are dated Tier-A live reads that a verifier re-read about 20
minutes later reproduced within 1%, but the overlap figures rest on one agent's unchecked judgment of small draws.
(S02.5, S02.2)

## Takeaway

The two biggest real-money prediction-market catalogs list many of the same real-world events, especially elections
and sports, but they often ask different questions about them, so "same event" is common while "same bet" is much
rarer.

## Findings


- **F1** [LB] Kalshi `GET /events?min_close_ts=<now>&with_nested_markets=true` paged to the end (cursor empty) and returned 13,038 events with at least one market in status `active`, holding 126,425 active markets. By Kalshi's own `category`: Sports 6,128, Elections 3,498, Entertainment 742, Politics 702, Economics 615, Financials 603, Climate and Weather 258, Science and Technology 174, Crypto 106, Companies 80, Mentions 75, Commodities 30, each other category 9 or fewer. The nested markets that were not active: initialized 87,770, finalized 3,620, inactive 478, determined 42, closed 4. — S02.5 (A), response fields `events[].category`, `events[].markets[].status`; fetched 2026-10-11, finished 19:50:40 UTC (a verifier re-read finishing 20:11:34 UTC gave 12,918 events and 123,658 active markets, every non-sports category within 2 of the counts above)
- **F2** [LB] Verifier re-reads of Kalshi `GET /events?status=open&limit=200`, paged until the cursor came back empty, returned 12,918 unique events (65 pages, 20:11:48 UTC) and 12,847 (65 pages, 20:17:51 UTC). The 20:11 count equals the count of events with an active market in the `min_close_ts` listing (F1) read one minute earlier. Series queries with `status=open` for KXMLBGAME (8), KXEPLGAME (21), KXETHD (3), KXNFLGAME (22), KXFEDDECISION (11), KXBTCD (3) and KXNHLGAME (51) returned no event missing from that listing, and the `min_close_ts` listing held the same series counts plus KXBTC15M 1. — S02.1 (A), S02.7 (A), S02.5 (A); counts computed from the responses, 2026-10-11
- **F3** [LB] Kalshi's OpenAPI spec describes `/events` as "Get all events. This endpoint excludes multivariate events." It gives `limit` a maximum of 200, a cursor, a `status` filter (unopened, open, closed, settled) and a `min_close_ts` filter for events with at least one market closing after that time. — S02.4 (A), `paths./events.get`
- **F4** [LB] Kalshi `GET /markets?status=open&limit=1000` returns almost only multivariate combos: a verifier pull stopped at a 401-page cap with the cursor still set, after 401,000 markets, of which 400,852 were multivariate (ticker prefix KXMVE or a non-empty `mve_collection_ticker`). The other 148 sat in 49 events, all of which were in the `status=open` events listing (F2). — S02.6 (A), S02.1 (A), fields `ticker`, `mve_collection_ticker`, `event_ticker`; 2026-10-11, finished 20:17:37 UTC
- **F5** [LB] Polymarket Gamma `GET /events/keyset?active=true&closed=false&limit=100`, paged with `after_cursor` until it ran out (221 pages), returned 22,025 events with at least one market that is `active` and not `closed`, holding 248,256 such markets. — S02.2 (A), fields `events[].markets[].active/closed`, `next_cursor`; finished 19:53:33 UTC (a pass 8 minutes earlier gave 22,048 and 248,895)
- **F6** [LB] Polymarket tag labels on those events (19:45 UTC pass): Sports 14,049, Games 12,743, Soccer 9,884, Crypto 3,970, Crypto Prices 3,716, Up or Down 3,615, 5M 2,370, 15M 780, 1H 347, Politics 2,456, Elections 1,692, Midterms 1,211, Geopolitics 473, Finance 424. — S02.2 (A), field `events[].tags[].label`
- **F7** [LB] Mapped to this file's five topics (method in Inferences), events, with active or open markets in parentheses, Kalshi vs Polymarket at 19:50-19:53 UTC: elections 3,498 (24,537) vs 1,829 (10,521); economics/Fed 615 (5,651) vs 202 (1,105); sports 6,128 (65,041) vs 14,082 (218,979); crypto 106 (4,522) vs 3,961 (5,949); other 2,691 (26,674) vs 1,951 (11,702). — S02.5 (A), S02.2 (A); counts computed from F1 and F5 responses (a verifier re-pull at 20:17 UTC with its own tag regexes gave Polymarket elections 1,829, crypto 3,973, sports 14,194, econ/Fed 168, other 1,946: the Polymarket split depends on the regexes)
- **F8** [LB] Two other public catalogs on the same day: Manifold `search-markets?filter=open` paged by `beforeTime` returned 39,757 open markets (BINARY 26,690, MULTIPLE_CHOICE 7,129, POLL 2,585, STONK 1,334, others smaller). PredictIt `marketdata/all` returned 203 markets. — S02.8 (A), 20:03 UTC; S01.7 (A), fetched shortly before 19:40 UTC
- **F9** [LB] Examples of listings present in both catalogs, as titled by each venue: Kalshi SENATEOHS-26 `Ohio Senate winner?` and Polymarket pm57682 `Ohio Senate Election Winner`; KXHOUSERACE-CA35-26 `CA-35 House winner?` and pm190922 `CA-35 House Election Winner`; KXNHLGAME-26OCT13VANNYI `Vancouver vs New York I` and pm1033575 `Canucks vs. Islanders` (slug nhl-van-nyi-2026-10-13); KXETH15M-26OCT111600 (3:45-4:00PM EDT) and pm1166385 `Ethereum Up or Down - October 11, 3:45PM-4:00PM ET`; KXBTCD-26OCT1116 `BTC price on Oct 11, 2026 at 4pm EDT?` and pm1172296 `Bitcoin above ___ on October 11, 4PM ET?`; KXHURPATHHAWAII-26DEC and pm743489; KXCMA-SOOTY26 and pm1040544 `CMA Song of the Year 2026`. — S02.5 (A), S02.2 (A), `title`, `sub_title`, `slug`
- **F10** [LB] Look-alikes where the subject matches but the occurrence or window does not: Kalshi KXSOLD-26OCT1117 (SOL price at 5pm EDT, Oct 11), where Polymarket's only same-day non-up/down SOL listing is pm1168507 `What price will Solana hit on October 11?`; KXCANOREOY-26DEC31 `Bank of Canada rate on December 31, 2026` vs pm991303 `Bank of Canada Decision in December?`; KXCBDECISIONMEXICO-26DEC17 (December meeting), where Polymarket lists only pm807644 `Bank of Mexico Decision in November?`; KXEGGS-OCT26 (October) vs pm1005016 `Price of Dozen Eggs in September?`. — S02.5 (A), S02.2 (A), titles
- **F11** [LB] The two venues cut events at different grain. Polymarket lists one game as several events: Grêmio vs Cruzeiro on 2026-10-18 has 6 (main, Halftime Result, Second Half Result, Exact Score, First Team to Score, More Markets), and its `Both Teams to Score` market sits inside More Markets. Kalshi lists one race or game as several proposition events: KXHOUSERACE-OH03-26 (winner), KXVOTEGENERAL-KXHOUSERACE-OH03-26JBEA and -26CDUL (one vote-percent event per candidate), KXMIDTERMMOV-* (margin) and KXMIDTERMVOTETURN-* (turnout). — S02.2 (A), S02.5 (A), `slug` prefixes and `event_ticker`
- **F12** [LB] At 20:00 UTC the Kalshi listing held 1 KXBTC15M event, while 2,370 Polymarket events carried the 5M tag and 780 the 15M tag (F6). 13 of the 15 Polymarket crypto events drawn were 5- or 15-minute up/down windows, for example pm1172119 `Bitcoin Up or Down - October 12, 2:00PM-2:05PM ET`. — S02.5 (A), S02.2 (A)
- **F13** A second pass at 20:00:56 UTC returned 12,987 Kalshi and 22,153 Polymarket events, net changes of -51 and +128 against the 19:50-19:53 pass. — S02.5 (A), S02.2 (A)
- **F14** Polymarket still returned some events as open after their dated end, for example pm782052 (end 2026-08-01) and pm1068540 (end 2026-10-01). — S02.2 (A), fields `endDate`, `markets[].active/closed`

## Inferences

<!-- Reasoning across findings. Says what follows and how sure; never what to choose. -->

**Method** (run by this agent, 2026-10-11, 19:40-20:03 UTC; one tool call can hold many paged HTTP requests to the
same endpoint):
- Catalogs come from F1 and F5. Topics:
  - Kalshi: its own `category`, mapped Elections → elections, Economics → econ/Fed, Sports → sports,
    Crypto → crypto, and everything else → other. Politics and Financials therefore land in other.
  - Polymarket: tag labels matched by regular expression in the order sports → crypto → elections → econ/Fed →
    other. The sports set includes `Games`, `Soccer`, league names and esports. The elections set is
    election(s)/midterm/primary/governor/mayor/presidential. The econ/Fed set is economy/Fed/FOMC/inflation/CPI/
    GDP/jobs/unemployment/interest rate/recession/PCE/payrolls.
- Draw: 15 events per venue and topic, simple random, using Python `random.Random(20261011)`.
- Candidates:
  - The 4 nearest listings on the other venue by TF-IDF cosine over the title, subtitle, first market labels,
    team codes taken from the ticker or slug, and dates.
  - Then, for every item still unresolved, a regular-expression search over the other venue's full catalog
    (second pass, 20:00 UTC).
- Judgment: one rater (this agent), using titles, tickers, slugs and market labels only. No rules were read.
  - **E** (same event): the other venue lists the same occurrence for the same subject (race, game, meeting, data
    release, asset and time window, award, or person and action), whatever question it asks.
  - **Q** (same question): E holds, and the question type and window also match (winner and winner, decision and
    decision, price at a time and price at a time). Thresholds and buckets may differ. Settlement sources were not
    checked; that is R03.

| Topic | Kalshi events | Polymarket events | Kalshi→PM E | Kalshi→PM Q | PM→Kalshi E | PM→Kalshi Q |
|---|---|---|---|---|---|---|
| elections | 3,498 | 1,829 | 12/15 | 5/15 | 13/15 | 10/15 |
| econ/Fed | 615 | 202 | 2/15 | 0/15 | 7/15 | 3/15 |
| sports | 6,128 | 14,082 | 11/15 | 5/15 (+4 unverified) | 5/15 (+1 unverified) | 2/15 (+1 unverified) |
| crypto | 106 | 3,961 | 6/15 (+2 unverified) | 6/15 (+2 unverified) | 0/15 | 0/15 |
| other | 2,691 | 1,951 | 5/15 | 4/15 | 4/15 | 3/15 |

<details><summary>The 150 drawn items and their judgments (for the human spot-check, U-06)</summary>

- Elections, Kalshi→PM:
  - E+Q: SENATEOHS-26↔pm57682; KXARGENTINAPRES-27↔pm435390; SENATEIA-26↔pm57647; KXMIDTERMMOV-FL27D↔pm812637;
    KXHOUSERACE-CA35-26↔pm190922.
  - E only: KXMIDTERMVOTETURN-VA05↔pm191668; KXAOCRUN-28↔pm872248 (Senate or President);
    KXHOUSEPOPVSGB-26NOV03↔pm215753; KXVOTEGENERAL-KXHOUSERACE-FL02-26AROG↔pm191400;
    KXVOTEGENERAL-KXHOUSERACE-CA33-26PAGU↔pm190920; KXVOTEGENERAL-GOVPARTYOK-26CMUN↔pm57186;
    KXVOTEGENERAL-KXHOUSERACE-GA14-26SHAR↔pm191438.
  - None: KXSTATEBALLOTMEASURE-WA, KXTRUMPRUN, SENATENH-28.
- Elections, PM→Kalshi:
  - E+Q: pm191578↔KXHOUSERACE-OH03-26; pm834425↔KXMIDTERMMOV-NM01D; pm191396↔KXHOUSERACE-CT03-26;
    pm828392↔KXMIDTERMMOV-MN08R/D; pm57646↔SENATEIL-26; pm33228↔KXPRESPARTY-2028; pm191609↔KXHOUSERACE-PA11-26;
    pm191633↔KXHOUSERACE-TN09-26; pm191442↔KXHOUSERACE-IA04-26; pm191533↔KXHOUSERACE-NC05-26.
  - E only: pm773641↔KXTRUMPENDORSE28-28MAR01 (window differs); pm850651↔KXHOUSERACE-GA11-26 (first-round outright
    vs winner); pm1149856↔KXTOCANTINSGOV-26OCT25 (margin vs winner).
  - None: pm886857, pm1160065.
- Econ/Fed, Kalshi→PM:
  - E only: KXIPODISCORD↔pm638260 (announce IPO vs IPO); KXCPINDEX-26OCT14↔pm1005014 (index level vs monthly change,
    one CPI release).
  - None: KXCBDECISIONMEXICO-26DEC17, KXADP-26OCT, KXBANKRUPTCY-26, KXLFPRATEEOY-35JAN05, KXCOBUILDPERMITS-27JAN27,
    KXEGGS-OCT26, KXNWSAYLOR-26, KXAKCRUDEOIL-27MAR01, KXCANOREOY-26DEC31, KXFEDEMPLOYEES-27FEB, KXGDPNOM-IND26,
    KXFED-27SEP, KXMNMEDDEVEMP-27APR30.
- Econ/Fed, PM→Kalshi:
  - E+Q: pm743552↔KXTRYFIREWARSH-27JAN01; pm770451↔KXFEDDECISION-27JAN; pm1116497↔KX2YRDIRLM-26OCT30L.
  - E only: pm1116451↔KXAAAGASM-26OCT31 (hit by vs level on date); pm955999↔KXFEDDECISION-26DEC (multi-meeting path);
    pm1035410↔KXUKEAINFL-26OCT21 (UK level vs UK-vs-euro); pm228565↔KXACQPYPL-27JAN01 (any part vs whole).
  - None: pm79208, pm961373, pm506331, pm648785, pm956930, pm226397, pm79041, pm147795.
- Sports, Kalshi→PM:
  - E+Q: KXAFCCLGAME-26OCT12ALNNEF↔pm1158219; KXBRASILEIROBTTS-26OCT18GPACRU↔bra-gre-cru-2026-10-18 More Markets
    (BTTS); KXUECLSPREAD-26OCT15TWETHU↔col-twe-thu-2026-10-15 (spread); KXJLEAGUEGAME-26OCT17CERKAW↔pm1126062;
    KXNHLGAME-26OCT13VANNYI↔pm1033575.
  - E, Q unverified: KXLALIGATOTAL-26OCT17RBBBAR↔pm1129055; KXMLSTOTAL-26OCT17HOUDAL↔mls-hou-dal-2026-10-17;
    KXWTAGTOTAL-26OCT11SINSAM↔pm1165239; KXNFL2H-26OCT11SFSEA↔pm941347 (2H spread found, 2H winner not).
  - E only: KXNHLSPREAD-26OCT13FLACBJ↔pm1033569 (no puck-line market); KXNCAAF1H-26OCT17LAMSELA↔pm1139687 (no
    first-half market).
  - None: KXNBAPLAYERHIGHSCORE-27ANY, KXESOCCERGAME-26OCT111656MILLANNI, KXNCAAFCFPPOLL-26NOV10R3,
    KXNCAAF1H-26OCT23ARMYTLSA.
- Sports, PM→Kalshi:
  - E+Q: pm1128883↔KXBUNDESLIGASPREAD-26OCT17UNIBVB; pm1153874↔KXT20MATCH-26OCT120200THAMYA.
  - E only: pm1118153↔KXEFLL1GAME-26OCT15MATOXU (halftime); pm1114673↔KXMLSGAME-26OCT14RSLSJ (corners);
    pm1133459↔KXKLEAGUEGAME-26OCT18JEOJEJ (halftime).
  - Unverified: pm830305 (Kelce receiving TDs) ↔ KXNFLSEASONRECTD-27C* (Kelce market not confirmed).
  - None: pm1168472, pm1148804 (Oct 29 game not yet on Kalshi), pm1167046, pm1068540, pm781479, pm1154634,
    pm782052, pm988839, pm1169413.
- Crypto, Kalshi→PM:
  - E+Q: KXHYPEMAXMON-HYPE-26OCT31↔pm1112416; KXBTCD-26OCT1116↔pm1172296; KXETH15M-26OCT111600↔pm1166385;
    KXETHD-26OCT1117↔pm1172877; KXXRPMAXY-XRP-26DEC31↔pm89530; KXSOLMAXY-27JAN01↔pm89525.
  - Unverified: KXZECMAXMON-ZEC-26OCT31; KXSOLD-26OCT1617 (the time of day of pm1159744 was not read).
  - None (only intraday-high or up/down listings found): KXSOLD-26OCT1117, KXSOLE-26OCT1117, KXXRP-26OCT1116,
    KXDOGE-26OCT1116, KXHYPE-26OCT1116, KXBNBD-26OCT1117, KXDOGE-26OCT1617.
- Crypto, PM→Kalshi:
  - None: 9 five-minute windows (pm1169935, pm1172119, pm1169936, pm1170946, pm1166928, pm1169291, pm1170140,
    pm1168235, pm1169027), 4 fifteen-minute windows (pm1167939, pm1171673, pm1171668, pm1171637), 1 hourly
    (pm1161910) and pm79077.
- Other, Kalshi→PM:
  - E+Q: KXNEWOUTBREAK-P-26↔pm95116; KXPAHLAVIVISITA↔pm139507; KXWARSHOUT-26JUL↔pm743529;
    KXLOWTSEA-26OCT12↔pm1163968.
  - E only: KXRTX5090MS-26NOV↔pm1111187 (monthly average vs month-end).
  - None: KXSBUX-27JANSTORES, KXCLAUDE-MYTH, KXNASDAQ100Y-26DEC31H1600, KXWINGA-27FEBDOMCOMP,
    KXHURRICANE-26DEC01EPACMAJ, KXALBUMRELEASE-26, KXTRUMPVISITTOWER-26OCT01, KXBROWNHUSTEDDEBATE-26,
    KXYTUBESUBSISHOWSPEED, KXDTEMMYS-EMERGING26.
- Other, PM→Kalshi:
  - E+Q: pm743489↔KXHURPATHHAWAII-26DEC; pm140376↔KXMACHADOVENEZUELA-26JUN29; pm1040544↔KXCMA-SOOTY26.
  - E only: pm73154↔KXPUTINZELMEET-29JAN01 (handshake vs meeting).
  - None: pm1158896, pm916848, pm964505, pm1161509, pm906873, pm1082452, pm1006090, pm1157404, pm1030121,
    pm1159021, pm424959.

</details>

- **I1** from F7 + the tally (elections): Formula: Kalshi E (12/15) × 3,498 ≈ 2,800; Kalshi Q (5/15) × 3,498 ≈ 1,200; Polymarket E (13/15) × 1,829 ≈ 1,600; Polymarket Q (10/15) × 1,829 ≈ 1,200 events. Assumptions: A-1 the 15-event draw represents its topic; A-2 the single agent's judgments are right; A-3 retrieval plus targeted search found every counterpart that exists. Range: Wilson 95% on the draw only: Kalshi E 1,900-3,300, Kalshi Q 530-2,000, Polymarket E 1,100-1,800, Polymarket Q 760-1,600; errors in A-2 and A-3 are not in the range. Elections give the highest same-question rate from the Polymarket side; medium-low confidence.
- **I2** from F7 + F10 + the tally (econ/Fed): Formula: Kalshi E (2/15) × 615 ≈ 82; Kalshi Q (0/15) × 615 = 0; Polymarket E (7/15) × 202 ≈ 94; Polymarket Q (3/15) × 202 ≈ 40 events. Assumptions: A-1 to A-3 as in I1. Range: Kalshi E 23-230, Kalshi Q 0-130, Polymarket E 50-140, Polymarket Q 14-91. Kalshi's economics catalog has a long tail (state jobs, commodity output, company net worth) that Polymarket does not list; the pairs found are Fed meetings, Treasury yields, CPI and named corporate actions; low confidence.
- **I3** from F7 + F11 + the tally (sports): Formula: Kalshi E (11/15) × 6,128 ≈ 4,500; Kalshi Q (5/15 verified, 9/15 counting unverified) × 6,128 ≈ 2,000 (3,700); Polymarket E (5/15) × 14,082 ≈ 4,700; Polymarket Q (2/15) × 14,082 ≈ 1,900 events. Assumptions: A-1 to A-3, plus A-4 Polymarket's per-game sub-events (F11) mean event counts overstate distinct games. Range: Kalshi E 2,900-5,500; Kalshi Q 930-3,600 (2,200-4,900 counting unverified); Polymarket E 2,100-8,200 (2,800-9,000 counting unverified); Polymarket Q 530-5,300. Sports is the largest overlap by count; low-medium confidence.
- **I4** from F7 + F12 + the tally (crypto): Formula: Kalshi E = Q (6/15) × 106 ≈ 42 (8/15 counting unverified ≈ 57); Polymarket E = Q (0/15) × 3,961 = 0 events. Assumptions: A-1 to A-3, plus A-5 the 5- and 15-minute Polymarket windows drawn (13 of 15) had no Kalshi listing at snapshot time. Range: Kalshi 21-68 (32-80 counting unverified); Polymarket 0-810 (Wilson upper bound for 0/15). Crypto overlap runs one way; medium confidence on the direction, low on size.
- **I5** from F7 + the tally (other): Formula: Kalshi E (5/15) × 2,691 ≈ 900; Kalshi Q (4/15) × 2,691 ≈ 720; Polymarket E (4/15) × 1,951 ≈ 520; Polymarket Q (3/15) × 1,951 ≈ 390 events. Assumptions: A-1 to A-3; "other" mixes Kalshi Politics, Financials, Entertainment and weather with Polymarket geopolitics, stocks, culture and weather. Range: Kalshi E 410-1,600, Kalshi Q 290-1,400, Polymarket E 210-1,000, Polymarket Q 140-880. Low confidence.
- **I6** from I1-I5 (totals): Formula: the sum of the I1-I5 base values gives Kalshi E ≈ 8,300 of 13,038 (64%), Kalshi Q ≈ 4,000 (30%), Polymarket E ≈ 6,900 of 22,025 (31%), Polymarket Q ≈ 3,500 (16%) events. Assumptions: as in I1-I5, with topic strata treated as independent. Range: the sum of per-topic Wilson bounds (wider than a joint interval): Kalshi E 5,300-10,600, Kalshi Q 1,800-7,200, Polymarket E 3,500-11,900, Polymarket Q 1,400-8,700. These figures count events, not matched pairs: one Kalshi race can match several Polymarket events and the reverse (F11). Medium confidence on order of magnitude.
- **I7** from F9 + F10 + F11 + the tally: event-level overlap overstates bet-level overlap. Of the 150 drawn events, 65 had a listing on the other venue for the identical occurrence, but only 38 (verified) for the identical question type and window. Most of the difference comes from three things: Kalshi's per-candidate vote-percent and turnout events, Polymarket's halftime, corners and exact-score sub-events, and different observation times for crypto prices. Not stated by any single source; high confidence in the direction, low in the size.
- **I8** from F1 + F2 + F3 + F4: on the verifier's 20:11 UTC read, Kalshi's two event listings agreed (12,918 from `status=open`, 12,918 with an active market from `min_close_ts`), while `/markets?status=open` does not give a count of single-event markets, because over 99% of what it returns is multivariate combos, which the events endpoint excludes (F3). An earlier pull of `status=open` returned far fewer events and could not be reproduced (Gaps), so whether the event listings stay in step over time is unknown. Medium confidence.
- **I9** from F6 + F12 + F14: catalog counts are not like-for-like. Polymarket counts include thousands of pre-created 5-minute and 15-minute windows (F12) and a number of events past their end date (F14), while Kalshi's count includes several events per election race (F11). Raw catalog totals therefore overstate how many distinct real-world occurrences each venue covers, by an amount this pass did not measure. Medium confidence.

## Gaps

- ⚠ Answered-when evidence missing: is there a verified, dated overlap count by topic? This pass has catalog counts by
  topic (F7) and agent-judged estimates from 15-event draws (I1-I6) with wide intervals — why: no human rater in an
  unattended run; draw size limited by budget — would close it: the human spot-checks the 150 judgments listed in
  Inferences (U-06), a blind second rater, and larger draws per topic
- ⚠ How much do E/Q judgments change once the rules are read? Judgments used titles, tickers, slugs and market labels
  only — why: settlement rules are R03's scope and were not read here — would close it: read both venues' rules for
  every E+Q pair (R03)
- ⚠ Do the 7 unverified items have counterparts? Sports Q for KXLALIGATOTAL, KXMLSTOTAL, KXWTAGTOTAL and KXNFL2H; PM
  pm830305 (Kelce); crypto KXZECMAXMON and KXSOLD-26OCT1617. The listings were found, but the specific market or
  observation time was not confirmed — why: cap discipline; the market labels needed were not printed — would close
  it: open each listing's markets and rules
- ⚠ At what time of day do Polymarket's daily `X above ___ on <date>?` crypto markets observe the price? v1's note
  says noon ET (v1, one-shot (main@9c3bf48), not re-measured) — why: unverified lead from v1 `research/overlap_census.md`,
  rules not read — would close it: read those markets' rules (R03)
- ⚠ Did the retrieval miss counterparts? Retrieval was title TF-IDF plus targeted regular expressions. Naming
  differences (Kalshi city names vs Polymarket nicknames, `Florida's 27th District` vs `FL-27`) hid some pairs in
  the first pass, so all shares are lower bounds of unknown size — why: no ground-truth recall check — would close
  it: a recall audit on a hand-built list of known pairs (R05 method)
- ⚠ How do market-level (contract-pair) overlap counts compare with these event-level counts? — why: not measured;
  out of budget — would close it: market-level matching over the F1/F5 snapshots
- ⚠ Would another topic taxonomy change the per-topic picture? The five topics are this file's mapping (Kalshi
  Politics and Financials, and Polymarket geopolitics and stocks, all fall into "other") — why: no shared taxonomy
  across venues — would close it: write-prd fixes the topic list
- ⚠ How much do Manifold (39,757 open markets) and PredictIt (203 markets) overlap with the two venues measured here,
  and may a US broker's clients use them? — why: not matched in this pass; eligibility is R01/R06 scope — would
  close it: the same draw-and-judge pass against those catalogs; R06 eligibility answer
- ⚠ Kalshi `status=open` events listing returned 5,000 (19:40 UTC) and 4,800 (about 19:47 UTC) events and missed
  open KXMLBGAME, KXEPLGAME and KXETHD events, with 0 for KXNFLGAME, KXFEDDECISION and KXBTCD: refuted on re-read —
  why: verifier pulls at 20:11 and 20:17 UTC returned 12,918 and 12,847 events and missed none of those series (F2);
  no saved responses show whether the earlier pull hit a transient API state or erred — would close it: repeated
  timed pulls of both event listings with responses saved (R08)
- ⚠ Kalshi `/markets?status=open` paged to the end at 220 pages with 219,000 markets, and its 58 single-event
  markets sat in 25 events absent from the `status=open` events listing: refuted on re-read — why: the verifier's
  pull at 20:17 UTC was still returning full pages at 401,000 markets, and all 49 single-event events it found were
  in the events listing (F4) — would close it: a full pull with no page cap, repeated at several times
- ⚠ Manifold `search-markets` refuses `offset` paging past 1,000: unsupported — why: a verifier request with
  `offset=1000&limit=100` at 20:18 UTC returned 100 markets — would close it: test offset and limit combinations
  against Manifold's API reference
- ⚠ How do the counts vary over time? This is one Sunday-afternoon snapshot (19:40-20:03 UTC); sports and crypto
  counts swing with schedules (F13) — why: one snapshot only — would close it: repeated snapshots across days (R08)
- ⚠ Did the payloads contain prompt-injection text? None was seen in the fields read (titles, slugs, tickers, tags,
  labels), but whole payloads were not read — why: payloads were parsed for named fields only — would close it: none
  needed unless free-text fields (rules, descriptions) are ingested later

## Handoffs

- write-prd: Is "equivalent" defined at the event level (same occurrence) or the question level (same proposition and window), given 65 of 150 drawn events had a same-occurrence listing but only 38 a same-question one? — rests on F9, F10, F11, I7
- write-prd: Which topics does the labelled set behind NS-01/SM-01 draw from, and is it stratified by topic, given elections show the highest same-question rate and sports the largest absolute overlap? — rests on F7, I1, I3, I6
- write-prd: Are 5-minute and 15-minute crypto windows in scope, given Polymarket lists thousands and the Kalshi listing held one current 15-minute BTC event? — rests on F6, F12, I4
- write-prd: Which topic taxonomy will metrics be reported under, given the venues' native categories and tags do not line up? — rests on F1, F6, F7
- design-hld: How does the Kalshi adapter enumerate the open catalog, given the `status=open` and `min_close_ts` event listings agreed on one read but their stability over time is untested, and `/markets?status=open` is over 99% multivariate combos? — rests on F1, F2, F3, F4, I8
- design-hld: Does matching run at market grain rather than event grain, given Polymarket splits one game into several sub-events and Kalshi splits one race into winner, margin, vote-percent and turnout events? — rests on F11, I6, I7
- design-hld: How often are catalogs re-pulled, given net counts moved by -51 (Kalshi) and +128 (Polymarket) in ten minutes and Polymarket keeps some past-dated events open? — rests on F13, F14
