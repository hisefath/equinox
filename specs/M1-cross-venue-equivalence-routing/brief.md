---
milestone: M1
slug: cross-venue-equivalence-routing
artifact: brief
written_by: sharpen-brief (software-factory-kit 0.1)
written_at: 2026-10-11 14:27 CDT (UTC-0500)
assignment: company info panel (partner portal, pasted 2026-10-11) · project brief Version 1 (partner portal, last updated 8/7/2026, pasted 2026-10-11) · Project Equinox PDF (2 pages, held by the human) · Gauntlet remote-phase syllabus §Technical Interview Expectations · specs/M1-cross-venue-equivalence-routing/decode.md (G-decode@d20fb08)
track_recommended: Full
track_decided_in: "G0 row in .handoff/TRACKER.md ## Approvals; this brief holds only the recommendation"
problem_allowlist: none
gate: G0
---

# M1 brief: Equinox, can we tell when two prediction markets are the same bet and route between them?

## Ask

> PEAK6 Labs needs an evidence-backed verdict, delivered today (2026-10-11 by 23:30 CDT), on whether equivalent
> prediction markets can be detected across two venues reliably enough for a venue-agnostic router to sit on top.
> The evidence is a small working system whose definition of "equivalent" is explicit and whose matching error is
> measured.

The human's original words: "So i want a fresh start, but to still keep the repo history. so we did the one shot prompt method, and i want that history to stay on the repo, but on a new branch from main, i want to restart the project with the new software factory kit process, for this peak 6 equinox project ." · "lets get it done today" (decode Q4)

## Problem

- (paraphrase, U-03) The same real-world event is listed on several venues, each with its own naming, structure and
  liquidity, and nothing tells you which listings are the same event or lets you compare their prices on equal
  terms. (PDF p1 §Overview)
- (paraphrase, U-03) Listings for one outcome can differ in name, expiry rules, contract design and price format,
  so comparing them across venues, or sending an order to the better one, is hard. (PDF p1 §Problem statement)
- (paraphrase, U-02) Fragmentation leads to inconsistent pricing and to the same integration work being repeated
  for each venue. Before investing further, the partner wants to learn whether normalizing across venues is
  realistic, what the architectural tradeoffs are, and how complex smart routing gets in practice. (brief
  §Business context)
- (paraphrase, U-02) Today each venue is handled by its own integration with nothing shared; the partner wants a
  reusable foundation that can take more venues. (brief §Current state, §Desired state)

**Restated, no solution or tech words:** Engineers at PEAK6 Labs who want to compare or trade the same event
across prediction-market venues cannot tell, without reading each listing's fine print, whether two listings are
really the same bet. Each venue also has to be wired in separately. They do this by hand today, one venue and one
listing at a time. That is slow, duplicates effort, and risks treating two different bets as one. Before spending
more, they need to know whether the matching can be trusted and how hard choosing a venue really gets.

- **Who has it:** PEAK6 Labs engineers evaluating non-traditional assets; the people who would later route
  customer orders across venues.
- **What they do today instead:** a separate integration per venue, with listings compared by eye.
- **Cost of doing nothing:** repeated integration effort, inconsistent prices, and, worst case, an order sent into
  a bet that settles differently from the one intended; or an investment decision made without evidence.

## Job stories

| ID | When… | I want… | so I can… | Who | Source |
|---|---|---|---|---|---|
| JS-01 | the same real-world event is listed on two venues under different names and rules | to know whether the two listings would settle the same way in every outcome | compare their prices as one bet, and never mix up two different bets | PEAK6 Labs engineer | REQ-03, REQ-04 |
| JS-02 | I have a hypothetical order for an outcome | to see which venue would get it, at what all-in cost, and the reason it won | trust and audit the choice | PEAK6 Labs engineer | REQ-05, REQ-06 |
| JS-03 | a venue is slow, unreachable or sends contradictory data | the decision to come out on time from the last good data, or be declined with a stated reason | rely on the decision step even when a venue misbehaves | PEAK6 Labs engineer | REQ-08, REQ-09 |
| JS-04 | I give the same inputs twice | exactly the same decision both times | reproduce and explain any past decision | PEAK6 Labs engineer | REQ-07 |
| JS-05 | we are deciding whether to invest further in cross-venue infrastructure | measured evidence of how often matching is wrong and where choosing a venue gets complicated | make the investment call on evidence rather than a demo | PEAK6 Labs lead | REQ-16, REQ-04 |
| JS-06 | a third venue needs adding later | to add it without touching the rules that choose a venue | reuse the foundation instead of repeating the integration work | PEAK6 Labs engineer | REQ-10, REQ-11, REQ-17 |

## Hiring partner

Who we are building for and what should impress them (`docs/METRICS.md` §5). Facts with a source; unknowns go to
Research questions. Not a partner project: one row `none: <why>`.

| Aspect | What we know | Source | What the project should show them |
|---|---|---|---|
| Who they are | (paraphrase, U-01) PEAK6: an investment firm of 1,001-2,000 people and a platinum hiring partner; core brands include Apex Fintech Solutions. Hiring for the AI Strategies incubator team (zero-to-one businesses on its fintech distribution, software transformations, AI services roll-ups); engineering team of 1-10 | panel §header, §About | A builder who can take an ambiguous infrastructure question to a defended answer alone |
| Who set the project | (paraphrase, U-02) PEAK6 Labs, exploring infrastructure for non-traditional assets including prediction markets; a named technical contact (name and email in the portal brief, not copied here) | brief §Business context, §Technical contact | A feasibility verdict their team could act on; ⚠ what PEAK6 Labs and Apex would do with it (→ R06) |
| What they value | (paraphrase, U-01) Entrepreneurial instinct and a strong work ethic; autonomy and ambiguous problem-solving both rated 5/5; mostly greenfield, rapid delivery and iteration; async, remote-first | panel §About, §Engineering environment | Gates and a decision journal that show judgment under ambiguity; a same-day delivery |
| How they grade | (paraphrase, U-03) How the problem is framed, how the system is decomposed, how ambiguity is handled, how decisions are justified; readability, modularity and documentation quality; UI polish does not count | PDF p2 §Evaluation criteria | Written reasoning first: the definition of "equivalent", the assumptions, the tradeoffs |
| Stack | (paraphrase, U-01) Python, TypeScript/JavaScript, Go; Postgres, SQL Server; React, Vue, Svelte; REST, OAuth, webhooks; Bun, Node.js, FastAPI; AWS, GCP, Kubernetes, CI/CD, Docker; agents. The project prefers Go and GCP | panel §Technical stack; brief §Technology | A stack choice argued against the stated preference at the design gate (decode T-6) |
| Interview | (paraphrase, U-01) 3-4 rounds, no coding assessment; every Gauntlet submission is followed by an interview defending architecture, verification, evaluation and metrics, including how each number could be gamed | panel §Interview process; syllabus | Every decision has an interview line (decision journal); every number states how it could be gamed |

## Value and metrics

Candidates for `write-prd`, not decisions (`docs/METRICS.md` §1-3). Targets are set in the PRD.

- **Underlying problem in the business's terms:** (paraphrase, U-02) inconsistent pricing across venues and
  integration work repeated per venue; the partner's own success words are a *reusable foundation* that supports
  many venues, and an answer to *whether normalization is realistically achievable* (brief §Business context).

| ID | Candidate metric | Role | Actionable or vanity ("if bad, what would we fix?") | Leading or lagging | Data that would measure it |
|---|---|---|---|---|---|
| NS-01 | Matching precision: the share of pairs the system calls "equivalent" that truly settle the same way in every outcome | North Star | actionable: if low, tighten the definition or the rules that reject look-alikes | lagging | a labelled set of live cross-venue pairs that the human spot-checks, judged blind to the system's score ⚠ (→ R05) |
| SM-01 | Matching recall: the share of truly equivalent labelled pairs the system finds | guards NS-01 (matching nothing scores perfect precision) | actionable: if low, widen candidate search or relax a rule | lagging | the same labelled set ⚠ (→ R05) |
| SM-02 | Decision reproducibility: replaying identical recorded inputs (also in shuffled order) gives identical decisions | guards the routing claims behind NS-01 (REQ-07) | actionable: if it ever differs, find the order- or rounding-dependent step | leading | repeated replays of a recorded snapshot |
| SM-03 | Fee-flip rate: the share of routed orders where the cheapest venue after fees differs from the cheapest before fees | explains why normalization matters to routing (JS-02, JS-05) | actionable: if near zero, price-only comparison would do and routing is simpler than feared | leading | the routing decision log over live matched pairs ⚠ (→ R04) |

Vanity traps considered and rejected: number of pairs found (no denominator; matching everything maximizes it) ·
markets ingested (scale, not correctness) · statement coverage % (says nothing about whether matches are right) ·
model-judged precision where the judge saw the system's own score (anchored, gameable; decode T-10).

## Appetite and deadlines

- **Now:** 2026-10-11 14:27 CDT (UTC-0500)
- **Appetite:** today, until the deadline: "lets get it done today" (decode Q4, approved at G-decode; journal D4)
- **Timebox:** 2026-10-11 13:45 CDT → 2026-10-11 23:30 CDT (scope bends, the time does not)

| Checkpoint | As written in the source | Local time (converted by date) | Source |
|---|---|---|---|
| Final submission | "tonight at 11:30pm" (no zone given; read as the human's local zone, CDT, in decode Q3, approved at G-decode) | 2026-10-11 23:30 CDT (UTC-0500) = 2026-10-12 04:30 UTC | the human, decode Q3 |
| main shows v2 | only when v2's `make check` is green and covers every REQ; until then main@9c3bf48 (v1) is what graders see | before 2026-10-11 23:30 CDT | journal D4 |

## Spec is law

Sources read in full: company info panel (5 sections: header, About, Technical stack, Engineering environment,
Interview process); project brief Version 1 (7 sections: PRD, Problem & context, Requirements & success criteria,
Technical contact, Technology, Submission & AI policy, Required deliverables); Project Equinox PDF (2 pages, 9
sections: Overview, Problem statement, Objectives, Scope, Core expectations, Technical flexibility, Deliverables,
Evaluation criteria, closing goal); syllabus §Technical Interview Expectations.
Requirements extracted: 30. Second pass: 2 new rows added (REQ-16, REQ-17).

| ID | Requirement (verbatim; uncleared source: `(paraphrase, U-##) <text>`) | Source | Deliverable (assignment's exact filename, or observable proof) | Owning phase / skill | Due | Status |
|---|---|---|---|---|---|---|
| REQ-01 | (paraphrase, U-02/U-03) Pull live market metadata and prices from at least two venues' public APIs | brief §Functional; PDF p1 §Objectives, §Scope | observable: a run that ingests both venues live | 3 write-prd → 9 implement-task | final | mapped |
| REQ-02 | (paraphrase, U-02/U-03) One internal market representation, independent of any venue's data format | brief §Functional; PDF p1 §Scope | the model in `docs/architecture/lld.md` and the code | 6 design-lld → 9 implement-task | final | mapped |
| REQ-03 | (paraphrase, U-02/U-03) Identify markets on different venues that may be the same real-world event | brief §Functional; PDF p1 §Scope | observable: proposed pairs with evidence | 3 write-prd → 9 implement-task | final | mapped |
| REQ-04 | (paraphrase, U-03) Define what "equivalent" means, justify the method (rules, heuristics, AI or a mix) and document it | PDF p2 §Core expectations | `docs/product/prd.md` (definition) and `docs/architecture/lld.md` (method), indexed in `README.md` | 3 write-prd, 6 design-lld, 14 garden-docs | final | mapped |
| REQ-05 | (paraphrase, U-02/U-03) Simulate routing a hypothetical order: weigh the available venues, decide, explain why that venue won | brief §Functional; PDF p2 §Core expectations | observable: a decision with its reason | 3 write-prd → 9 implement-task | final | mapped |
| REQ-06 | (paraphrase, U-02/U-03) Log and explain both matching and routing decisions | brief §Functional; PDF p1 §Objectives | observable: a decision log with reasons | 3 write-prd → 9 implement-task | final | mapped |
| REQ-07 | (paraphrase, U-02) Routing logic is deterministic | brief §Performance benchmarks | observable: a replay test (SM-02) | 7 define-acceptance | final | mapped |
| REQ-08 | (paraphrase, U-02) Routing never blocks on external calls | brief §Performance benchmarks | observable: a test with a hung venue | 7 define-acceptance | final | mapped |
| REQ-09 | (paraphrase, U-02/U-03) Degrade gracefully on API failure or inconsistent data; document assumptions where data is incomplete or ambiguous | brief §Performance benchmarks; PDF p2 §Core expectations | failure-injection tests + an assumptions register in `docs/product/prd.md` | 3 write-prd, 7 define-acceptance | final | mapped |
| REQ-10 | (paraphrase, U-02/U-03) Separate venue integration, normalization, equivalence detection and routing | brief §Code quality; PDF p2 §Core expectations | `docs/architecture/hld.md` boundaries, enforced in code | 5 design-hld | final | mapped |
| REQ-11 | (paraphrase, U-02/U-03) The routing layer holds no venue-specific logic or assumptions | brief §Code quality; PDF p2 §Core expectations | observable: a structural test on the routing layer | 5 design-hld, 7 define-acceptance | final | mapped |
| REQ-12 | (paraphrase, U-02) Assumptions documented; reasonable test coverage; clean, readable code | brief §Code quality | assumptions register; `make check` green | 3 write-prd, 9 implement-task | final | mapped |
| REQ-13 | (paraphrase, U-02/U-03) Go preferred, Python acceptable; other tools if justified | brief §Technology; PDF p2 §Technical flexibility | a decision at G2 citing this preference | 5 design-hld | final | mapped |
| REQ-14 | (paraphrase, U-02/U-03) GCP preferred, local deployment acceptable; clarity matters more than infrastructure sophistication | brief §Technology; PDF p2 §Technical flexibility | `docs/runbooks/deploy.md` and a live URL: the human chose Railway (Q2, P-10), not the preferred GCP; ⚠ design-hld must justify the host against this preference (→ G2) | 5 design-hld, 13 ship-milestone (+ U-09) | final | mapped |
| REQ-15 | (paraphrase, U-02/U-03) AI is optional; if used, disclose it and the reasoning behind it | brief §Submission & AI policy; PDF p2 §Technical flexibility, §Deliverables | `docs/AI_USAGE.md` (same file as REQ-21) | supervisor + 12 report-milestone | final | mapped |
| REQ-16 | (paraphrase, U-02) Answer whether cross-venue normalization is realistically achievable, what the architectural tradeoffs are, and how complex smart routing gets | brief §Business context | a stated verdict per question with evidence, in `README.md` and `docs/product/prd.md` success criteria | 3 write-prd, 14 garden-docs | final | mapped |
| REQ-17 | (paraphrase, U-02) Desired state: a reusable foundation that can support multiple venues | brief §Desired state | observable: adding a venue touches no routing code (REQ-11) | 5 design-hld | final | mapped |
| REQ-18 | "Source Code" | brief §Required deliverables | the repository | 9 implement-task | final | mapped |
| REQ-19 | "Technical Documentation" | brief §Required deliverables | `docs/` (prd.md, hld.md, lld.md, diagrams), indexed in `docs/README.md` and `README.md` | 3, 5, 6, 14 garden-docs | final | mapped |
| REQ-20 | "Demo Video" | brief §Required deliverables | `docs/demo/out/*` + an uploaded video link submitted with the repo link (Q1) | 17 write-demo-script, 18 record-demo (+ U-07) | final | mapped |
| REQ-21 | "AI Usage Log" | brief §Required deliverables | `docs/AI_USAGE.md` | supervisor (dated entries as work happens) + a section in each 12 report-milestone | final | mapped |
| REQ-22 | "Test Results" | brief §Required deliverables | `docs/milestones/M<N>.md` evidence sections, linked from `README.md` | 12 report-milestone, 14 garden-docs | final | mapped |
| REQ-23 | "Deployment Guide" | brief §Required deliverables | `docs/runbooks/deploy.md` (Railway, Q2), linked from `README.md` | 13 ship-milestone | final | mapped |
| REQ-24 | (paraphrase, U-03) A working prototype | PDF p2 §Deliverables | observable: `make check` green and a live run | 9 implement-task | final | mapped |
| REQ-25 | (paraphrase, U-03) Setup instructions | PDF p2 §Deliverables | `README.md` + `AGENTS.md` Commands | 14 garden-docs | final | mapped |
| REQ-26 | (paraphrase, U-03) A brief architecture overview | PDF p2 §Deliverables | `docs/architecture/hld.md`, summarized in `README.md` | 5 design-hld, 14 garden-docs | final | mapped |
| REQ-27 | (paraphrase, U-03) A written explanation of the equivalence logic | PDF p2 §Deliverables | `docs/architecture/lld.md` §equivalence, linked from `README.md` | 6 design-lld, 14 garden-docs | final | mapped |
| REQ-28 | (paraphrase, U-03) A written explanation of the routing logic | PDF p2 §Deliverables | `docs/architecture/lld.md` §routing, linked from `README.md` | 6 design-lld, 14 garden-docs | final | mapped |
| REQ-29 | (paraphrase, U-03) Out of scope: real-money trading, wallet integration, regulatory implementation, production UI; UI polish does not count | PDF p2 §Scope; brief §Functional | non-goals in `docs/product/prd.md` | 3 write-prd | final | mapped |
| REQ-30 | (paraphrase, syllabus) An interview follows the submission: defend architecture, verification and evaluation choices, production thinking and each metric, including how it could be gamed | syllabus §Technical Interview Expectations | the decision journal's interview lines; private quiz notes | X log-decision, 16 quiz-me | after submission | mapped |

Conflicts between sources: the portal marks AI documentation optional (§Submission & AI policy) yet lists an AI
usage log among the required deliverables, and the PDF requires disclosure if AI was used. AI is used, so every
source requires the log (REQ-15, REQ-21); nothing left to decide.

## User-owned steps

Things only the human can do. Earliest-blocking first. Every source outside the repo adds a row: "May the brief quote
<source> word for word, and may a copy be committed? ⚠ confidential? (→ G0)".

| ID | Step | Why only the human | Due | Blocks |
|---|---|---|---|---|
| U-01 | May the brief quote the company info panel word for word, and may a copy be committed? ⚠ confidential? (→ G0) | the partner's material; the repo is public | G0 | quoting in all docs |
| U-02 | May the brief quote the project brief Version 1 word for word, and may a copy be committed? ⚠ confidential? (→ G0) | same | G0 | quoting in all docs |
| U-03 | May the brief quote the Project Equinox PDF word for word, and may a copy be committed? ⚠ confidential? (→ G0) | same | G0 | quoting in all docs |
| U-04 | Say yes to pushing milestone branches (GitLab + GitHub mirror) and opening a draft merge request per milestone | pushing is human-only (WORKFLOW §2 rule 8) | before M1 closes | CI runs, MR review, REQ-18 |
| U-05 | Make the GitLab project public (it was created private by the v1 push) | repo visibility is human-only | before 2026-10-11 23:30 CDT | graders opening REQ-18 |
| U-06 | Spot-check the labelled pairs behind NS-01 and SM-01 | the human is the independent rater (decode T-10) | before the metric is reported | NS-01, SM-01 |
| U-07 | Record the narrated demo video and upload it | your voice, face and account | before 2026-10-11 23:30 CDT | REQ-20 |
| U-08 | Approve each merge into main (only once v2 is green and covers every REQ, D4) | merging is human-only | before 2026-10-11 23:30 CDT | REQ-18, REQ-24 |
| U-09 | Railway (Q2): create or approve the Equinox project and service in your Railway account, approve its spend, hold any secret values, and say yes to each deploy naming its SHA (the agent may run non-secret setup only after a recorded yes) | your account, money and authority | before 13 ship-milestone | REQ-14, REQ-23, REQ-24 |
| U-10 | Submit on the portal: the public GitLab project link as the primary link and the demo as an uploaded video link; `README.md` opens with a deliverables index (Q1) | your account | 2026-10-11 23:30 CDT | everything |
| U-11 | Optional: email the technical contact with clarifying questions (e.g. which venues, their idea of "equivalent"); default no today, assumptions are documented instead | sending messages is human-only | — | REQ-04 |
| U-12 | Optional: install gitleaks so the pre-push hook scans for secrets | installs are human-only | before the first push | U-04 |

## Track

**Recommended: Full**, because WORKFLOW.md §3 puts a new product, a graded project or a capstone on Full.
Greenfield or brownfield: greenfield (v1's code is removed on this branch and only cited as evidence, D3).
Track announced by the supervisor: Full (TRACKER M1-S01, M1-S02).
Existing PRD: none.

| Option | What you get | What it costs, in plain words | Fits this ask? |
|---|---|---|---|
| Full | Every phase from this brief to ship: PRD, UX, HLD, LLD, diagrams, acceptance, tasks, gates | about a dozen gates today, each a short answer from you; the most planning before code | yes: a graded partner project restarted to practise the whole workflow (D1) |
| Standard | Starts from an existing PRD: design delta, acceptance, tasks, build | needs a PRD that does not exist; skipping it means the definition of "equivalent" is never decided by you | no: no PRD exists |
| Fix | One failing repro and the smallest fix, ≤2 files, no API/data/auth/payment change | nothing to fix; there is no code | no |
| Do nothing | No build; the time goes elsewhere | main stays v1, the one-shot build you didn't decide, and the restart ends | no: contradicts D1 and D4 |

## Research questions

For `gather-evidence`. Questions only: no answers, no recommended solutions.

| ID | Question | Decision it unblocks | Answered when | From |
|---|---|---|---|---|
| R01 | Which prediction-market venues offer public, unauthenticated access to market metadata and prices with depth, and what are their rate limits, terms of use and data formats? | which two venues (REQ-01) | each candidate venue's official docs, plus one live request each | REQ-01 |
| R02 | How much do the leading venues' open markets overlap today (the same real-world events), and in which topics? | whether matching matters at scale and where to measure it (JS-01, NS-01) | a dated live count by topic | JS-01 |
| R03 | How does each candidate venue define settlement (resolution source, time windows, cut-off times, ties, cancellations, carve-outs), and where do look-alike listings differ? | the definition of "equivalent" (REQ-04) | each venue's rulebook or contract terms, plus documented cases where look-alikes settled differently | REQ-04, decode T-2 |
| R04 | What are each venue's fee schedules, rounding rules and price formats, and how much do fees move the all-in cost of an order? | how routing compares venues; SM-03 (REQ-05) | each venue's published fee schedule with worked examples | REQ-05, SM-03 |
| R05 | What data could measure NS-01: how do existing tools and published research detect equivalent markets and report accuracy, and what makes a labelled set defensible (size, sampling, a blind second rater) and how could it be gamed? | the North Star's data source and method (NS-01, SM-01) | cited papers or tool docs with reported accuracy, and a sampling method | NS-01, decode T-10 |
| R06 | Who is the partner (PEAK6 Labs; Apex Fintech Solutions' work on non-traditional assets), what would cross-venue routing serve, and which venues may a US-regulated broker's clients use? | the hiring-partner lens and venue eligibility (REQ-16) | company pages, filings or press, and each venue's regulatory status | Hiring partner ⚠ |
| R07 | Market: who already aggregates or routes across prediction markets, how many users and how much volume do they have, and who is the ideal customer for cross-venue routing? | how the feasibility verdict is positioned against what exists (REQ-16) | named competitors with user or volume figures and sources | REQ-16 |
| R08 | Usage and load: how many markets and price updates each venue carries, when markets are busiest (peak hours and events), and what request budget the rate limits allow | refresh cadence and the no-blocking budget in design (REQ-08) | measured counts from live calls and published limits | REQ-08 |

## Parking lot

Solution ideas heard during discovery, kept verbatim and not evaluated. Design phases may pick them up.

| ID | Idea (verbatim) | Raised by | When | Need behind it |
|---|---|---|---|---|
| P-01 | "Go standard library only" (v1, main@9c3bf48) | agent (decode parking lot) | 2026-10-11 | REQ-13 |
| P-02 | "Hexagonal ports and adapters" (v1) | agent (decode parking lot) | 2026-10-11 | JS-06 |
| P-03 | "A canonical binary-proposition model with integer micro-dollar money and fees as a curve" (v1) | agent (decode parking lot) | 2026-10-11 | JS-01, JS-02 |
| P-04 | "A copy-on-write snapshot store; routing as a pure function" (v1) | agent (decode parking lot) | 2026-10-11 | JS-03, JS-04 |
| P-05 | "TF-IDF blocking plus veto rules plus one-to-one assignment for matching" (v1) | agent (decode parking lot) | 2026-10-11 | JS-01 |
| P-06 | "A reviewed mapping table fed by offline LLM adjudication" (v1) | agent (decode parking lot) | 2026-10-11 | JS-01, decode T-13 |
| P-07 | "Record and replay of venue responses" (v1) | agent (decode parking lot) | 2026-10-11 | JS-04 |
| P-08 | "A Docker image targeting Cloud Run" (v1) | agent (decode parking lot) | 2026-10-11 | REQ-14 |
| P-09 | "Kalshi and Polymarket as the two venues" (v1) | agent (decode parking lot) | 2026-10-11 | REQ-01 |
| P-10 | "no, we're deploying to railway" | human (Q2) | 2026-10-11 | a live URL graders can open (REQ-14, REQ-23, REQ-24); the human already deploys Cerbo there |

## Interview log

One question at a time, each with a recommendation; answers verbatim, with secret or personal values replaced by
`[redacted: <what it was>]`. Pass 1, 2026-10-11.

| Q# | Question | Recommended (and why) | Answer (verbatim) | Landed in |
|---|---|---|---|---|
| Q1 | What exactly do you submit, and where: a portal form with the repo link and a video link? Which repo link do graders get? | The GitLab project (made public, U-05) as the primary link, the demo as an uploaded video link, and `README.md` opening with a deliverables index, because graders open what is linked and a private repo scores zero | "yes" (2026-10-11 14:32 CDT) | U-10, REQ-18-REQ-23 |
| Q2 | Live GCP deployment, or local run plus a deployment guide? | Local run plus a guide that covers GCP, with a live GCP deployment only if you already have a GCP project with billing, because the PDF accepts local, values clarity over infrastructure, and spend is yours (U-09) | "no, we're deploying to railway" (2026-10-11 14:33 CDT) | P-10, REQ-14, REQ-23, U-09 |
