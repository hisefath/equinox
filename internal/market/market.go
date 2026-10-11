// Package market holds the venue-neutral types every other package shares: money, fees, order books and the
// canonical market (one binary proposition). It depends on the standard library only (lld.md BE-1).
package market

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
)

// Amount is money in integer micro-dollars (one millionth of a US dollar). Never a float (HLD INV-3).
type Amount int64

const (
	Micro  Amount = 1
	Cent   Amount = 10_000
	Dollar Amount = 1_000_000
)

// ParseAmount parses a decimal dollar string such as "0.585" exactly. It rejects more than 6 decimals, signs and
// overflow, so a venue's price never silently rounds.
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty amount")
	}
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if len(frac) > 6 {
		return 0, fmt.Errorf("amount %q has more than 6 decimals", s)
	}
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("amount %q is not a plain decimal", s)
		}
	}
	if len(whole) > 12 {
		return 0, fmt.Errorf("amount %q is too large", s)
	}
	var w, f int64
	for _, r := range whole {
		w = w*10 + int64(r-'0')
	}
	frac += strings.Repeat("0", 6-len(frac))
	for _, r := range frac {
		f = f*10 + int64(r-'0')
	}
	return Amount(w*int64(Dollar) + f), nil
}

// String renders the amount as dollars with up to 6 decimals and no trailing zeros ("0.585", "1", "-0.01").
func (a Amount) String() string {
	sign := ""
	if a < 0 {
		sign, a = "-", -a
	}
	s := fmt.Sprintf("%d.%06d", a/Dollar, a%Dollar)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return sign + s
}

// Side is the side of a binary proposition an order buys.
type Side string

const (
	Yes Side = "yes"
	No  Side = "no"
)

// Key identifies one binary proposition on one venue: "<venue>:<venue id>[:<outcome index>]".
type Key string

// Level is one price level of an order book.
type Level struct {
	Price Amount `json:"price"`
	Qty   int64  `json:"qty"`
}

// Book is the YES side of one market's order book at receipt time. A NO ask at price p is a YES bid at $1 - p.
type Book struct {
	Key     Key       `json:"key"`
	Bids    []Level   `json:"bids"` // best (highest) first
	Asks    []Level   `json:"asks"` // best (lowest) first
	AsOf    time.Time `json:"as_of"`
	Crossed bool      `json:"crossed"`
}

// NewBook sorts and merges levels, drops empty or out-of-range levels and flags a crossed book (lld.md books).
func NewBook(key Key, bids, asks []Level, asOf time.Time) (Book, error) {
	if asOf.IsZero() {
		return Book{}, fmt.Errorf("book %s has no time", key)
	}
	b := Book{Key: key, Bids: merge(bids, true), Asks: merge(asks, false), AsOf: asOf.UTC()}
	if len(b.Bids) > 0 && len(b.Asks) > 0 && b.Bids[0].Price >= b.Asks[0].Price {
		b.Crossed = true
	}
	return b, nil
}

func merge(levels []Level, desc bool) []Level {
	by := map[Amount]int64{}
	for _, l := range levels {
		if l.Qty <= 0 || l.Price <= 0 || l.Price >= Dollar {
			continue
		}
		by[l.Price] += l.Qty
	}
	out := make([]Level, 0, len(by))
	for p, q := range by {
		out = append(out, Level{Price: p, Qty: q})
	}
	sort.Slice(out, func(i, j int) bool {
		if desc {
			return out[i].Price > out[j].Price
		}
		return out[i].Price < out[j].Price
	})
	return out
}

// AsksFor returns the levels an order on side buys from, cheapest first: YES asks, or NO asks derived from YES bids.
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

// FeeTerms describes a taker fee as data: fee = qty × rate × p^PExp × (1-p)^QExp, rounded up to RoundTo per order.
// Both venues' published schedules fit this curve (R04); unknown terms make a market untradable (LINV-1).
type FeeTerms struct {
	RatePPM int64  `json:"rate_ppm"` // rate in parts per million, e.g. 70000 = 0.07
	PExp    int    `json:"p_exp"`
	QExp    int    `json:"q_exp"`
	RoundTo Amount `json:"round_to"` // smallest fee unit, e.g. Cent; fees round up to it
}

// Known reports whether every fee field is set.
func (f FeeTerms) Known() bool { return f.RatePPM >= 0 && f.RoundTo > 0 && f.PExp >= 0 && f.QExp >= 0 }

// TakerFee is the fee for buying qty contracts at price, computed exactly with big integers and rounded up.
func (f FeeTerms) TakerFee(qty int64, price Amount) Amount {
	if qty <= 0 || f.RatePPM == 0 {
		return 0
	}
	p := big.NewInt(int64(price))
	q := big.NewInt(int64(Dollar - price))
	num := new(big.Int).Mul(big.NewInt(qty), big.NewInt(f.RatePPM)) // qty × rate(ppm)
	num.Mul(num, new(big.Int).Exp(p, big.NewInt(int64(f.PExp)), nil))
	num.Mul(num, new(big.Int).Exp(q, big.NewInt(int64(f.QExp)), nil))
	// num is in units of 1e-6 (rate) × Dollar^(PExp+QExp); convert to micro-dollars.
	den := new(big.Int).Exp(big.NewInt(int64(Dollar)), big.NewInt(int64(f.PExp+f.QExp)), nil)
	den.Mul(den, big.NewInt(1_000_000))
	den.Div(den, big.NewInt(int64(Dollar))) // result in micro-dollars
	micro := ceilDiv(num, den)
	unit := big.NewInt(int64(f.RoundTo))
	rounded := new(big.Int).Mul(ceilDiv(micro, unit), unit)
	return Amount(rounded.Int64())
}

func ceilDiv(a, b *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(a, b, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// Unknown marks a settlement part the parser could not read; a pair with any Unknown part is never equivalent (LINV-2).
const Unknown = "unknown"

// Terms are the comparable settlement parts of one proposition (lld.md D20).
type Terms struct {
	Subject    string   `json:"subject"`    // who or what the question is about, normalized
	Predicate  string   `json:"predicate"`  // win, above, below, cut, hike, ...
	Comparator string   `json:"comparator"` // ">", ">=", "<", "<=", "=", "range", or Unknown
	Threshold  string   `json:"threshold"`  // normalized number, or "" when the question has none
	Source     string   `json:"source"`     // resolution source, or Unknown
	Statistic  string   `json:"statistic"`  // e.g. "close", "60s average", or "" when not a price question
	WindowEnd  string   `json:"window_end"` // the date the outcome is decided by (YYYY-MM-DD), or Unknown
	EdgeRules  []string `json:"edge_rules"` // carve-outs that change payout (e.g. "death", "tie 50/50")
}

// Market is one binary proposition on one venue in venue-neutral form (lld.md markets).
type Market struct {
	Key              Key       `json:"key"`
	Venue            string    `json:"venue"`
	EventID          string    `json:"event_id"`
	Question         string    `json:"question"`
	Outcome          string    `json:"outcome"`
	RulesText        string    `json:"rules_text"`
	RulesFingerprint string    `json:"rules_fingerprint"`
	Terms            Terms     `json:"terms"`
	Topic            string    `json:"topic"`
	OpensAt          time.Time `json:"opens_at"`
	ClosesAt         time.Time `json:"closes_at"`
	ResolvesAt       time.Time `json:"resolves_at"`
	Volume24h        int64     `json:"volume_24h"`
	Tradable         bool      `json:"tradable"`
	UntradableReason string    `json:"untradable_reason,omitempty"`
	Fee              FeeTerms  `json:"fee"`
	MinQty           int64     `json:"min_qty"`
	BookRef          string    `json:"book_ref"`
	ReceivedAt       time.Time `json:"received_at"`
}

// Topics are the system's own topic labels (A-6).
var Topics = map[string]bool{"elections": true, "economics": true, "sports": true, "crypto": true, "other": true}

// NewMarket validates m and fills defaults; an invalid record is never held (lld.md ck_markets_* checks).
func NewMarket(m Market) (Market, error) {
	if m.Key == "" || m.Venue == "" {
		return Market{}, errors.New("market needs a key and a venue")
	}
	if strings.TrimSpace(m.Question) == "" {
		return Market{}, fmt.Errorf("%s: empty question", m.Key)
	}
	if m.Outcome == "" {
		m.Outcome = "Yes"
	}
	if !Topics[m.Topic] {
		m.Topic = "other"
	}
	if m.MinQty < 1 {
		m.MinQty = 1
	}
	if m.ClosesAt.IsZero() {
		return Market{}, fmt.Errorf("%s: no close time", m.Key)
	}
	m.OpensAt, m.ClosesAt, m.ReceivedAt = m.OpensAt.UTC(), m.ClosesAt.UTC(), m.ReceivedAt.UTC()
	if !m.ResolvesAt.IsZero() {
		m.ResolvesAt = m.ResolvesAt.UTC()
	}
	for _, p := range []*string{&m.Terms.Comparator, &m.Terms.Source, &m.Terms.WindowEnd} {
		if *p == "" {
			*p = Unknown
		}
	}
	if !m.Fee.Known() {
		m.Tradable, m.UntradableReason = false, "fee terms unknown"
	}
	if !m.Tradable && m.UntradableReason == "" {
		m.UntradableReason = "not tradable"
	}
	if m.Tradable {
		m.UntradableReason = ""
	}
	if m.RulesFingerprint == "" {
		m.RulesFingerprint = Fingerprint(m.RulesText, m.Terms)
	}
	return m, nil
}

// Active reports whether a market belongs in scope: traded in the last 24 hours and lasts at least 1 hour (D11).
func (m Market) Active() (bool, string) {
	if m.Volume24h <= 0 {
		return false, "inactive"
	}
	if !m.OpensAt.IsZero() && m.ClosesAt.Sub(m.OpensAt) < time.Hour {
		return false, "too_short"
	}
	return true, ""
}

// MarshalJSON writes an amount as an exact decimal dollar string ("0.585"), as the API contract specifies.
func (a Amount) MarshalJSON() ([]byte, error) { return []byte(`"` + a.String() + `"`), nil }

// UnmarshalJSON reads a decimal dollar string (or a JSON number in micro-dollars, for older files).
func (a *Amount) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) >= 2 && s[0] == '"' {
		neg := strings.HasPrefix(s[1:len(s)-1], "-")
		v, err := ParseAmount(strings.TrimPrefix(s[1:len(s)-1], "-"))
		if neg {
			v = -v
		}
		*a = v
		return err
	}
	var n int64
	_, err := fmt.Sscan(s, &n)
	*a = Amount(n)
	return err
}
