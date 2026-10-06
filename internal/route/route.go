// Package route simulates where to send a hypothetical order among venues that list an equivalent market.
//
// Route is a pure function of (order, quotes, policy, now): it performs no I/O, reads no clock, uses
// no randomness and integer money only, so the same inputs always produce the same Decision, down to the
// byte. It imports only the canonical market package and never branches on a venue's identity: fees
// arrive as a market.FeeCurve, liquidity as a market.Book, and venue health as a plain string.
package route

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

// MaxOrderQty bounds a simulated order. It keeps every intermediate product comfortably inside int64.
const MaxOrderQty = 1_000_000

// Order is a hypothetical taker BUY of Qty contracts of Side.
type Order struct {
	Side  market.Side   `json:"side"`
	Qty   int64         `json:"qty"`
	Limit market.Amount `json:"limit,omitempty"` // worst acceptable price per contract before fees; 0 = none
}

// Quote is one venue's market for the equivalent proposition, as last seen by ingestion.
type Quote struct {
	Market market.Market `json:"market"`
	Book   market.Book   `json:"book"`
	// Unhealthy is empty when the venue's last refresh succeeded, else the reason it didn't.
	Unhealthy string `json:"unhealthy,omitempty"`
}

// Policy holds the knobs a desk would tune. None of them name a venue.
type Policy struct {
	MaxBookAge time.Duration `json:"max_book_age"` // older books are excluded rather than trusted
	Split      bool          `json:"split"`        // allow one order to be split across venues
}

// Fill is a simulated execution on one market.
type Fill struct {
	Venue    string        `json:"venue"`
	MarketID string        `json:"market_id"`
	Qty      int64         `json:"qty"`
	Notional market.Amount `json:"notional"` // Σ price × qty, before fees
	Fee      market.Amount `json:"fee"`
	Total    market.Amount `json:"total"`  // notional + fee
	AllIn    market.Amount `json:"all_in"` // Total / Qty, rounded up
	Worst    market.Amount `json:"worst_price"`

	rawFee market.Amount // Σ per-execution fees before the venue's per-order rounding
}

// Evaluation records how one candidate fared, including why it was excluded.
type Evaluation struct {
	Venue    string        `json:"venue"`
	MarketID string        `json:"market_id"`
	Eligible bool          `json:"eligible"`
	Excluded string        `json:"excluded,omitempty"`
	BestAsk  market.Amount `json:"best_ask,omitempty"`
	Alone    *Fill         `json:"alone,omitempty"` // what this venue would do on its own for the whole order
}

// Status of a decision.
const (
	Filled   = "filled"
	Partial  = "partial"
	Rejected = "rejected"
)

// Decision is the full, self-explaining output of Route.
type Decision struct {
	ID          string        `json:"id"` // hash of the inputs: equal IDs mean identical inputs
	At          time.Time     `json:"at"`
	Order       Order         `json:"order"`
	Policy      Policy        `json:"policy"`
	Status      string        `json:"status"`
	Filled      int64         `json:"filled"`
	Total       market.Amount `json:"total"`
	AllIn       market.Amount `json:"all_in"`
	Allocations []Fill        `json:"allocations"`
	Evaluations []Evaluation  `json:"evaluations"`
	Explanation []string      `json:"explanation"`
}

// Route decides where to send o. It never fails: an unroutable order yields a Rejected decision that says why.
func Route(o Order, quotes []Quote, p Policy, now time.Time) Decision {
	quotes = slices.Clone(quotes)
	slices.SortFunc(quotes, func(a, b Quote) int {
		return cmp.Or(cmp.Compare(a.Market.Venue, b.Market.Venue), cmp.Compare(a.Market.ID, b.Market.ID))
	})
	d := Decision{ID: decisionID(o, quotes, p, now), At: now, Order: o, Policy: p, Allocations: []Fill{}}
	d.say("order: BUY %d %s%s, split %s, max book age %s", o.Qty, o.Side, limitText(o.Limit), onOff(p.Split), p.MaxBookAge)

	if err := o.validate(); err != nil {
		return d.reject("invalid order: %v", err)
	}

	var eligible []Quote
	for i, q := range quotes {
		ev := Evaluation{Venue: q.Market.Venue, MarketID: q.Market.ID}
		asks := q.Book.AsksFor(o.Side)
		if len(asks) > 0 {
			ev.BestAsk = asks[0].Price
		}
		ev.Excluded = exclusion(q, asks, o, p, now)
		if i > 0 && quotes[i-1].Market.Key() == q.Market.Key() {
			ev.Excluded = "duplicate quote for the same market"
		}
		if ev.Excluded == "" {
			if f := fill(q, asks, o.Qty, o.Limit); f.Qty < q.Market.MinQty {
				ev.Excluded = fmt.Sprintf("fillable size %d below venue minimum order %d", f.Qty, q.Market.MinQty)
			} else {
				ev.Eligible, ev.Alone = true, &f
				eligible = append(eligible, q)
			}
		}
		d.Evaluations = append(d.Evaluations, ev)
		if ev.Eligible {
			d.say("%s/%s: eligible; alone fills %d at all-in %s/contract (fees %s)",
				ev.Venue, ev.MarketID, ev.Alone.Qty, ev.Alone.AllIn, ev.Alone.Fee)
		} else {
			d.say("%s/%s: excluded: %s", ev.Venue, ev.MarketID, ev.Excluded)
		}
	}
	if len(eligible) == 0 {
		return d.reject("no eligible venue")
	}

	best := bestSingle(d.Evaluations)
	plan := []Fill{*best.Alone}
	if p.Split && len(eligible) > 1 {
		if s := split(eligible, o); better(s, plan) {
			d.say("split beats best single venue (%s/%s: %d at %s): %s",
				best.Venue, best.MarketID, best.Alone.Qty, best.Alone.Total, savings(s, plan))
			plan = s
		} else {
			d.say("split considered; no improvement over best single venue, keeping one child order")
		}
	}

	d.Allocations = plan
	for _, f := range plan {
		d.Filled += f.Qty
		d.Total += f.Total
	}
	d.AllIn = perContract(d.Total, d.Filled)
	d.Status = Filled
	if d.Filled < o.Qty {
		d.Status = Partial
	}
	for _, f := range plan {
		d.say("route %d to %s/%s: notional %s + fees %s = %s (all-in %s/contract, worst price %s)",
			f.Qty, f.Venue, f.MarketID, f.Notional, f.Fee, f.Total, f.AllIn, f.Worst)
	}
	if len(plan) == 1 {
		if rb := runnerUp(d.Evaluations, plan[0]); rb != nil {
			d.say("next best: %s/%s at all-in %s/contract for %d", rb.Venue, rb.MarketID, rb.Alone.AllIn, rb.Alone.Qty)
		}
	}
	d.say("result: %s %d/%d contracts, total %s, all-in %s/contract", d.Status, d.Filled, o.Qty, d.Total, d.AllIn)
	return d
}

func (o Order) validate() error {
	switch {
	case o.Side != market.Yes && o.Side != market.No:
		return fmt.Errorf("side must be yes or no, got %q", o.Side)
	case o.Qty <= 0 || o.Qty > MaxOrderQty:
		return fmt.Errorf("qty must be in 1..%d, got %d", MaxOrderQty, o.Qty)
	case o.Limit < 0 || o.Limit >= market.Dollar:
		return fmt.Errorf("limit must be below $1, got %s", o.Limit)
	}
	return nil
}

// exclusion returns why q cannot take part, or "" if it can.
func exclusion(q Quote, asks []market.Level, o Order, p Policy, now time.Time) string {
	switch {
	case q.Unhealthy != "":
		return "venue unhealthy: " + q.Unhealthy
	case q.Market.Untradable != "":
		return "market not tradable: " + q.Market.Untradable
	}
	if err := q.Book.Validate(); err != nil {
		return "untrusted book: " + err.Error()
	}
	if age := now.Sub(q.Book.AsOf); p.MaxBookAge > 0 && age > p.MaxBookAge {
		return fmt.Sprintf("stale book: age %s exceeds %s", age.Round(time.Second), p.MaxBookAge)
	}
	switch {
	case len(asks) == 0:
		return fmt.Sprintf("no %s offers on the book", o.Side)
	case o.Limit > 0 && asks[0].Price > o.Limit:
		return fmt.Sprintf("best ask %s above limit %s", asks[0].Price, o.Limit)
	}
	return ""
}

// fill walks asks best-first up to qty and the limit. Each price level is a separate execution; fees
// are summed across executions and rounded once per order the way the venue's FeeCurve says.
func fill(q Quote, asks []market.Level, qty int64, limit market.Amount) Fill {
	f := Fill{Venue: q.Market.Venue, MarketID: q.Market.ID}
	for _, l := range asks {
		if f.Qty == qty || (limit > 0 && l.Price > limit) {
			break
		}
		take := min(l.Qty, qty-f.Qty)
		f.add(q.Market.Fee, l.Price, take)
	}
	return f
}

func (f *Fill) add(fee market.FeeCurve, price market.Amount, qty int64) {
	f.Qty += qty
	f.Notional += price * market.Amount(qty)
	f.rawFee += fee.Fee(price, qty)
	f.Fee = fee.Round(f.rawFee)
	f.Total = f.Notional + f.Fee
	f.AllIn = perContract(f.Total, f.Qty)
	f.Worst = max(f.Worst, price)
}

// bestSingle ranks venues that could take the order alone: most contracts filled, then lowest all-in
// cost per contract, then venue and market id so that ties resolve the same way every time.
func bestSingle(evals []Evaluation) *Evaluation {
	var best *Evaluation
	for i := range evals {
		e := &evals[i]
		if e.Eligible && (best == nil || cheaper(*e.Alone, *best.Alone) < 0) {
			best = e
		}
	}
	return best
}

func runnerUp(evals []Evaluation, chosen Fill) *Evaluation {
	var rb *Evaluation
	for i := range evals {
		e := &evals[i]
		if e.Eligible && (e.Venue != chosen.Venue || e.MarketID != chosen.MarketID) && (rb == nil || cheaper(*e.Alone, *rb.Alone) < 0) {
			rb = e
		}
	}
	return rb
}

// cheaper orders fills: more contracts first, then lower cost per contract compared exactly by
// cross-multiplying (no division, no rounding), then venue and market id.
func cheaper(a, b Fill) int {
	return cmp.Or(
		cmp.Compare(b.Qty, a.Qty),
		cmp.Compare(int64(a.Total)*b.Qty, int64(b.Total)*a.Qty),
		cmp.Compare(a.Venue, b.Venue),
		cmp.Compare(a.MarketID, b.MarketID),
	)
}

// split fills the order greedily from a consolidated book of every eligible venue, cheapest
// fee-inclusive price first. A venue whose share would fall below its minimum order size is dropped and
// the allocation recomputed; that loop ends because each pass removes a venue.
func split(eligible []Quote, o Order) []Fill {
	for {
		fills := greedy(eligible, o)
		kept := eligible[:0:0]
		for _, q := range eligible {
			if f, ok := fills[q.Market.Key()]; !ok || f.Qty >= q.Market.MinQty {
				kept = append(kept, q)
			}
		}
		if len(kept) == len(eligible) {
			out := make([]Fill, 0, len(fills))
			for _, q := range eligible {
				if f, ok := fills[q.Market.Key()]; ok {
					out = append(out, *f)
				}
			}
			return out
		}
		eligible = kept
	}
}

type slot struct {
	key   int64 // cost of 1,000,000 contracts at this level, fees included: exact enough to rank levels
	q     *Quote
	level market.Level
}

func greedy(eligible []Quote, o Order) map[string]*Fill {
	const ref = 1_000_000
	var slots []slot
	for i := range eligible {
		q := &eligible[i]
		for _, l := range q.Book.AsksFor(o.Side) {
			if o.Limit > 0 && l.Price > o.Limit {
				break
			}
			key := int64(l.Price)*ref + int64(q.Market.Fee.Fee(l.Price, ref))
			slots = append(slots, slot{key, q, l})
		}
	}
	slices.SortFunc(slots, func(a, b slot) int {
		return cmp.Or(cmp.Compare(a.key, b.key), cmp.Compare(a.q.Market.Venue, b.q.Market.Venue),
			cmp.Compare(a.q.Market.ID, b.q.Market.ID), cmp.Compare(a.level.Price, b.level.Price))
	})
	fills := map[string]*Fill{}
	left := o.Qty
	for _, s := range slots {
		if left == 0 {
			break
		}
		f := fills[s.q.Market.Key()]
		if f == nil {
			f = &Fill{Venue: s.q.Market.Venue, MarketID: s.q.Market.ID}
			fills[s.q.Market.Key()] = f
		}
		take := min(s.level.Qty, left)
		f.add(s.q.Market.Fee, s.level.Price, take)
		left -= take
	}
	return fills
}

// better reports whether plan a fills more, or fills the same for strictly less money.
func better(a, b []Fill) bool {
	qa, ta := sum(a)
	qb, tb := sum(b)
	return qa > qb || (qa == qb && ta < tb)
}

func savings(split, single []Fill) string {
	qs, ts := sum(split)
	q1, t1 := sum(single)
	if qs > q1 {
		return fmt.Sprintf("fills %d more contracts", qs-q1)
	}
	return fmt.Sprintf("saves %s on %d contracts", t1-ts, qs)
}

func sum(fs []Fill) (qty int64, total market.Amount) {
	for _, f := range fs {
		qty += f.Qty
		total += f.Total
	}
	return
}

func perContract(total market.Amount, qty int64) market.Amount {
	if qty == 0 {
		return 0
	}
	return (total + market.Amount(qty) - 1) / market.Amount(qty)
}

// decisionID hashes the canonical (already sorted) inputs, so a reviewer can prove two decisions saw
// identical inputs, and a replay of logged inputs can be checked against the logged decision.
func decisionID(o Order, quotes []Quote, p Policy, now time.Time) string {
	b, _ := json.Marshal(struct {
		O Order
		Q []Quote
		P Policy
		T time.Time
	}{o, quotes, p, now.UTC()})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:8])
}

func (d *Decision) say(format string, args ...any) {
	d.Explanation = append(d.Explanation, fmt.Sprintf(format, args...))
}

func (d Decision) reject(format string, args ...any) Decision {
	d.Status = Rejected
	d.say("result: rejected: "+format, args...)
	return d
}

func limitText(l market.Amount) string {
	if l == 0 {
		return " at market"
	}
	return " limit " + l.String()
}

func onOff(b bool) string {
	if b {
		return "allowed"
	}
	return "off"
}
