# Decision journal

Why each design choice was made and what it cost (software-factory-kit WORKFLOW.md §9). Committed, append-only.

- Written only by the `log-decision` skill, after every gate. It records decisions the **human** made; an entry is
  never `accepted` without the human's quoted words.
- Never renumber or delete an entry. Change your mind by adding a new D-row and marking the old one
  `superseded-by D<m>`. Rejected options stay visible.
- One-way (Type 1) decisions are also promoted to `docs/decisions/NNNN-<slug>.md`.
- Who said yes, word for word, lives in `.handoff/TRACKER.md ## Approvals`; this file holds the reasoning.
- Topics group decisions in the project review, `docs/decisions/REVIEW.md`, which
  `scripts/decision-views.sh` generates from this ledger. Changing the list is itself a decision.

Topics: Product · Data · Security · Money · Infrastructure · Process

## Entry schema

Copy this block for each new entry (D-ids increase by one, starting at D1):

```markdown
### D<n> <title>  [status: proposed | accepted | superseded-by D<m> | rejected]
- Date (`date`) · phase · milestone · Decided by: human · Proposed by: <skill> (<harness>, <model>) · artifact@<sha>
- Context and forces (FR/NFR ids)
- Options: always includes "do nothing" and the agent's first proposal ("instead of")
- Choice · Why (Y-statement: in context C, facing F, we chose X over Y to get Z, accepting W)
- Cost accepted · Reversibility (Type 1 one-way / Type 2 two-way) · Confidence (low/med/high)
- Revisit if · Evidence (file:line, R/S ids) · Interview line (one sentence you'd say out loud)
- Review: Topic <topic> · Weight <●●●|●●○|●○○> · One-liner: <at most 15 words>
```

## Entries

<!-- log-decision appends D1 below this line. -->

### D1 Restart Equinox on a branch; v1 stays in main  [status: accepted]
- Date 2026-10-11 14:03 CDT · phase 0 (setup, init-repo) · milestone M1 · Decided by: human, decision: restart on a new branch from main; v1 stays in main's history 2026-10-11 13:45 CDT: "So i want a fresh start, but to still keep the repo history. so we did the one shot prompt method, and i want that history to stay on the repo, but on a new branch from main, i want to restart the project with the new software factory kit process, for this peak 6 equinox project ." · Proposed by: supervisor (Claude Code, claude-opus-5-5), the human's instruction; no options card was shown · artifact: main@9c3bf48
- Context and forces: v1 was built one-shot in a single unattended session (2026-10-05/06), so its decisions were the AI's, not the human's. The human wants to practise software-factory-kit end to end on a graded partner project, but keep v1's history visible (decode.md T-3, T-12).
- Options:
  - Do nothing — ⚠ not offered at the gate (keep v1 on main as the only build)
  - Restart on a new branch from main, delete v1 on that branch, keep v1 in main's history (agent's first proposal — none recorded; this is the human's instruction) — cost: v1 is not browsable on the branch; every phase is re-run — chosen
- Choice: restart on `milestone/m1-research` from main@9c3bf48 · Why: In the context of a graded project whose v1 decisions were made by the AI alone, facing an interview where the human must defend every decision, we chose a fresh branch run through the kit's gates over keeping v1 as the build to get decisions the human made and can defend, accepting that v1 is reachable only through history.
- Cost accepted: v1's code and docs are gone from the branch (commit 59ff700); everything is rebuilt phase by phase. · Reversibility: Type 2 two-way — main@9c3bf48 is untouched, and dropping the branch restores the old state · Confidence: ⚠ not stated by the human or an artifact
- Revisit if: ⚠ what would make the human submit v1 instead of v2? · Evidence: commits 59ff700, 82e0ef7; approval decision: restart on a new branch from main; v1 stays in main's history 2026-10-11 13:45 CDT · Interview line: "I built Equinox one-shot first, then restarted on a branch so every v2 decision passes a human gate, with v1 kept in history."
- Review: Topic Process · Weight ●●○ · One-liner: Restart via the kit on a branch over keeping v1; costs a rebuild.

### D2 Frame Equinox as a feasibility verdict  [status: accepted]
- Date 2026-10-11 14:03 CDT · phase 0.5 decode-prompt · milestone M1 · Decided by: human, G-decode 2026-10-11 14:02 CDT: "approve" (interview Q1: "A") · Proposed by: decode-prompt (Claude Code, claude-opus-5-5) · artifact: specs/M1-cross-venue-equivalence-routing/decode.md@d20fb08
- Context and forces: The brief comes from PEAK6 Labs asking whether cross-venue normalization and routing are viable, while the company panel describes the AI Strategies team hiring AI engineers, so there may be two audiences (QB-1, QB-2, QB-3). The framing decides what the brief and PRD optimize for (REQ-04, REQ-19; T-1, T-13).
- Options:
  - Do nothing / simplest (shown as "Do nothing"): keep the literal requirements list as the frame — cost: features with no verdict, the one-shot trap again — rejected: the human chose A
  - A. Feasibility answer leads (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: less flashy, no AI features just for show — chosen
  - B. AI showcase leads — cost: risks the brief's no-polish, show-reasoning bar; AI inside the system strains "deterministic" — rejected: the human chose A
  - C. Equal weight — cost: diluted, two North Stars in the PRD — rejected: the human chose A
- Choice: A, build the brief on QB-1 with QB-2 as its core and carry the 14 trap guards · Why: In the context of a partner asking a go/no-go feasibility question, facing a second, AI-focused hiring audience, we chose to lead with the feasibility answer over an AI showcase or an equal split to get a brief graded on framing, ambiguity and justification, accepting no AI features added for show (QB-3 is shown through the process and the AI usage log).
- Cost accepted: no AI features just for show; the hiring signal must come from how the system is built and explained. · Reversibility: Type 2 two-way — the frame can be revised at G0 or G1 before any code exists · Confidence: high (QB-1 confidence in decode.md)
- Revisit if: ⚠ what signal (e.g. a reply from the technical contact) would make QB-3 lead? · Evidence: specs/M1-cross-venue-equivalence-routing/decode.md:49-53, :117; approval G-decode 2026-10-11 14:02 CDT · Interview line: "I framed Equinox as a feasibility verdict, because the partner asked whether this is viable, not for a product, and graded reasoning over polish."
- Review: Topic Product · Weight ●●○ · One-liner: Feasibility verdict leads over an AI showcase; no AI features for show.

### D3 Cite v1 as a prior experiment only  [status: accepted]
- Date 2026-10-11 14:03 CDT · phase 0.5 decode-prompt · milestone M1 · Decided by: human, G-decode 2026-10-11 14:02 CDT: "approve" (interview Q2: "A") · Proposed by: decode-prompt (Claude Code, claude-opus-5-5) · artifact: specs/M1-cross-venue-equivalence-routing/decode.md@d20fb08
- Context and forces: v1 holds useful evidence (venue API behaviour, fee formulas, a live overlap census, and labelled precision 1.0 vs about 0.63 on the first live audit), but its decisions were the AI's, made in one unattended session (T-12, T-3).
- Options:
  - Do nothing / simplest (shown as "Do nothing"): leave it undecided — cost: every later phase guesses and the copy-v1 trap stays open — rejected: the human chose A
  - A. Cite v1 as a prior experiment (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: some care with labels; re-checking facts takes time — chosen
  - B. Clean room — cost: redoes research that already exists and loses the 100%-vs-63% lesson — rejected: the human chose A
  - C. Evolve v1's code — cost: planning gets bent around code the AI already wrote, which defeats the restart — rejected: the human chose A
- Choice: A, no v1 code or design carries over; v1 facts may be cited, labelled "v1, one-shot", and re-verified; v1 numbers are never quoted for v2; a v1 design idea comes back only as one option at G2/G3 · Why: In the context of a restart meant to give the human decisions they can defend, facing real evidence locked in v1, we chose to cite v1 as a prior experiment over a clean room or evolving its code to keep the evidence without inheriting the decisions, accepting the work of labelling and re-checking.
- Cost accepted: labelling and re-verifying every v1 fact before use; no v1 code shortcut. · Reversibility: Type 2 two-way — a later gate can still bring a v1 idea back as an option · Confidence: ⚠ not stated in decode.md
- Revisit if: ⚠ which v1 component, if any, would be worth reusing as code? · Evidence: specs/M1-cross-venue-equivalence-routing/decode.md:87, :118; approval G-decode 2026-10-11 14:02 CDT · Interview line: "I treated my one-shot v1 as a prior experiment: I kept its evidence, like labelled precision overstating live precision, but re-made every decision myself."
- Review: Topic Process · Weight ●●○ · One-liner: Cite v1 as evidence over reusing its code; costs re-checking facts.

### D4 Finish v2 today; main changes only when green  [status: accepted]
- Date 2026-10-11 14:03 CDT · phase 0.5 decode-prompt · milestone M1 · Decided by: human, G-decode 2026-10-11 14:02 CDT: "approve" (interview Q3: "yeah deadlines tonight at 11:30pm"; Q4: "i dont want you t really be concerned about the deadline, lets get it done today") · Proposed by: decode-prompt (Claude Code, claude-opus-5-5) · artifact: specs/M1-cross-venue-equivalence-routing/decode.md@d20fb08
- Context and forces: The deadline is 2026-10-11 23:30 CDT, about 9.5 h after the restart began. The kit merges every milestone into main, and the restart branch deleted v1's files, so a docs-only merge would leave main with no working prototype (REQ-16, REQ-17; T-14).
- Options:
  - Do nothing / simplest (shown as "Do nothing"): keep planning as if there's no deadline — cost: same outcome as C — rejected: superseded by the human's "lets get it done today"
  - A. Submit v1, show the restart as work in progress (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: the submitted code is the one-shot build; the restart is "what I'm doing next" — rejected: "i dont want you t really be concerned about the deadline, lets get it done today"
  - B. Compressed v2 with v1 as a fallback, reusing v1's code — cost: about 9 h of gates nonstop; reusing v1 code reverses Q2 — rejected: the human chose to finish v2 today with Q2 standing
  - C. v2 from scratch, submit whatever exists — cost: likely incomplete at 23:30 — chosen in part: v2 is finished today from scratch, read with the T-14 guard that main keeps v1 until v2 is green and covers every REQ
- Choice: finish v2 through the kit today; milestone MRs stack on the restart branch, and main@9c3bf48 stays what graders see until v2's `make check` is green and covers every REQ · Why: In the context of a same-day deadline and a restart the human wants to complete, facing a kit that merges every milestone into main, we chose to finish v2 today over submitting v1 to get a submission built through human gates, accepting a compressed day with quick, recommended-default gates.
- Cost accepted: a compressed day of gates; main is not updated per milestone until v2 is complete; the deadline is not a planning input. · Reversibility: Type 2 two-way — main@9c3bf48 stays the fallback until the human says otherwise · Confidence: ⚠ not stated in decode.md
- Revisit if: ⚠ what state of v2 would make the human submit main@9c3bf48 instead? · Evidence: specs/M1-cross-venue-equivalence-routing/decode.md:89, :119-120; approval G-decode 2026-10-11 14:02 CDT · Interview line: "I chose to finish the gated rebuild the same day, but kept v1 on main until v2 passed every check, so graders never saw a broken build."
- Review: Topic Process · Weight ●●○ · One-liner: Finish v2 today over submitting v1; main keeps v1 until green.

### D5 Docs-only make check until the design gate  [status: accepted]
- Date 2026-10-11 14:27 CDT · phase 0 (setup, init-repo) · milestone M1 · Decided by: human, G-init 2026-10-11 14:26 CDT: "1" · Proposed by: supervisor (Claude Code, claude-opus-5-5) · artifact: scaffold@82e0ef7
- Context and forces: `make check` is the single pass/fail gate the pre-push hook and CI run, and the scaffold's placeholder fails on purpose. Go is the partner's preference and v1's language, but D3 and trap T-6 hold the stack open until the design gate (G2).
- Options:
  - Do nothing / simplest (shown as "Do nothing"): leave the failing placeholder — cost: the pre-push hook blocks every push and CI stays red — rejected: the human chose 1
  - 1. Docs-only checks until the design gate (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: the checks prove little until code exists — chosen
  - 2. Go checks now (gofmt, go vet, go test) — cost: picks the stack before design, against T-6 and D3, making G2 a formality — rejected: the human chose 1
- Choice: 1, `make check` = workflow scripts parse + embedded diagrams match their sources + decision review current; AGENTS.md says the stack is decided at the design gate and lists the project rules · Why: In the context of a docs-only first milestone, facing a partner preference for Go that design has not yet weighed, we chose docs-only checks over Go checks now to keep the design gate a real decision, accepting a gate that proves little until code exists.
- Cost accepted: until the stack is chosen, `make check` guards only scripts, diagrams and the decision review; one Makefile edit is owed at G2/G3. · Reversibility: Type 2 two-way — the Makefile's three recipes are replaced when the stack is chosen · Confidence: ⚠ not stated in the G-init card
- Revisit if: the design gate picks a stack (then lint, types and test get that stack's tools) · Evidence: Makefile:15-25, AGENTS.md:6-11, :57-63; approval G-init 2026-10-11 14:26 CDT · Interview line: "Before choosing a stack, my one pass/fail check only guarded docs and scripts, so the design gate stayed a real choice rather than a rubber stamp."
- Review: Topic Process · Weight ●○○ · One-liner: Docs-only checks until design over Go checks now; checks prove little early.

### D6 Run Equinox on the Full track  [status: accepted]
- Date 2026-10-11 14:36 CDT · phase 1 sharpen-brief · milestone M1 · Decided by: human, G0 2026-10-11 14:35 CDT: "approve and /gather-evidence" · Proposed by: sharpen-brief (Claude Code, claude-opus-5-5) · artifact: specs/M1-cross-venue-equivalence-routing/brief.md@c0e0111
- Context and forces: A graded partner project restarted to practise the whole workflow (D1), due 2026-10-11 23:30 CDT (D4), with 30 requirements across two deliverable lists, 8 research questions and no PRD yet (REQ-04, REQ-16). The approval also accepts the brief's frame (feasibility verdict, D2), the North Star candidate (matching precision) and the Q1/Q2 answers (public GitLab + video link; deploy to Railway, parked as P-10), and leaves U-01..U-03 at the default: partner documents are paraphrased, never quoted or committed.
- Options:
  - Do nothing / simplest (shown as "Do nothing"): no build; the time goes elsewhere — cost: main stays v1, the one-shot build the human did not decide, and the restart ends — rejected: contradicts D1 and D4
  - Full (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: about a dozen gates today, each a short answer; the most planning before code — chosen
  - Standard — cost: needs a PRD that does not exist; skipping it means the human never decides what "equivalent" means — rejected: no PRD exists
  - Fix — cost: nothing to fix; there is no code — rejected: no code
- Choice: Full · Why: In the context of a graded partner project restarted to practise the workflow end to end, facing a same-day deadline, we chose the Full track over Standard to get a PRD in which the human personally defines "equivalent", accepting about a dozen short gates today.
- Cost accepted: every phase runs (research, PRD, design, acceptance, tasks, build, review, ship) with a gate at each; scope, not time, bends if the day runs short. · Reversibility: Type 2 two-way — a track may move up mid-flight; dropping to Standard after the PRD exists is a supervisor re-route · Confidence: ⚠ not stated in the brief
- Revisit if: ⚠ what progress by when would make the human cut phases? · Evidence: specs/M1-cross-venue-equivalence-routing/brief.md:172; approval G0 2026-10-11 14:35 CDT · Interview line: "Even with a same-day deadline I ran the full gated track, because the PRD is where I decide what 'equivalent' means, and that is what's graded."
- Review: Topic Process · Weight ●●○ · One-liner: Full track over Standard; about a dozen gates in one day.

### D7 Equivalent means same payout in every outcome  [status: accepted]
- Date 2026-10-11 16:07 CDT · phase 3 write-prd · milestone M2 · Decided by: human, G1 2026-10-11 16:06 CDT: "approve, also push and /design-hld" (interview Q1: "ok") · Proposed by: write-prd (Claude Code, claude-opus-5-5) · artifact: docs/product/prd.md@40eb5d5
- Context and forces: The assignment requires a defined and justified "equivalent" (REQ-04) and it decides the North Star's true positives. R02 found 65 of 150 drawn events had a same-event listing on the other venue but only 38 a same-question one; R03 found look-alikes that settled on opposite sides over event definition, carve-outs, timing and source (FR-004, FR-005, FR-006).
- Options:
  - Do nothing / simplest — ⚠ not offered at the gate (leave "equivalent" undefined until design)
  - Same payout in every outcome the rules name, edge cases included; payout timing may differ as a caveat; anything else that differs is "related, not routable" (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: fewer routable pairs; edge-case rules must be read — chosen
  - Same event is enough — cost: about 1.7x more pairs, many of them different bets — rejected: the human chose the recommendation
  - Main outcome only, edge-case differences as caveats — cost: a carve-out like Khamenei 2026 would be routed — rejected: the human chose the recommendation
  - Payout timing also breaks equivalence — cost: drops identical-outcome pairs such as the 2024 election (about 75 days apart) — rejected: the human chose the recommendation
- Choice: same payout in every outcome, timing as caveat · Why: In the context of a router that may send money only into the bet the customer intended, facing look-alike listings that have settled on opposite sides, we chose the strict settlement-level definition over same-event or main-outcome matching to keep false matches out, accepting fewer routable pairs and the work of reading edge-case rules.
- Cost accepted: fewer routable pairs; every pair needs its edge-case rules compared; capital lock-up differences are disclosed, not prevented. · Reversibility: Type 2 two-way — loosening later means relabelling the sample and re-measuring, not rebuilding · Confidence: ⚠ not stated in the PRD
- Revisit if: measured recall (SC-002) shows the strict rules miss most true pairs · Evidence: docs/product/prd.md:137; R02, R03; approval G1 2026-10-11 16:06 CDT · Interview line: "I defined 'equivalent' as paying the same in every outcome, not the same event, because look-alike markets have settled on opposite sides."
- Review: Topic Product · Weight ●●● · One-liner: Same payout in every outcome over same-event matching; fewer routable pairs.

### D8 Kalshi and polymarket.com, eligibility as data  [status: accepted]
- Date 2026-10-11 16:07 CDT · phase 3 write-prd · milestone M2 · Decided by: human, G1 2026-10-11 16:06 CDT: "approve, also push and /design-hld" (interview Q2: "ok") · Proposed by: write-prd (Claude Code, claude-opus-5-5) · artifact: docs/product/prd.md@40eb5d5
- Context and forces: Two venues with public metadata and prices are required (REQ-01). R01 confirmed unauthenticated order-book depth live for Kalshi and polymarket.com; Polymarket US is documented but unconfirmed live. R06: polymarket.com takes no new US orders and is not a CFTC designated contract market, while Kalshi and Polymarket US are; regulatory implementation is out of scope (REQ-29) (FR-035, FR-036, FR-037, C-1).
- Options:
  - Do nothing / simplest — ⚠ not offered at the gate
  - Kalshi + polymarket.com, customer-region eligibility as declared data, ineligible venues left out with the reason; Polymarket US named as the next venue (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: an eligibility list to maintain and disclose as not legal advice — chosen
  - Kalshi + Polymarket US — cost: the second venue's live data and overlap are unproven today — rejected: the human chose the recommendation
  - Kalshi + polymarket.com, ignoring eligibility — cost: would route a US customer to a venue they cannot use — rejected: the human chose the recommendation
  - Eligibility as an outright ban on polymarket.com — cost: only one venue left for US customers, nothing to route between — rejected: the human chose the recommendation
- Choice: Kalshi + polymarket.com with eligibility as data · Why: In the context of a partner whose broker platform must respect where customers may trade, facing one venue that bars US orders and another whose regulated twin is unproven live, we chose two venues with confirmed public depth plus eligibility as data over a regulated-only pair to keep the build feasible today, accepting an eligibility list that is declared, dated and not legal advice.
- Cost accepted: a declared, dated eligibility list; the regulated-only pairing is deferred to a third venue. · Reversibility: Type 2 two-way — venues sit behind the venue-integration boundary and routing reads eligibility as data (REQ-11) · Confidence: ⚠ not stated in the PRD
- Revisit if: Polymarket US's public book is confirmed live and its overlap with Kalshi is measured · Evidence: docs/product/prd.md:168; R01, R02, R06; approval G1 2026-10-11 16:06 CDT · Interview line: "I picked the two venues with public depth I could verify, then made 'who may use this venue' data the router reads, so a US order never lands offshore."
- Review: Topic Product · Weight ●●○ · One-liner: Kalshi plus polymarket.com with eligibility data; maintain a dated list.

### D9 Blind sampled precision with recall and kappa  [status: accepted]
- Date 2026-10-11 16:07 CDT · phase 3 write-prd · milestone M2 · Decided by: human, G1 2026-10-11 16:06 CDT: "approve, also push and /design-hld" (interview Q3: "ok") · Proposed by: write-prd (Claude Code, claude-opus-5-5) · artifact: docs/product/prd.md@40eb5d5
- Context and forces: The North Star (matching precision) must survive the interview question of how it could be gamed (REQ-30). R05: precision is gamed by matching less or by raters seeing the system's output; kappa, not percent agreement, with about 30 comparisons as a floor (SC-001, SC-002, SC-003, FR-028 to FR-034).
- Options:
  - Do nothing / simplest (shown as "Report the number with no target"): honest, but nothing marks the number good or bad — rejected: the human chose the recommendation
  - Fixed-seed sample of up to 100 equivalent pairs, AI rater blind, the human labels 30 blind, kappa reported, labels frozen before tuning; hard gate 0.90 (v1 one-shot 0.879 as a prior experiment), aspirational 0.95 (preprint under 2% false positives); recall as the guard (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: about 30 careful labels from the human today — chosen
  - The human labels all 100 — cost: about 1-2 hours of reading today — rejected: the human chose the recommendation
  - AI labels only — cost: a model grading a model — rejected: the human chose the recommendation
- Choice: blind sampled precision, gate 0.90, aspire 0.95, recall and kappa guards · Why: In the context of an interview that probes how a number could be gamed, facing a cheap AI rater that may share the matcher's blind spots, we chose a blind sample with a human cross-check over AI-only or all-human labels to get a defensible number today, accepting 30 human labels and a single-day sample.
- Cost accepted: about 30 blind labels by the human; a sample, not a census; a hard gate that, if missed, turns the verdict into "routing needs a reviewed pair list". · Reversibility: Type 2 two-way — the sample size and targets can change before measurement runs · Confidence: ⚠ not stated in the PRD
- Revisit if: kappa between the human and the AI rater falls below 0.60 (SC-003) · Evidence: docs/product/prd.md:191; R05 F5, D3; approval G1 2026-10-11 16:06 CDT · Interview line: "Precision is measured on a blind random sample I cross-checked myself, with recall alongside it, so you can't score well by matching less."
- Review: Topic Product · Weight ●●○ · One-liner: Blind sampled precision with recall and kappa guards; costs 30 human labels.

### D10 AI only offline, as an adjudicator  [status: accepted]
- Date 2026-10-11 16:07 CDT · phase 3 write-prd · milestone M2 · Decided by: human, G1 2026-10-11 16:06 CDT: "approve, also push and /design-hld" (interview Q4: "ok") · Proposed by: write-prd (Claude Code, claude-opus-5-5) · artifact: docs/product/prd.md@40eb5d5
- Context and forces: AI-assisted matching is allowed (REQ-04) and the hiring team values applied AI (decode QB-3), but routing must be deterministic and never block on outside calls (REQ-07, REQ-08). R03: the decisive differences sit in free-text rules (FR-038 to FR-041, NOGO-1).
- Options:
  - Do nothing / simplest (shown as "No AI in the product"): fully deterministic — cost: rules-text differences are the hardest part for plain rules, and the AI signal is lost — rejected: the human chose the recommendation
  - Offline only: deterministic rules first, AI adjudicates the rest, verdicts stored with model, prompt version and reason, reviewable and frozen before routing; routing never calls a model; works with AI off (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: a review step and stored verdicts to maintain — chosen
  - AI at decision time — cost: answers change between runs and add delay, breaking REQ-07/08 — rejected: the human chose the recommendation
  - AI checks every candidate pair — cost: expensive over about 375,000 open markets; a model graded by a model — rejected: the human chose the recommendation
- Choice: AI offline only, as an adjudicator · Why: In the context of deterministic, non-blocking routing, facing settlement differences written in free text, we chose offline AI adjudication with frozen, reviewable verdicts over no AI or AI at decision time to catch rules-text differences without making routing depend on a model, accepting a review step and stored verdicts.
- Cost accepted: stored AI verdicts to review; matching coverage depends on the adjudicator being available (the system reports undecided pairs when it is off). · Reversibility: Type 2 two-way — the adjudicator is optional (FR-038 is Optional) · Confidence: ⚠ not stated in the PRD
- Revisit if: SC-003 shows the AI rater and the human disagree, since the adjudicator and the rater may share blind spots (RISK-3) · Evidence: docs/product/prd.md:171; R03; approval G1 2026-10-11 16:06 CDT · Interview line: "AI reads the fine print offline and its verdicts are frozen and reviewable; the router itself never calls a model, so decisions replay exactly."
- Review: Topic Product · Weight ●●○ · One-liner: Offline AI adjudication over runtime AI or none; adds a review step.

### D11 Consider active markets only  [status: accepted]
- Date 2026-10-11 16:07 CDT · phase 3 write-prd · milestone M2 · Decided by: human, G1 2026-10-11 16:06 CDT: "approve, also push and /design-hld" (interview Q5: "ok") · Proposed by: write-prd (Claude Code, claude-opus-5-5) · artifact: docs/product/prd.md@40eb5d5
- Context and forces: R08 measured about 126,000 open Kalshi and 249,000 Polymarket markets, of which 17.8% and 7.9% traded in 24 hours; a full Polymarket sweep took about 2,500 requests and 12 minutes, and a repeat sweep was refused (HTTP 403). R02: thousands of 5- and 15-minute windows (FR-001, FR-003, NG-5, NFR-004).
- Options:
  - Do nothing / simplest (shown as "The whole open catalog"): most complete — cost: about 85% of markets do not trade; slow sweeps risk blocks — rejected: the human chose the recommendation
  - Open markets with at least one trade in the last 24 hours and a life of at least 1 hour, left-out counts reported (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: quiet long-tail markets are not matched — chosen
  - Top N by volume per venue — cost: an arbitrary cut biased toward sports — rejected: the human chose the recommendation
  - A few topics only — cost: hides how matching does on the long tail — rejected: the human chose the recommendation
- Choice: active markets only, with left-out counts reported · Why: In the context of a feasibility verdict about routable listings, facing a catalog where most markets never trade and sweeps risk blocks, we chose active markets over the whole catalog to keep refreshes within limits and matches tradable, accepting that quiet markets go unmatched.
- Cost accepted: markets with no trade in 24 hours and sub-hour windows are never matched; the report always says how many were left out. · Reversibility: Type 2 two-way — the activity window is a stated rule that can widen · Confidence: ⚠ not stated in the PRD
- Revisit if: the verdict needs evidence on quiet, long-tail markets · Evidence: docs/product/prd.md:134; R02, R08; approval G1 2026-10-11 16:06 CDT · Interview line: "I matched only markets that traded in the last day, because a market nobody trades has no price worth routing to, and I report what I left out."
- Review: Topic Data · Weight ●○○ · One-liner: Active markets only over the full catalog; quiet markets go unmatched.

### D12 Approve the Equinox PRD at G1  [status: accepted]
- Date 2026-10-11 16:07 CDT · phase 3 write-prd · milestone M2 · Decided by: human, G1 2026-10-11 16:06 CDT: "approve, also push and /design-hld" · Proposed by: write-prd (Claude Code, claude-opus-5-5) · artifact: docs/product/prd.md@40eb5d5
- Context and forces: The PRD fixes scope, non-goals and success measures before design (REQ-04, REQ-16, REQ-29): 7 stories (4 P1), 44 FRs, 7 NFRs, 4 SCs, 38 AEs, milestones M4 tracer bullet, M5 trust, M6 reach. Two open questions remained: reading public data before the venues' terms are read (→ R01), and the US-regulated venue's fee formula (→ R04); seven agent assumptions (A-1 to A-3, A-5 to A-8) awaited confirmation.
- Options:
  - Do nothing / simplest (shown as "Stop"): nothing proceeds — cost: no design work starts — rejected: the human approved
  - Approve, taking both open-question recommendations (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: later changes need a PRD edit, a fresh G1 and design rework — chosen
  - Revise — cost: time now — rejected: the human approved
- Choice: Approve; both open questions take their recommendations (read public data for a non-commercial prototype; the human downloads the fee schedule by hand for design-hld); A-1 to A-3 and A-5 to A-8 confirmed · Why: In the context of a same-day appetite, facing a PRD whose choices all trace to the human's answers or cited research, we chose to approve over revising to start design now, accepting that the two open items are settled by recommendation rather than new evidence.
- Cost accepted: the venues' terms of use stay unread for now; FR-009's fee formula for the US-regulated venue waits on the human's download. · Reversibility: Type 2 two-way — a clarify-spec pass and a fresh G1 can change any requirement · Confidence: ⚠ not stated in the PRD
- Revisit if: the venues' terms forbid the prototype's reads, or the fee schedule changes the routing rules · Evidence: docs/product/prd.md:280; approval G1 2026-10-11 16:06 CDT · Interview line: "I froze the requirements only after five decisions of my own, each traced to research, so design starts from what I chose, not what the AI guessed."
- Review: Topic Process · Weight ●●○ · One-liner: PRD approved with both open items as recommended; terms of use unread.

### D13 Build Equinox in Go, standard library only  [status: accepted]
- Date 2026-10-11 16:18 CDT · phase 5 design-hld · milestone M3 · Decided by: human, G2 D13 2026-10-11 16:17 CDT: "yes, use go" (quoting option b "Go, standard library only (brief P-01).") · Proposed by: design-hld (Claude Code, claude-opus-5-5) · artifact: docs/architecture/hld.md@5d35114
- Context and forces: The partner prefers Go and accepts Python (REQ-13); the router must replay exactly (FR-013, NFR-001), money must never be held in floating point (INV-3), and a background refresher must run beside routing (FR-020); the stack was left open until this gate (AGENTS.md:57). Proposed as docs/architecture/hld.md D13.
- Options:
  - Do nothing / simplest: Python 3 scripts with the standard library — cost: about the same build and run; floats by default, extra structure for a long-running service, not the preferred language — rejected: the human chose Go
  - Go 1.27, standard library only (brief P-01) (agent's recommendation) (agent's first proposal) — cost: about 1 extra build-day of text tooling; same run cost; live runs on this Mac go through a container — chosen
  - TypeScript on Bun or Node — cost: about the same; floating-point numbers by default put INV-3 at risk; not the stated preference — rejected: the human chose Go
- Choice: Go, standard library only · Why: In the context of a partner whose stack prefers Go and a router that must be exactly reproducible, facing a same-day build, we chose Go with only its standard library over Python or TypeScript to get one static binary, safe concurrency between refresh and routing, and exact integer money, accepting more hand-written text matching and container-only live runs on this Mac.
- Cost accepted: text matching written by hand with no third-party libraries; live venue calls on this Mac only from a container (AGENTS.md:58). · Reversibility: Type 1 one-way — a language is on the Type 1 list; changing it later means a rewrite · Confidence: high
- Revisit if: matching quality needs a text library Go lacks (SC-001 below its gate with rules-text misses as the cause) · Evidence: docs/architecture/hld.md:249; `go version` → go1.27.1 checked 2026-10-11; approval G2 D13 2026-10-11 16:17 CDT; ADR: docs/decisions/0013-build-equinox-in-go-standard-library-only.md · Interview line: "I picked Go because PEAK6 runs Go, and its integers and goroutines made deterministic routing beside a live refresher easy to prove."
- Review: Topic Infrastructure · Weight ●●● · One-liner: Go standard library over Python or TypeScript; more hand-written matching.

### D14 One service: background refresher, immutable snapshots  [status: accepted]
- Date 2026-10-11 16:19 CDT · phase 5 design-hld · milestone M3 · Decided by: human, G2 D14 2026-10-11 16:19 CDT: "approved for the recc," · Proposed by: design-hld (Claude Code, claude-opus-5-5) · artifact: docs/architecture/hld.md@5d35114
- Context and forces: Routing must never contact a venue (FR-020, NFR-002) and must replay exactly (FR-013); a failing venue keeps its last good data (FR-021); reviewers need a live page (FR-042, C-2); about 10 people use it and a full refresh takes about 4 min (hld §8). Proposed as docs/architecture/hld.md D14.
- Options:
  - Do nothing / simplest: the command line fetches venue data for each run, then routes — cost: least build; each order waits on the venues (breaks FR-020) and there is no live page (breaks FR-042) — rejected: fails two requirements
  - One service process: a background refresher polls both venues, matches after each catalog refresh and publishes an immutable snapshot; routing reads only the current snapshot; the same binary is the command line (brief P-04) (agent's recommendation) (agent's first proposal) — cost: about 1 build-day, about $7 a month; about 4 min of warming up after a restart — chosen
  - Two processes: an ingest worker writes snapshots to a shared store; a routing service reads it — cost: about 2 extra build-days, about $15-20 a month; a third container and a sync protocol — rejected: the human chose the recommendation
- Choice: one service with a background refresher and immutable snapshots · Why: In the context of routing that must never wait on a venue and must replay exactly, facing a 10-person audience and a one-day appetite, we chose one service that refreshes in the background and publishes immutable snapshots over per-order fetching or a split ingest service to get non-blocking, reproducible routing with one thing to run, accepting a short warm-up after each restart.
- Cost accepted: one instance, so a restart means about 4 min of "warming up" with no decisions; refresher and router share one machine's memory. · Reversibility: Type 2 two-way — splitting ingest from routing later is the documented 10K-user step (hld §12) · Confidence: med
- Revisit if: refresh duration nears 12 min, or reviewers need the page during restarts (hld §8 signals) · Evidence: docs/architecture/hld.md:268; approval G2 D14 2026-10-11 16:19 CDT · Interview line: "Routing reads an immutable snapshot that a background refresher swaps in, so it never blocks on a venue and the same snapshot always gives the same answer."
- Review: Topic Infrastructure · Weight ●●○ · One-liner: One service with snapshots over a split ingest; a restart warm-up.

### D15 Decision log on a volume, reviewed data in git  [status: accepted]
- Date 2026-10-11 16:20 CDT · phase 5 design-hld · milestone M3 · Decided by: human, G2 D15 2026-10-11 16:20 CDT: "approved for the recc" · Proposed by: design-hld (Claude Code, claude-opus-5-5) · artifact: docs/architecture/hld.md@5d35114
- Context and forces: Every decided or refused order must be readable later, across restarts (FR-012, NFR-006); the router's trusted inputs must be frozen and reviewable (FR-033, FR-035, FR-039, FR-041) and decisions must replay (FR-013); about 90 decisions a day ≈ 66 MB a year; cost ceiling about $10 a month (hld §8, Q-5). Proposed as docs/architecture/hld.md D15.
- Options:
  - Do nothing / simplest: keep everything in memory — cost: none; the decision log is lost on every restart (breaks NFR-006) and reviewed inputs have no history — rejected: fails NFR-006
  - An append-only decision log file on a Railway volume, plus reviewed reference data as versioned files in the repository, shipped in the image (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: about 0.5 build-day; volume about $0.15 per GB-month — chosen
  - A managed Postgres database on Railway — cost: about 1 extra build-day; about $5-8 a month; reviewed inputs lose their merge-request review trail — rejected: the human chose the recommendation
- Choice: append-only log file on a volume, reviewed data in git · Why: In the context of a decision log of about 66 MB a year and inputs that must be frozen and reviewed, facing a cost ceiling around $10 a month, we chose an append-only file on a volume plus versioned files in git over a database or memory only to keep every input reviewable and tied to a commit with nothing extra to run, accepting file scans for reads and a redeploy to change reference data.
- Cost accepted: decisions are read by scanning a file; changing eligibility, verdicts or labels needs a reviewed merge request and a redeploy. · Reversibility: Type 2 two-way — the append-only log can be loaded into a database later (hld §12, 100K tier) · Confidence: med
- Revisit if: someone needs to query decisions by field, or the log exceeds 1 GB · Evidence: docs/architecture/hld.md:287; railway.com/pricing checked 2026-10-11; approval G2 D15 2026-10-11 16:20 CDT · Interview line: "Every input the router trusts is a reviewed file in git, and every decision is appended to a log before the answer goes out, so any decision can be replayed."
- Review: Topic Data · Weight ●●○ · One-liner: Log file plus git-reviewed data over Postgres; reads scan a file.

### D16 Offline AI through exported files and a harness  [status: accepted]
- Date 2026-10-11 16:22 CDT · phase 5 design-hld · milestone M3 · Decided by: human, G2 D16 2026-10-11 16:22 CDT: "yes lets do it" (quoting option b "exported files plus an agent harness") · Proposed by: design-hld (Claude Code, claude-opus-5-5) · artifact: docs/architecture/hld.md@5d35114
- Context and forces: D10 allows AI only offline with stored, reviewable, frozen verdicts and no model call while routing (FR-038 to FR-041); the AI rater labels the sample blind (FR-031); the product holds no secrets (NFR-007); there is no budget or key for a model API. Between the gate card and this answer the human asked about mocking the venue APIs to save time, and the supervisor re-recommended option a for time; the human then chose the original recommendation. Proposed as docs/architecture/hld.md D16.
- Options:
  - Do nothing / simplest: no AI adjudicator; pairs the rules cannot settle stay not routable (FR-040) — cost: none; fewer routable pairs; US-6 cut — rejected: "yes lets do it" for option b (it was the supervisor's revised recommendation for time)
  - Exported files plus an agent harness: the command line exports pair cards, an agent harness (Claude Code on the human's plan) writes a verdict file with model, prompt version, time and reason, the command line imports it after checks, and the file is reviewed in a merge request (agent's first proposal — no instead-of recorded; the original recommendation) — cost: about 0.5 build-day; $0 new spend; a person starts each round — chosen
  - The command line calls a model provider's API with a key during an offline job — cost: about 0.5 build-day plus per-pair model cost; a secret and a new monthly spend — rejected: the human chose option b
- Choice: exported files plus an agent harness · Why: In the context of AI that may only work offline and verdicts that must be frozen and reviewable, facing no budget or key for a model API, we chose an export → agent harness → reviewed import path over an in-product API call or no AI to get reviewable AI verdicts with no secret and no new spend, accepting that a person starts each round.
- Cost accepted: a person starts each adjudication and rating round; verdicts reach routing only after a reviewed merge request and a redeploy. · Reversibility: Type 2 two-way — an API call can replace the harness later without changing the verdict file · Confidence: med
- Revisit if: adjudication rounds become frequent enough that a person starting each one is the bottleneck · Evidence: docs/architecture/hld.md:306; brief P-06; approval G2 D16 2026-10-11 16:22 CDT · Interview line: "The AI never runs inside the product: it gets an export, returns a verdict file with its reasons, and a person reviews that file before the router can use it."
- Review: Topic Infrastructure · Weight ●●○ · One-liner: AI via exported files and a harness over an API key; manual rounds.

### D17 Recorded real venue responses as mocks, live switch  [status: accepted]
- Date 2026-10-11 16:22 CDT · phase 5 design-hld · milestone M3 · Decided by: human, decision: venue mocks 2026-10-11 16:22 CDT: "yes i agree" (quoting option a "Recorded real responses as the mock, plus a live switch (recommended).") · Proposed by: supervisor (Claude Code, claude-opus-5-5), chat options card · artifact: docs/architecture/hld.md@5d35114
- Context and forces: The human asked for mock venue APIs modeled on the two venues to save time; the assignment requires live market data from two venues (REQ-01, C-1); the HLD already records real venue responses for tests and CI (CN-3, §11 CI row) and reads the public, keyless APIs live (L-2, L-3).
- Options:
  - Do nothing / simplest (shown as "c) Mocks only, no live data") — cost: fails REQ-01 and needs a PRD change plus a fresh G1 — rejected: the human chose option a
  - a) Recorded real responses as the mock, plus a live switch (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: no extra build time; the reader is pointed at recordings or the real address — chosen
  - b) Hand-written fake APIs modeled on the two venues, live added later — cost: more build time to invent shapes and data; REQ-01 fails if live never lands — rejected: the human chose option a
- Choice: recorded real responses as the mock, plus a live switch · Why: In the context of a same-day appetite and an assignment that requires live data, facing a request to mock the venues, we chose recorded real responses with a live switch over invented fake APIs or mocks only to get fast, offline development and tests with exact venue shapes, accepting that recordings age and must be re-recorded.
- Cost accepted: recordings are a dated snapshot of each venue's responses and must be re-recorded when a venue changes its format; live runs on this Mac still go through a container. · Reversibility: Type 2 two-way — the switch is a setting · Confidence: high
- Revisit if: a venue changes its response format, or tests start failing only against live data · Evidence: docs/architecture/hld.md:126, :346; brief P-07; approval decision: venue mocks 2026-10-11 16:22 CDT · Interview line: "Tests and development run on recorded real venue responses, so they are fast and exact, and the same reader goes live with one switch, because the assignment demands live data."
- Review: Topic Data · Weight ●○○ · One-liner: Recorded real responses with a live switch over invented mocks; recordings age.

### D18 Approve the Equinox HLD at G2  [status: accepted]
- Date 2026-10-11 16:30 CDT · phase 5 design-hld · milestone M3 · Decided by: human, G2 (closing) 2026-10-11 16:30 CDT: "approve, also push and /design-lld" · Proposed by: design-hld (Claude Code, claude-opus-5-5) · artifact: docs/architecture/hld.md@e74350c
- Context and forces: The HLD sets 4 containers, 11 connections, 4 trust boundaries in three non-overlapping zones (revised at a G2 Revise), 9 invariants and 7 open questions, with D13-D16 answered on 5d35114 and the diagrams embedded afterwards; the closing answer binds every earlier G2 answer to the final file (FR-001 to FR-043, NFR-001 to NFR-007). The human had sent the same words once before the diagrams were embedded; that early answer was not recorded because the artifact did not exist yet.
- Options:
  - Do nothing / simplest (shown as "Stop"): nothing proceeds — cost: no low-level design starts — rejected: the human approved
  - Approve the embedded HLD with D13-D16 as answered (agent's recommendation) (agent's first proposal — no instead-of recorded) — cost: later architecture changes need an HLD revision and a fresh G2 — chosen
  - Revise — cost: time now — rejected: the human approved
- Choice: Approve · Why: In the context of a same-day appetite, facing an HLD whose decisions were each answered and whose trust zones were already revised once, we chose to approve the embedded file over another revision to start low-level design now, accepting that the seven open questions are settled by measurement in M4 or by their recommended answers.
- Cost accepted: the open questions (usage, memory, latency target, availability, cost ceiling, model API cost, the US-regulated venue's fee formula) stay open until M4 measurement or the human's answer. · Reversibility: Type 2 two-way — a design-hld revision and a fresh G2 can change any part · Confidence: ⚠ not stated in the HLD
- Revisit if: the M4 tracer bullet measures memory, refresh time or paging stability outside the §8 budget · Evidence: docs/architecture/hld.md@e74350c; approval G2 (closing) 2026-10-11 16:30 CDT · Interview line: "I approved the architecture only after every decision was answered and the diagrams found and fixed a gap in my own trust boundaries."
- Review: Topic Process · Weight ●●○ · One-liner: HLD approved after a trust-zone fix; open questions measured in M4.

### D19 Match at the grain of one binary proposition  [status: accepted]
- Date 2026-10-11 17:14 CDT · phase 6 design-lld · milestone M3 · Decided by: human, G3 D19 2026-10-11 17:14 CDT: "approve D19" · Proposed by: design-lld (Claude Code, claude-opus-5-5) · artifact: docs/architecture/lld.md@d7cc8b2
- Context and forces: An order buys one YES/NO contract and D7 defines equivalence by payout (FR-004, FR-006); R02 found the venues group events differently (one splits a game into several sub-events, the other splits a race into winner, margin and turnout events). Proposed as docs/architecture/lld.md D19.
- Options:
  - Do nothing / simplest: match whole events by title — cost: least build; an event holds many different bets, so routing cannot say which contract to buy (breaks D7) — rejected: the human chose the recommendation
  - One binary proposition per row: each market, and each outcome of a named-outcome market, becomes one row (brief P-03) (agent's recommendation) (agent's first proposal) — cost: about 0.5 build-day; more rows to compare — chosen
  - Event first, then propositions inside matched events — cost: about 1 build-day; one wrong event match hides every correct proposition inside it — rejected: the human chose the recommendation
- Choice: one binary proposition per row · Why: In the context of a router that buys one YES/NO contract and a definition of equivalence by payout, facing venues that group events differently, we chose matching at the grain of one binary proposition over event-level matching to compare exactly what gets bought, accepting more rows to compare.
- Cost accepted: more rows to compare (each outcome of a named-outcome market is its own row). · Reversibility: Type 2 two-way — event grouping can be added on top later as a candidate filter · Confidence: high
- Revisit if: matching time per refresh nears the 4-minute budget (HLD §8) · Evidence: docs/architecture/lld.md:912; brief P-03; R02; approval G3 D19 2026-10-11 17:14 CDT · Interview line: "I matched individual yes/no propositions, because that is what an order buys, and the venues group events differently."
- Review: Topic Data · Weight ●●○ · One-liner: Proposition-level matching over event-level; more rows to compare.
