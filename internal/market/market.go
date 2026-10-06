// Package market is Equinox's canonical, venue-independent model of a binary prediction market.
//
// Every venue adapter maps its own schema into these types, and nothing venue-specific crosses this
// boundary. Matching and routing depend only on this package, which is what keeps them venue-agnostic.
package market

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"
)

// Side is the outcome a contract pays out on.
type Side string

const (
	Yes Side = "yes"
	No  Side = "no"
)

// Market is one binary proposition on one venue: it pays $1 if YES resolves true, otherwise $0.
//
// Multi-outcome events ("Who wins the 2028 election?") are decomposed into one Market per outcome,
// which is how both Kalshi and Polymarket structure them underneath.
type Market struct {
	Venue    string    `json:"venue"`           // opaque venue id, e.g. "kalshi"; never interpreted downstream
	ID       string    `json:"id"`              // venue-native market id (Kalshi ticker, Polymarket market id)
	EventID  string    `json:"event_id"`        // venue-native grouping id
	Event    string    `json:"event,omitempty"` // event title: context the market title often omits ("Baltimore wins")
	Question string    `json:"question"`        // the proposition, as close to "Will X happen?" as the venue gives
	Outcome  string    `json:"outcome,omitempty"`
	Rules    string    `json:"rules,omitempty"` // resolution criteria text
	Category string    `json:"category,omitempty"`
	Close    time.Time `json:"close"`              // when trading stops
	Resolves time.Time `json:"resolves,omitempty"` // when the outcome is expected to be known, if the venue says
	URL      string    `json:"url,omitempty"`

	BookRef string   `json:"book_ref"` // opaque handle the adapter needs to fetch the order book
	Fee     FeeCurve `json:"fee"`
	Tick    Amount   `json:"tick"`
	MinQty  int64    `json:"min_qty"` // smallest order the venue accepts, in contracts
	// Untradable is empty when the market can take orders, else the reason it can't (closed, order
	// book disabled, a fee schedule we can't price...). Such markets are still matched, never routed.
	Untradable string `json:"untradable,omitempty"`

	// Top of book from the listing endpoint (0 = no quote). Display only: routing uses Book.
	YesBid Amount `json:"yes_bid"`
	YesAsk Amount `json:"yes_ask"`
}

// Key identifies a market globally.
func (m Market) Key() string { return m.Venue + ":" + m.ID }

// Level is one price level of an order book.
type Level struct {
	Price Amount `json:"price"`
	Qty   int64  `json:"qty"`
}

// Book is the YES order book of one market, best price first on each side.
//
// Only the YES side is stored. A NO contract is the complement of a YES contract, so buying NO at
// $1−p is the same trade as hitting a YES bid at p, and NO asks are derived from YES bids. Kalshi publishes
// exactly this (YES bids and NO bids); Polymarket's CLOB mirrors the two outcome tokens' books the same way.
type Book struct {
	MarketKey string    `json:"market_key"`
	Bids      []Level   `json:"bids"` // descending price
	Asks      []Level   `json:"asks"` // ascending price
	AsOf      time.Time `json:"as_of"`
}

// AsksFor returns the levels a buyer of side lifts, best (cheapest) first.
func (b Book) AsksFor(side Side) []Level {
	if side == Yes {
		return b.Asks
	}
	out := make([]Level, len(b.Bids))
	for i, l := range b.Bids {
		out[i] = Level{Price: Dollar - l.Price, Qty: l.Qty}
	}
	return out
}

// Normalize sorts both sides best-first, merges duplicate prices and drops empty or out-of-range
// levels. Adapters call it so that every Book downstream has one shape regardless of venue ordering
// (Polymarket returns asks worst-first, for example).
func (b Book) Normalize() Book {
	b.Bids = clean(b.Bids, func(x, y Level) int { return cmp.Compare(y.Price, x.Price) })
	b.Asks = clean(b.Asks, func(x, y Level) int { return cmp.Compare(x.Price, y.Price) })
	return b
}

func clean(levels []Level, cmp func(x, y Level) int) []Level {
	out := make([]Level, 0, len(levels))
	for _, l := range levels {
		if l.Qty > 0 && l.Price > 0 && l.Price < Dollar {
			out = append(out, l)
		}
	}
	slices.SortStableFunc(out, cmp)
	merged := out[:0]
	for _, l := range out {
		if n := len(merged); n > 0 && merged[n-1].Price == l.Price {
			merged[n-1].Qty = min(merged[n-1].Qty, math.MaxInt64-l.Qty) + l.Qty // saturate, never wrap
			continue
		}
		merged = append(merged, l)
	}
	return merged
}

// Validate reports why a normalized book can't be trusted, or nil. A crossed book (best bid ≥ best
// ask) is the usual symptom of a stale or half-updated snapshot.
func (b Book) Validate() error {
	if len(b.Bids) == 0 && len(b.Asks) == 0 {
		return fmt.Errorf("empty book")
	}
	if len(b.Bids) > 0 && len(b.Asks) > 0 && b.Bids[0].Price >= b.Asks[0].Price {
		return fmt.Errorf("crossed book: bid %s >= ask %s", b.Bids[0].Price, b.Asks[0].Price)
	}
	if b.AsOf.IsZero() {
		return fmt.Errorf("book has no timestamp")
	}
	return nil
}
