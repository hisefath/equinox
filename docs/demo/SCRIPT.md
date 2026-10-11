# Demo video script (about 5 minutes)

Narrated, terminal-first (decode T-7: a silent scroll shows no reasoning). Everything runs offline on the recorded
real venue responses (`-replay testdata/live-20261011`, the default), so every number below repeats on every take.
Before recording: a large terminal font, an empty `data/` (or delete `data/decisions.jsonl`), and the repo root as
the working directory.

## 0. Build (off camera, or 5 s on camera)

**Type:** `go build -o bin/equinox ./cmd/equinox`
**Screen:** nothing; it returns to the prompt.

## 1. Framing (30 s)

**Show:** README.md title and first paragraph.
**Say:** "PEAK6 Labs asked a feasibility question: can prediction markets on different venues be normalized into one
form, what does that cost us, and how hard is routing on top of it? Equinox is the evidence: a small working system
that reads Kalshi and Polymarket, decides which markets are truly the same bet, and simulates routing an order between
them. It places no orders. I'll end with the verdict."

## 2. One refresh over both venues (40 s)

**Type:** `./bin/equinox refresh -top 10`
**Screen:** one line per venue (`listed · considered · inactive · too short · unreadable`), then
`pairs: N equivalent · N related, not routable · N undecided (for offline AI) · N near misses`, then the top pairs.
**Say:** "Each venue lists thousands of markets; we only consider the active ones, those that traded in the last
24 hours and last at least an hour. Of those, this many pairs pay the same side in every outcome, this many are
related but not routable, and this many the parser couldn't read, so they're left undecided, never guessed."

## 3. One equivalent pair and its evidence (40 s)

**Show:** scroll the refresh output to `96fd378f2b486591` (Messi, Ballon d'Or).
**Screen:** the Kalshi question `⇄` the Polymarket question, then `evidence: [...]` and any `caveats:`.
**Say:** "Here's an equivalent pair: Messi winning the Ballon d'Or on both venues. The evidence lists each settlement
part we compared (the question, the source, the window, the threshold and the edge-case rules) and they all agree.
Only payout timing may differ, and if it does it shows up as a caveat."

## 4. A rejected look-alike (30 s)

**Show:** the `near miss:` lines at the bottom of the same output; point at the Rotten Tomatoes pair.
**Screen:** `near miss: <question A> ⇄ <question B> — threshold differs` (or the part the line names).
**Say:** "This is the dangerous case: two markets that read almost the same but use different Rotten Tomatoes score
thresholds. A score between the two pays YES on one venue and NO on the other, so the system names the differing
part and refuses to route across it."

## 5. Routing that pair: YES, 500 contracts (60 s)

**Type:** `./bin/equinox route -pair 96fd378f2b486591 -side yes -qty 500 -region US`
**Screen:** JSON decision: `"venue": "kalshi"`, `all_in` $21.35, and in `venues` Polymarket with
`"eligible": false` and its reason.
**Say:** "For a US customer only Kalshi is eligible: Polymarket isn't open to US customers, and that's data, not a
hard-coded rule. So the order goes to Kalshi at $21.35 all-in, and the decision says why the other venue was left out."

**Type:** `./bin/equinox route -pair 96fd378f2b486591 -side yes -qty 500 -region non-US`
**Screen:** `"venue": "polymarket"`, `all_in` $11.32274, `deciding_rule` lowest all-in cost, `saving` $10.03, and
Kalshi's own all-in cost in `venues`.
**Say:** "For a non-US customer both venues are eligible, so the rule is the lowest all-in cost after fees: Polymarket
at $11.32274, a saving of $10.03 over Kalshi. Money is integer micro-dollars, which is why you see five decimals."

**Type:** `./bin/equinox decisions -n 2`
**Screen:** the two decisions just made, newest first, read back from `data/decisions.jsonl`.
**Say:** "Every decision, including refusals, is written to the log before it's returned, so we can audit it later."

## 6. Determinism (25 s)

**Type:**
```bash
r() { ./bin/equinox route -pair 96fd378f2b486591 -side yes -qty 500 -region non-US \
      | grep -v -e decision_id -e logged_at -e request_id -e snapshot_as_of; }
diff <(r) <(r) && echo IDENTICAL
```
**Screen:** `IDENTICAL`.
**Say:** "Same order, same market data, same decision, down to the micro-dollar. Only the IDs and timestamps differ.
Routing is a pure function over a frozen snapshot, never calls a venue and never calls a model."

## 7. Graceful failure (40 s)

**Type** (a replay with the offshore venue's recordings missing, outside the repo):
```bash
mkdir -p /tmp/eqx-down && cp -R testdata/live-20261011/kalshi /tmp/eqx-down/
./bin/equinox refresh -replay /tmp/eqx-down -top 3
```
**Screen:** `kalshi healthy ...`, `polymarket unhealthy ...` with `last error: ...`, and `0 equivalent`. No crash.
**Say:** "Now the offshore venue is down. The system doesn't crash or guess: it marks that venue unhealthy, shows the
error, and keeps serving the other venue."

**Type:** `go test ./internal/web -run VenueDown -v`
**Screen:** `--- PASS: TestVenueDownKeepsLastGoodData`.
**Say:** "In the running server, a venue that goes down mid-session keeps its last good data, is marked unhealthy,
and ages out of decisions after 90 seconds. This test cuts a venue off and checks that orders still get an
explained decision that names the unhealthy venue."

## 8. Measured matching quality (30 s)

**Show:** `docs/TEST_RESULTS.md`, the measurement section.
**Screen:** precision with its confidence interval, recall, and Cohen's kappa: [numbers from TEST_RESULTS.md].
**Say:** "Pair counts say nothing about correctness, so we measured. A fixed-seed random sample of equivalent pairs,
labelled blind by an AI rater that never sees our verdict, gives precision of [numbers from TEST_RESULTS.md] against a
0.90 gate. Recall is the guard against matching too little, and kappa against the human's blind labels on 30 of the
same pairs checks that we aren't just letting a model grade a model."

## 9. Where AI is and isn't used, then the verdict (30 s)

**Show:** `docs/AI_USAGE.md`, the Summary table.
**Say (15 s):** "AI built this under human gates: every decision is in the journal. Inside the product there is no
AI in routing. AI only adjudicates undecided pairs offline, a person reviews those verdicts, and the system works
with AI turned off."
**Say (close):** "So, the verdict for PEAK6 Labs: yes, venues can be normalized, but 'same title' is not 'same bet';
the cost is a strict definition of equivalent, settlement parsing, and measuring the error rather than asserting it.
Once that's done, routing itself is the easy part: a small deterministic function over eligibility, depth and fees."
