---
milestone: M1
decoded: Sun Oct 11 13:47:37 CDT 2026
sources: [company info panel (pasted 2026-10-11), project brief Version 1 (partner portal, last updated 8/7/2026, pasted 2026-10-11), Project Equinox PDF (2 pages), Gauntlet remote-phase syllabus §Technical Interview Expectations, v1 one-shot build (main@9c3bf48), software-factory-kit FINDINGS.md]
gate: G-decode
---

# Decode: Equinox, cross-venue prediction-market equivalence and routing

The partner documents are not cleared for a public repo, so every row below paraphrases them and cites a page or
section, with no word-for-word quotes.

## Surface ask

Build a small working prototype that pulls live market data from two prediction-market venues and converts it into
one internal market model that does not depend on either venue's data format. It must find markets on the two
venues that refer to the same real-world event and simulate where a hypothetical order should go. Every match and
routing decision must be logged with its reasoning. Routing has to be deterministic, must never wait on a network
call, and must survive bad or missing data. The deliverables are code, docs, a demo video, test results, an AI usage
log and a deployment guide, plus written explanations of the equivalence and routing logic. Go and GCP are
preferred, but Python and a local deployment are acceptable.

| Req | Requirement (one line, paraphrased) | Source (page/section) |
|---|---|---|
| REQ-01 | Pull live market metadata and prices from at least two venues' public APIs | brief §Functional requirements; PDF p1 §Objectives, §Scope |
| REQ-02 | One internal market representation, independent of any venue's data format | brief §Functional requirements; PDF p1 §Scope |
| REQ-03 | Identify markets on different venues that may be the same real-world event | brief §Functional requirements; PDF p1 §Scope |
| REQ-04 | Define what "equivalent" means, justify the method (rules, heuristics, AI, or a mix) and document it | PDF p2 §Core expectations |
| REQ-05 | Simulate routing a hypothetical order: weigh the available venues, decide, and explain why that venue won | brief §Functional requirements; PDF p2 §Core expectations |
| REQ-06 | Log and explain both matching decisions and routing decisions | brief §Functional requirements; PDF p1 §Objectives |
| REQ-07 | Routing logic is deterministic | brief §Performance benchmarks |
| REQ-08 | Routing never blocks on external calls | brief §Performance benchmarks |
| REQ-09 | Degrade gracefully on API failure or inconsistent data, and document assumptions wherever data is incomplete or ambiguous | brief §Performance benchmarks; PDF p2 §Core expectations |
| REQ-10 | Separate venue integration, normalization, equivalence detection and routing | brief §Code quality; PDF p2 §Core expectations |
| REQ-11 | The routing layer holds no venue-specific logic or assumptions | brief §Code quality; PDF p2 §Core expectations |
| REQ-12 | Assumptions documented; reasonable test coverage; clean, readable code | brief §Code quality |
| REQ-13 | Go preferred, Python acceptable; other tools only if justified | brief §Technology; PDF p2 §Technical flexibility |
| REQ-14 | GCP preferred, local deployment acceptable; clarity matters more than infrastructure sophistication | brief §Technology; PDF p2 §Technical flexibility |
| REQ-15 | Using AI is optional; if it is used, disclose it and the reasoning behind it | PDF p2 §Technical flexibility, §Deliverables; brief §Submission & AI policy |
| REQ-16 | Portal deliverables: source code, technical documentation, demo video, test results, AI usage log, deployment guide | brief §Required deliverables |
| REQ-17 | PDF deliverables: a working prototype, setup instructions, a short architecture overview, written explanations of the equivalence logic and the routing logic, and AI disclosure | PDF p2 §Deliverables |
| REQ-18 | Out of scope: real-money trading, wallets, regulatory implementation, production UI; UI polish does not count | PDF p2 §Scope; brief §Functional requirements |
| REQ-19 | An interview follows the submission: defend the architecture, the verification and evaluation choices, production thinking, and each metric (including how it could be gamed) | syllabus §Technical Interview Expectations |

## Question behind the question

| QB | Hypothesis: what this project really tests | Evidence | Confidence |
|---|---|---|---|
| QB-1 | **It is a feasibility study, and the build is the evidence.** The partner wants answers it can act on before investing in three questions: can normalization across venues actually be done, what are the architectural tradeoffs, and how complex does routing become in practice. The working prototype is the proof behind those answers; it is not the product. | The brief's business context lists those three questions as what it wants answered before spending more (brief §Business context). The PDF frames the work as an exploratory prototype, explicitly not a trading product, and says it weighs architectural thinking, tradeoff awareness and clear reasoning over polish (PDF p1 §Overview, §Objectives). The evaluation criteria grade framing, decomposition, handling of ambiguity and justification (PDF p2 §Evaluation criteria). Its closing line restates the goal as judging whether the infrastructure is viable (PDF p2). | high |
| QB-2 | **The real test is defining "equivalent" under ambiguity and measuring how often detection is wrong.** A false match sends an order into a different bet, so graders will probe whether the builder knows that look-alike markets can settle differently. They will also check that the error rate was measured, not asserted. | The PDF says venues differ in naming, expiry logic, contract design and price format, and calls inferring equivalence the core challenge (PDF p1 §Problem statement). It requires a definition of "equivalent" and a justified method (PDF p2 §Core expectations). The brief only asks for *potentially* equivalent markets (brief §Functional requirements). The syllabus says interviews ask how each metric could be gamed. v1's own record shows a labelled-set precision of 1.0 against a first live audit of about 0.63 (v1 test results §4, main@9c3bf48). | high |
| QB-3 | **It doubles as the technical screen for an AI-native engineering hire.** The partner runs no coding assessment, so this project and its interview are the coding signal. The hiring team values autonomy, comfort with ambiguity, speed and applied AI (agents), so it will look at how deliberately AI was used, both in building and possibly inside matching, and where AI was kept out (deterministic routing). | No coding assessment, 3-4 interview rounds (panel §Interview process). The panel describes an incubator team hiring AI engineers (panel §About), rates autonomy and ambiguous problem-solving 5/5 (panel §Engineering environment) and lists agents under applied AI (panel §Technical stack). The PDF allows AI-assisted matching but requires its reasoning to be documented (PDF p2). The portal labels the brief AI-accelerated (brief header). **Tension:** the panel describes the AI Strategies team, while the brief comes from PEAK6 Labs (non-traditional assets), so there may be two audiences. | med |

**Recommended framing for the brief:** Give PEAK6 Labs an evidence-backed verdict on whether cross-venue
equivalence can be detected reliably enough for a venue-agnostic router to sit on top of it. The evidence is a
small working system in which the definition of "equivalent" is explicit and the matching error is measured.
Every judgment call, including where AI is and isn't used, must be written down and defensible in the interview.
(QB-1, sharpened by QB-2; QB-3 shapes how the process and AI usage are shown.)

## What graders / the partner will look at first

1. **The written reasoning.** The problem framing, the definition of "equivalent" and its justification, the
   architecture overview and the tradeoffs (PDF p2 §Deliverables, §Evaluation criteria).
2. **The demo video.** One match with its evidence, one routing decision with its "why", and graceful failure.
3. **The code's shape.** Separation of the four layers, a routing layer with no venue logic, and readability
   (brief §Code quality).
4. **Evidence.** Test results proving determinism, non-blocking routing and graceful failure, and a measured
   matching-error number with its method.
5. **The AI usage log.** What AI did, why, and where it was deliberately kept out.
6. **The interview.** Defend every decision and every number, including how each number could be gamed (syllabus).
   A live cloud URL is *not* required (PDF p2 §Technical flexibility).

## Traps

| T | Trap | Why it's tempting | Cost if we fall in | Evidence | Guard (phase / skill / check) |
|---|---|---|---|---|---|
| T-1 | **Literal reading: building a feature checklist ("an aggregator", "a smart router") instead of answering three feasibility questions** | The functional requirements read like a checklist, and "smart routing" sounds like optimizing execution | Graders see features but no verdict; the three business questions go unanswered | brief §Business context (the three questions); PDF p2 §Core expectations (reasoning matters more than execution quality) | `sharpen-brief` frames the problem as the three questions (G0); `write-prd` has one success criterion per question, each requiring a stated verdict with evidence (G1) |
| T-2 | **Literal reading: equating "similar titles" with "equivalent"** | Titles look alike, text similarity is quick to build, and it produces many matches | False matches route money into a different bet, the exact risk the partner cares about. Look-alike markets have settled in opposite directions (v1 cites a 2024 shutdown market). v1's labelled precision of 1.0 fell to about 0.63 on its first live audit | PDF p1 §Problem statement; v1 equivalence doc §1 and test results §4 (main@9c3bf48) | `write-prd` defines equivalence by settlement (same event, source, window, threshold) before any method is chosen (G1); `define-acceptance` writes look-alike hard negatives that must be rejected |
| T-3 | **Hidden grading criteria: the interview is the final grade** | It is easy to optimize the repo and treat docs as output. AI-written reasoning reads well | The builder cannot defend choices they did not make. In v1 the AI made the decisions in one unattended session, and the journal was private | syllabus §Technical Interview Expectations; panel §Interview process (no coding assessment); v1 kickoff (autonomous, journal gitignored) | The human decides at every gate (G0-G5); `log-decision` writes an interview line per decision; `quiz-me` runs before submission |
| T-4 | **Hidden grading criteria: two deliverable lists** | Each list looks complete on its own | Building to one list misses items on the other: setup instructions and the separate equivalence and routing write-ups (PDF), or the demo video, test results and deployment guide (portal) | brief §Required deliverables vs PDF p2 §Deliverables | `sharpen-brief` maps the union into its REQ table with exact deliverable paths (G0); `report-milestone` checks every REQ before G5 |
| T-5 | **Unstated non-functional bar: "deterministic", "non-blocking" and "graceful" are named but not defined** | It is tempting to claim them without proof. Market data changes every second, so "deterministic" is ambiguous | One interview question ("deterministic with respect to what?") collapses the claim. Silent float or ordering drift flips decisions | brief §Performance benchmarks; v1 fixed real ordering-dependent drift (v1 test results §5) | `write-prd` turns each into a measurable non-functional requirement (inputs pinned, failure kinds listed) (G1); `define-acceptance` writes a same-input-same-decision test and failure-injection tests (down venue, partial data, stale or crossed prices) |
| T-6 | **Wrong stack / partner mismatch** | Python's text and ML ecosystem looks attractive for matching, and the hiring panel lists Python first | It weakens the partner signal, or the choice has to be defended against a stated preference | brief §Technology and PDF p2 §Technical flexibility (Go and GCP preferred); panel §Technical stack (Python, TypeScript, Go; AWS and GCP) | No stack is chosen before `design-hld`; G2 records the partner preference as a force in the decision |
| T-7 | **Demo vs depth** | UI polish is worth nothing, so the demo becomes a silent terminal scroll. Or the opposite: building a dashboard | The graders' first look does not show the reasoning, or the timebox burns on UI. v1's video was an unnarrated recording | PDF p2 §Evaluation criteria; brief §Functional requirements; v1 demo script (main@9c3bf48) | `write-demo-script` (G-demo): narrated; shows a match with its evidence, a rejected look-alike, a routing decision and its "why", graceful failure, and the measured error |
| T-8 | **Deliverable gaps: deadline, video spec and submission place unknown; the v1 GitLab repo is still private** | The prompt states no deadline or video length | A late or missing deliverable, or graders who cannot open the repo | brief (no dates); v1 state (GitLab project created private by push) | ⚠ deadline at G-decode; `sharpen-brief` writes the deadline with its time zone and the human-only steps (U-##) (G0) |
| T-9 | **Scope creep / gold-plating** | Real order placement, live streaming feeds, more than two venues, arbitrage alerts, a UI, cloud infrastructure and AI on every request all look impressive | The timebox is spent on what isn't graded. Arbitrage framing also invites regulatory questions the brief excluded | PDF p2 §Scope (out-of-scope list), §Technical flexibility | `write-prd` non-goals (G1); `slice-tasks` tracer bullet first (G4) |
| T-10 | **Unverifiable success: no matching-quality metric in the prompt; "reasonable coverage" undefined** | Reporting a coverage % and "found N pairs" is easy. So is letting a model judge the model | N pairs says nothing about correctness, and coverage says nothing about matching. Model judges that saw the matcher's own score anchor on it (v1 audits 1-2), which is a gameable number | brief §Code quality; syllabus (how could each number be gamed); v1 AI usage log §4 (main@9c3bf48) | `sharpen-brief`/`write-prd` set a North Star with a named data source (`docs/METRICS.md`); `gather-evidence` builds a labelled set that the **human** spot-checks; the method states how the number could be gamed and what stops it |
| T-11 | **Domain landmines** | Each one looks like a detail | Wrong prices or wrong matches. (a) The same title can hide a different resolution source, time window, threshold inclusivity or cut-off time zone. (b) Price formats (cents vs probability) and yes/no complements; multi-outcome vs binary structures. (c) Fees differ by venue and can flip the decision. (d) Float rounding and map-order drift. (e) Stale, crossed or partial order books, pagination and rate limits. (f) Whether a US firm's clients may use each venue (regulated exchange vs offshore): a routing-eligibility question the partner, a regulated brokerage group, will notice even though regulatory implementation is out of scope. (g) Data terms of use | PDF p1 §Problem statement; PDF p2 §Scope (regulatory out of scope); panel §About (Apex Fintech Solutions); v1 fee-flip and timestamp findings (main@9c3bf48) | `gather-evidence` R-questions per venue (R03-R04, R06); `write-prd` assumptions register (G1); `design-hld` invariants for money, time and determinism (G2); `define-acceptance` edge cases (G4) |
| T-12 | **Copying v1 / prior work** | v1 already has code, docs and a headline number (out-of-sample precision 0.879). Copying it forward is fast | The factory becomes a paper exercise. The human defends decisions they never made. v1 numbers get quoted for a v2 system that never measured them. The other way round, ignoring v1 discards real evidence: API facts, the live overlap census, and the lesson that labelled precision overstated live precision | main@9c3bf48; the human's restart message (TRACKER Approvals 2026-10-11 13:45 CDT) | G-decode Q2 sets the v1 policy; `gather-evidence` cites v1 only as a labelled prior experiment and re-verifies its facts; any v1 design re-enters `design-hld`/`design-lld` as one option among others, never the default; every number is re-measured in v2 |
| T-13 | **Unstated non-functional bar: AI in the matching path vs deterministic routing** | AI-assisted matching is explicitly allowed and the hiring team is AI-focused; or, the other way round, banning AI entirely is safe | A model consulted on every request makes decisions non-reproducible and slow, and sends data to a third party. Keeping AI out entirely may miss the hiring signal (QB-3) | PDF p2 §Core expectations; brief §Performance benchmarks; panel §Technical stack (agents) | ⚠ G1 (moved from interview Q5); `write-prd` states where AI may sit (G1); `design-hld` makes "routing reads only frozen inputs" an invariant (G2) |
| T-14 | **Deliverable gaps: a same-day restart, and a docs-only merge that empties main** | The kit merges every finished milestone into main, and the restart branch deleted v1's files | The deadline is 2026-10-11 23:30 CDT and the human chose to finish v2 today (Q4). If the docs-only M1 merged into main first, main would hold no working prototype until v2's code lands; a compressed day also tempts skipping gates | the human's Q3 and Q4 answers; 59ff700 deletes the v1 files on this branch; WORKFLOW.md §8 | Milestones still close in order with their gates, but **main is updated only once v2's `make check` is green and covers every REQ**: until then milestone MRs stack on the restart branch and main@9c3bf48 stays what graders see; gates stay one decision per reply, with recommended defaults so each answer is quick |

Categories checked (each is a T row above or n/a with a reason): literal reading (T-1, T-2) · hidden grading
criteria (T-3, T-4) · unstated non-functional bar (T-5, T-13) · wrong stack / partner mismatch (T-6) · demo vs depth
(T-7) · deliverable gaps (T-8) · scope creep (T-9) · unverifiable success (T-10) · domain landmines (T-11) · copying
v1 / prior work (T-12, T-14).

## Open questions

- ⚠ AI inside the product (offline, results frozen before routing) or only in the build process? (→ G1, moved from interview Q5)
- ⚠ Is a live GCP deployment worth the human's money and time, or is local enough? Spend is human-only (→ G0)
- ⚠ Should the human email the technical contact with clarifying questions (e.g. which venues, their definition of equivalent)? Sending messages is human-only (→ G0)

## Research questions for gather-evidence

- R01: Which prediction-market venues offer public, unauthenticated access to both market metadata and prices (with depth)? What are their rate limits, terms of use and data formats?
- R02: How much do the leading venues' open markets actually overlap today (same real-world events), and in which topics?
- R03: How does each candidate venue define settlement (resolution source, time windows, cut-off times, ties, cancellations, carve-outs)? Where do equivalent-looking markets differ?
- R04: What are each venue's fee schedules and rounding rules, and how much do fees move the all-in cost of an order?
- R05: Prior art: how do existing aggregators, arbitrage tools and published research detect equivalent markets, and what accuracy do they report?
- R06: Who is the partner (PEAK6 Labs; the parent's brokerage-infrastructure business)? What would cross-venue routing be used for, and which venues can a US-regulated broker's clients legally reach?
- R07: Who already aggregates or routes across prediction markets (competitors and their users), how big is the traded volume, and when are markets busiest (load profile for design)?
- R08: What makes a matching-quality measurement defensible (labelled-set size, sampling, a second rater, blind judging), and how could it be gamed?

## Interview log

| Q | Question | Recommended | Human's answer (verbatim) |
|---|---|---|---|
| Q1 | Which framing leads: QB-1 (feasibility verdict for PEAK6 Labs, build as evidence) or QB-3 (AI-native showcase for the hiring team)? | QB-1 leads, with QB-2 as its core; QB-3 is shown through the process and AI usage log, not by adding AI features | "A" (2026-10-11 13:56 CDT; option A = feasibility answer leads) |
| Q2 | v1 policy: clean room, cited prior experiment, or evolve v1's code? | Cited prior experiment: no v1 code or design is carried over by default; v1 facts may be cited, labelled "v1, one-shot", and re-verified; v1 numbers are never quoted for v2 | "A" (2026-10-11 13:57 CDT; option A = cite v1 as a prior experiment) |
| Q3 | Deadline (with time zone) and appetite in working days? | ⚠ no default: the prompt has no date | "yeah deadlines tonight at 11:30pm" (2026-10-11 13:59 CDT; read as 2026-10-11 23:30 CDT, the machine's zone, about 9.5 h away; appetite = until then) |
| Q4 | With about 9.5 h left, what gets submitted tonight: v1 (main) with the factory restart shown as in-progress, a compressed v2 with v1 as the fallback, or a from-scratch v2? | Submit v1 (main@9c3bf48) tonight, unchanged as the working prototype; the restart continues on its branch as visible process work. Tonight's hours go to the planning phases on the branch plus the interview defense, and nothing merges into main before the submission | "i dont want you t really be concerned about the deadline, lets get it done today" (2026-10-11 14:01 CDT; read as: finish v2 through the kit today, not v1) |
| Q5 | AI inside the product (offline, frozen before routing) or build process only? | Allowed only offline: AI may propose or adjudicate matches ahead of time, the results are frozen and reviewed, and routing never calls a model; final call at G1/G2 | not asked: moved to `write-prd` (G1), which decides where AI may sit, to save a round trip today |

## Parking lot

v1 approaches (main@9c3bf48). These are candidates only, re-decided at G2/G3 if they come back at all:
- Go standard library only.
- Hexagonal ports and adapters.
- A canonical binary-proposition model with integer micro-dollar money and fees as a curve.
- A copy-on-write snapshot store; routing as a pure function.
- TF-IDF blocking plus veto rules plus one-to-one assignment for matching.
- A reviewed mapping table fed by offline LLM adjudication.
- Record and replay of venue responses.
- A Docker image targeting Cloud Run.
- Kalshi and Polymarket as the two venues.
