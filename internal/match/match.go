// Package match decides which markets on different venues are the same bet.
//
// Equivalent means: for every state of the world, A resolves YES if and only if B resolves YES. That is
// only knowable from resolution rules written in prose, so the matcher works in two stages:
//
//  1. Recall. TF-IDF cosine similarity over normalized text (event + question + outcome), with an
//     inverted index on rare tokens so that only plausible pairs are scored at all.
//  2. Precision. Hard vetoes on structured fields that text similarity is blind to: thresholds,
//     comparator shape (> vs ≥ vs exactly vs range vs touch), explicit dates, months and years,
//     mutually exclusive scopes (Senate vs House, cut vs hike, YoY vs MoM, ALCS vs World Series),
//     resolution oracles, leagues, teams and outcome entities. A single veto rejects a pair whatever
//     its score.
//
// Survivors are assigned one-to-one per venue pair (greedy by score, deterministic tie-breaks) and
// tiered: "equivalent" pairs may be routed; "review" pairs are plausible but need a human. Every pair,
// and every high-scoring rejection, carries the evidence for the decision.
package match

import (
	"cmp"
	"fmt"
	"maps"
	"math"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

// Tiers.
const (
	Equivalent = "equivalent" // routable
	Review     = "review"     // plausible; a human (or offline LLM) must confirm before routing
)

// Options tunes the matcher. Zero values take the defaults below, which were calibrated on the
// hand-labelled pair set in testdata (see docs/EQUIVALENCE.md).
type Options struct {
	MinScore   float64       // candidates below this cosine are ignored (default 0.30)
	AutoScore  float64       // at or above this, a vetting survivor is "equivalent" (default 0.50)
	MaxDF      float64       // tokens in more than this share of markets are not used for blocking (0.05)
	Candidates int           // candidates scored per market after blocking (25)
	MaxGap     time.Duration // resolution times further apart than this can't be the same event (180 days)
	MinContext float64       // similarity still required with the outcome label removed (default 0.30)
	RareDF     float64       // a subject token in at most this share of markets is distinctive (0.002)
}

func (o Options) withDefaults() Options {
	o.MinScore = cmp.Or(o.MinScore, 0.30)
	o.AutoScore = cmp.Or(o.AutoScore, 0.50)
	o.MaxDF = cmp.Or(o.MaxDF, 0.05)
	o.Candidates = cmp.Or(o.Candidates, 25)
	o.MaxGap = cmp.Or(o.MaxGap, 180*24*time.Hour)
	o.MinContext = cmp.Or(o.MinContext, 0.30)
	o.RareDF = cmp.Or(o.RareDF, 0.002)
	return o
}

// Pair is a proposed equivalence between markets on two venues.
type Pair struct {
	ID       string        `json:"id"`
	Tier     string        `json:"tier"`
	Score    float64       `json:"score"`
	A        market.Market `json:"a"`
	B        market.Market `json:"b"`
	Evidence []string      `json:"evidence"`
	Caveats  []string      `json:"caveats,omitempty"`
	Reviewed string        `json:"reviewed,omitempty"` // source of a reviewer's verdict, if any
}

// Rejection is a similar-looking pair that a veto ruled out: the cases that show why text
// similarity alone is not enough.
type Rejection struct {
	A     string  `json:"a"`
	B     string  `json:"b"`
	AText string  `json:"a_text"`
	BText string  `json:"b_text"`
	Score float64 `json:"score"`
	Veto  string  `json:"veto"`
}

// Result is the full output of a matching run.
type Result struct {
	Pairs      []Pair         `json:"pairs"`
	NearMisses []Rejection    `json:"near_misses"`
	Markets    map[string]int `json:"markets"`  // per venue
	Compared   int            `json:"compared"` // candidate pairs fully scored
	Vetoed     map[string]int `json:"vetoed"`   // veto kind -> count
}

type doc struct {
	f    features
	vec  map[string]float64 // L2-normalized tf-idf
	ctx  map[string]float64 // the same, over context tokens only
	rare []string           // subject tokens rare in the corpus: names and places, not phrasing
}

// Verdict is the matcher's judgement of one specific pair, with its reasons.
type Verdict struct {
	Score    float64  `json:"score"`
	Context  float64  `json:"context"`        // similarity with the outcome labels removed
	Tier     string   `json:"tier,omitempty"` // empty when vetoed or too dissimilar
	Veto     string   `json:"veto,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
	Caveats  []string `json:"caveats,omitempty"`
}

// Corpus holds term statistics over a set of markets, so that a single pair can be judged in the
// same context a full run would see.
type Corpus struct {
	df map[string]int
	n  float64
}

// NewCorpus computes term statistics for ms.
func NewCorpus(ms []market.Market) *Corpus {
	c := &Corpus{df: map[string]int{}, n: float64(len(ms))}
	for _, d := range extractAll(ms) {
		for t := range d.f.tokens {
			c.df[t]++
		}
	}
	return c
}

// Explain judges one pair, so a reviewer can ask why two markets did or did not match.
func (c *Corpus) Explain(a, b market.Market, o Options) Verdict {
	o = o.withDefaults()
	da, db := c.doc(extract(a), o), c.doc(extract(b), o)
	v := Verdict{Score: round(cosine(da.vec, db.vec)), Context: round(cosine(da.ctx, db.ctx))}
	if v.Veto = vet(da, db, o); v.Veto != "" || v.Score < o.MinScore {
		return v
	}
	p := makePair(candidate{da, db, v.Score}, o)
	v.Tier, v.Evidence, v.Caveats = p.Tier, p.Evidence, p.Caveats
	return v
}

func (c *Corpus) doc(f features, o Options) *doc {
	return &doc{f: f, vec: weigh(f.tokens, c.df, c.n), ctx: weigh(f.context, c.df, c.n), rare: rare(f, c.df, c.n, o)}
}

// extractAll runs feature extraction on every core: it is independent per market and dominates run time.
func extractAll(ms []market.Market) []*doc {
	docs := make([]*doc, len(ms))
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	for w := range workers {
		wg.Go(func() {
			for i := w; i < len(ms); i += workers {
				docs[i] = &doc{f: extract(ms[i])}
			}
		})
	}
	wg.Wait()
	return docs
}

// Match finds equivalent markets across every pair of venues present in ms.
func Match(ms []market.Market, o Options) Result {
	o = o.withDefaults()
	docs := extractAll(ms)
	byVenue := map[string][]*doc{}
	df := map[string]int{}
	for _, d := range docs {
		byVenue[d.f.m.Venue] = append(byVenue[d.f.m.Venue], d)
		for t := range d.f.tokens {
			df[t]++
		}
	}
	n := float64(len(ms))
	c := &Corpus{df: df, n: n}
	for _, docs := range byVenue {
		for i, d := range docs {
			docs[i] = c.doc(d.f, o)
		}
		slices.SortFunc(docs, func(a, b *doc) int { return cmp.Compare(a.f.m.Key(), b.f.m.Key()) })
	}

	res := Result{Markets: map[string]int{}, Vetoed: map[string]int{}, Pairs: []Pair{}, NearMisses: []Rejection{}}
	venues := make([]string, 0, len(byVenue))
	for v, docs := range byVenue {
		venues = append(venues, v)
		res.Markets[v] = len(docs)
	}
	slices.Sort(venues)
	for i := range venues {
		for j := i + 1; j < len(venues); j++ {
			matchVenues(byVenue[venues[i]], byVenue[venues[j]], df, n, o, &res)
		}
	}
	slices.SortFunc(res.Pairs, func(a, b Pair) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.ID, b.ID))
	})
	slices.SortFunc(res.NearMisses, func(a, b Rejection) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.A, b.A), cmp.Compare(a.B, b.B))
	})
	res.NearMisses = res.NearMisses[:min(len(res.NearMisses), 200)]
	return res
}

type candidate struct {
	a, b  *doc
	score float64
}

func matchVenues(as, bs []*doc, df map[string]int, n float64, o Options, res *Result) {
	// Inverted index over B on rare tokens only: common tokens ("2026", "win") would make every market
	// a candidate for every other, and rare ones (names, numbers, team ids) carry the identity.
	index := map[string][]int{}
	for j, d := range bs {
		for t := range d.vec {
			if float64(df[t])/n <= o.MaxDF {
				index[t] = append(index[t], j)
			}
		}
	}
	var survivors []candidate
	for _, a := range as {
		partial := map[int]float64{}
		for _, t := range slices.Sorted(maps.Keys(a.vec)) {
			for _, j := range index[t] {
				partial[j] += a.vec[t] * bs[j].vec[t]
			}
		}
		top := make([]int, 0, len(partial))
		for j := range partial {
			top = append(top, j)
		}
		slices.SortFunc(top, func(x, y int) int { return cmp.Or(cmp.Compare(partial[y], partial[x]), cmp.Compare(x, y)) })
		for _, j := range top[:min(len(top), o.Candidates)] {
			b := bs[j]
			score := cosine(a.vec, b.vec)
			if score < o.MinScore {
				continue
			}
			res.Compared++
			if veto := vet(a, b, o); veto != "" {
				kind, _, _ := strings.Cut(veto, ":")
				res.Vetoed[kind]++
				if score >= o.AutoScore {
					res.NearMisses = append(res.NearMisses, Rejection{A: a.f.m.Key(), B: b.f.m.Key(),
						AText: label(a.f.m), BText: label(b.f.m), Score: round(score), Veto: veto})
				}
				continue
			}
			survivors = append(survivors, candidate{a, b, score})
		}
	}
	// One-to-one: a market can be the same bet as at most one market on another venue. Greedy by score
	// with total-order tie-breaks; this alone removes "one market matched to three strikes" errors.
	slices.SortFunc(survivors, func(x, y candidate) int {
		return cmp.Or(cmp.Compare(y.score, x.score), cmp.Compare(x.a.f.m.Key(), y.a.f.m.Key()), cmp.Compare(x.b.f.m.Key(), y.b.f.m.Key()))
	})
	used := map[string]bool{}
	for _, c := range survivors {
		ka, kb := c.a.f.m.Key(), c.b.f.m.Key()
		if used[ka] || used[kb] {
			continue
		}
		used[ka], used[kb] = true, true
		res.Pairs = append(res.Pairs, makePair(c, o))
	}
}

func makePair(c candidate, o Options) Pair {
	a, b := c.a.f, c.b.f
	p := Pair{ID: a.m.Key() + "~" + b.m.Key(), Score: round(c.score), A: a.m, B: b.m, Tier: Review}
	if c.score >= o.AutoScore {
		p.Tier = Equivalent
	}
	p.Evidence = append(p.Evidence, fmt.Sprintf("text similarity %.2f; shared terms: %s", c.score, strings.Join(shared(c.a.vec, c.b.vec, 6), ", ")))
	if len(a.teams) > 0 && len(b.teams) > 0 {
		p.Evidence = append(p.Evidence, "same team: "+strings.Join(intersect(a.teams, b.teams), ", "))
	}
	if len(a.outcome) > 0 && len(b.outcome) > 0 {
		p.Evidence = append(p.Evidence, "outcomes agree: "+strings.Join(intersect(a.outcome, b.outcome), ", "))
	}
	if len(a.nums) > 0 {
		p.Evidence = append(p.Evidence, "thresholds agree: "+strings.Join(a.nums, ", "))
	}
	if a.shape != "" && a.shape == b.shape {
		p.Evidence = append(p.Evidence, "comparator agrees: "+a.shape)
	}
	if len(a.dates) > 0 && len(b.dates) > 0 {
		p.Evidence = append(p.Evidence, "explicit dates agree within a day")
	}
	for g, v := range a.scopes {
		if b.scopes[g] == v {
			p.Evidence = append(p.Evidence, g+" agrees: "+v)
		}
	}
	slices.Sort(p.Evidence[1:])

	if (a.shape == "") != (b.shape == "") {
		p.Caveats = append(p.Caveats, fmt.Sprintf("comparator stated on one side only (%q vs %q)", a.shape, b.shape))
	}
	if (len(a.sources) == 0) != (len(b.sources) == 0) {
		p.Caveats = append(p.Caveats, "resolution source identified on one side only")
	}
	if len(a.years) > 0 && len(b.years) > 0 && yearGap(a.years, b.years) == 1 {
		p.Caveats = append(p.Caveats, fmt.Sprintf("years named differ by one (%v vs %v)", a.years, b.years))
	}
	if gap := absDur(a.resolves.Sub(b.resolves)); gap > 7*24*time.Hour {
		p.Caveats = append(p.Caveats, fmt.Sprintf("venues expect resolution %.0f days apart (settlement timing / capital lock-up differs)", gap.Hours()/24))
	}
	return p
}

// vet returns the first reason a and b cannot be the same proposition, or "".
func vet(da, db *doc, o Options) string {
	a, b := da.f, db.f
	switch {
	case len(a.nums) > 0 && len(b.nums) > 0 && !slices.Equal(a.nums, b.nums):
		return fmt.Sprintf("threshold: %v vs %v", a.nums, b.nums)
	case a.shape != "" && b.shape != "" && a.shape != b.shape:
		return fmt.Sprintf("comparator: %s vs %s", a.shape, b.shape)
	case len(a.nums) > 0 && threshold(a) != threshold(b) && !line(a) && !line(b):
		// "6.1+" vs "6.1", "at least one cut" vs "one cut": the side without a comparator means exactly.
		return fmt.Sprintf("comparator: %q vs %q", a.shape, b.shape)
	case len(a.dates) > 0 && len(b.dates) > 0 && !datesMeet(a.dates, b.dates):
		return fmt.Sprintf("date: day %v vs day %v of year", a.dates, b.dates)
	case len(a.dates) > 0 && len(b.dates) > 0 && !datesEqual(a.dates, b.dates) && absDur(a.m.Close.Sub(b.m.Close)) > 12*time.Hour:
		// "Before Nov 1" and "by Oct 31" are the same deadline, but "before Oct 23" and "by Oct 23" are not.
		// When the dates are a day apart, the venues' own trading cut-offs must agree.
		return fmt.Sprintf("deadline: dates a day apart and trading closes %.0fh apart", absDur(a.m.Close.Sub(b.m.Close)).Hours())
	case len(a.months) > 0 && len(b.months) > 0 && len(intersectInt(a.months, b.months)) == 0:
		return fmt.Sprintf("month: %v vs %v", a.months, b.months)
	case len(a.years) > 0 && len(b.years) > 0 && yearGap(a.years, b.years) > 1:
		// One year apart is often naming, not substance ("2027 Congress" vs "2026 midterms"); real
		// one-year differences are caught by the timing check below and flagged as a caveat.
		return fmt.Sprintf("year: %v vs %v", a.years, b.years)
	case len(a.sources) > 0 && len(b.sources) > 0 && len(intersect(a.sources, b.sources)) == 0:
		return fmt.Sprintf("resolution source: %v vs %v", a.sources, b.sources)
	case len(a.leagues) > 0 && len(b.leagues) > 0 && len(intersect(a.leagues, b.leagues)) == 0:
		return fmt.Sprintf("league: %v vs %v", a.leagues, b.leagues)
	case len(a.teams) > 0 && len(b.teams) > 0 && len(intersect(a.teams, b.teams)) == 0:
		return fmt.Sprintf("team: %v vs %v", a.teams, b.teams)
	case len(a.outcome) > 0 && len(b.outcome) > 0 && len(intersect(a.outcome, b.outcome)) == 0:
		return fmt.Sprintf("outcome: %v vs %v", a.outcome, b.outcome)
	case !slices.Equal(a.ranks, b.ranks):
		return fmt.Sprintf("rank: %v vs %v", a.ranks, b.ranks)
	case !slices.Equal(a.mods, b.mods):
		return fmt.Sprintf("modifier: %v vs %v", a.mods, b.mods)
	case threshold(a) && len(b.nums) == 0 || threshold(b) && len(a.nums) == 0:
		return "threshold: stated on one side only"
	case !a.resolves.IsZero() && !b.resolves.IsZero() && absDur(a.resolves.Sub(b.resolves)) > o.MaxGap:
		return fmt.Sprintf("timing: resolutions %.0f days apart", absDur(a.resolves.Sub(b.resolves)).Hours()/24)
	}
	for _, g := range slices.Sorted(maps.Keys(a.scopes)) {
		if v, ok := b.scopes[g]; ok && v != a.scopes[g] {
			return fmt.Sprintf("scope (%s): %s vs %s", g, a.scopes[g], v)
		}
	}
	// Each side names something rare that the other never mentions: Greenland vs Alberta, two different
	// candidates in the same race. One-sided extras are fine ("Francis James Allison Oyague").
	if ua, ub := missing(da.rare, db.f.tokens), missing(db.rare, da.f.tokens); len(ua) > 0 && len(ub) > 0 {
		return fmt.Sprintf("subject: %v vs %v", ua, ub)
	}
	// Text similarity is dominated by rare names, so "Spain wins the Women's World Cup" resembles
	// "Spain participates in Eurovision". With the outcome label taken out, the rest of the question
	// must still say the same thing.
	if c := cosine(da.ctx, db.ctx); c < o.MinContext {
		return fmt.Sprintf("context: question similarity %.2f without the outcome label", c)
	}
	return ""
}

func rare(f features, df map[string]int, n float64, o Options) []string {
	var out []string
	for _, t := range f.subject {
		if float64(df[t]) <= max(o.RareDF*n, 1) && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	slices.Sort(out)
	return out
}

// missing returns the tokens of ts that never occur in other.
func missing(ts []string, other map[string]float64) []string {
	var out []string
	for _, t := range ts {
		if other[t] == 0 {
			out = append(out, t)
		}
	}
	return out
}

// line reports whether f is a points line (spread, total), where the comparator is implied by the
// market type rather than written: "Spread: Ravens (-7.5)" means "wins by more than 7.5".
func line(f features) bool {
	return f.tokens["spread"] > 0 || f.tokens["ou"] > 0 || f.tokens["handicap"] > 0 || f.tokens["total"] > 0
}

func datesEqual(a, b []int) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}
	return false
}

// threshold reports whether f states a numeric condition (a number with a comparator).
func threshold(f features) bool {
	return len(f.nums) > 0 && f.shape != "" && f.shape != "="
}

// datesMeet reports whether some pair of explicit dates is within one day, on a circular calendar so
// that "before Jan 1, 2027" meets "by December 31".
func datesMeet(a, b []int) bool {
	for _, x := range a {
		for _, y := range b {
			d := x - y
			if d < 0 {
				d = -d
			}
			if min(d, 365-d) <= 1 {
				return true
			}
		}
	}
	return false
}

// weigh builds an L2-normalized tf-idf vector. Keys are visited in sorted order: float addition is
// not associative, and map order would otherwise make near-tied scores flip between runs.
func weigh(tf map[string]float64, df map[string]int, n float64) map[string]float64 {
	v := make(map[string]float64, len(tf))
	var norm float64
	for _, t := range slices.Sorted(maps.Keys(tf)) {
		w := tf[t] * (math.Log((n+1)/float64(df[t]+1)) + 1)
		v[t] = w
		norm += w * w
	}
	norm = math.Sqrt(norm)
	for t := range v {
		v[t] /= norm
	}
	return v
}

// cosine of two normalized vectors, summed in sorted-key order so the result is reproducible.
func cosine(a, b map[string]float64) float64 {
	if len(b) < len(a) {
		a, b = b, a
	}
	var s float64
	for _, t := range slices.Sorted(maps.Keys(a)) {
		s += a[t] * b[t]
	}
	return s
}

// shared returns the terms contributing most to the similarity, for the evidence line.
func shared(a, b map[string]float64, k int) []string {
	var ts []string
	for t := range a {
		if _, ok := b[t]; ok {
			ts = append(ts, t)
		}
	}
	slices.SortFunc(ts, func(x, y string) int { return cmp.Or(cmp.Compare(a[y]*b[y], a[x]*b[x]), cmp.Compare(x, y)) })
	return ts[:min(len(ts), k)]
}

func label(m market.Market) string {
	if m.Outcome != "" && !strings.Contains(m.Question, m.Outcome) {
		return m.Question + " [" + m.Outcome + "]"
	}
	return m.Question
}

func intersect(a, b []string) []string {
	var out []string
	for _, x := range a {
		if slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

// yearGap is the smallest distance between any year named by a and any named by b.
func yearGap(a, b []int) int {
	best := math.MaxInt
	for _, x := range a {
		for _, y := range b {
			best = min(best, max(x-y, y-x))
		}
	}
	return best
}

func intersectInt(a, b []int) []int {
	var out []int
	for _, x := range a {
		if slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func round(f float64) float64 { return math.Round(f*1000) / 1000 }
