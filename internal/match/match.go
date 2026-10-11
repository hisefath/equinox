// Package match turns two venues' markets into one-to-one pairs with a verdict, evidence and caveats
// (lld.md BE-7, D19, D20). It is a pure function of its inputs: every iteration is over sorted slices.
package match

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/refdata"
)

// Pair verdicts (SM-1).
const (
	Undecided  = "undecided"
	Equivalent = "equivalent"
	Related    = "related"
	Stale      = "stale"
)

// Parts are the settlement parts Compare checks, in the order evidence is written (FR-004, FR-006).
var Parts = []string{"question", "source", "statistic", "window", "threshold", "edge rule"}

// Window is how far apart two markets' close times may be and still be candidates (Q-3 default).
const Window = 7 * 24 * time.Hour

// maxDF is the most markets a token may appear in and still count as rare.
// ponytail: fixed cut-off; tune from the M5 measurement (Q-3) and freeze into the rules version.
const maxDF = 50

var (
	ErrUnknownPair  = errors.New("unknown pair")
	ErrNotRoutable  = errors.New("pair is not routable")
	ErrRulesChanged = errors.New("rules changed since matching")
)

// Verdict is the result of comparing one pair.
type Verdict struct {
	Verdict       string   `json:"verdict"`
	DifferingPart string   `json:"differing_part,omitempty"`
	Evidence      []string `json:"evidence"`
	Caveats       []string `json:"caveats"`
	Score         int64    `json:"score"` // similarity in parts per thousand
}

// Pair is one proposed match between a market on each venue (lld.md pairs). AKey < BKey by string.
type Pair struct {
	PairID        string     `json:"pair_id"`
	AKey          market.Key `json:"a_key"`
	BKey          market.Key `json:"b_key"`
	Verdict       string     `json:"verdict"`
	DecidedBy     string     `json:"decided_by"`
	DifferingPart string     `json:"differing_part,omitempty"`
	Evidence      []string   `json:"evidence"`
	Caveats       []string   `json:"caveats"`
	Score         int64      `json:"score"`
	FingerprintA  string     `json:"fingerprint_a"`
	FingerprintB  string     `json:"fingerprint_b"`
	RulesVersion  string     `json:"rules_version"`
	MatchedAt     time.Time  `json:"matched_at"`
}

// NearMiss is a rejected look-alike with the reason it was rejected (FR-007).
type NearMiss struct {
	Pair   Pair   `json:"pair"`
	Reason string `json:"reason"`
}

// PairSet is one refresh's pairs, ordered by (verdict, score desc, pair_id), plus near misses by (score desc, pair_id).
type PairSet struct {
	Pairs      []Pair     `json:"pairs"`
	NearMisses []NearMiss `json:"near_misses"`
}

// Matcher matches markets under one rules version (INV-9).
type Matcher struct{ rulesVersion string }

// New returns a matcher stamping rulesVersion on every pair.
func New(rulesVersion string) *Matcher { return &Matcher{rulesVersion: rulesVersion} }

// Match finds candidates by rare shared tokens within the close window, compares each, applies AI verdicts and
// human overrides whose fingerprints still match, then assigns equivalents one-to-one by score, then pair id.
func (m *Matcher) Match(a, b []market.Market, ref refdata.Set) PairSet {
	all := append(slices.Clone(a), b...)
	slices.SortFunc(all, func(x, y market.Market) int { return strings.Compare(string(x.Key), string(y.Key)) })
	var matchedAt time.Time
	toks := make([][]string, len(all))
	df := map[string]int{}
	for i, mk := range all {
		toks[i] = tokens(mk.Question + " " + mk.Outcome)
		for _, t := range toks[i] {
			df[t]++
		}
		if mk.ReceivedAt.After(matchedAt) {
			matchedAt = mk.ReceivedAt
		}
	}
	inA := map[market.Key]bool{}
	for _, mk := range a {
		inA[mk.Key] = true
	}
	// Inverted index over rare tokens: token -> indices into all, ascending.
	index := map[string][]int{}
	for i := range all {
		for _, t := range toks[i] {
			if df[t] <= maxDF {
				index[t] = append(index[t], i)
			}
		}
	}

	var pairs []Pair
	for i, x := range all {
		var cands []int
		for _, t := range toks[i] {
			for _, j := range index[t] {
				if j > i && inA[all[j].Key] != inA[x.Key] {
					cands = append(cands, j)
				}
			}
		}
		slices.Sort(cands)
		for _, j := range slices.Compact(cands) {
			y := all[j]
			if d := x.ClosesAt.Sub(y.ClosesAt); d > Window || d < -Window {
				continue
			}
			pairs = append(pairs, m.decide(x, y, ref, matchedAt))
		}
	}

	// Greedy one-to-one over equivalents (uq_pairs_one_to_one).
	slices.SortFunc(pairs, func(p, q Pair) int {
		if p.Score != q.Score {
			return int(q.Score - p.Score)
		}
		return strings.Compare(p.PairID, q.PairID)
	})
	var ps PairSet
	taken := map[market.Key]string{}
	for _, p := range pairs {
		switch {
		case p.Verdict == Equivalent && (taken[p.AKey] != "" || taken[p.BKey] != ""):
			k := p.AKey
			if taken[k] == "" {
				k = p.BKey
			}
			ps.NearMisses = append(ps.NearMisses, NearMiss{p, fmt.Sprintf("%s already paired in %s", k, taken[k])})
		case p.Verdict == Equivalent:
			taken[p.AKey], taken[p.BKey] = p.PairID, p.PairID
			ps.Pairs = append(ps.Pairs, p)
		case p.Verdict == Related:
			ps.NearMisses = append(ps.NearMisses, NearMiss{p, reason(p.DifferingPart)})
			ps.Pairs = append(ps.Pairs, p)
		default:
			ps.Pairs = append(ps.Pairs, p)
		}
	}
	slices.SortStableFunc(ps.Pairs, func(p, q Pair) int { return strings.Compare(p.Verdict, q.Verdict) })
	return ps
}

func (m *Matcher) decide(x, y market.Market, ref refdata.Set, at time.Time) Pair {
	v := Compare(x, y)
	p := Pair{
		PairID: pairID(x.Key, y.Key), AKey: x.Key, BKey: y.Key,
		Verdict: v.Verdict, DecidedBy: "rules", DifferingPart: v.DifferingPart,
		Evidence: v.Evidence, Caveats: v.Caveats, Score: v.Score,
		FingerprintA: x.RulesFingerprint, FingerprintB: y.RulesFingerprint,
		RulesVersion: m.rulesVersion, MatchedAt: at,
	}
	k := refdata.PK{PairID: p.PairID, FingerprintA: p.FingerprintA, FingerprintB: p.FingerprintB}
	if ai, ok := ref.Verdicts[k]; ok && p.Verdict == Undecided {
		p.Verdict, p.DecidedBy, p.DifferingPart = ai.Verdict, "ai", ai.DifferingPart
	}
	if o, ok := ref.Overrides[k]; ok {
		p.DecidedBy = "human"
		if o.Verdict == "equivalent" {
			p.Verdict, p.DifferingPart = Equivalent, ""
		} else if p.Verdict != Related {
			p.Verdict, p.DifferingPart = Related, "human review"
		}
	}
	return p
}

func reason(part string) string {
	if part == "human review" {
		return "rejected by human review"
	}
	return part + " differs"
}

// Routable returns the pair if it is equivalent and both markets' current fingerprints match those at matching
// time (INV-4, FR-015, FR-027). cur maps each market key to its current rules fingerprint.
func (p PairSet) Routable(id string, cur map[market.Key]string) (Pair, error) {
	for _, q := range p.Pairs {
		if q.PairID != id {
			continue
		}
		switch {
		case q.Verdict == Related:
			return q, fmt.Errorf("%w: %s", ErrNotRoutable, reason(q.DifferingPart))
		case q.Verdict != Equivalent:
			return q, fmt.Errorf("%w: %s", ErrNotRoutable, q.Verdict)
		case cur[q.AKey] != q.FingerprintA || cur[q.BKey] != q.FingerprintB:
			return q, ErrRulesChanged
		}
		return q, nil
	}
	return Pair{}, ErrUnknownPair
}

// Compare decides one pair by its settlement parts: any unknown part on either side -> undecided (LINV-2); else
// the first differing part -> related; else equivalent, with a caveat when the payout times differ (FR-004..006).
func Compare(x, y market.Market) Verdict {
	if x.Key > y.Key {
		x, y = y, x
	}
	v := Verdict{Score: score(x, y)}
	px, py := parts(x.Terms), parts(y.Terms)
	for i := range Parts {
		if px[i] == market.Unknown || py[i] == market.Unknown {
			v.Verdict = Undecided
			return v
		}
	}
	for i, name := range Parts {
		if px[i] != py[i] {
			v.Verdict, v.DifferingPart = Related, name
			return v
		}
		v.Evidence = append(v.Evidence, fmt.Sprintf("%s agrees: %s", name, px[i]))
	}
	v.Verdict = Equivalent
	if !x.ResolvesAt.IsZero() && !y.ResolvesAt.IsZero() && !x.ResolvesAt.Equal(y.ResolvesAt) {
		d := x.ResolvesAt.Sub(y.ResolvesAt).Abs()
		if d >= 24*time.Hour {
			v.Caveats = []string{fmt.Sprintf("pays out about %d days apart", (d+12*time.Hour)/(24*time.Hour))}
		} else {
			v.Caveats = []string{fmt.Sprintf("pays out about %d hours apart", (d+30*time.Minute)/time.Hour)}
		}
	}
	return v
}

// parts renders Terms as one normalized string per entry of Parts, or market.Unknown.
func parts(t market.Terms) []string {
	n := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	known := func(s string) string {
		if s = n(s); s == "" || s == market.Unknown {
			return market.Unknown
		}
		return s
	}
	question := market.Unknown
	if s, p := known(t.Subject), known(t.Predicate); s != market.Unknown && p != market.Unknown {
		question = s + " " + p
	}
	statistic := n(t.Statistic) // "" = not a price question, a known value
	if statistic == "" {
		statistic = "none"
	}
	threshold := "none" // no threshold: the comparator is irrelevant
	if th := n(t.Threshold); th != "" {
		if c := known(t.Comparator); th == market.Unknown || c == market.Unknown {
			threshold = market.Unknown
		} else {
			threshold = c + " " + th
		}
	}
	edge := make([]string, 0, len(t.EdgeRules))
	for _, r := range t.EdgeRules {
		edge = append(edge, n(r))
	}
	slices.Sort(edge)
	edges := strings.Join(edge, "; ")
	if slices.Contains(edge, market.Unknown) {
		edges = market.Unknown
	} else if edges == "" {
		edges = "none"
	}
	return []string{question, known(t.Source), statistic, known(t.WindowEnd), threshold, edges}
}

var stop = map[string]bool{"a": true, "an": true, "the": true, "will": true, "be": true, "by": true, "in": true,
	"at": true, "of": true, "on": true, "to": true, "its": true, "after": true, "before": true, "for": true, "or": true,
	"and": true, "yes": true, "no": true}

// tokens normalizes text to sorted, de-duplicated lowercase words without stop words.
func tokens(s string) []string {
	f := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	out := f[:0]
	for _, w := range f {
		if !stop[w] {
			out = append(out, w)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// score is the Jaccard similarity of the two questions' tokens, in parts per thousand (integer, deterministic).
func score(x, y market.Market) int64 {
	tx, ty := tokens(x.Question+" "+x.Outcome), tokens(y.Question+" "+y.Outcome)
	inter := 0
	for _, t := range tx {
		if _, ok := slices.BinarySearch(ty, t); ok {
			inter++
		}
	}
	union := len(tx) + len(ty) - inter
	if union == 0 {
		return 0
	}
	return int64(inter * 1000 / union)
}

// pairID is the first 16 hex chars of SHA-256 over the two keys in sorted order.
func pairID(x, y market.Key) string {
	if x > y {
		x, y = y, x
	}
	h := sha256.Sum256([]byte(string(x) + "\n" + string(y)))
	return hex.EncodeToString(h[:])[:16]
}
