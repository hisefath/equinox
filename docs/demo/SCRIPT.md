# Demo video script (about 5 minutes)

Narrated (decode T-7: a silent scroll shows no reasoning). The story follows the page's three steps (pick a pair,
place a test order, see where it goes and why), then switches to the terminal for the proof: determinism, failure
handling and measured quality. Every number repeats on every take, because both the hosted page and the local
binary replay the same recorded real venue responses (`testdata/live-20261011`).

**Before recording**
- Browser: https://equinox-production-92b0.up.railway.app, zoomed to 110-125%. Or run it locally with
  `./bin/equinox serve` and open http://localhost:8080; the numbers are the same.
- Terminal: large font, the repo root as the working directory, `go build -o bin/equinox ./cmd/equinox` already
  run.
- A tall window, so the order ticket stays on screen beside the pairs.

## 1. Framing (30 s): the page's top section

**Show:** the hero ("Same bet. Best venue.") and the three numbered step cards.
**Say:** "PEAK6 Labs asked a feasibility question: can prediction markets on different venues be normalized into one
form, what does that cost, and how hard is routing on top of it? Equinox is the evidence. It reads Kalshi and
Polymarket, decides which markets are truly the same bet, and simulates where an order should go. It places no
orders. The page follows three steps, and I'll end with the verdict."

**Point at** the stats line under the step cards: about 19,000 and 20,700 active markets, **12** equivalent pairs,
**2,116** look-alikes rejected.
**Say:** "Out of roughly forty thousand active markets, only twelve pairs pay out the same on both venues. That ratio
is the first finding: same title is not same bet."

## 2. Step 1, pick a matched pair (45 s)

**Scroll** to "1 Pick a matched pair" and find the Messi card ("Who will win the Ballon d'Or in 2026? — Lionel
Messi" ⇄ "Will Lionel Messi win the 2026 Ballon d'Or?").
**Point at** the "Agrees on" line (statistic, window, threshold, edge rule) and the "Worth checking" line.
**Say:** "Each card is one market on each venue. 'Agrees on' lists the settlement parts we compared, and every one
matches. 'Worth checking' is what the parser couldn't confirm, here the exact wording and the source. Those pairs
went through an offline AI review that a person approved before they could be routed."

**Click** "Show the closest rejected look-alikes" and point at the Ocasio-Cortez row: Kalshi asks about the
Democratic **Presidential** nominee, Polymarket about the **Vice**-Presidential nominee, marked *question differs*.
**Say:** "This is the dangerous case. The two read almost the same, but one is about the presidential nominee and
the other about the vice-presidential one, so they can pay opposite ways. The system names the part that differs and
never routes across it."

## 3. Step 2, place a test order (30 s)

**Click** "Route this pair →" on the Messi card. The card turns to "Selected ✓" and the order ticket shows the
Messi market.
**Set:** Buy YES, 500 contracts, Customer lives: **Outside the US**. Leave max price empty.
**Say:** "I'm placing a test order: buy YES, five hundred contracts, for a customer outside the US. Note the hint:
US customers can only use Kalshi. That's eligibility as data, not a hard-coded rule."

## 4. Step 3, where it goes and why (60 s)

**Click** "Route order →".
**Screen:** "Send it to **Polymarket**", **$11.32274**, "for 500 yes contracts at $0.022 each, plus $0.32274 in
fees", "Why: lowest total cost after fees · saves $10.02726", and two venue cards: Polymarket *Chosen*, Kalshi *Not
chosen* at $21.35.
**Say:** "Both venues are allowed for this customer, so the rule is the lowest total cost after fees: Polymarket at
eleven dollars thirty-two, saving about ten dollars over Kalshi. Money is integer micro-dollars, which is why you see
five decimals. Every venue gets a reason, including the losers."

**Click** "Try it for a customer in the US →".
**Screen:** "Send it to **Kalshi**", **$21.35**, "Why: it is the only venue this customer may use"; the Polymarket
card shows *Not allowed* and "not open to US customers".
**Say:** "Same order, US customer. Polymarket isn't open to US customers, so it's excluded and the order goes to
Kalshi at twenty-one thirty-five. The decision says exactly why the other venue was left out."

**Click** "decision log" in the "Why these two markets are the same bet" box (or the nav link).
**Say:** "Every decision, refusals included, is written to this log before the answer comes back, so it can be
audited later."

## 5. Determinism (25 s): terminal

**Type:**
```bash
r() { ./bin/equinox route -pair 96fd378f2b486591 -side yes -qty 500 -region non-US \
      | grep -v -e decision_id -e logged_at -e request_id -e snapshot_as_of; }
diff <(r) <(r) && echo IDENTICAL
```
**Screen:** `IDENTICAL`.
**Say:** "Same order and same market data give the same decision, down to the micro-dollar; only IDs and timestamps
differ. Routing is a pure function over a frozen snapshot: it never calls a venue and never calls a model."

## 6. Graceful failure (40 s): terminal

**Type** (a replay with the offshore venue's recordings missing):
```bash
mkdir -p /tmp/eqx-down && cp -R testdata/live-20261011/kalshi /tmp/eqx-down/
./bin/equinox refresh -replay /tmp/eqx-down -top 3
```
**Screen:** `kalshi healthy ...`, `polymarket unhealthy ...` with `last error: ...`, and `0 equivalent`. No crash.
**Say:** "Now the offshore venue is down. The system doesn't crash or guess: it marks that venue unhealthy, shows
the error, and keeps serving the other venue. On the page, the green dots under the order ticket would turn red."

**Type:** `go test ./internal/web -run VenueDown -v`
**Screen:** `--- PASS: TestVenueDownKeepsLastGoodData`.
**Say:** "In the running server, a venue that drops mid-session keeps its last good data, is marked unhealthy and
ages out of decisions after 90 seconds. This test cuts a venue off and checks that orders still get an explained
decision."

## 7. Measured matching quality (30 s)

**Show:** `docs/TEST_RESULTS.md`, section 3.
**Screen:** precision 12/12 = 1.00 (95% interval 0.76-1.00), recall 12/15 = 0.80, a three-model blind quorum
unanimous on all 30 pairs.
**Say:** "Pair counts say nothing about correctness, so we measured. A blind rater checked a fixed-seed sample:
twelve out of twelve correct. But twelve is small, so the lower bound is 0.76, under our 0.90 gate: not proven yet.
Recall is 0.80; the three misses write a threshold differently. A blind quorum of three models agreed on all thirty
pairs, but they are one vendor's models, so a human rater is the next step."

## 8. Where AI is and isn't used, then the verdict (30 s)

**Show:** `docs/AI_USAGE.md`, the Summary table.
**Say:** "AI built this under human gates, and every decision is in the journal. Inside the product there is no AI
in routing: AI only reviews undecided pairs offline, a person approves those verdicts, and the system works with AI
turned off."
**Say (close, back on the page's hero):** "So, the verdict for PEAK6 Labs: yes, venues can be normalized, but same
title is not same bet. The cost is a strict definition of equivalent, settlement parsing, and measuring the error
instead of asserting it. Once that's done, routing is the easy part: a small deterministic function over
eligibility, depth and fees."
