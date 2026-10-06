package market

import (
	"testing"
	"time"
)

func usd(s string) Amount {
	a, err := ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestParseAmount(t *testing.T) {
	for in, want := range map[string]Amount{
		"0.5230": 523_000, "0.022": 22_000, "1": Dollar, "1.0000": Dollar, ".5": 500_000, "0.000001": 1,
		"0.12345600": 123_456, "-0.25": -250_000,
	} {
		got, err := ParseAmount(in)
		if err != nil || got != want {
			t.Errorf("ParseAmount(%q) = %v, %v; want %v", in, int64(got), err, int64(want))
		}
	}
	for _, bad := range []string{"", "abc", "0.1234567", "1.2.3"} {
		if _, err := ParseAmount(bad); err == nil {
			t.Errorf("ParseAmount(%q) should fail", bad)
		}
	}
	for a, want := range map[Amount]string{523_000: "0.523", 17_500: "0.0175", Dollar: "1.00", 1_716_000: "1.716", -250_000: "-0.25"} {
		if got := a.String(); got != want {
			t.Errorf("String(%d) = %q, want %q", int64(a), got, want)
		}
	}
}

func TestParseQtyFloors(t *testing.T) {
	for in, want := range map[string]int64{"595.01": 595, "5.82": 5, "0.99": 0, "100": 100} {
		if got, err := ParseQty(in); err != nil || got != want {
			t.Errorf("ParseQty(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := ParseQty("-1"); err == nil {
		t.Error("negative qty should fail")
	}
}

func TestFeeCurve(t *testing.T) {
	kalshi := FeeCurve{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: Cent}
	cases := []struct {
		name     string
		curve    FeeCurve
		p        string
		qty      int64
		raw, due string // before and after per-order rounding
	}{
		// Kalshi's published example: 100 contracts at 50¢ -> 0.07*100*0.5*0.5 = $1.75
		{"kalshi 100@0.50", kalshi, "0.50", 100, "1.75", "1.75"},
		// Kalshi docs' fee-rounding example: one $0.055 contract -> $0.00363825 -> ceil to micro, then 1¢
		{"kalshi docs example", kalshi, "0.055", 1, "0.003639", "0.01"},
		// 100 @ 0.43 -> 1.7157 -> $1.72 (research worked example)
		{"kalshi 100@0.43", kalshi, "0.43", 100, "1.7157", "1.72"},
		{"zero rate", FeeCurve{}, "0.50", 100, "0", "0"},
		{"price at bound", kalshi, "1", 100, "0", "0"},
		// Polymarket docs table: politics rate 0.04, 100 shares at 0.50 -> $1.00; at 0.30 -> $0.84
		{"polymarket 100@0.50", FeeCurve{RatePPM: 40_000, PExp: 1, QExp: 1, RoundTo: 10}, "0.50", 100, "1", "1"},
		{"polymarket 100@0.30", FeeCurve{RatePPM: 40_000, PExp: 1, QExp: 1, RoundTo: 10}, "0.30", 100, "0.84", "0.84"},
		// Polymarket research example: 100 NO at 0.976, rate 0.04 -> 0.093696 -> $0.09370 at 5 dp
		{"polymarket 5dp rounding", FeeCurve{RatePPM: 40_000, PExp: 1, QExp: 1, RoundTo: 10}, "0.976", 100, "0.093696", "0.0937"},
		// legacy crypto_15_min: rate 0.25, exponent 2 at 0.5 -> 100*0.25*0.0625 = $1.5625
		{"polymarket exponent 2", FeeCurve{RatePPM: 250_000, PExp: 2, QExp: 2, RoundTo: 10}, "0.50", 100, "1.5625", "1.5625"},
		// flat 1% of $1 per contract (a+b = 0) -> 100 * 0.01 = $1.00
		{"flat per-contract", FeeCurve{RatePPM: 10_000}, "0.50", 100, "1", "1"},
	}
	for _, c := range cases {
		raw := c.curve.Fee(usd(c.p), c.qty)
		if raw != usd(c.raw) || c.curve.Round(raw) != usd(c.due) {
			t.Errorf("%s: Fee = %s, rounded %s; want %s, %s", c.name, raw, c.curve.Round(raw), c.raw, c.due)
		}
	}
}

func TestBookNormalizeAndAsksFor(t *testing.T) {
	l := func(p string, q int64) Level { return Level{usd(p), q} }
	b := Book{
		Bids: []Level{l("0.40", 10), l("0.45", 5), l("0.45", 5), {0, 9}, l("0.30", 0)},
		Asks: []Level{l("0.60", 1), l("0.55", 2), {Dollar, 3}},
		AsOf: time.Unix(1, 0),
	}.Normalize()
	if len(b.Bids) != 2 || b.Bids[0] != l("0.45", 10) || b.Bids[1] != l("0.40", 10) {
		t.Fatalf("bids = %v", b.Bids)
	}
	if len(b.Asks) != 2 || b.Asks[0] != l("0.55", 2) {
		t.Fatalf("asks = %v", b.Asks)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	// Buying NO lifts complemented YES bids: best YES bid 0.45 -> NO ask 0.55.
	if no := b.AsksFor(No); no[0] != l("0.55", 10) || no[1] != l("0.60", 10) {
		t.Fatalf("no asks = %v", no)
	}
	crossed := Book{Bids: []Level{l("0.60", 1)}, Asks: []Level{l("0.50", 1)}, AsOf: time.Unix(1, 0)}
	if crossed.Validate() == nil {
		t.Fatal("crossed book should be invalid")
	}
}
