---
id: R05
question: "What data could measure NS-01: how do existing tools and published research detect equivalent markets and report accuracy, and what makes a labelled set defensible (size, sampling, a blind second rater) and how could it be gamed?"
asked_in: specs/M1-cross-venue-equivalence-routing/brief.md#research-questions
unblocks: "the North Star's data source and method (NS-01, SM-01)"
answered_when: "cited papers or tool docs with reported accuracy, and a sampling method"
status: partial
confidence: medium
supersedes: none
date: 2026-10-11
budget: 15/15 tool calls
researched_by: Claude Code (gather-evidence subagent, files-only), claude-opus-5-5
---

# R05 What data could measure matching precision (NS-01), and how is that measurement defended or gamed?

## Answer

Published detectors build candidate pairs from same-date or same-topic blocks, verify them with an LLM, and report
accuracy one stage at a time: a peer-reviewed within-Polymarket study manually checked all 374 LLM-flagged dependent
pairs and confirmed 13 (S05.3), the one cross-venue study found is a preprint reporting a verification false-positive
rate below 2% on an unstated sample and κ=0.94 on 1,000 verification-stage labels from one annotator (S05.2,
non-LB), and pmxt's matching docs publish no accuracy figure (S05.4). The data the brief names for NS-01 is a labelled
set of the pairs the system calls "equivalent", judged blind to the system's score (S05.14); McHugh treats Cohen's
kappa, not percent agreement, as the chance-corrected agreement measure, with about 30 comparisons as a floor (S05.5).
The number can be gamed by matching less (S05.13, S05.14), by labelling with the system's output in view, which in
one annotation study raised agreement and pulled a rater's errors toward the tool's (S05.7), and, per one preprint
survey, by scoring on data the method was tuned on (S05.8, non-LB); confidence medium, because the only cross-venue
accuracy figure is a single preprint and the sampling and leakage sources were read only as abstracts.

## Takeaway

Matching tools report accuracy stage by stage rather than as one end-to-end precision number, and that precision can
be inflated by matching less or by letting the people who label the pairs see the system's answer.

## Findings

- **F1** [LB] Saguillo et al. (peer-reviewed, AFT 2025) detect dependent markets *within Polymarket only*, not across venues. Their pipeline pairs markets that share a topic and end date, then has the DeepSeek-R1-Distill-Qwen-32B LLM enumerate joint resolutions. For the US-election group they ran 46,360 pairs, and 1,576 came out as dependent. An automatic subset checker kept 374 of those, and a manual check of all 374 confirmed 13 (11 NegRisk-NegRisk, 2 NegRisk-Single). As printed, the checker's split lists only 4 NegRisk-NegRisk pairs, so the per-type counts are internally inconsistent; the 374 and 13 totals are stated. Most false positives were weaker dependencies, or the LLM conflating election outcomes such as popular vote vs electoral college. — S05.3 (A), §5.2 p.27:14; S05.10 (B), abstract
- **F2** [LB] The same paper reports accuracy per stage, on a census or a small random sample. Its single-market LLM step ran on 128 NegRisk markets, where a valid output must mark exactly one true condition per vector: 4 returned no valid JSON and 101 passed all checks, reported as 81.45% (that is 101 of the 124 valid outputs) (§5.1). Topic classification was 92% accurate on a random sample of 100 hand-labelled instances (§4.1.1). — S05.3 (A), §4.1.1 p.27:10, §5.1 p.27:13
- **F3** [LB] pmxt's Router returns clusters built from pairwise edges. Each edge carries a relation (`identity`, `subset`, `superset`, `overlap`, `disjoint`) and a confidence from 0 to 1, filterable with `minConfidence`. The page does not describe the matching method or say how confidence is computed or calibrated. It publishes no accuracy, precision, recall, evaluation method or labelled set. Every example passes a `pmxt_live_` key. — S05.4 (A), Router > Analyze > Find Similar Markets (`fetchMatchedMarketClusters`, Parameters, Relation Types)
- **F4** Gebele & Matthes (arXiv preprint, Jan 2026, not peer-reviewed) match across 10 venues, Kalshi and Polymarket among them. Pairs must share one of 20 LLM-assigned categories and overlap in time; retrieval uses text-embedding-3-large with k=20, followed by two LLM passes, under a three-way taxonomy (equivalent, subset, independent). They report that structural filtering kept 100% of validated relations, retrieval at k=20 captured 99.9%, and LLM verification brought the false-positive rate below 2% on an unstated sample. Human annotation used two regimes, one annotator each: a stratified sample of 1,000 candidate pairs, which the authors note is heavily skewed toward true negatives and where agreement was effectively perfect, and 1,000 verification-stage instances with inter-rater agreement κ=0.94; no precision value is given. Residual errors are mostly false negatives. A New York weather-station case was misclassified by both the human and the framework. About 8% of Polymarket's and about 2% of Kalshi's markets have an equivalent elsewhere. — S05.2 (B), §4.1-§4.4, §5.2; S05.9 (B), abstract. Single non-peer-reviewed source: not load-bearing (see Gaps)
- **F5** [LB] Percent agreement does not account for chance agreement and may overestimate true agreement. Cohen's kappa corrects for chance and is for two raters (Fleiss's kappa for three or more). The author treats any kappa below 0.60 as inadequate and calls Cohen's bands possibly too lenient; her Table 3 rates kappa 0.60-0.79 as moderate, with 35-63% of data reliable (her separate example: 61% percent agreement leaves almost 40% of the data faulty). The heuristic is at least 30 comparisons; a worked example with N=222 (kappa 0.85) gives a 95% CI of 0.78-0.92. She calls kappa's assumptions about rater independence poorly supported. — S05.5 (A), Table 3, Figure 3 worked example, sections on percent agreement, kappa and sample size, Conclusions; S05.15 (A), same article, Europe PMC full text
- **F6** [LB] Two annotators corrected an automatic part-of-speech tagger's pre-annotations. The more accurate annotator scored 98.1 unaided but 95.8 when correcting an 81.6%-accurate tagger's output (Table 5); the authors attribute that drop to a lapse in concentration. With the best (96.4%) tagger, that annotator's accuracy held (98.4 to 98.2, Table 4) but their errors became systematic in the direction of the tagger's own systematic error (§4.3, Table 6), while the other annotator improved (94.1 to 97.1). Inter-annotator agreement π was 0.947 on 50 sentences without pre-annotation and 0.983 on 25 with the best tagger's (Table 3). The authors conclude that pre-annotation raised both accuracy and agreement, but brings "biases that should be identified and notified to the annotators" (§5). — S05.7 (A), §4.3, §5, Tables 1, 3-6; S05.11 (A), venue record
- **F7** Under extreme class imbalance between matching and non-matching pairs, sound evaluation of entity resolution needs very many labels. An adaptive importance sampler (OASIS) cut labelling by up to 83% across several datasets without losing estimate accuracy, and its precision, recall and F-measure estimates converge to the true values. — S05.6 (B), abstract (preprint page as read; venue not confirmed)
- **F8** A survey found data-leakage errors in 329 papers across 17 fields, leading in some cases to wildly overoptimistic conclusions. It presents a taxonomy of 8 leakage types, not named in the abstract. — S05.8 (B), abstract (preprint page as read)
- **F9** [LB] The project's NS-01 is matching precision: the proportion of system-called "equivalent" pairs that truly settle the same way in every outcome. SM-01, matching recall, guards it because matching nothing scores perfect precision. The named data is a labelled set of live cross-venue pairs that the human spot-checks blind to the system's score. Rejected vanity traps include pair counts and model-judged precision where the judge saw the system's score. The kit's tests require every metric to be actionable, labelled leading or lagging, and backed by named data; supporting metrics must explain the North Star or guard it from gaming; targets trace to a constraint, benchmark or requirement and are marked hard gate or aspirational. — S05.14 (A), §Value and metrics; S05.13 (A), §2-§4
- **F10** v1, one-shot (main@9c3bf48), not re-measured: v1 reported labelled precision of 1.0 against about 0.63 on its first live audit, and model judges that saw the matcher's score anchored on it. v1's notes also planned a hand-labelled set of 100-300 pairs with hard negatives, reported precision and recall separately rather than F1, and named Dome (sports-only identifier joins) and open-source bots matching on fuzzy-title thresholds. — S05.12 (C, colour only), T-2, T-10; S05.1 (C, colour only), §0, §1, §3.5

## Inferences

- **I1** from F1 + F2 + F3 + F4: in the sources read, accuracy is reported per pipeline stage (retrieval recall, verification false-positive rate, manual confirmation counts). None reports end-to-end precision on a frozen, held-out labelled set of Kalshi–Polymarket pairs, and the one deployed tool read (pmxt) reports nothing. Fairly sure for these sources; a wider sweep could still find one.
- **I2** from F1 + F4: a precision figure means little without its denominator. Formula: precision = confirmed pairs / pairs output at that stage. Assumptions: Saguillo's denominator is the 374 checker-passed pairs; Gebele's under-2% false-positive rate is after two LLM passes on an unstated sample. Range: 13/374 = 3.5% confirmed at Saguillo's stage, versus above 98% implied after Gebele's second pass. The two studies also use different relation definitions (dependent vs equivalent), so their numbers are not comparable. Sure.
- **I3** from F9 + F1 + F4 + F7, grading the candidate metrics with METRICS.md. NS-01 is actionable (a false match points to the rule or field that let it through), lagging, and measurable here only once the system outputs pairs and a person labels them. Saguillo's full manual check of all 374 flagged pairs is a precedent for labelling every output pair rather than a random subset when the output is a few hundred. SM-01 is lagging, and its denominator (true pairs the system missed) cannot come from the system's own output. It needs pairs drawn from outside the output, and because about 2-8% of markets have a counterpart (F4), a uniform random draw of cross-venue pairs yields few positives; F7 reports non-uniform (importance-weighted) draws as the answer to that imbalance. Rater agreement (kappa) checks label quality; it is not a product metric. Moderately sure.
- **I4** from F5 + F9: how much a clean result proves depends on n. Formula: with zero false matches among n labelled output pairs, the exact one-sided 95% upper bound on the false-match rate is 1 - 0.05^(1/n), about 3/n. Assumptions: pairs independent, labels correct, binomial model. This is arithmetic only; its textbook source was unread (see Gaps). Range: n=30 → false-match rate up to 9.5% (precision ≥ 0.905); n=100 → up to 3.0% (≥ 0.970); n=300 → up to 1.0% (≥ 0.990). Under F5's 30-comparison heuristic, a kappa from about 30 double-rated pairs would carry a wide interval (F5's N=222 example already spans 0.78-0.92). Sure of the arithmetic; the thresholds that matter are a PRD decision.
- **I5** from F5 + F6 + F8 + F9 + F10: the gaming routes these sources support are:
  - shrinking the denominator: fewer pairs called equivalent raises precision; F9 names SM-01 as the guard;
  - letting raters see the system's verdict or score: in F6, agreement rose and one rater's errors lined up with the tool's systematic error when raters corrected its output, so high agreement alone does not show the raters judged independently of the system;
  - reporting percent agreement instead of kappa (F5);
  - evaluating on pairs the rules or thresholds were tuned against (F8, B only; F10's 1.0-to-0.63 drop is consistent, C);
  - a reported κ whose raters are unclear (F4: one annotator per stage).

  Each route is described by at least one source; how it is blocked is a PRD and design choice. Moderately sure.

## Gaps

- ⚠ [LB] Is there a peer-reviewed or second independent cross-venue (Kalshi–Polymarket) matching accuracy figure? — why: the only one found is Gebele & Matthes, an arXiv preprint (S05.2, B); Saguillo (A) is within one venue and measures dependency, not equivalence — would close: read a peer-reviewed version of Gebele & Matthes, or another independent study; option: run anthropic-skills:deep-research for this question
- ⚠ What does Gebele's κ=0.94 compare, given one annotator per regime, and on what sample is the under-2% false-positive rate measured? — why: §4.4 names one annotator per regime and says only "across the validated relations" for the stage rates; the appendix was searched by keyword, not read in full — would close: read the appendix of https://arxiv.org/html/2601.01706v1 in full, or ask the authors
- ⚠ What is the sourced upper bound on error rate when zero false matches are seen in n pairs (rule of three)? — why: unread: https://www.bmj.com/content/311/7005/619 (HTTP 403); I4 is arithmetic only — would close: read Eypasch et al. 1995 (BMJ) or Hanley & Lippman-Hand 1983 (JAMA), or Brown, Cai & DasGupta 2001 (Statistical Science) on binomial intervals
- ⚠ Are OASIS (S05.6) and the leakage survey (S05.8) peer-reviewed, and is tuning on the test set one of the 8 named leakage types? — why: only arXiv abstract pages were read; neither page showed a journal reference (cap reached) — would close: read the PVLDB version of Marchant & Rubinstein and the Patterns version of Kapoor & Narayanan
- ⚠ Can an LLM serve as the second rater, and how well does it agree with a human on market equivalence? — why: no source read reports LLM-versus-human agreement for this task (Gebele gives none) — would close: a study reporting LLM-human kappa on market pairs, or a measured pilot in this project
- ⚠ How big is the NS-01 denominator (pairs the system will call equivalent today)? — why: no live venue call was made in this pass (budget spent on literature); it depends on a matcher that does not exist yet; R02 owns live overlap counts — would close: R02's dated live count, then the system's first output
- ⚠ Could pmxt's matched clusters serve as an independent reference set, and what does its confidence mean? — why: the docs give no method, calibration or accuracy, and the endpoint examples use a live API key (not requested) — would close: pmxt's own evaluation write-up, or a key the human chooses to obtain
- ⚠ Page text addressed agents and was ignored: the pmxt docs header tells readers to fetch an llms.txt documentation index before exploring further — why: page text addressed agents and was ignored — would close: nothing to close; recorded for the supervisor

## Handoffs

- write-prd: Is NS-01's denominator every pair the system outputs as "equivalent" (a full check, as Saguillo did for 374 pairs), or a random sample of them, and at what output size does the full check switch to a sample? — rests on F1, F2, I3, I4
- write-prd: Who is the second rater for the labelled set (a second person, a model, or none, with that limitation stated), and is agreement reported as Cohen's kappa with its interval rather than percent agreement? — rests on F4, F5, I4
- write-prd: Is every rater blind to the system's verdict and score while labelling, and are labels frozen before any rule or threshold is tuned against them? — rests on F6, F8, F10, I5
- write-prd: For SM-01, which pool supplies the true pairs the system missed: near-threshold candidates, a random sample within same-date or same-topic blocks, or both? — rests on F4, F7, I3
- write-prd: Is the NS-01 target grounded in a published benchmark (only Gebele's under-2% false-positive rate, a single preprint) or in the cost of one false match (an order sent into a different bet), and is it a hard gate or aspirational? — rests on F4, F9, I2
- write-prd: Which relation vocabulary do labels use (pmxt's five relations, or Gebele's equivalent/subset/independent), and does only identity/equivalent count as a true positive for NS-01? — rests on F3, F4
- design-hld: Where does the labelled set live, and how is it versioned so the scoring set stays separate from any set used to tune matching rules? — rests on F8, F10, I5
- design-hld: If a model proposes labels while the set is built, how is its output kept out of the rater's view during labelling? — rests on F6, I5
