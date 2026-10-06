package match

import "testing"

func TestApplyReviews(t *testing.T) {
	pairs := []Pair{{ID: "a~b", Tier: Equivalent}, {ID: "c~d", Tier: Review}, {ID: "e~f", Tier: Equivalent}}
	got := ApplyReviews(pairs, []PairReview{
		{Pair: "a~b", Verdict: "not_equivalent", Source: "analyst", Note: "different deadline"},
		{Pair: "c~d", Verdict: Equivalent, Source: "analyst"},
	})
	if got[0].Tier != Rejected || got[1].Tier != Equivalent || got[1].Reviewed != "analyst" || got[2].Reviewed != "" {
		t.Fatalf("got %+v", got)
	}
	if pairs[0].Tier != Equivalent {
		t.Fatal("input mutated")
	}
	if rs, err := LoadReviews("testdata/does-not-exist.json"); err != nil || rs != nil {
		t.Fatalf("missing file should mean no reviews: %v %v", rs, err)
	}
}
