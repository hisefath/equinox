package match

import (
	"testing"

	"github.com/hisefath/equinox/internal/market"
)

// D26: only a same-direction threshold difference is a payout pair, and the harder condition is the riskier side.
func TestPayout(t *testing.T) {
	m := func(key, cmp, thr string) market.Market {
		return market.Market{Key: market.Key(key), Terms: market.Terms{Comparator: cmp, Threshold: thr}}
	}
	rel := Pair{Verdict: Related, DifferingPart: "threshold", Score: 600}
	cases := []struct {
		name   string
		p      Pair
		a, b   market.Market
		ok     bool
		harder market.Key
	}{
		{"above: higher bar is harder", rel, m("a", ">", "85"), m("b", ">=", "50"), true, "a"},
		{"above: strict beats inclusive at the same number", rel, m("a", ">=", "50"), m("b", ">", "50"), true, "b"},
		{"below: lower bar is harder", rel, m("a", "<", "10"), m("b", "<=", "20"), true, "a"},
		{"opposite directions are not comparable", rel, m("a", ">", "85"), m("b", "<", "50"), false, ""},
		{"same condition: nothing to trade off", rel, m("a", ">=", "50"), m("b", ">=", "50"), false, ""},
		{"equivalent pairs are not payout pairs", Pair{Verdict: Equivalent, Score: 900}, m("a", ">", "85"), m("b", ">=", "50"), false, ""},
		{"another part differs", Pair{Verdict: Related, DifferingPart: "source", Score: 900}, m("a", ">", "85"), m("b", ">=", "50"), false, ""},
		{"weak similarity is not trusted", Pair{Verdict: Related, DifferingPart: "threshold", Score: 100}, m("a", ">", "85"), m("b", ">=", "50"), false, ""},
	}
	for _, c := range cases {
		pp, ok := Payout(c.p, c.a, c.b)
		if ok != c.ok || (ok && pp.Harder != c.harder) {
			t.Errorf("%s: ok=%v harder=%s", c.name, ok, pp.Harder)
		}
	}
	if pp, _ := Payout(rel, m("a", ">", "85"), m("b", ">=", "50")); pp.A != "above 85" || pp.B != "50 or more" {
		t.Errorf("descriptions: %q %q", pp.A, pp.B)
	}
}
