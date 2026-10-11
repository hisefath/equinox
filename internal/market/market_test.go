package market

import (
	"testing"
	"time"
)

func TestParseAmount(t *testing.T) {
	for in, want := range map[string]Amount{"0.585": 585_000, "1": Dollar, ".01": Cent, "0.000001": 1} {
		got, err := ParseAmount(in)
		if err != nil || got != want {
			t.Errorf("ParseAmount(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "-0.1", "0.1234567", "1e3", "abc"} {
		if _, err := ParseAmount(bad); err == nil {
			t.Errorf("ParseAmount(%q) accepted", bad)
		}
	}
	if s := Amount(585_000).String(); s != "0.585" {
		t.Errorf("String = %q", s)
	}
}

// Worked examples from the venues' fee documents (R04): at 50c, 100 contracts cost $1.75 at a 0.07 rate rounded
// to the cent, and $1.00 at a 0.04 rate rounded to 5 decimals.
func TestTakerFeeWorkedExamples(t *testing.T) {
	cases := []struct {
		f     FeeTerms
		qty   int64
		price Amount
		want  Amount
	}{
		{FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: Cent}, 100, 500_000, 1_750_000},
		{FeeTerms{RatePPM: 40_000, PExp: 1, QExp: 1, RoundTo: 10}, 100, 500_000, 1_000_000},
		{FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: Cent}, 1, 10_000, Cent}, // 0.000693 rounds up to 1 cent
		{FeeTerms{RatePPM: 0, RoundTo: Cent}, 100, 500_000, 0},
	}
	for _, c := range cases {
		if got := c.f.TakerFee(c.qty, c.price); got != c.want {
			t.Errorf("TakerFee(%d @ %v, %+v) = %v; want %v", c.qty, c.price, c.f, got, c.want)
		}
	}
}

func TestBookNormalizeAndNoSide(t *testing.T) {
	b, err := NewBook("v:1", []Level{{400_000, 5}, {420_000, 3}, {420_000, 2}}, []Level{{450_000, 7}, {0, 9}}, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if b.Bids[0] != (Level{420_000, 5}) || len(b.Asks) != 1 || b.Crossed {
		t.Fatalf("book = %+v", b)
	}
	if no := b.AsksFor(No); no[0] != (Level{580_000, 5}) {
		t.Fatalf("NO asks = %+v", no)
	}
	if _, err := NewBook("v:1", nil, nil, time.Time{}); err == nil {
		t.Fatal("undated book accepted")
	}
	crossed, _ := NewBook("v:2", []Level{{600_000, 1}}, []Level{{590_000, 1}}, time.Unix(1, 0))
	if !crossed.Crossed {
		t.Fatal("crossed book not flagged")
	}
}

func TestUnknownFeeMakesMarketUntradable(t *testing.T) {
	m, err := NewMarket(Market{Key: "v:1", Venue: "v", Question: "Q?", ClosesAt: time.Unix(10, 0), Tradable: true})
	if err != nil {
		t.Fatal(err)
	}
	if m.Tradable || m.UntradableReason != "fee terms unknown" {
		t.Fatalf("market = %+v", m)
	}
}
