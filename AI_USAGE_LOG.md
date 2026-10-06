# AI Usage Log

The brief requires that AI usage be disclosed, along with the reasoning behind it. This file covers
both: how AI was used to *build* Equinox, and why the *running system* uses no AI.

## 1. Summary

| | |
|---|---|
| Developer | Sefath Chowdhury |
| AI tool | Claude Code (model: Claude Opus 5.5), in the Claude desktop app |
| How it was used | The developer wrote the brief and the working instructions (stack judgement, docs-first, a decision journal, public GitLab plus a GitHub mirror). Claude Code then worked autonomously for one long session (2026-10-05 → 06): research, design, implementation, tests, evaluation and docs. Sub-agents ran research, three live-precision audits and a final four-part review |
| AI in the runtime path | **None.** The matcher and router are deterministic Go code; no model is called at runtime |
| AI in evaluation | LLM judges labelled live matched pairs to *measure* the matcher's precision (§4). Their verdicts seed `reviews/pairs.json`, marked as LLM-sourced |
| Human review still needed | See §6 |

## 2. Why the running system uses no AI

The PRD requires **deterministic routing**, and the matcher feeds the router. An LLM in the matching
path would:
- make outputs non-reproducible;
- add per-pair cost and latency (about 1.2M candidates are scored per refresh);
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
| Research (parallel sub-agents) | Four researchers: Kalshi API, Polymarket API, prior art, and a **live overlap census** that hand-labelled 55 cross-venue pairs. Two skeptic agents then re-verified every load-bearing API fact | The skeptics confirmed 23 facts and refuted 3, none of which the design depended on (maker-fee scaling, a category detail, and Polymarket's market count, re-estimated at about 257k rather than 70–85k). Reports: `research/*.md` |
| Design | PRD, system design, assumptions register, Mermaid diagrams, architecture and tradeoffs | Diagrams rendered with mermaid-cli to validate syntax |
| Implementation | All Go code in `cmd/` and `internal/` | `go vet`, `gofmt`, tests, the race detector, live runs |
| Matcher iteration | Built the matcher against the labelled set, ran it on live data, audited its output, and added vetoes for each false-positive family | Labelled-set gate plus three audits (§4) |
| Final review | Four reviewers (requirements compliance, money/routing correctness, pipeline correctness, docs accuracy), each followed by an adversarial verifier | Confirmed findings fixed with regression tests; one finding refuted with the venue's own docs ([`docs/TEST_RESULTS.md`](docs/TEST_RESULTS.md) §7) |
| Docs | All documents in `docs/`, README, this log | The docs-accuracy reviewer checked every number, path and offline command |

### Mistakes the AI made and caught

Each was found by a test, a live run, an audit or a review sub-agent:

1. **Wrong Polymarket fee formula.** The first draft had an extra factor of p. Research found the
   documented formula. Fixed before any routing code depended on it, and the docs' fee table is now
   reproduced in tests.
2. **Live runs blocked locally.** A freshly built Go binary timed out on every HTTPS call while `curl`
   worked. The cause was the Mac's outbound firewall dropping new binaries. Workaround: live runs inside
   Docker. Documented in Troubleshooting.
3. **Non-deterministic matching.** Float sums in map order flipped near-tied scores between runs. Fixed,
   and now tested.
4. **Over-optimism.** The labelled set said precision 1.000; the first live audit said 0.63. That is why
   the audits exist, and why the out-of-sample number (0.88), not the labelled one, is the headline.
5. **Parsing bugs.**
   - Adjacent numbers were swallowed (found in an audit).
   - `$1,000` was read the same as `$1,000,000`, and "2026-27" seasons were read as ranges (both found in
     the final review).
6. **Polymarket book timestamps were misread** as fetch time. They are last-change time, which biased
   routing toward Kalshi. Found in the final review.
7. **A 31 MB Chromium core dump** from the diagram renderer was committed. Found in the final review and
   removed. It remains in the history of the first pushed commit, because rewriting pushed history is the
   developer's decision. It contains no secrets.
8. **The docs overstated the audit method.** They said the judges were "blind" to tier and score and
   that verdicts came from "2 independent judges". In audits 1 and 2 the judges' inputs included tier and
   score (with an instruction to ignore them), and confirmations had a single judge. Found by the
   docs-accuracy reviewer. The docs were corrected, and audit 3 was run genuinely blind.

## 4. LLM-as-evaluator: the live audits

- **Why:** labelling hundreds of live pairs by hand wasn't possible in the time available. Independent
  LLM judges reading both venues' rules text against a written strict definition give a defensible
  precision estimate, provided the method is disclosed and the raw verdicts can be checked.
- **Method:**
  - stratified random samples with fixed seeds;
  - six judges per audit;
  - one skeptic re-judges every negative, and a false positive counts only if both agree;
  - audits 2 and 3 used pairs never used for tuning;
  - **audits 1–2 showed judges the matcher's tier and score (an anchoring risk); audit 3 did not.**
- **Artifacts:** `research/audit/` (exploratory, 130 pairs), `research/audit2/` (out of sample, 120),
  `research/audit3/` (blind, 50). Each has the exact inputs (`chunk*.json`) and every verdict with a
  one-sentence reason (`labels.json`).
- **Use of the verdicts:**
  - to *report* precision ([`docs/TEST_RESULTS.md`](docs/TEST_RESULTS.md) §4);
  - to guide which vetoes to build, from audit 1 only;
  - to seed `reviews/pairs.json`: 297 verdicts, of which 197 confirmations come from one judge and 100
    rejections from a judge and a skeptic agreeing. The `source` field says which.
- **Limits:** the judges read rule *excerpts*, truncated at 700 characters, and can be wrong. Treat the
  verdicts as a strong second opinion, not ground truth.

## 5. Approximate AI compute

| Run | Sub-agents | Sub-agent tokens |
|---|---|---|
| Research + verification workflow | 6 | about 1.20M |
| Live audit 1 | 8 | about 0.54M |
| Live audit 2 (out of sample) | 8 | about 0.52M |
| Live audit 3 (blind, final matcher) | 8 | about 0.45M |
| Final review (4 reviewers + 4 verifiers) | 8 | about 1.45M |
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
6. **Whether to purge the core dump from git history** (requires a force-push to both remotes).
