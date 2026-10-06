# Equivalence: what "the same market" means, and how Equinox decides it

## 1. Definition

> Two binary contracts **A** (venue X) and **B** (venue Y) are **equivalent** if and only if, for every
> plausible state of the world, A resolves YES exactly when B resolves YES.

In practice that means all of the following hold:

1. **Same subject.** The same person, party, team, asset or index, after aliasing (Kalshi "Milwaukee" =
   Polymarket "Brewers").
2. **Same predicate.** Win, be nominated, qualify, appear on the ballot, be relegated, leave office and
   be "the next" office-holder are all different.
3. **Same threshold and comparator.** `>` ≠ `≥` ≠ "exactly" ≠ a range ≠ "reaches at any time". A different
   strike is a different contract.
4. **Same window.** A different deadline is a different contract. Wording conventions are reconciled:
   "before Nov 1" = "by Oct 31".
5. **Same scope.** Senate ≠ House; ALCS ≠ World Series; core CPI ≠ CPI; YoY ≠ MoM; the Women's World Cup ≠
   the World Cup.
6. **Compatible resolution source.** A CF Benchmarks BRTI average ≠ a Binance 1-minute candle; Central
   Park ≠ LaGuardia.

Differences in **settlement timing** do not break equivalence. Kalshi pays when the winner is sworn in;
Polymarket pays when the race is called. The YES condition is the same, so this is reported as a
**caveat**, because it changes how long capital is locked up.

### Why so strict

A false match makes the router treat two different bets as one. Best execution then sends money into
the wrong contract, or leaves an unhedged position. A missed match only gives up a price improvement.
The history of "identical" markets that resolved in opposite directions makes this concrete:

| Event | Kalshi | Polymarket | Why they diverged |
|---|---|---|---|
| US government shutdown in 2024 | **No** | **Yes** | One required an observed OPM shutdown status; the other counted a missed funding deadline |
| Khamenei out (2026) | settled at **$0.02** under a death carve-out | **Yes** | An exception clause existed on one venue only |
| 2024 presidential election | paid at **inauguration** | paid on **network calls** | Timing and source differed: a subset relation, not identity |

(Sources and raw API captures: [`research/prior_art.md`](../research/prior_art.md).)

So Equinox is built **precision-first**. It would rather miss a pair than invent one, and it puts a
reviewed mapping table between the matcher and the router.

## 2. Methodology

The design is a classic record-linkage pipeline adapted to prediction markets
([`diagrams/matching-pipeline.mmd`](diagrams/matching-pipeline.mmd)). The code is
[`internal/match`](../internal/match).

### 2.1 Normalization (`text.go`)

Each canonical market contributes `event title + question + outcome label`. Normalization:

- **Case, punctuation and accents.** Lowercase; fold accented letters ("Erdoğan" → "erdogan"); keep
  digit hyphens ("MI-07"), drop word hyphens ("year-over-year"). Comparators become words: `≥` → "at
  least", `>` → "above", `↑` → "reach".
- **Venue phrasing to canonical tokens.** "Federal Reserve"/"FOMC" → `fed`; "bps"/"basis points" → `bp`;
  "no change"/"maintains rate" → `hold`; "decrease"/"lower" → `cut`; "Pro Baseball" → `mlb`; "World
  Series" → `mlb championship`; "Democratic"/"Democratics" → `democrat`.
- **A light stemmer**, so "dissenting"/"dissent" and "released"/"release" meet.
- **Team resolution.** 124 MLB/NFL/NBA/NHL teams map city labels (Kalshi: "Tampa Bay", "New York Y") and
  nicknames (Polymarket: "Rays", "Yankees") to team ids. Nicknames are read first. A bare city resolves
  to every team it could mean, so "Tampa Bay" alone could be the Rays, the Buccaneers or the Lightning.

### 2.2 Features (`extract`)

| Feature | Example |
|---|---|
| TF-IDF tokens (outcome tokens weighted ×2) | `fed`, `hold`, `oct`, `meet` |
| Context tokens: everything except the outcome label | Used to check that the *question* still matches once the shared name is removed |
| Numbers, excluding years and date parts | "$84,000" → `84000`; "6-9M" → `6, 9000000`; "Gemini 4.0" → `4` |
| Comparator shape | `>` `≥` `<` `≤` `=` `range` `touch` |
| Explicit dates, bare months, years | "before Nov 1, 2026" → day 305 |
| Scopes | rate action, office, predicate, competition, stage, indicator, measure, direction, segment |
| Ranks | "#2", "top half", "top-ranked" |
| Modifiers | core, fantasy, women's, any, final, meeting, announce/confirm, relative ordering ("next to", "first to"), negated outcomes ("Neither", "No IPO") |
| Oracles from the rules text | Binance, CF Benchmarks/BRTI, Chainlink, LaGuardia, Central Park… |
| Teams and leagues | Leagues are pinned by nicknames ("Ravens" → NFL) or league words |
| Rare subject tokens | Question words in at most 0.2% of markets: names and places |

### 2.3 Candidate generation (blocking) and scoring

- An inverted index covers tokens that appear in at most 5% of markets, so common words never create
  candidates.
- For each market, the 25 best partial matches are fully scored by TF-IDF cosine. Pairs below 0.30 are
  dropped.
- In the live run this scored **97,658** candidates out of a possible 193 million pairs.

### 2.4 Vetoes: where precision comes from

A single veto rejects a pair whatever its score. In evaluation order:

| Veto | Rejects | Live example it caught |
|---|---|---|
| threshold | different numbers | Spread −2.5 vs −1.5; "#2" vs "#1 on the Hot 100" |
| comparator | different shape | "Above 2.4%" vs "2.4%"; "$84,000 or above" (≥) vs "above $84,000" (>); "above $95k" vs "reach $95k" |
| comparator, one-sided | one side states `>`/`≥`/range, the other doesn't (except spreads/totals) | "Astra 6.1+" vs "Astra 6.1"; "at least one cut" vs "one cut" |
| date | explicit dates more than a day apart | Merz "before Nov 1" vs "before Nov 30" |
| deadline | dates a day apart *and* trading cut-offs more than 12 h apart | "before Oct 23" vs "by Oct 23" |
| month / year | bare months disjoint; years ≥ 2 apart | Fed October vs December; Senate 2028 vs 2026 |
| resolution source | different known oracles | BRTI vs Binance |
| league / team | different league or team | NFL Raiders playoffs vs WNBA Aces; Atlanta vs the "Ravens" outcome |
| outcome | outcome labels share no entity | Andy Barr vs Julia Letlow |
| rank | different or one-sided rank | "#2 seed" vs champion; "top half" vs champion |
| modifier | present on one side only | fantasy D/ST vs Super Bowl; "any state" vs Texas; "next leader out" vs "leaves by date" |
| timing | expected resolutions more than 180 days apart | |
| scope | different values in a scope group | cut vs hike; Senate vs House; VP vs President; nominee vs win; ALCS vs World Series; CPI vs PCE |
| subject | each side names something rare the other never mentions | Greenland vs Alberta independence |
| context | under 0.30 similarity once the outcome label is removed | "Spain wins the Women's World Cup" vs "Spain participates in Eurovision" |

Every rejected candidate with a high score is kept as a **near miss**, with its veto. These are the most
instructive output of the system (`/near-misses`, or `equinox scan -near-misses N`).

### 2.5 Assignment and tiers

- Surviving candidates are assigned **one-to-one per venue pair**: greedy by score, ties broken by
  market keys.
- A pair scoring ≥ 0.50 is **`equivalent`** (routable). Otherwise it is **`review`**, which is never
  routed by default.
- Each pair carries **evidence** (shared terms, agreeing team, outcome, threshold, comparator, scopes) and
  **caveats** (comparator stated on one side only, oracle identified on one side only, resolution times
  far apart).

### 2.6 The reviewed mapping table

`reviews/pairs.json` holds verdicts: `{pair, verdict, source, note}`.
- A confirming verdict promotes a pair (even from `review`).
- A rejecting verdict blocks it (tier `rejected`).
- With `-require-review`, **only confirmed pairs route**. This is the production setting.

The committed table holds 247 verdicts from the two live audits (§3.3), marked
`source: llm-audit: 2 independent judges`. A human reviewer should spot-check them before relying on
them.

## 3. Evaluation

All numbers are for the frozen matcher in this repository. Raw outputs are in
[`test-results/`](../test-results/) and [`research/audit*/`](../research/).

### 3.1 Hand-labelled live pairs (CI gate)

There are 55 cross-venue pairs, hand-labelled from a live census
([`research/overlap_census.md`](../research/overlap_census.md)):
- **32 equivalent**, across Fed, elections, geopolitics, tech, economics, sports and entertainment.
- **23 hard negatives**, chosen to fool text similarity: threshold vs bucket, different date window,
  different resolution source, inverse polarity, boundary inclusivity, related-but-different.

Each pair goes through the **real adapters** from its raw API JSON and is judged against term statistics
from the full **60,234-market** recorded corpus.

| | Result |
|---|---|
| Precision | **1.000** (0 of 23 hard negatives accepted, each rejected for the intended reason) |
| Recall | **0.844** (27 of 32) |
| In context (full pipeline, 60k markets, siblings competing) | 26 found, **0 false positives** |

`TestLabelledPairs` fails the build if any labelled non-equivalent pair is accepted, or if recall drops
below 0.75.

### 3.2 Why a second evaluation was needed

The labelled set is mostly **templated** markets: Fed buckets, Senate races, game lines. The first live
run showed a different failure mode. Pairs shared a rare *name*, but the *question* was different:
- Spain in the Women's World Cup vs Spain in Eurovision.
- Warnock for VP vs Warnock for President.
- Kalshi's "top half of the EPL" ladder vs Polymarket's "wins the EPL".

To measure that, live matches were audited.

### 3.3 Live audits (independent judges)

Method:
- Take a stratified random sample of live pairs.
- Each pair is judged against the definition above by an independent LLM judge reading both venues'
  question, outcome and rules excerpt.
- Every "not equivalent" or "uncertain" verdict is re-judged by a second, skeptical reviewer. Only
  agreeing negatives count as false positives.
- The judges never see the matcher's tier or score.

| Audit | Matcher version | Sample | `equivalent`-tier precision | `review`-tier precision |
|---|---|---|---|---|
| 1 (exploratory) | first version | 100 equivalent + 30 review | **0.626** (62 / 99) | 0.233 |
| 1, re-scored after fixes | final | same pairs (**in-sample**, optimistic) | 0.945 (52 / 55) | — |
| **2 (out of sample)** | **final** | 96 equivalent + 24 review, **none seen before** | **0.872** (82 / 94; 95% CI 0.79–0.93) | 0.167 |

Reading these results:
- **The tiers mean something.** `equivalent` is right about 87% of the time; `review` about 17%. Not
  routing `review` is correct.
- **Audit-driven vetoes worked.** Out-of-sample precision rose from 0.63 to 0.87.
- **The remaining 13% is mostly rules-level**, which vocabulary cannot reach. Examples:
  - announce vs complete an IPO;
  - first round vs runoff;
  - Polymarket's Fed "by the December meeting" window vs Kalshi's "by December 31";
  - "next PM" vs the deadline of a coalition-formation window;
  - in-person meeting vs "phone call counts".

### 3.4 Failure taxonomy (all audited false positives)

| Pattern | Share | Fixable deterministically? |
|---|---|---|
| Different predicate on the same subject (top half vs title, relegated vs win, sold vs win) | high, before fixes | **Yes**: predicate/rank vetoes added |
| Relative vs absolute ("next leader out" vs "leaves by date") | medium | **Yes**: ordering modifier added |
| Different subject sharing a token (Greenland vs Alberta; Eagles the band vs Eagles the team) | medium | Partly: rare-subject veto; the band/team homonym remains |
| Announce vs complete; agreed vs signed; any-time vs snapshot | medium | **No**: needs a reading of the rules text |
| Window boundaries (Dec 31 vs December FOMC; Aug 2028 vs Mar 2028) | medium | Partly: needs date parsing of the rules text |
| Composition order (D-House/R-Senate vs D-Senate/R-House) | low | Yes, with a composition parser (not built) |
| Counting units (one 50 bp hike = 1 event on Kalshi, 2 steps on Polymarket) | low | Needs domain knowledge |

## 4. Known gaps (recall)

- **Open-ended markets with placeholder dates.** Kalshi stamps "next PM of Romania" with a resolution
  date about 17 years out; Polymarket uses a near date. The 180-day timing veto then rejects identical
  questions (the top near misses in the live run). Fix: treat resolution dates more than 5 years out as
  "no deadline". Deliberately **not** applied after the audit, so that the measured numbers describe the
  shipped code.
- **Product vs company** ("Gemini" vs "Google"). Not aliased on purpose: the labelled set contains a
  look-alike pair with a different resolution source.
- **Structural equivalence across market types.** "Wins the ALDS" = "advances to the ALCS". A stage veto
  blocks this.
- **Low-similarity sports series markets** land in `review` (scores 0.32–0.45) and need a reviewer.

## 5. Path to production

1. Keep the deterministic matcher as a **candidate generator and high-precision filter**.
2. Send `equivalent` and `review` pairs to **offline adjudication**: an LLM reads both rule sets and
   produces a structured comparison of the YES regions, with a human confirming. The result is written to
   the **reviewed mapping table**. LLMs stay out of the routing path, which stays deterministic.
3. Route production orders only with `-require-review`.
4. Feed every confirmed false positive back as a **regression fixture**, as was done for the eight live
   false positives in `TestLiveFalsePositivesAreVetoed`.
5. Re-run the out-of-sample audit on every matcher change, and track precision and recall per category.
