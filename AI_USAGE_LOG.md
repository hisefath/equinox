# AI Usage Log

The brief requires that AI usage be disclosed, along with the reasoning behind it. This file covers
both: how AI was used to *build* Equinox, and why the *running system* uses no AI.

## 1. Summary

| | |
|---|---|
| Developer | Sefath Chowdhury |
| AI tool | Claude Code (model: Claude Opus 5.5), in the Claude desktop app |
| How it was used | The developer wrote the brief and the working instructions (stack judgement, docs-first, a decision journal, public GitLab plus a GitHub mirror). Claude Code then worked autonomously for one long session (2026-10-05 → 06). It did the research, design, implementation, tests, evaluation and docs. Sub-agents ran research, live-precision audits and the final compliance review |
| AI in the runtime path | **None.** The matcher and router are deterministic Go code; no model is called at runtime |
| AI in evaluation | LLM judges labelled live matched pairs to *measure* the matcher's precision (§4). Their labels seed `reviews/pairs.json` and are marked as LLM-sourced |
| Human review still needed | See §6 |

## 2. Why the running system uses no AI

The PRD requires **deterministic routing**, and the matcher feeds the router. An LLM in the matching
path would:
- make outputs non-reproducible;
- add per-pair cost and latency (tens of thousands of candidate pairs per refresh);
- send market data to a third party;
- be hard to audit ("why did these match?").

Published LLM matching pipelines also need multiple verification passes to reach low false-positive
rates (Gebele & Matthes 2026: <2% false positives after two LLM passes; Saguillo et al. 2025: 81% correct
on dependency extraction). See [`research/prior_art.md`](research/prior_art.md).

So the system uses explainable, deterministic matching, and **LLMs appear only offline**: as evaluators
now, and as the proposed adjudicator that feeds the reviewed mapping table in production
([`docs/EQUIVALENCE.md`](docs/EQUIVALENCE.md) §5).

## 3. Development log

| Phase | What the AI did | How it was checked |
|---|---|---|
| Kickoff | Read the PDF and the PRD. Probed both venues' APIs live with `curl` to see real payloads before designing anything | Live responses |
| Research (parallel sub-agents) | Four researchers: Kalshi API (endpoints, fixed-point fields, the bids-only book, the fee schedule and its overrides), Polymarket API (keyset paging, JSON-in-string fields, worst-first books, the `feeSchedule` curve), prior art (aggregators, academic matching work, divergent resolutions, smart-order-routing concepts), and a **live overlap census** that hand-labelled 55 cross-venue pairs. Two skeptic agents then re-verified every load-bearing API fact | The skeptics confirmed 23 facts and refuted 3, none of which the design depended on (maker-fee scaling, a category detail, and Polymarket's market count, which turned out to be about 257k rather than 80k). Reports: `research/*.md` |
| Design | PRD, system design, assumptions register, Mermaid diagrams, architecture and tradeoffs | Diagrams rendered with mermaid-cli to validate syntax |
| Implementation | All Go code in `cmd/` and `internal/` | `go vet`, `gofmt`, tests, the race detector, live runs |
| Matcher iteration | Built the matcher against the labelled set, ran it on live data, sampled its output, and added vetoes for each observed false-positive family | Labelled-set gate plus independent audits (§4) |
| Docs | All documents in `docs/`, README, this log | A final compliance-review sub-agent pass against the PDF and PRD |

### Mistakes the AI made and caught

These are worth knowing; each was found by a test, a live run or a sub-agent:

1. **Wrong Polymarket fee formula.** The first draft assumed `C × p × rate × (p(1−p))^e`. Research found
   the documented formula, `C × rate × (p(1−p))^e`. Fixed before any routing code depended on it, and the
   docs' fee table is now reproduced in tests.
2. **Live runs blocked locally.** A freshly built Go binary timed out on every HTTPS call while `curl`
   worked. The cause was the Mac's outbound firewall dropping new binaries. Workaround: live runs inside
   Docker. Documented in Troubleshooting.
3. **Non-deterministic matching.** Float sums in map order flipped near-tied scores between runs. Found
   by a diff of two scans, fixed, and now tested.
4. **A regex that swallowed adjacent numbers**, so `"1 (25 bps)"` parsed as `{1}`. Found while
   investigating an audited false positive.
5. **Over-optimism.** The labelled set said precision 1.000; the first live audit said 0.63. The labelled
   set alone was not a sufficient test. That is why the audit process exists, and why the out-of-sample
   number (0.87), not the labelled one, is the headline.

## 4. LLM-as-evaluator: the live audits

- **Why:** labelling hundreds of live pairs by hand wasn't possible in the time available. Independent
  LLM judges reading both venues' rules text against a written strict definition give a defensible
  precision estimate, provided the method is disclosed and the raw labels can be checked.
- **Method:**
  - stratified random samples with fixed seeds;
  - six judges per audit, each blind to the matcher's tier and score;
  - every negative re-judged by a skeptic, and only agreeing negatives counted;
  - an out-of-sample audit on pairs never used for tuning.
- **Artifacts:**
  - `research/audit/` (exploratory, 130 pairs);
  - `research/audit2/` (out of sample, 120 pairs);
  - each `labels.json` holds every verdict with a one-sentence reason.
- **Use of the labels:**
  - to *report* precision ([`docs/TEST_RESULTS.md`](docs/TEST_RESULTS.md));
  - to guide which vetoes to build, from audit 1 only (audit 2 was never used for tuning);
  - to seed `reviews/pairs.json` (247 verdicts, `source: llm-audit: 2 independent judges`).
- **Limits:** the judges read rule *excerpts*, truncated at 700 characters, and can be wrong. Treat the
  labels as a strong second opinion, not ground truth.

## 5. Approximate AI compute

| Run | Sub-agents | Sub-agent tokens |
|---|---|---|
| Research + verification workflow | 6 | about 1.20M |
| Live audit 1 | 8 | about 0.54M |
| Live audit 2 (out of sample) | 8 | about 0.52M |
| Final compliance review | see its report | |
| Main session (design, code, docs) | 1 | not itemized |

No paid venue APIs or credentials were used. All venue data came from public, unauthenticated endpoints.

## 6. What the developer should review personally

1. **The definition of equivalence** ([`docs/EQUIVALENCE.md`](docs/EQUIVALENCE.md) §1), especially the
   decision to treat settlement-timing differences as caveats rather than vetoes.
2. **A sample of the 55 labelled pairs** in `research/labelled_pairs.json`. They are the CI gate.
3. **A sample of `reviews/pairs.json`.** These are LLM verdicts and become routing permissions under
   `-require-review`.
4. **The fee parameters** for any venue or series that matters commercially, checked against the venues'
   current fee schedules.
5. **The scoping assumptions** (A9: 30 Kalshi pages, the 3,000 most-traded Polymarket markets).
