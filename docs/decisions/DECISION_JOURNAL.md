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
