// Package route decides one hypothetical order across venue-neutral quotes as a pure function (lld.md BE-8).
// It imports only internal/market and the standard library without net (INV-1) and never names a venue (INV-7).
package route

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

// DefaultMaxAge is the book freshness limit used when a Policy leaves MaxAge unset (lld.md SEQ-3: 90 s).
const DefaultMaxAge = 90 * time.Second

// Outcomes (ck_decisions_outcome). Refused is set by the caller for a pair that cannot be routed.
const (
	Filled    = "filled"
	Partial   = "partial"
	NotFilled = "not_filled"
	Refused   = "refused"
)

// Deciding rules (ck_decisions_rule, FR-010).
const (
	RuleMostFilled   = "most_filled"
	RuleLowestAllIn  = "lowest_all_in"
	RuleTieBreak     = "tie_break"
	RuleOnlyEligible = "only_eligible"
)

// Order is one hypothetical buy on a pair (lld.md decisions.order).
type Order struct {
	PairID         string         `json:"pair_id"`
	Side           market.Side    `json:"side"`
	Quantity       int64          `json:"quantity"`
	Limit          *market.Amount `json:"limit,omitempty"`
	CustomerRegion string         `json:"customer_region"`
}

// FieldError names one invalid order field as a JSON Pointer (lld.md §8 validation).
type FieldError struct {
	Pointer string `json:"pointer"`
	Detail  string `json:"detail"`
}

// FieldErrors is the list of invalid fields Validate returns.
type FieldErrors []FieldError

func (e FieldErrors) Error() string {
	s := make([]string, len(e))
	for i, f := range e {
		s[i] = f.Pointer + ": " + f.Detail
	}
	return strings.Join(s, "; ")
}

// Validate returns nil or every invalid field (FR-016, FR-037).
func (o Order) Validate() error {
	var errs FieldErrors
	if o.Quantity < 1 {
		errs = append(errs, FieldError{"#/quantity", "quantity must be at least 1"})
	}
	if o.Side != market.Yes && o.Side != market.No {
		errs = append(errs, FieldError{"#/side", "side must be yes or no"})
	}
	if o.Limit != nil && (*o.Limit < 0 || *o.Limit > market.Dollar) {
		errs = append(errs, FieldError{"#/limit", "limit must be between 0 and 1"})
	}
	switch o.CustomerRegion {
	case "US", "non-US":
	case "":
		errs = append(errs, FieldError{"#/customer_region", "customer region required"})
	default:
		errs = append(errs, FieldError{"#/customer_region", "customer region must be US or non-US"})
	}
	if errs == nil {
		return nil
	}
	return errs
}

// Quote is everything the router may know about one venue's market for the pair, as data (FR-014). Venue is an
// opaque id; the caller fills eligibility for the order's region, health and tradability from the snapshot.
type Quote struct {
	Venue            string          `json:"venue"`
	Key              market.Key      `json:"key"`
	Book             market.Book     `json:"book"`
	Fee              market.FeeTerms `json:"fee"`
	MinQty           int64           `json:"min_qty"`
	Healthy          bool            `json:"healthy"`
	HealthReason     string          `json:"health_reason,omitempty"`
	Tradable         bool            `json:"tradable"`
	UntradableReason string          `json:"untradable_reason,omitempty"`
	Eligible         bool            `json:"eligible"`
}

// Policy holds the routing limits.
type Policy struct {
	MaxAge time.Duration `json:"max_age"` // books older than this at asOf are left out (FR-022); 0 = DefaultMaxAge
}

// VenueResult says how one venue fared: chosen, lost, or left out with the reason (FR-011).
type VenueResult struct {
	Venue     string         `json:"venue"`
	MarketKey market.Key     `json:"market_key"`
	Eligible  bool           `json:"eligible"`
	Reason    string         `json:"reason"`
	Filled    int64          `json:"filled"`
	AllIn     *market.Amount `json:"all_in,omitempty"`
}

// Decision is one decided order (lld.md decisions). Route fills the routing fields; the caller fills LoggedAt,
// RequestID, Evidence, Caveats and, for a refusal, ProblemType.
type Decision struct {
	DecisionID   string         `json:"decision_id"`
	LoggedAt     time.Time      `json:"logged_at"`
	RequestID    string         `json:"request_id"`
	Order        Order          `json:"order"`
	SnapshotAsOf time.Time      `json:"snapshot_as_of"`
	Outcome      string         `json:"outcome"`
	Venue        string         `json:"venue,omitempty"`
	MarketKey    market.Key     `json:"market_key,omitempty"`
	FilledQty    int64          `json:"filled_qty"`
	UnfilledQty  int64          `json:"unfilled_qty"`
	AvgPrice     *market.Amount `json:"avg_price,omitempty"`
	Fees         market.Amount  `json:"fees"`
	AllIn        market.Amount  `json:"all_in"`
	DecidingRule string         `json:"deciding_rule,omitempty"`
	Saving       *market.Amount `json:"saving,omitempty"`
	Venues       []VenueResult  `json:"venues"`
	Evidence     []string       `json:"evidence"`
	Caveats      []string       `json:"caveats"`
	ProblemType  string         `json:"problem_type,omitempty"`
}

type fill struct {
	q                     Quote
	reason                string // non-empty: left out
	filled                int64
	notional, fees, allIn market.Amount
}

// Route decides o across quotes at the snapshot time asOf. It never errors, never reads a clock and gives the same
// decision whatever order the quotes arrive in (INV-2). A malformed order must be rejected by Validate first.
func Route(o Order, quotes []Quote, p Policy, asOf time.Time) Decision {
	if p.MaxAge <= 0 {
		p.MaxAge = DefaultMaxAge
	}
	asOf = asOf.UTC()
	qs := slices.Clone(quotes)
	// Total order: venue id, market key, then full content, so shuffled input sorts identically (INV-2).
	slices.SortFunc(qs, func(a, b Quote) int {
		ea, _ := json.Marshal(a)
		eb, _ := json.Marshal(b)
		return cmp.Or(strings.Compare(a.Venue, b.Venue), strings.Compare(string(a.Key), string(b.Key)), strings.Compare(string(ea), string(eb)))
	})

	d := Decision{Order: o, SnapshotAsOf: asOf, Outcome: NotFilled, UnfilledQty: o.Quantity,
		Venues: []VenueResult{}, Evidence: []string{}, Caveats: []string{}}
	d.DecisionID = decisionID(o, qs, p, asOf)

	fills := make([]fill, len(qs))
	var cands []*fill
	for i, q := range qs {
		fills[i] = try(o, q, p, asOf)
		if fills[i].reason == "" {
			cands = append(cands, &fills[i])
		}
	}
	// Most filled, then lowest all-in, then the fixed order (venue id, market key) the quotes are already in.
	slices.SortStableFunc(cands, func(a, b *fill) int {
		return cmp.Or(cmp.Compare(b.filled, a.filled), cmp.Compare(a.allIn, b.allIn))
	})

	var win *fill
	if len(cands) > 0 {
		win = cands[0]
		d.Outcome, d.Venue, d.MarketKey = Filled, win.q.Venue, win.q.Key
		if win.filled < o.Quantity {
			d.Outcome = Partial
		}
		d.FilledQty, d.UnfilledQty = win.filled, o.Quantity-win.filled
		avg := win.notional / market.Amount(win.filled)
		d.AvgPrice, d.Fees, d.AllIn = &avg, win.fees, win.allIn
		switch {
		case len(cands) == 1:
			d.DecidingRule = RuleOnlyEligible
		case cands[1].filled != win.filled:
			d.DecidingRule = RuleMostFilled
		case cands[1].allIn != win.allIn:
			d.DecidingRule = RuleLowestAllIn
			s := cands[1].allIn - win.allIn
			d.Saving = &s
		default:
			d.DecidingRule = RuleTieBreak
			s := market.Amount(0)
			d.Saving = &s
		}
	}
	winReason := map[string]string{RuleOnlyEligible: "only eligible venue", RuleMostFilled: "most contracts filled",
		RuleLowestAllIn: "lowest all-in cost", RuleTieBreak: "tie on all-in cost, broken by fixed order"}[d.DecidingRule]

	for i := range fills {
		f := &fills[i]
		r := VenueResult{Venue: f.q.Venue, MarketKey: f.q.Key, Eligible: f.reason == "", Reason: f.reason, Filled: f.filled}
		if f.reason == "" {
			allIn := f.allIn
			r.AllIn = &allIn
			switch {
			case f == win:
				r.Reason = winReason
			case f.filled < win.filled:
				r.Reason = "fewer contracts filled"
			case f.allIn > win.allIn:
				r.Reason = "higher all-in cost"
			default:
				r.Reason = "tie on all-in cost, broken by fixed order"
			}
		}
		d.Venues = append(d.Venues, r)
	}
	return d
}

// try fills o against one quote, or returns the reason the venue is left out (FR-017, FR-022, FR-023, FR-036).
func try(o Order, q Quote, p Policy, asOf time.Time) fill {
	f := fill{q: q}
	asks := q.Book.AsksFor(o.Side)
	switch {
	case !q.Eligible:
		f.reason = "not open to " + o.CustomerRegion + " customers"
	case !q.Healthy:
		f.reason = "venue unhealthy: " + q.HealthReason
	case !q.Tradable:
		f.reason = q.UntradableReason
		if f.reason == "" {
			f.reason = "not tradable"
		}
	case q.Book.AsOf.IsZero():
		f.reason = "undated order book"
	case asOf.Sub(q.Book.AsOf) > p.MaxAge:
		f.reason = fmt.Sprintf("data %d s old", int64(asOf.Sub(q.Book.AsOf)/time.Second))
	case q.Book.Crossed:
		f.reason = "crossed order book"
	case len(asks) == 0:
		f.reason = "empty order book"
	case o.Quantity < q.MinQty:
		f.reason = fmt.Sprintf("below minimum size %d", q.MinQty)
	case o.Limit != nil && asks[0].Price > *o.Limit:
		f.reason = "best ask above limit"
	}
	if f.reason != "" {
		return f
	}
	// ponytail: fee is charged per price level walked, each level rounded up; per-order rounding if a venue differs.
	for _, l := range asks {
		if f.filled == o.Quantity || (o.Limit != nil && l.Price > *o.Limit) {
			break
		}
		n := min(l.Qty, o.Quantity-f.filled)
		f.filled += n
		f.notional += l.Price * market.Amount(n)
		f.fees += q.Fee.TakerFee(n, l.Price)
	}
	f.allIn = f.notional + f.fees
	return f
}

// decisionID is the first 16 hex chars of SHA-256 over the canonical inputs (lld.md decisions.decision_id).
func decisionID(o Order, sorted []Quote, p Policy, asOf time.Time) string {
	b, _ := json.Marshal(struct {
		O Order
		Q []Quote
		P Policy
		T time.Time
	}{o, sorted, p, asOf})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:16]
}
