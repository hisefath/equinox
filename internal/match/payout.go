package match

import (
	"fmt"
	"strconv"

	"github.com/hisefath/equinox/internal/market"
)

// PayoutPair describes a related pair that differs only in its threshold, in the same direction (both "above" or
// both "below"), so one market's YES pays in a subset of the other's situations (D26). Such a pair is never the
// same bet; it may be routed only under the "higher payout" goal, and every decision names the difference.
type PayoutPair struct {
	Harder market.Key // pays YES in fewer situations: the riskier bet, usually cheaper
	Easier market.Key
	A, B   string // what each side's YES needs, e.g. "above 75" and "58 or more"
}

// Payout reports whether p is a payout pair and how its two sides differ. a and b are p's markets.
func Payout(p Pair, a, b market.Market) (PayoutPair, bool) {
	if p.Verdict != Related || p.DifferingPart != "threshold" || p.Score < payoutMinScore {
		return PayoutPair{}, false
	}
	ta, tb := a.Terms, b.Terms
	da, oka := direction(ta.Comparator)
	db, okb := direction(tb.Comparator)
	xa, err1 := strconv.ParseFloat(ta.Threshold, 64)
	xb, err2 := strconv.ParseFloat(tb.Threshold, 64)
	if !oka || !okb || da != db || err1 != nil || err2 != nil {
		return PayoutPair{}, false
	}
	// For "above", a higher bar is harder; at the same number, strict ">" is harder than ">=". Mirror for "below".
	harderA := xa > xb || (xa == xb && ta.Comparator == ">" && tb.Comparator == ">=")
	if da < 0 {
		harderA = xa < xb || (xa == xb && ta.Comparator == "<" && tb.Comparator == "<=")
	}
	if xa == xb && ta.Comparator == tb.Comparator {
		return PayoutPair{}, false // same condition after all: nothing to trade off
	}
	pp := PayoutPair{Harder: b.Key, Easier: a.Key, A: needs(ta), B: needs(tb)}
	if harderA {
		pp.Harder, pp.Easier = a.Key, b.Key
	}
	return pp, true
}

func direction(c string) (int, bool) {
	switch c {
	case ">", ">=":
		return 1, true
	case "<", "<=":
		return -1, true
	}
	return 0, false
}

func needs(t market.Terms) string {
	switch t.Comparator {
	case ">":
		return "above " + t.Threshold
	case ">=":
		return t.Threshold + " or more"
	case "<":
		return "below " + t.Threshold
	case "<=":
		return t.Threshold + " or less"
	}
	return fmt.Sprintf("%s %s", t.Comparator, t.Threshold)
}

// payoutMinScore is the similarity a related pair needs before its threshold difference is trusted as the only
// difference (parts per thousand). Below it, the two questions may be about different things (FR-005 candidates).
var payoutMinScore int64 = 400
