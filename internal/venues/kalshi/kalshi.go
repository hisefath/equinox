// Package kalshi reads the US-regulated venue's open markets, series fee terms and order books (lld.md BE-3).
package kalshi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/venue"
)

// Name is the venue id used in keys.
const Name = "kalshi"

// Adapter implements venue.Venue for Kalshi's public trade API.
type Adapter struct {
	c    *fetch.Client
	base string
	now  func() time.Time

	PageSize int // markets per page; default 1000 (the API maximum)
	MaxPages int // 0 reads every page; tests and fixtures set a small number
}

var _ venue.Venue = (*Adapter)(nil)

// New returns an adapter reading base, e.g. https://api.elections.kalshi.com/trade-api/v2.
func New(c *fetch.Client, base string, now func() time.Time) *Adapter {
	if now == nil {
		now = time.Now
	}
	return &Adapter{c: c, base: strings.TrimRight(base, "/"), now: now, PageSize: 1000}
}

func (a *Adapter) Name() string { return Name }

type rawMarket struct {
	Ticker              string      `json:"ticker"`
	EventTicker         string      `json:"event_ticker"`
	Title               string      `json:"title"`
	YesSubTitle         string      `json:"yes_sub_title"`
	Status              string      `json:"status"`
	OpenTime            time.Time   `json:"open_time"`
	CloseTime           time.Time   `json:"close_time"`
	ExpectedExpiration  *time.Time  `json:"expected_expiration_time"`
	Volume24h           string      `json:"volume_24h_fp"`
	RulesPrimary        string      `json:"rules_primary"`
	RulesSecondary      string      `json:"rules_secondary"`
	EarlyCloseCondition string      `json:"early_close_condition"`
	StrikeType          string      `json:"strike_type"`
	FloorStrike         json.Number `json:"floor_strike"`
	CapStrike           json.Number `json:"cap_strike"`
}

type series struct {
	Ticker        string      `json:"ticker"`
	Category      string      `json:"category"`
	FeeType       string      `json:"fee_type"`
	FeeMultiplier json.Number `json:"fee_multiplier"`
	Sources       []struct {
		Name string `json:"name"`
	} `json:"settlement_sources"`
}

// Catalog reads open, non-combo markets page by page and returns the active ones (FR-001, FR-024).
func (a *Adapter) Catalog(ctx context.Context) ([]market.Market, venue.CatalogStats, error) {
	var (
		out    []market.Market
		stats  venue.CatalogStats
		seen   = map[string]bool{}
		cache  = map[string]*series{}
		cursor string
	)
	// One request lists every series with its fee terms (about 15,000), instead of one request per series.
	// ponytail: refetched every refresh; cache with a TTL if refreshes get frequent.
	if a.MaxPages == 0 {
		var all struct {
			Series []*series `json:"series"`
		}
		if err := a.c.GetJSON(ctx, a.base+"/series", &all); err != nil {
			var refused fetch.ErrRefused
			if ctx.Err() != nil || errors.As(err, &refused) {
				return nil, stats, err
			}
			// otherwise fall back to one request per series below
		}
		for _, s := range all.Series {
			cache[s.Ticker] = s
		}
	}
	for page := 0; a.MaxPages == 0 || page < a.MaxPages; page++ {
		q := url.Values{"status": {"open"}, "mve_filter": {"exclude"}, "limit": {fmt.Sprint(a.PageSize)}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var resp struct {
			Cursor  string            `json:"cursor"`
			Markets []json.RawMessage `json:"markets"`
		}
		if err := a.c.GetJSON(ctx, a.base+"/markets?"+q.Encode(), &resp); err != nil {
			return nil, stats, err
		}
		for _, raw := range resp.Markets {
			var r rawMarket
			if err := json.Unmarshal(raw, &r); err != nil || r.Ticker == "" || seen[r.Ticker] {
				stats.Unread()
				continue
			}
			seen[r.Ticker] = true
			vol, err := venue.Contracts(r.Volume24h)
			if err != nil {
				stats.Unread()
				continue
			}
			// Cheap pre-check, so series are fetched only for active markets.
			if ok, _ := (market.Market{Volume24h: vol, OpensAt: r.OpenTime, ClosesAt: r.CloseTime}).Active(); !ok {
				stats.Count(market.Market{Volume24h: vol, OpensAt: r.OpenTime, ClosesAt: r.CloseTime})
				continue
			}
			sTicker, _, _ := strings.Cut(r.EventTicker, "-")
			s, ok := cache[sTicker]
			if !ok {
				s = &series{}
				if err := a.c.GetJSON(ctx, a.base+"/series/"+url.PathEscape(sTicker), &struct {
					Series *series `json:"series"`
				}{s}); err != nil {
					// A refusal or a cancelled refresh is the venue's problem; anything else (e.g. a 404 for one
					// series) only costs that series its fee terms, so its markets become untradable (FR-024).
					var refused fetch.ErrRefused
					if ctx.Err() != nil || errors.As(err, &refused) {
						return nil, stats, err
					}
					s = &series{}
				}
				cache[sTicker] = s
			}
			m, err := a.toMarket(r, vol, s)
			if err != nil {
				stats.Unread()
				continue
			}
			if stats.Count(m) {
				out = append(out, m)
			}
		}
		cursor = resp.Cursor
		if cursor == "" {
			break
		}
	}
	return out, stats, nil
}

func (a *Adapter) toMarket(r rawMarket, vol int64, s *series) (market.Market, error) {
	now := a.now()
	m := market.Market{
		Key:       market.Key(Name + ":" + r.Ticker),
		Venue:     Name,
		EventID:   r.EventTicker,
		Question:  r.Title,
		Outcome:   r.YesSubTitle,
		RulesText: strings.TrimSpace(r.RulesPrimary + "\n\n" + r.RulesSecondary),
		Topic:     venue.Topic(s.Category),
		OpensAt:   r.OpenTime, ClosesAt: r.CloseTime,
		Volume24h:  vol,
		Tradable:   r.Status == "active" && r.CloseTime.After(now),
		Fee:        fee(s),
		MinQty:     1,
		BookRef:    r.Ticker,
		ReceivedAt: now,
	}
	if r.ExpectedExpiration != nil {
		m.ResolvesAt = *r.ExpectedExpiration
	}
	switch {
	case r.Status != "active":
		m.UntradableReason = "status " + r.Status
	case !m.Tradable:
		m.UntradableReason = "past close"
	}
	m.Terms = terms(r, s, m)
	return market.NewMarket(m)
}

// fee is the quadratic taker fee 0.07 × series fee_multiplier × p × (1-p), rounded up to the cent.
// ponytail: assumption pending the unread fee schedule (lld.md Q-1); event-level overrides are not read.
func fee(s *series) market.FeeTerms {
	unknown := market.FeeTerms{RatePPM: -1}
	if !strings.HasPrefix(s.FeeType, "quadratic") {
		return unknown
	}
	mult, err := market.ParseAmount(s.FeeMultiplier.String()) // multiplier in millionths
	if err != nil {
		return unknown
	}
	return market.FeeTerms{RatePPM: 70_000 * int64(mult) / 1_000_000, PExp: 1, QExp: 1, RoundTo: market.Cent}
}

// terms reads the settlement parts from Kalshi's structured fields (R03 F2).
func terms(r rawMarket, s *series, m market.Market) market.Terms {
	t := market.Terms{
		Subject:   venue.Normalize(firstNonEmpty(r.YesSubTitle, r.Title)),
		Predicate: market.Unknown, Comparator: market.Unknown,
		Source:    market.Unknown,
		WindowEnd: m.ClosesAt.Format(time.DateOnly),
		EdgeRules: venue.EdgeRules(r.RulesPrimary + "\n" + r.RulesSecondary + "\n" + r.EarlyCloseCondition),
	}
	if !m.ResolvesAt.IsZero() {
		t.WindowEnd = m.ResolvesAt.Format(time.DateOnly)
	}
	switch r.StrikeType {
	case "greater":
		t.Predicate, t.Comparator, t.Threshold = "above", ">", r.FloorStrike.String()
	case "greater_or_equal":
		t.Predicate, t.Comparator, t.Threshold = "above", ">=", r.FloorStrike.String()
	case "less":
		t.Predicate, t.Comparator, t.Threshold = "below", "<", r.CapStrike.String()
	case "less_or_equal":
		t.Predicate, t.Comparator, t.Threshold = "below", "<=", r.CapStrike.String()
	case "between":
		t.Predicate, t.Comparator, t.Threshold = "between", "range", r.FloorStrike.String()+"-"+r.CapStrike.String()
	case "structured":
		if strings.Contains(strings.ToLower(r.Title), " win") {
			t.Predicate, t.Comparator = "win", "="
		}
	}
	if t.Comparator != market.Unknown && t.Comparator != "=" && (t.Threshold == "" || strings.HasPrefix(t.Threshold, "-") || strings.HasSuffix(t.Threshold, "-")) {
		t.Predicate, t.Comparator, t.Threshold = market.Unknown, market.Unknown, "" // strike missing: unsure
	}
	var names []string
	for _, src := range s.Sources {
		if n := strings.TrimSpace(src.Name); n != "" {
			names = append(names, strings.ToLower(n))
		}
	}
	if len(names) > 0 {
		sort.Strings(names)
		t.Source = strings.Join(names, "; ")
	}
	return t
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Books reads each market's order book. Kalshi lists YES and NO bids only; a NO bid at p is a YES ask at $1 - p.
// Partial results are returned with the first error.
func (a *Adapter) Books(ctx context.Context, keys []market.Key) (map[market.Key]market.Book, error) {
	out := map[market.Key]market.Book{}
	var first error
	for _, k := range keys {
		ticker, ok := strings.CutPrefix(string(k), Name+":")
		if !ok {
			continue
		}
		var resp struct {
			Book struct {
				Yes [][]string `json:"yes_dollars"`
				No  [][]string `json:"no_dollars"`
			} `json:"orderbook_fp"`
		}
		if err := a.c.GetJSON(ctx, a.base+"/markets/"+url.PathEscape(ticker)+"/orderbook", &resp); err != nil {
			if first == nil {
				first = err
			}
			var refused fetch.ErrRefused
			if errors.As(err, &refused) || ctx.Err() != nil {
				break // honour the wait: no more requests to this venue now (FR-025)
			}
			continue
		}
		bids, err1 := levels(resp.Book.Yes, false)
		asks, err2 := levels(resp.Book.No, true)
		if err1 != nil || err2 != nil {
			continue // unreadable book: skipped (FM-4)
		}
		if b, err := market.NewBook(k, bids, asks, a.now()); err == nil {
			out[k] = b
		}
	}
	return out, first
}

func levels(pairs [][]string, flip bool) ([]market.Level, error) {
	out := make([]market.Level, 0, len(pairs))
	for _, p := range pairs {
		if len(p) != 2 {
			return nil, fmt.Errorf("level %v", p)
		}
		price, err := market.ParseAmount(p[0])
		if err != nil {
			return nil, err
		}
		qty, err := venue.Qty(p[1])
		if err != nil {
			return nil, err
		}
		if flip {
			price = market.Dollar - price
		}
		out = append(out, market.Level{Price: price, Qty: qty})
	}
	return out, nil
}
