package route

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

var now = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

var quadratic = market.FeeCurve{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: market.Cent}

func quote(venue string, fee market.FeeCurve, asks ...market.Level) Quote {
	return Quote{
		Market: market.Market{Venue: venue, ID: venue + "-m", Fee: fee},
		Book:   market.Book{Asks: asks, Bids: []market.Level{lv("0.10", 10)}, AsOf: now.Add(-time.Second)},
	}
}

// usd parses a dollar string; tests read better in dollars than in micro-dollars.
func usd(s string) market.Amount {
	a, err := market.ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

func lv(price string, qty int64) market.Level { return market.Level{Price: usd(price), Qty: qty} }

var policy = Policy{MaxBookAge: 30 * time.Second}

func TestPicksCheapestVenue(t *testing.T) {
	d := Route(Order{Side: market.Yes, Qty: 100},
		[]Quote{quote("a", market.FeeCurve{}, lv("0.52", 500)), quote("b", market.FeeCurve{}, lv("0.51", 500))}, policy, now)
	assertRouted(t, d, Filled, "b")
	if d.Total != 100*usd("0.51") {
		t.Errorf("total = %s", d.Total)
	}
}

func TestFeesCanFlipTheDecision(t *testing.T) {
	// a: 0.50 + 7% quadratic fee ($1.75 per 100) = 0.5175 all-in; b: 0.51 with no fee wins.
	d := Route(Order{Side: market.Yes, Qty: 100},
		[]Quote{quote("a", quadratic, lv("0.5", 500)), quote("b", market.FeeCurve{}, lv("0.51", 500))}, policy, now)
	assertRouted(t, d, Filled, "b")
	if a := d.Evaluations[0].Alone; a.Fee != usd("1.75") || a.AllIn != usd("0.5175") {
		t.Errorf("venue a alone = %+v", a)
	}
}

func TestExclusions(t *testing.T) {
	stale := quote("stale", market.FeeCurve{}, lv("0.4", 500))
	stale.Book.AsOf = now.Add(-time.Minute)
	down := quote("down", market.FeeCurve{}, lv("0.4", 500))
	down.Unhealthy = "HTTP 503 after 3 attempts"
	closed := quote("closed", market.FeeCurve{}, lv("0.4", 500))
	closed.Market.Untradable = "status closed"
	crossed := quote("crossed", market.FeeCurve{}, lv("0.4", 500))
	crossed.Book.Bids = []market.Level{lv("0.45", 1)}
	empty := quote("empty", market.FeeCurve{})
	empty.Book.Bids = nil
	small := quote("small", market.FeeCurve{}, lv("0.4", 500))
	small.Market.MinQty = 1000
	ok := quote("ok", market.FeeCurve{}, lv("0.6", 500))

	d := Route(Order{Side: market.Yes, Qty: 100}, []Quote{stale, down, closed, crossed, empty, small, ok}, policy, now)
	assertRouted(t, d, Filled, "ok")
	want := map[string]string{
		"stale": "stale book", "down": "venue unhealthy", "closed": "not tradable", "crossed": "crossed book",
		"empty": "empty book", "small": "below venue minimum",
	}
	for _, e := range d.Evaluations {
		if w, bad := want[e.Venue]; bad && (e.Eligible || !strings.Contains(e.Excluded, w)) {
			t.Errorf("%s: eligible=%v excluded=%q, want %q", e.Venue, e.Eligible, e.Excluded, w)
		}
	}
}

func TestPrefersFullFillThenSplits(t *testing.T) {
	quotes := []Quote{quote("a", market.FeeCurve{}, lv("0.5", 100)), quote("b", market.FeeCurve{}, lv("0.52", 1000))}
	order := Order{Side: market.Yes, Qty: 300}

	single := Route(order, quotes, policy, now)
	assertRouted(t, single, Filled, "b") // a is cheaper but can only fill 100 of 300

	p := policy
	p.Split = true
	split := Route(order, quotes, p, now)
	if split.Status != Filled || len(split.Allocations) != 2 {
		t.Fatalf("split = %+v", split.Allocations)
	}
	if split.Allocations[0].Qty != 100 || split.Allocations[1].Qty != 200 || split.Total != 100*usd("0.50")+200*usd("0.52") {
		t.Errorf("split allocations = %+v", split.Allocations)
	}
	if !split.mentions("split beats best single venue") {
		t.Errorf("explanation should justify the split: %v", split.Explanation)
	}
}

func TestSplitDropsVenueBelowMinimum(t *testing.T) {
	a := quote("a", market.FeeCurve{}, lv("0.5", 3)) // cheapest, but only 3 contracts deep
	a.Market.MinQty = 1
	b := quote("b", market.FeeCurve{}, lv("0.51", 1000))
	b.Market.MinQty = 50 // the remainder after a's 3 must still be >= 50
	d := Route(Order{Side: market.Yes, Qty: 52}, []Quote{a, b}, Policy{MaxBookAge: time.Minute, Split: true}, now)
	if d.Status != Filled || d.Filled != 52 {
		t.Fatalf("decision = %+v", d)
	}
	for _, f := range d.Allocations {
		if f.Venue == "b" && f.Qty < 50 {
			t.Errorf("b got %d, below its minimum", f.Qty)
		}
	}
}

func TestLimitPriceAndPartialFill(t *testing.T) {
	d := Route(Order{Side: market.Yes, Qty: 300, Limit: usd("0.51")},
		[]Quote{quote("a", market.FeeCurve{}, lv("0.5", 100), lv("0.52", 500))}, policy, now)
	assertRouted(t, d, Partial, "a")
	if d.Filled != 100 {
		t.Errorf("filled = %d", d.Filled)
	}
	d = Route(Order{Side: market.Yes, Qty: 10, Limit: usd("0.40")}, []Quote{quote("a", market.FeeCurve{}, lv("0.5", 100))}, policy, now)
	if d.Status != Rejected || !strings.Contains(d.Evaluations[0].Excluded, "above limit") {
		t.Errorf("want rejection above limit, got %s %q", d.Status, d.Evaluations[0].Excluded)
	}
}

func TestBuyNoLiftsComplementedYesBids(t *testing.T) {
	q := quote("a", market.FeeCurve{}, lv("0.6", 10))
	q.Book.Bids = []market.Level{lv("0.55", 40), lv("0.5", 40)} // NO asks: 0.45 x40, 0.50 x40
	d := Route(Order{Side: market.No, Qty: 50}, []Quote{q}, policy, now)
	assertRouted(t, d, Filled, "a")
	if d.Total != 40*usd("0.45")+10*usd("0.50") || d.Allocations[0].Worst != usd("0.50") {
		t.Errorf("allocation = %+v", d.Allocations[0])
	}
}

func TestTieBreakIsByVenueID(t *testing.T) {
	d := Route(Order{Side: market.Yes, Qty: 10},
		[]Quote{quote("zeta", market.FeeCurve{}, lv("0.5", 100)), quote("alpha", market.FeeCurve{}, lv("0.5", 100))}, policy, now)
	assertRouted(t, d, Filled, "alpha")
}

func TestRejectsInvalidOrdersAndNoVenues(t *testing.T) {
	q := []Quote{quote("a", market.FeeCurve{}, lv("0.5", 100))}
	for _, o := range []Order{{Side: "maybe", Qty: 1}, {Side: market.Yes}, {Side: market.Yes, Qty: MaxOrderQty + 1}, {Side: market.Yes, Qty: 1, Limit: market.Dollar}} {
		if d := Route(o, q, policy, now); d.Status != Rejected {
			t.Errorf("%+v should be rejected", o)
		}
	}
	if d := Route(Order{Side: market.Yes, Qty: 1}, nil, policy, now); d.Status != Rejected || !d.mentions("no eligible venue") {
		t.Errorf("no quotes: %+v", d)
	}
}

// Determinism: input order and repetition must not change a single byte of the decision.
func TestDeterministic(t *testing.T) {
	p := Policy{MaxBookAge: time.Minute, Split: true}
	qs := []Quote{
		quote("c", quadratic, lv("0.5", 50), lv("0.53", 500)),
		quote("a", market.FeeCurve{RatePPM: 40_000, PExp: 1, QExp: 1, RoundTo: 10}, lv("0.505", 80), lv("0.54", 500)),
		quote("b", market.FeeCurve{}, lv("0.515", 60)),
	}
	o := Order{Side: market.Yes, Qty: 400}
	want := mustJSON(t, Route(o, qs, p, now))
	perms := [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	for i := 0; i < 20; i++ {
		perm := perms[i%len(perms)]
		got := mustJSON(t, Route(o, []Quote{qs[perm[0]], qs[perm[1]], qs[perm[2]]}, p, now))
		if got != want {
			t.Fatalf("permutation %v changed the decision:\n%s\nvs\n%s", perm, got, want)
		}
	}
}

// Architecture fitness test: the routing layer may import only the canonical model and the standard
// library, and its source must not mention any venue.
func TestRouteIsVenueAgnostic(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, venue := range []string{"kalshi", "polymarket", "poly", "clob", "gamma"} {
			if strings.Contains(strings.ToLower(string(src)), venue) {
				t.Errorf("%s mentions %q", f, venue)
			}
		}
		ast, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range ast.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, ".") && path != "github.com/hisefath/equinox/internal/market" {
				t.Errorf("%s imports %s; route may depend only on internal/market", f, path)
			}
		}
	}
}

func assertRouted(t *testing.T, d Decision, status, venue string) {
	t.Helper()
	if d.Status != status || len(d.Allocations) != 1 || d.Allocations[0].Venue != venue {
		t.Fatalf("want %s to %s, got %s %+v\n%s", status, venue, d.Status, d.Allocations, strings.Join(d.Explanation, "\n"))
	}
}

func (d Decision) mentions(s string) bool {
	return strings.Contains(strings.Join(d.Explanation, "\n"), s)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
