package venue

import (
	"slices"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

func TestCountKeepsTheSumInvariant(t *testing.T) {
	open := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	var s CatalogStats
	kept := []bool{
		s.Count(market.Market{Volume24h: 5, OpensAt: open, ClosesAt: open.Add(2 * time.Hour)}),
		s.Count(market.Market{Volume24h: 0, OpensAt: open, ClosesAt: open.Add(2 * time.Hour)}),
		s.Count(market.Market{Volume24h: 5, OpensAt: open, ClosesAt: open.Add(30 * time.Minute)}),
	}
	s.Unread()
	if !slices.Equal(kept, []bool{true, false, false}) {
		t.Fatalf("kept %v", kept)
	}
	want := CatalogStats{Listed: 4, Considered: 1, Inactive: 1, TooShort: 1, Unreadable: 1}
	if s != want || s.Listed != s.Considered+s.Inactive+s.TooShort+s.Unreadable {
		t.Fatalf("got %+v want %+v", s, want)
	}
}

func TestEdgeRules(t *testing.T) {
	got := EdgeRules("If the game is postponed... canceled entirely... resolve 50-50. Overtime is included.")
	if want := []string{"cancellation", "overtime", "postponement", "tie 50/50"}; !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := EdgeRules("plain rules"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestContracts(t *testing.T) {
	for in, want := range map[string]int64{"0.00": 0, "0.01": 1, "12": 12, "2323427.756063003": 2323428} {
		if got, err := Contracts(in); err != nil || got != want {
			t.Errorf("Contracts(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "x", "-1", "NaN"} {
		if _, err := Contracts(bad); err == nil {
			t.Errorf("Contracts(%q) accepted", bad)
		}
	}
}

func TestTopic(t *testing.T) {
	for in, want := range map[string]string{"Elections": "elections", "Politics": "elections", "Economics": "economics",
		"Financials": "economics", "Sports": "sports", "Crypto": "crypto", "Climate and Weather": "other", "": "other"} {
		if got := Topic(in); got != want {
			t.Errorf("Topic(%q) = %q want %q", in, got, want)
		}
	}
}
