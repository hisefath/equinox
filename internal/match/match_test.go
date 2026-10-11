package match

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/refdata"
)

var dec = time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)

func mk(t *testing.T, key, q string, tm market.Terms, mods ...func(*market.Market)) market.Market {
	t.Helper()
	m := market.Market{Key: market.Key(key), Venue: key[:1], Question: q, Terms: tm, ClosesAt: dec,
		OpensAt: dec.Add(-30 * 24 * time.Hour), Volume24h: 10, Tradable: true, RulesText: "rules of " + key}
	for _, f := range mods {
		f(&m)
	}
	m, err := market.NewMarket(m)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func fedTerms() market.Terms {
	return market.Terms{Subject: "fed funds rate", Predicate: "cut", Comparator: "=", Threshold: "25",
		Source: "fomc statement", WindowEnd: "2026-12-10"}
}

func btcTerms(stat string) market.Terms {
	return market.Terms{Subject: "btc", Predicate: "above", Comparator: ">", Threshold: "120000",
		Source: "cf benchmarks", Statistic: stat, WindowEnd: "2026-12-10"}
}

func iowaTerms() market.Terms {
	return market.Terms{Subject: "lee", Predicate: "win iowa senate", Source: "certified result", WindowEnd: "2026-11-03"}
}

func TestCompareAE002Equivalent(t *testing.T) {
	a := mk(t, "a:fed", "Will the Fed cut rates by 25 bps at its December 2026 meeting?", fedTerms())
	b := mk(t, "b:fed", "Fed decreases rates by 25 bps after December 2026 meeting?", fedTerms())
	v := Compare(a, b)
	if v.Verdict != Equivalent || v.DifferingPart != "" || len(v.Caveats) != 0 {
		t.Fatalf("got %+v", v)
	}
	if len(v.Evidence) != len(Parts) {
		t.Errorf("want one evidence line per part, got %v", v.Evidence)
	}
}

func TestCompareAE003TimingCaveat(t *testing.T) {
	a := mk(t, "a:iowa", "Candidate Lee wins the 2026 Iowa Senate race", iowaTerms(),
		func(m *market.Market) { m.ResolvesAt = time.Date(2027, 1, 3, 0, 0, 0, 0, time.UTC) })
	b := mk(t, "b:iowa", "Lee wins Iowa Senate 2026", iowaTerms(),
		func(m *market.Market) { m.ResolvesAt = time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC) })
	v := Compare(a, b)
	if v.Verdict != Equivalent || !reflect.DeepEqual(v.Caveats, []string{"pays out about 60 days apart"}) {
		t.Fatalf("got %+v", v)
	}
}

func TestCompareAE004StatisticDiffers(t *testing.T) {
	a := mk(t, "a:btc", "BTC above $120,000 at 5 PM ET", btcTerms("60s average"))
	b := mk(t, "b:btc", "BTC above $120,000 at 5 PM ET", btcTerms("1m close"))
	v := Compare(a, b)
	if v.Verdict != Related || v.DifferingPart != "statistic" {
		t.Fatalf("got %+v", v)
	}
}

func TestCompareAE005ThresholdDiffers(t *testing.T) {
	more := fedTerms()
	more.Comparator = ">"
	a := mk(t, "a:fed", "Fed cuts by more than 25 bps in December", more)
	b := mk(t, "b:fed", "Fed cuts by 25 bps in December", fedTerms())
	v := Compare(a, b)
	if v.Verdict != Related || v.DifferingPart != "threshold" {
		t.Fatalf("got %+v", v)
	}
	ps := New("r1").Match([]market.Market{a}, []market.Market{b}, refdata.Set{})
	if len(ps.NearMisses) != 1 || ps.NearMisses[0].Reason != "threshold differs" {
		t.Fatalf("near misses %+v", ps.NearMisses)
	}
}

func TestCompareUnknownIsUndecided(t *testing.T) {
	for _, part := range []func(*market.Terms){
		func(x *market.Terms) { x.Subject = "" },
		func(x *market.Terms) { x.Predicate = market.Unknown },
		func(x *market.Terms) { x.Source = "" },
		func(x *market.Terms) { x.Statistic = market.Unknown },
		func(x *market.Terms) { x.WindowEnd = "" },
		func(x *market.Terms) { x.Comparator = "" },
		func(x *market.Terms) { x.Threshold = market.Unknown },
		func(x *market.Terms) { x.EdgeRules = []string{market.Unknown} },
	} {
		tm := fedTerms()
		part(&tm)
		a := mk(t, "a:fed", "Fed cut December", tm)
		b := mk(t, "b:fed", "Fed cut December", fedTerms())
		if v := Compare(a, b); v.Verdict != Undecided {
			t.Errorf("terms %+v: got %+v", tm, v)
		}
	}
}

func TestCompareEdgeRuleOrderIrrelevant(t *testing.T) {
	x, y := fedTerms(), fedTerms()
	x.EdgeRules = []string{"Tie 50/50", "death"}
	y.EdgeRules = []string{"death", "tie 50/50"}
	if v := Compare(mk(t, "a:1", "Fed cut", x), mk(t, "b:1", "Fed cut", y)); v.Verdict != Equivalent {
		t.Fatalf("got %+v", v)
	}
	y.EdgeRules = []string{"death"}
	if v := Compare(mk(t, "a:1", "Fed cut", x), mk(t, "b:1", "Fed cut", y)); v.DifferingPart != "edge rule" {
		t.Fatalf("got %+v", v)
	}
}

func TestMatchCloseWindow(t *testing.T) {
	a := mk(t, "a:fed", "Fed cut 25 bps December", fedTerms())
	b := mk(t, "b:fed", "Fed cut 25 bps December", fedTerms(), func(m *market.Market) { m.ClosesAt = dec.Add(8 * 24 * time.Hour) })
	late := a
	late.ClosesAt = dec.Add(16 * 24 * time.Hour) // the lower key closes later this time
	for _, xs := range [][2]market.Market{{a, b}, {late, mk(t, "b:fed", "Fed cut 25 bps December", fedTerms())}} {
		if ps := New("r1").Match(xs[:1], xs[1:], refdata.Set{}); len(ps.Pairs) != 0 {
			t.Fatalf("pair outside the 7-day window: %+v", ps.Pairs)
		}
	}
	b.ClosesAt = dec.Add(-6 * 24 * time.Hour)
	if ps := New("r1").Match([]market.Market{a}, []market.Market{b}, refdata.Set{}); len(ps.Pairs) != 1 {
		t.Fatalf("pair inside the window missing: %+v", ps.Pairs)
	}
}

func TestMatchOneToOne(t *testing.T) {
	a := mk(t, "a:fed", "Fed cut 25 bps December", fedTerms())
	b1 := mk(t, "b:fed1", "Fed cut 25 bps December", fedTerms())
	b2 := mk(t, "b:fed2", "Fed cut 25 bps December meeting", fedTerms())
	ps := New("r1").Match([]market.Market{a}, []market.Market{b1, b2}, refdata.Set{})
	var eq []Pair
	for _, p := range ps.Pairs {
		if p.Verdict == Equivalent {
			eq = append(eq, p)
		}
	}
	if len(eq) != 1 || eq[0].BKey != "b:fed1" {
		t.Fatalf("want exactly a:fed-b:fed1 (higher score), got %+v", eq)
	}
	if len(ps.NearMisses) != 1 || ps.NearMisses[0].Pair.BKey != "b:fed2" {
		t.Fatalf("loser should be a near miss: %+v", ps.NearMisses)
	}
	if _, err := ps.Routable(ps.NearMisses[0].Pair.PairID, nil); !errors.Is(err, ErrUnknownPair) {
		t.Errorf("one-to-one loser must not be routable: %v", err)
	}
}

func TestMatchTieBreaksByPairID(t *testing.T) {
	a := mk(t, "a:fed", "Fed cut 25 bps December", fedTerms())
	var bs []market.Market
	for _, k := range []string{"b:x", "b:y", "b:z"} {
		bs = append(bs, mk(t, k, "Fed cut 25 bps December", fedTerms()))
	}
	ps := New("r1").Match([]market.Market{a}, bs, refdata.Set{})
	want := pairID("a:fed", "b:x")
	for _, k := range []market.Key{"b:y", "b:z"} {
		if id := pairID("a:fed", k); id < want {
			want = id
		}
	}
	if eq := equivalents(ps); len(eq) != 1 || eq[0].PairID != want {
		t.Fatalf("want lowest pair id %s, got %+v", want, eq)
	}
}

func equivalents(ps PairSet) []Pair {
	var out []Pair
	for _, p := range ps.Pairs {
		if p.Verdict == Equivalent {
			out = append(out, p)
		}
	}
	return out
}

func corpus(t *testing.T) ([]market.Market, []market.Market) {
	more := fedTerms()
	more.Comparator = ">"
	unk := btcTerms("")
	unk.Source = ""
	a := []market.Market{
		mk(t, "a:fed", "Will the Fed cut rates by 25 bps in December?", fedTerms()),
		mk(t, "a:fedmore", "Fed cuts by more than 25 bps in December", more),
		mk(t, "a:btc", "BTC above $120,000 at 5 PM ET", btcTerms("60s average")),
		mk(t, "a:iowa", "Lee wins Iowa Senate", iowaTerms()),
		mk(t, "a:eth", "ETH above 5000 by December", unk),
	}
	b := []market.Market{
		mk(t, "b:fed", "Fed decreases rates by 25 bps in December", fedTerms()),
		mk(t, "b:fed2", "Fed lowers rates 25 bps December", fedTerms()),
		mk(t, "b:btc", "BTC above $120,000 at 5 PM ET", btcTerms("1m close")),
		mk(t, "b:iowa", "Iowa Senate: Lee wins", iowaTerms()),
		mk(t, "b:eth", "ETH above 5000 December", btcTerms("")),
	}
	return a, b
}

func TestMatchDeterministicUnderInputOrder(t *testing.T) {
	a, b := corpus(t)
	want := New("r1").Match(a, b, refdata.Set{})
	if len(want.Pairs) == 0 {
		t.Fatal("corpus produced no pairs")
	}
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 20; i++ {
		r.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
		r.Shuffle(len(b), func(i, j int) { b[i], b[j] = b[j], b[i] })
		if got := New("r1").Match(a, b, refdata.Set{}); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d differs:\n%+v\n%+v", i, got, want)
		}
		if got := New("r1").Match(b, a, refdata.Set{}); !reflect.DeepEqual(got, want) {
			t.Fatalf("swapped venues differ on run %d", i)
		}
	}
	for _, p := range want.Pairs {
		if p.AKey > p.BKey || p.RulesVersion != "r1" {
			t.Errorf("bad pair %+v", p)
		}
	}
}

func findPair(t *testing.T, ps PairSet, a, b market.Key) Pair {
	t.Helper()
	for _, p := range ps.Pairs {
		if p.AKey == a && p.BKey == b {
			return p
		}
	}
	t.Fatalf("no pair %s %s in %+v", a, b, ps.Pairs)
	return Pair{}
}

func TestOverrideWinsOverAIVerdict(t *testing.T) {
	a, b := corpus(t)
	base := New("r1").Match(a, b, refdata.Set{})
	eth := findPair(t, base, "a:eth", "b:eth")
	if eth.Verdict != Undecided {
		t.Fatalf("eth should be undecided by rules: %+v", eth)
	}
	k := refdata.PK{PairID: eth.PairID, FingerprintA: eth.FingerprintA, FingerprintB: eth.FingerprintB}
	ref := refdata.Set{Verdicts: map[refdata.PK]refdata.Verdict{k: {Verdict: "equivalent"}}}
	if p := findPair(t, New("r1").Match(a, b, ref), "a:eth", "b:eth"); p.Verdict != Equivalent || p.DecidedBy != "ai" {
		t.Fatalf("AI verdict not applied: %+v", p)
	}
	ref.Overrides = map[refdata.PK]refdata.Override{k: {Verdict: "not_equivalent"}}
	p := findPair(t, New("r1").Match(a, b, ref), "a:eth", "b:eth")
	if p.Verdict != Related || p.DecidedBy != "human" || p.DifferingPart != "human review" {
		t.Fatalf("override did not win: %+v", p)
	}
}

func TestAIVerdictOnlyForUndecided(t *testing.T) {
	a, b := corpus(t)
	btc := findPair(t, New("r1").Match(a, b, refdata.Set{}), "a:btc", "b:btc")
	k := refdata.PK{PairID: btc.PairID, FingerprintA: btc.FingerprintA, FingerprintB: btc.FingerprintB}
	ref := refdata.Set{Verdicts: map[refdata.PK]refdata.Verdict{k: {Verdict: "equivalent"}}}
	if p := findPair(t, New("r1").Match(a, b, ref), "a:btc", "b:btc"); p.Verdict != Related || p.DecidedBy != "rules" {
		t.Fatalf("AI verdict overrode a rules verdict: %+v", p)
	}
}

func TestStaleFingerprintIgnored(t *testing.T) {
	a, b := corpus(t)
	eth := findPair(t, New("r1").Match(a, b, refdata.Set{}), "a:eth", "b:eth")
	k := refdata.PK{PairID: eth.PairID, FingerprintA: "old", FingerprintB: eth.FingerprintB}
	ref := refdata.Set{
		Verdicts:  map[refdata.PK]refdata.Verdict{k: {Verdict: "equivalent"}},
		Overrides: map[refdata.PK]refdata.Override{k: {Verdict: "equivalent"}},
	}
	if p := findPair(t, New("r1").Match(a, b, ref), "a:eth", "b:eth"); p.Verdict != Undecided || p.DecidedBy != "rules" {
		t.Fatalf("stale verdict applied: %+v", p)
	}
}

func TestRoutable(t *testing.T) {
	a, b := corpus(t)
	ps := New("r1").Match(a, b, refdata.Set{})
	cur := map[market.Key]string{}
	for _, m := range append(a, b...) {
		cur[m.Key] = m.RulesFingerprint
	}
	iowa := findPair(t, ps, "a:iowa", "b:iowa")
	if p, err := ps.Routable(iowa.PairID, cur); err != nil || p.PairID != iowa.PairID {
		t.Fatalf("iowa should route: %v", err)
	}
	btc := findPair(t, ps, "a:btc", "b:btc")
	_, err := ps.Routable(btc.PairID, cur)
	if !errors.Is(err, ErrNotRoutable) || err.Error() != "pair is not routable: statistic differs" {
		t.Errorf("btc: %v", err)
	}
	for _, k := range []market.Key{"a:iowa", "b:iowa"} {
		old := cur[k]
		cur[k] = "rewritten"
		if _, err := ps.Routable(iowa.PairID, cur); !errors.Is(err, ErrRulesChanged) {
			t.Errorf("rules change on %s: %v", k, err)
		}
		cur[k] = old
	}
	if _, err := ps.Routable("nope", cur); !errors.Is(err, ErrUnknownPair) {
		t.Errorf("unknown: %v", err)
	}
}
