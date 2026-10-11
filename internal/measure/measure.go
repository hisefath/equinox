// Package measure draws seeded samples, produces blind rater cards and scores labels into precision (with a Wilson
// 95% interval), recall and Cohen's kappa (lld.md BE-13, D9, SC-001..SC-003).
package measure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
)

// Card is what a rater sees: both venues' questions and rules only — never the verdict, score or evidence (INV-8).
type Card struct {
	PairID       string `json:"pair_id"`
	AQuestion    string `json:"a_question"`
	AOutcome     string `json:"a_outcome"`
	ARules       string `json:"a_rules"`
	ACloses      string `json:"a_closes"`
	BQuestion    string `json:"b_question"`
	BOutcome     string `json:"b_outcome"`
	BRules       string `json:"b_rules"`
	BCloses      string `json:"b_closes"`
	FingerprintA string `json:"fingerprint_a"`
	FingerprintB string `json:"fingerprint_b"`
}

// Sample is one measurement draw (lld.md samples).
type Sample struct {
	SampleID       string    `json:"sample_id"`
	Seed           uint64    `json:"seed"`
	RulesVersion   string    `json:"rules_version"`
	DrawnAt        time.Time `json:"drawn_at"`
	PrecisionPairs []string  `json:"precision_pairs"`
	RecallPool     []string  `json:"recall_pool"`
	Cards          []Card    `json:"cards"`
}

// Label is one rater's label for one card (lld.md labels).
type Label struct {
	SampleID     string    `json:"sample_id"`
	PairID       string    `json:"pair_id"`
	Rater        string    `json:"rater"` // "ai" or "human"
	Label        string    `json:"label"` // "equivalent" or "not_equivalent"
	LabelledAt   time.Time `json:"labelled_at"`
	RulesVersion string    `json:"rules_version"`
}

// Draw takes up to n equivalent pairs and a recall pool (near misses plus undecided/related pairs), shuffled with a
// recorded seed so the same seed draws the same sample (FR-028, FR-032).
func Draw(ps match.PairSet, markets map[market.Key]market.Market, seed uint64, n int, rulesVersion string, at time.Time) Sample {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var eq, pool []match.Pair
	for _, p := range ps.Pairs {
		if p.Verdict == match.Equivalent {
			eq = append(eq, p)
		} else {
			pool = append(pool, p)
		}
	}
	for _, nm := range ps.NearMisses {
		pool = append(pool, nm.Pair)
	}
	byID := func(a, b match.Pair) int { return cmpStr(a.PairID, b.PairID) }
	slices.SortFunc(eq, byID)
	slices.SortFunc(pool, byID)
	pool = slices.CompactFunc(pool, func(a, b match.Pair) bool { return a.PairID == b.PairID })
	rng.Shuffle(len(eq), func(i, j int) { eq[i], eq[j] = eq[j], eq[i] })
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	eq = eq[:min(n, len(eq))]
	pool = pool[:min(n, len(pool))]
	s := Sample{SampleID: fmt.Sprintf("%s-%d", at.UTC().Format("20060102"), seed), Seed: seed, RulesVersion: rulesVersion, DrawnAt: at.UTC()}
	for _, p := range eq {
		s.PrecisionPairs = append(s.PrecisionPairs, p.PairID)
		s.Cards = append(s.Cards, card(p, markets))
	}
	for _, p := range pool {
		s.RecallPool = append(s.RecallPool, p.PairID)
		s.Cards = append(s.Cards, card(p, markets))
	}
	return s
}

func cmpStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func card(p match.Pair, ms map[market.Key]market.Market) Card {
	a, b := ms[p.AKey], ms[p.BKey]
	return Card{PairID: p.PairID, AQuestion: a.Question, AOutcome: a.Outcome, ARules: a.RulesText, ACloses: a.ClosesAt.Format(time.RFC3339),
		BQuestion: b.Question, BOutcome: b.Outcome, BRules: b.RulesText, BCloses: b.ClosesAt.Format(time.RFC3339),
		FingerprintA: p.FingerprintA, FingerprintB: p.FingerprintB}
}

// ImportLabels reads JSON-lines labels and rejects the file on any invalid line (FR-033).
func ImportLabels(s Sample, r io.Reader) ([]Label, error) {
	in := map[string]bool{}
	for _, c := range s.Cards {
		in[c.PairID] = true
	}
	var out []Label
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for line := 1; sc.Scan(); line++ {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var l Label
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		switch {
		case !in[l.PairID]:
			return nil, fmt.Errorf("line %d: pair %s is not in sample %s", line, l.PairID, s.SampleID)
		case l.Rater != "ai" && l.Rater != "human":
			return nil, fmt.Errorf("line %d: rater must be ai or human", line)
		case l.Label != "equivalent" && l.Label != "not_equivalent":
			return nil, fmt.Errorf("line %d: label must be equivalent or not_equivalent", line)
		}
		l.SampleID, l.RulesVersion = s.SampleID, s.RulesVersion
		if l.LabelledAt.IsZero() {
			l.LabelledAt = time.Now().UTC()
		}
		out = append(out, l)
	}
	return out, sc.Err()
}

// Report is a measurement result.
type Report struct {
	SampleID       string  `json:"sample_id"`
	Seed           uint64  `json:"seed"`
	RulesVersion   string  `json:"rules_version"`
	PrecisionN     int     `json:"precision_n"`
	PrecisionTrue  int     `json:"precision_true"`
	Precision      float64 `json:"precision"`
	PrecisionLow   float64 `json:"precision_ci_low"`
	PrecisionHigh  float64 `json:"precision_ci_high"`
	RecallTruePool int     `json:"recall_true_in_pool"`
	RecallFound    int     `json:"recall_found"`
	Recall         float64 `json:"recall"`
	KappaN         int     `json:"kappa_n"`
	Kappa          float64 `json:"kappa"`
	Warning        string  `json:"warning,omitempty"`
}

// Score turns labels into the report. The human label wins where both raters labelled a pair; the AI label covers
// the rest. It refuses labels drawn under another rules version (INV-9). Recall counts true pairs in the recall pool
// as missed (the system did not call them equivalent) against the true pairs it did find in the precision sample.
func Score(s Sample, labels []Label, currentRules string) (Report, error) {
	if s.RulesVersion != currentRules {
		return Report{}, fmt.Errorf("labels drawn under rules %s, current is %s; draw and label a new sample", s.RulesVersion, currentRules)
	}
	ai, human := map[string]string{}, map[string]string{}
	for _, l := range labels {
		if l.RulesVersion != s.RulesVersion {
			return Report{}, fmt.Errorf("label for %s is from rules %s", l.PairID, l.RulesVersion)
		}
		if l.Rater == "human" {
			human[l.PairID] = l.Label
		} else {
			ai[l.PairID] = l.Label
		}
	}
	final := func(id string) (string, bool) {
		if v, ok := human[id]; ok {
			return v, true
		}
		v, ok := ai[id]
		return v, ok
	}
	r := Report{SampleID: s.SampleID, Seed: s.Seed, RulesVersion: s.RulesVersion}
	for _, id := range s.PrecisionPairs {
		if v, ok := final(id); ok {
			r.PrecisionN++
			if v == "equivalent" {
				r.PrecisionTrue++
			}
		}
	}
	missed := 0
	for _, id := range s.RecallPool {
		if v, ok := final(id); ok && v == "equivalent" {
			missed++
		}
	}
	r.RecallTruePool, r.RecallFound = missed+r.PrecisionTrue, r.PrecisionTrue
	if r.RecallTruePool > 0 {
		r.Recall = float64(r.RecallFound) / float64(r.RecallTruePool)
	}
	if r.PrecisionN > 0 {
		r.Precision = float64(r.PrecisionTrue) / float64(r.PrecisionN)
		r.PrecisionLow, r.PrecisionHigh = Wilson(r.PrecisionTrue, r.PrecisionN)
		if r.Precision < 0.90 {
			r.Warning = "precision is below the 0.90 hard gate: routing needs a human-reviewed pair list (FR-034)"
		}
	}
	var both [][2]string
	for id, h := range human {
		if a, ok := ai[id]; ok {
			both = append(both, [2]string{h, a})
		}
	}
	r.KappaN = len(both)
	r.Kappa = Kappa(both)
	return r, nil
}

// Wilson is the Wilson score 95% interval for k successes in n trials.
func Wilson(k, n int) (float64, float64) {
	if n == 0 {
		return 0, 0
	}
	const z = 1.959964
	p, nf := float64(k)/float64(n), float64(n)
	den := 1 + z*z/nf
	c := (p + z*z/(2*nf)) / den
	h := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / den
	return math.Max(0, c-h), math.Min(1, c+h)
}

// Kappa is Cohen's kappa for two raters over binary labels; 1 when both always agree on one class.
func Kappa(pairs [][2]string) float64 {
	if len(pairs) == 0 {
		return 0
	}
	n := float64(len(pairs))
	var agree, aEq, bEq float64
	for _, p := range pairs {
		if p[0] == p[1] {
			agree++
		}
		if p[0] == "equivalent" {
			aEq++
		}
		if p[1] == "equivalent" {
			bEq++
		}
	}
	po := agree / n
	pe := (aEq/n)*(bEq/n) + (1-aEq/n)*(1-bEq/n)
	if pe == 1 {
		return 1
	}
	return (po - pe) / (1 - pe)
}
