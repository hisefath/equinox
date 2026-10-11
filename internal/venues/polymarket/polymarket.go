// Package polymarket reads the offshore venue's open markets in 24-hour-volume order, their fee schedules and
// their order books (lld.md BE-4).
package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/venue"
)

// Name is the venue id used in keys.
const Name = "polymarket"

// booksPerPost is how many tokens one POST /books asks for.
// ponytail: fixed 50; the docs list no batch limit, raise it if the venue accepts more.
const booksPerPost = 50

// Adapter implements venue.Venue for Polymarket's Gamma (catalog) and CLOB (books) APIs.
type Adapter struct {
	c                     *fetch.Client
	catalogBase, bookBase string
	now                   func() time.Time

	PageSize int // markets per keyset page; default 100 (the API clamps to 100, R08 F4)
	MaxPages int // 0 reads until the first zero-volume market

	mu     sync.Mutex
	tokens map[market.Key]string // book token per key, from the latest Catalog
}

var _ venue.Venue = (*Adapter)(nil)

// New returns an adapter, e.g. New(c, "https://gamma-api.polymarket.com", "https://clob.polymarket.com", time.Now).
func New(c *fetch.Client, catalogBase, bookBase string, now func() time.Time) *Adapter {
	if now == nil {
		now = time.Now
	}
	return &Adapter{c: c, catalogBase: strings.TrimRight(catalogBase, "/"), bookBase: strings.TrimRight(bookBase, "/"),
		now: now, PageSize: 100, tokens: map[market.Key]string{}}
}

func (a *Adapter) Name() string { return Name }

type rawMarket struct {
	ID               string      `json:"id"`
	Question         string      `json:"question"`
	Description      string      `json:"description"`
	ResolutionSource string      `json:"resolutionSource"`
	StartDate        time.Time   `json:"startDate"`
	EndDate          time.Time   `json:"endDate"`
	Outcomes         string      `json:"outcomes"`     // JSON-encoded string array
	ClobTokenIDs     string      `json:"clobTokenIds"` // JSON-encoded string array
	Volume24h        json.Number `json:"volume24hr"`
	Active           bool        `json:"active"`
	Closed           bool        `json:"closed"`
	AcceptingOrders  bool        `json:"acceptingOrders"`
	EnableOrderBook  bool        `json:"enableOrderBook"`
	OrderMinSize     json.Number `json:"orderMinSize"`
	FeesEnabled      bool        `json:"feesEnabled"`
	FeeType          string      `json:"feeType"`
	FeeSchedule      *struct {
		Rate     json.Number `json:"rate"`
		Exponent json.Number `json:"exponent"`
	} `json:"feeSchedule"`
	SportsMarketType string `json:"sportsMarketType"`
	Events           []struct {
		ID string `json:"id"`
	} `json:"events"`
}

// Catalog pages the open-market listing in 24-hour-volume order, stops at the first market with zero volume and
// de-duplicates by id (FR-001, FM-8). A named-outcome listing becomes one market per outcome (D19).
func (a *Adapter) Catalog(ctx context.Context) ([]market.Market, venue.CatalogStats, error) {
	var (
		out    []market.Market
		stats  venue.CatalogStats
		seen   = map[string]bool{}
		tokens = map[market.Key]string{}
		cursor string
	)
pages:
	for page := 0; a.MaxPages == 0 || page < a.MaxPages; page++ {
		q := url.Values{"closed": {"false"}, "limit": {fmt.Sprint(a.PageSize)}, "order": {"volume24hr"}, "ascending": {"false"}}
		if cursor != "" {
			q.Set("after_cursor", cursor)
		}
		var resp struct {
			NextCursor string            `json:"next_cursor"`
			Markets    []json.RawMessage `json:"markets"`
		}
		if err := a.c.GetJSON(ctx, a.catalogBase+"/markets/keyset?"+q.Encode(), &resp); err != nil {
			return nil, stats, err
		}
		for _, raw := range resp.Markets {
			var r rawMarket
			if err := json.Unmarshal(raw, &r); err != nil || r.ID == "" || seen[r.ID] {
				stats.Unread()
				continue
			}
			seen[r.ID] = true
			ms, err := a.toMarkets(r)
			if err != nil {
				stats.Unread()
				continue
			}
			zero := false
			for _, m := range ms {
				zero = m.Volume24h == 0
				if stats.Count(m) {
					out = append(out, m)
					tokens[m.Key] = m.BookRef
				}
			}
			if zero {
				break pages // listing is in volume order: everything after is inactive too
			}
		}
		cursor = resp.NextCursor
		if cursor == "" {
			break
		}
	}
	a.mu.Lock()
	a.tokens = tokens
	a.mu.Unlock()
	return out, stats, nil
}

func (a *Adapter) toMarkets(r rawMarket) ([]market.Market, error) {
	var outcomes, tokens []string
	if err := json.Unmarshal([]byte(r.Outcomes), &outcomes); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(r.ClobTokenIDs), &tokens); err != nil {
		return nil, err
	}
	if len(outcomes) < 2 || len(outcomes) != len(tokens) {
		return nil, fmt.Errorf("%s: %d outcomes, %d tokens", r.ID, len(outcomes), len(tokens))
	}
	vol := int64(0)
	if r.Volume24h != "" {
		v, err := venue.Contracts(r.Volume24h.String())
		if err != nil {
			return nil, err
		}
		vol = v
	}
	minQty := int64(1)
	if r.OrderMinSize != "" {
		if v, err := venue.Contracts(r.OrderMinSize.String()); err == nil && v > 0 {
			minQty = v
		}
	}
	now := a.now()
	base := market.Market{
		Venue:     Name,
		Question:  r.Question,
		RulesText: r.Description,
		Topic:     topic(r),
		OpensAt:   r.StartDate, ClosesAt: r.EndDate,
		Volume24h: vol,
		Fee:       fee(r),
		MinQty:    minQty,
		Tradable:  true, ReceivedAt: now,
	}
	if len(r.Events) > 0 {
		base.EventID = r.Events[0].ID
	}
	switch {
	case r.Closed || !r.Active:
		base.Tradable, base.UntradableReason = false, "closed"
	case !r.AcceptingOrders || !r.EnableOrderBook:
		base.Tradable, base.UntradableReason = false, "not accepting orders"
	case !r.EndDate.After(now):
		base.Tradable, base.UntradableReason = false, "past its end"
	}
	yesNo := len(outcomes) == 2 && strings.EqualFold(outcomes[0], "yes") && strings.EqualFold(outcomes[1], "no")
	var out []market.Market
	for i, o := range outcomes {
		m := base
		m.Outcome, m.BookRef = o, tokens[i]
		m.Key = market.Key(fmt.Sprintf("%s:%s:%d", Name, r.ID, i))
		if yesNo {
			m.Key = market.Key(Name + ":" + r.ID)
		}
		m.Terms = terms(r, o, yesNo)
		nm, err := market.NewMarket(m)
		if err != nil {
			return nil, err
		}
		out = append(out, nm)
		if yesNo {
			break // the NO token is the other side of the same proposition
		}
	}
	return out, nil
}

// fee reads feeSchedule: fee = qty × rate × p × (1-p), rounded to 5 decimals (R04 F7, F9, F10).
func fee(r rawMarket) market.FeeTerms {
	unknown := market.FeeTerms{RatePPM: -1}
	if r.FeeSchedule == nil {
		if r.FeesEnabled {
			return unknown
		}
		return market.FeeTerms{RatePPM: 0, PExp: 1, QExp: 1, RoundTo: 10 * market.Micro} // fee-free market
	}
	if e := r.FeeSchedule.Exponent.String(); e != "" && e != "1" {
		return unknown // only the documented exponent 1 fits the curve with certainty
	}
	rate, err := market.ParseAmount(r.FeeSchedule.Rate.String()) // millionths = ppm
	if err != nil {
		return unknown
	}
	return market.FeeTerms{RatePPM: int64(rate), PExp: 1, QExp: 1, RoundTo: 10 * market.Micro}
}

func topic(r rawMarket) string {
	if r.SportsMarketType != "" {
		return "sports"
	}
	return venue.Topic(r.FeeType)
}

var (
	numRe     = `\$?([0-9][0-9,]*(?:\.[0-9]+)?)\s*([kmb]\b)?`
	betweenRe = regexp.MustCompile(`(?i)^will (.+?) be between ` + numRe + ` and ` + numRe)
	cmpRe     = regexp.MustCompile(`(?i)^will (.+?) (?:be |close |trade |finish )?(above|over|greater than|more than|at least|reach|hit|below|under|less than|at most) ` + numRe)
	winRe     = regexp.MustCompile(`(?i)^will (.+?) win\b`)
)

var comparators = map[string][2]string{
	"above": {"above", ">"}, "over": {"above", ">"}, "greater than": {"above", ">"}, "more than": {"above", ">"},
	"at least": {"above", ">="}, "reach": {"above", ">="}, "hit": {"above", ">="},
	"below": {"below", "<"}, "under": {"below", "<"}, "less than": {"below", "<"}, "at most": {"below", "<="},
}

// terms reads settlement parts from the free-text question; anything it cannot read stays Unknown (R03, D20).
// ponytail: a few hand-written patterns; unread questions go to the AI path (FR-038).
func terms(r rawMarket, outcome string, yesNo bool) market.Terms {
	t := market.Terms{
		Subject:   venue.Normalize(r.Question),
		Predicate: market.Unknown, Comparator: market.Unknown,
		Source:    market.Unknown,
		WindowEnd: r.EndDate.UTC().Format(time.DateOnly),
		EdgeRules: venue.EdgeRules(r.Description),
	}
	if s := strings.TrimSpace(r.ResolutionSource); s != "" {
		t.Source = strings.ToLower(s)
	}
	q := strings.TrimSpace(r.Question)
	switch {
	case !yesNo:
		t.Subject = venue.Normalize(outcome)
		if r.SportsMarketType == "moneyline" {
			t.Predicate, t.Comparator = "win", "="
		}
	case betweenRe.MatchString(q):
		m := betweenRe.FindStringSubmatch(q)
		lo, ok1 := number(m[2], m[3])
		hi, ok2 := number(m[4], m[5])
		if ok1 && ok2 {
			t.Subject, t.Predicate, t.Comparator, t.Threshold = venue.Normalize(m[1]), "between", "range", lo+"-"+hi
		}
	case cmpRe.MatchString(q):
		m := cmpRe.FindStringSubmatch(q)
		if n, ok := number(m[3], m[4]); ok {
			c := comparators[strings.ToLower(m[2])]
			t.Subject, t.Predicate, t.Comparator, t.Threshold = venue.Normalize(m[1]), c[0], c[1], n
		}
	case winRe.MatchString(q):
		t.Subject, t.Predicate, t.Comparator = venue.Normalize(winRe.FindStringSubmatch(q)[1]), "win", "="
	}
	return t
}

// number normalizes "$120,000", "120k" or "1.5m" to a plain decimal string.
func number(digits, suffix string) (string, bool) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(digits, ",", ""), 64)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(suffix) {
	case "k":
		f *= 1e3
	case "m":
		f *= 1e6
	case "b":
		f *= 1e9
	}
	return strconv.FormatFloat(f, 'f', -1, 64), true
}

// Books posts the tokens of the given keys (from the latest Catalog) in batches. Partial results are returned with
// the first error.
func (a *Adapter) Books(ctx context.Context, keys []market.Key) (map[market.Key]market.Book, error) {
	a.mu.Lock()
	byToken := map[string][]market.Key{}
	var order []string
	for _, k := range keys {
		if t, ok := a.tokens[k]; ok {
			if _, dup := byToken[t]; !dup {
				order = append(order, t)
			}
			byToken[t] = append(byToken[t], k)
		}
	}
	a.mu.Unlock()
	out := map[market.Key]market.Book{}
	var first error
	for i := 0; i < len(order); i += booksPerPost {
		batch := order[i:min(i+booksPerPost, len(order))]
		body := make([]map[string]string, len(batch))
		for j, t := range batch {
			body[j] = map[string]string{"token_id": t}
		}
		var resp []struct {
			AssetID string `json:"asset_id"`
			Bids    []rawLevel
			Asks    []rawLevel
		}
		if err := a.c.PostJSON(ctx, a.bookBase+"/books", body, &resp); err != nil {
			if first == nil {
				first = err
			}
			var refused fetch.ErrRefused
			if errors.As(err, &refused) || ctx.Err() != nil {
				break // honour the wait: no more requests to this venue now (FR-025)
			}
			continue
		}
		asOf := a.now()
		for _, b := range resp {
			bids, err1 := levels(b.Bids)
			asks, err2 := levels(b.Asks)
			if err1 != nil || err2 != nil {
				continue // unreadable book: skipped (FM-4)
			}
			for _, k := range byToken[b.AssetID] {
				if book, err := market.NewBook(k, bids, asks, asOf); err == nil {
					out[k] = book
				}
			}
		}
	}
	return out, first
}

type rawLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

func levels(raw []rawLevel) ([]market.Level, error) {
	out := make([]market.Level, 0, len(raw))
	for _, l := range raw {
		p, err := market.ParseAmount(l.Price)
		if err != nil {
			return nil, err
		}
		q, err := venue.Qty(l.Size)
		if err != nil {
			return nil, err
		}
		out = append(out, market.Level{Price: p, Qty: q})
	}
	return out, nil
}
