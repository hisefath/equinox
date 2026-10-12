package route

import (
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

var now = time.Date(2026, 10, 11, 16, 0, 0, 0, time.UTC)

func amt(s string) market.Amount {
	a, err := market.ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

func book(key market.Key, bids, asks []market.Level, age time.Duration) market.Book {
	b, err := market.NewBook(key, bids, asks, now.Add(-age))
	if err != nil {
		panic(err)
	}
	return b
}

// quote is a healthy, tradable, eligible venue asking price for qty YES contracts with a zero fee.
func quote(venue string, price string, qty int64) Quote {
	key := market.Key(venue + ":m1")
	return Quote{
		Venue: venue, Key: key, Healthy: true, Tradable: true, Eligible: true, MinQty: 1,
		Fee:  market.FeeTerms{RoundTo: market.Cent},
		Book: book(key, []market.Level{{Price: amt("0.40"), Qty: 500}}, []market.Level{{Price: amt(price), Qty: qty}}, 10*time.Second),
	}
}

func order(qty int64) Order {
	return Order{PairID: "p1", Side: market.Yes, Quantity: qty, CustomerRegion: "non-US"}
}

var policy = Policy{MaxAge: 90 * time.Second}

func result(t *testing.T, d Decision, venue string) VenueResult {
	t.Helper()
	for _, v := range d.Venues {
		if v.Venue == venue {
			return v
		}
	}
	t.Fatalf("no result for %s in %+v", venue, d.Venues)
	return VenueResult{}
}

func TestFeeFlipsDecision(t *testing.T) {
	a, b := quote("v1", "0.58", 2000), quote("v2", "0.585", 2000)
	d := Route(order(100), []Quote{a, b}, policy, now)
	if d.Venue != "v1" || d.DecidingRule != RuleLowestAllIn {
		t.Fatalf("without fees want v1 lowest_all_in, got %s %s", d.Venue, d.DecidingRule)
	}
	a.Fee = market.FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: market.Cent} // 0.07 curve on v1
	d = Route(order(100), []Quote{a, b}, policy, now)
	if d.Venue != "v2" || d.DecidingRule != RuleLowestAllIn {
		t.Fatalf("with v1 fee want v2, got %s %s", d.Venue, d.DecidingRule)
	}
	if d.Outcome != Filled || d.FilledQty != 100 || d.AllIn != amt("58.5") || d.Fees != 0 || d.AvgPrice == nil || *d.AvgPrice != amt("0.585") {
		t.Fatalf("bad fill %+v", d)
	}
	if d.Saving == nil || *d.Saving != amt("58")+a.Fee.TakerFee(100, amt("0.58"))-amt("58.5") {
		t.Fatalf("bad saving %v", d.Saving)
	}
	if r := result(t, d, "v1"); r.Reason != "higher all-in cost" || r.AllIn == nil || *r.AllIn != amt("59.71") {
		t.Fatalf("loser result %+v", r)
	}
}

func TestExclusionReasons(t *testing.T) {
	limit := amt("0.60")
	cases := []struct {
		name   string
		edit   func(*Quote, *Order)
		reason string
	}{
		{"ineligible region", func(q *Quote, o *Order) { q.Eligible, o.CustomerRegion = false, "US" }, "not open to US customers"},
		{"unhealthy", func(q *Quote, _ *Order) { q.Healthy, q.HealthReason = false, "timed out" }, "venue unhealthy: timed out"},
		{"untradable", func(q *Quote, _ *Order) { q.Tradable, q.UntradableReason = false, "market paused" }, "market paused"},
		{"stale", func(q *Quote, _ *Order) { q.Book.AsOf = now.Add(-95 * time.Second) }, "data 95 s old"},
		{"crossed", func(q *Quote, _ *Order) {
			q.Book = book(q.Key, []market.Level{{Price: amt("0.62"), Qty: 10}}, []market.Level{{Price: amt("0.60"), Qty: 10}}, time.Second)
		}, "crossed order book"},
		{"empty", func(q *Quote, _ *Order) { q.Book.Asks = nil }, "empty order book"},
		{"undated", func(q *Quote, _ *Order) { q.Book.AsOf = time.Time{} }, "undated order book"},
		{"below min qty", func(q *Quote, _ *Order) { q.MinQty = 500 }, "below minimum size 500"},
		{"above limit", func(_ *Quote, o *Order) { o.Limit = &limit }, "best ask above limit"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bad, good := quote("v1", "0.70", 1000), quote("v2", "0.59", 1000)
			o := order(100)
			c.edit(&bad, &o)
			d := Route(o, []Quote{bad, good}, policy, now)
			r := result(t, d, "v1")
			if r.Eligible || r.Reason != c.reason || r.Filled != 0 {
				t.Fatalf("v1 result %+v, want reason %q", r, c.reason)
			}
			if d.Venue != "v2" || d.DecidingRule != RuleOnlyEligible || d.Saving != nil {
				t.Fatalf("want v2 only_eligible, got %+v", d)
			}
		})
	}
}

func TestMostFilledBeatsCheaper(t *testing.T) {
	d := Route(order(1000), []Quote{quote("v1", "0.50", 400), quote("v2", "0.60", 1000)}, policy, now)
	if d.Venue != "v2" || d.DecidingRule != RuleMostFilled || d.Saving != nil {
		t.Fatalf("got %+v", d)
	}
	if r := result(t, d, "v1"); r.Reason != "fewer contracts filled" || r.Filled != 400 {
		t.Fatalf("loser %+v", r)
	}
}

func TestTieBreakByVenueThenKey(t *testing.T) {
	d := Route(order(100), []Quote{quote("vb", "0.589", 1000), quote("va", "0.589", 1000)}, policy, now)
	if d.Venue != "va" || d.DecidingRule != RuleTieBreak || d.Saving == nil || *d.Saving != 0 {
		t.Fatalf("got %+v", d)
	}
	if r := result(t, d, "vb"); r.Reason != "tie on all-in cost, broken by fixed order" {
		t.Fatalf("loser %+v", r)
	}
	x, y := quote("v1", "0.589", 1000), quote("v1", "0.589", 1000)
	x.Key, y.Key = "v1:b", "v1:a"
	if d := Route(order(100), []Quote{x, y}, policy, now); d.MarketKey != "v1:a" {
		t.Fatalf("same venue: want key v1:a, got %s", d.MarketKey)
	}
}

func TestPartialFill(t *testing.T) {
	limit := amt("0.60")
	q := quote("v1", "0.58", 300)
	q.Book.Asks = append(q.Book.Asks, market.Level{Price: amt("0.60"), Qty: 100}, market.Level{Price: amt("0.61"), Qty: 900})
	o := order(1000)
	o.Limit = &limit
	d := Route(o, []Quote{q}, policy, now)
	if d.Outcome != Partial || d.FilledQty != 400 || d.UnfilledQty != 600 || d.AllIn != amt("234") || *d.AvgPrice != amt("0.585") {
		t.Fatalf("got %+v", d)
	}
}

func TestNotFilled(t *testing.T) {
	limit := amt("0.60")
	o := order(100)
	o.Limit = &limit
	d := Route(o, []Quote{quote("v1", "0.70", 1000), quote("v2", "0.75", 1000)}, policy, now)
	if d.Outcome != NotFilled || d.Venue != "" || d.FilledQty != 0 || d.UnfilledQty != 100 || d.DecidingRule != "" || d.AvgPrice != nil {
		t.Fatalf("got %+v", d)
	}
	for _, r := range d.Venues {
		if r.Reason != "best ask above limit" {
			t.Fatalf("venue %+v", r)
		}
	}
	if d := Route(o, nil, policy, now); d.Outcome != NotFilled || d.Venues == nil {
		t.Fatalf("no quotes: %+v", d)
	}
}

func TestNoSide(t *testing.T) {
	// NO asks come from YES bids at $1 - p: v1 bids 0.40 (NO at 0.60), v2 bids 0.45 (NO at 0.55).
	a, b := quote("v1", "0.70", 10), quote("v2", "0.70", 10)
	b.Book.Bids = []market.Level{{Price: amt("0.45"), Qty: 500}}
	o := order(100)
	o.Side = market.No
	d := Route(o, []Quote{a, b}, policy, now)
	if d.Venue != "v2" || d.FilledQty != 100 || *d.AvgPrice != amt("0.55") {
		t.Fatalf("got %+v", d)
	}
}

func TestDeterministicUnderShuffle(t *testing.T) {
	qs := []Quote{quote("v1", "0.58", 50), quote("v2", "0.585", 2000), quote("v3", "0.59", 2000)}
	qs[0].Key = "v1:z"
	qs = append(qs, quote("v1", "0.57", 40))
	want := Route(order(100), qs, policy, now)
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 20; i++ {
		s := append([]Quote(nil), qs...)
		r.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
		got := Route(order(100), s, policy, now)
		if got.DecisionID != want.DecisionID || got.Venue != want.Venue || got.MarketKey != want.MarketKey || len(got.Venues) != len(want.Venues) {
			t.Fatalf("shuffle %d: %+v vs %+v", i, got, want)
		}
		for j := range got.Venues {
			if got.Venues[j].MarketKey != want.Venues[j].MarketKey || got.Venues[j].Reason != want.Venues[j].Reason {
				t.Fatalf("shuffle %d venue %d differs", i, j)
			}
		}
	}
	if len(want.DecisionID) != 16 {
		t.Fatalf("decision id %q", want.DecisionID)
	}
	if other := Route(order(101), qs, policy, now); other.DecisionID == want.DecisionID {
		t.Fatal("different order, same decision id")
	}
}

func TestValidate(t *testing.T) {
	if err := order(1).Validate(); err != nil {
		t.Fatal(err)
	}
	over := amt("1.01")
	bad := Order{PairID: "p1", Side: "maybe", Quantity: 0, Limit: &over}
	err := bad.Validate()
	fe, ok := err.(FieldErrors)
	if !ok || len(fe) != 4 {
		t.Fatalf("got %v", err)
	}
	want := map[string]string{"#/quantity": "quantity must be at least 1", "#/side": "side must be yes or no",
		"#/limit": "limit must be between 0 and 1", "#/customer_region": "customer region required"}
	for _, f := range fe {
		if want[f.Pointer] != f.Detail {
			t.Errorf("%s: %q", f.Pointer, f.Detail)
		}
	}
	o := order(1)
	o.CustomerRegion = "EU"
	if o.Validate() == nil {
		t.Fatal("unknown region accepted")
	}
}

func TestRouteImportsOnlyMarket(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range strings.Fields(string(out)) {
		if p == "net" || strings.HasPrefix(p, "net/") {
			t.Errorf("route depends on %s", p)
		}
		first, _, _ := strings.Cut(p, "/")
		if strings.Contains(first, ".") && p != "github.com/hisefath/equinox/internal/market" && p != "github.com/hisefath/equinox/internal/route" {
			t.Errorf("route depends on non-standard %s", p)
		}
	}
}

func TestRouteNamesNoVenue(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range []string{"kal" + "shi", "poly" + "market"} {
			if strings.Contains(strings.ToLower(string(b)), v) {
				t.Errorf("%s names venue %s", f, v)
			}
		}
	}
}

// D26: under the higher-payout goal, most paid back per dollar wins even when another venue fills more; the
// default goal is unchanged and keeps its decision id.
func TestHigherPayoutGoal(t *testing.T) {
	cheap, dear := quote("ashby", "0.45", 60), quote("birch", "0.70", 100)
	o := order(100)
	best := Route(o, []Quote{cheap, dear}, policy, now)
	if best.Venue != "birch" || best.DecidingRule != RuleMostFilled || best.Payout != nil {
		t.Fatalf("best price: %s %s %v", best.Venue, best.DecidingRule, best.Payout)
	}
	o.Goal = GoalHigherPayout
	d := Route(o, []Quote{dear, cheap}, policy, now)
	if d.Venue != "ashby" || d.DecidingRule != RuleHighestPayout || d.Payout == nil || *d.Payout != 60*market.Dollar {
		t.Fatalf("higher payout: %s %s %v", d.Venue, d.DecidingRule, d.Payout)
	}
	if r := result(t, d, "birch"); r.Reason != "lower payout per dollar" {
		t.Fatalf("loser reason %q", r.Reason)
	}
	if d.DecisionID == best.DecisionID {
		t.Fatal("the goal must be part of the decision id")
	}
	o.Goal = "max_risk"
	if err := o.Validate(); err == nil {
		t.Fatal("unknown goal accepted")
	}
}
