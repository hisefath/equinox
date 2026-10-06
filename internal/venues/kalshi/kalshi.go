// Package kalshi adapts Kalshi's public Trade API v2 to the canonical market model.
//
// Everything Kalshi-specific lives here: endpoints, field names, the "bids only" order book, the
// quadratic fee schedule and its series/event overrides. Nothing downstream knows Kalshi exists.
package kalshi

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
)

// DefaultBase is Kalshi's public API (market data needs no authentication).
const DefaultBase = "https://api.elections.kalshi.com/trade-api/v2"

// Adapter implements ingest.Venue for Kalshi.
type Adapter struct {
	Base     string
	HTTP     *fetch.Client
	MaxPages int              // /events pages of 200 events each; 0 = no cap
	Now      func() time.Time // stamps books, which Kalshi doesn't timestamp; injectable for replay

	mu       sync.Mutex
	series   map[string]series // fee schedules change rarely and the list is ~19 MB: cache it
	seriesAt time.Time         // and refresh it after SeriesTTL
}

// SeriesTTL bounds how long cached series metadata (category, fee schedule) is trusted.
const SeriesTTL = time.Hour

// New returns an adapter with Kalshi's documented Basic-tier pacing (~20 req/s; we stay at 10).
func New(c *fetch.Client, maxPages int) *Adapter {
	return &Adapter{Base: DefaultBase, HTTP: c, MaxPages: maxPages, Now: time.Now}
}

func (a *Adapter) Name() string { return "kalshi" }

type series struct {
	Ticker        string   `json:"ticker"`
	Category      string   `json:"category"`
	FeeType       string   `json:"fee_type"`
	FeeMultiplier *float64 `json:"fee_multiplier"`
}

type event struct {
	EventTicker  string   `json:"event_ticker"`
	SeriesTicker string   `json:"series_ticker"`
	Title        string   `json:"title"`
	Category     string   `json:"category"`
	FeeTypeOver  string   `json:"fee_type_override"`
	FeeMultOver  *float64 `json:"fee_multiplier_override"`
	Markets      []mkt    `json:"markets"`
}

type mkt struct {
	Ticker         string `json:"ticker"`
	MarketType     string `json:"market_type"`
	Title          string `json:"title"`
	YesSubTitle    string `json:"yes_sub_title"`
	Status         string `json:"status"`
	CloseTime      string `json:"close_time"` // times as strings: one odd value must not fail a page
	ExpectedExpiry string `json:"expected_expiration_time"`
	Occurrence     string `json:"occurrence_datetime"`
	YesBid         string `json:"yes_bid_dollars"`
	YesAsk         string `json:"yes_ask_dollars"`
	YesBidSize     string `json:"yes_bid_size_fp"`
	YesAskSize     string `json:"yes_ask_size_fp"`
	RulesPrimary   string `json:"rules_primary"`
	MVECollection  string `json:"mve_collection_ticker"`
}

// Markets crawls open events with their nested markets. /events excludes multivariate combo markets
// by design, which are not single propositions and can never be equivalent to another venue's market.
func (a *Adapter) Markets(ctx context.Context) ([]market.Market, ingest.Stats, error) {
	var st ingest.Stats
	fees, err := a.seriesIndex(ctx)
	if err != nil {
		return nil, st, fmt.Errorf("kalshi series: %w", err)
	}
	var out []market.Market
	cursor := ""
	for page := 0; a.MaxPages == 0 || page < a.MaxPages; page++ {
		q := url.Values{"status": {"open"}, "with_nested_markets": {"true"}, "limit": {"200"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var resp struct {
			Events []event `json:"events"`
			Cursor *string `json:"cursor"`
		}
		if err := a.HTTP.GetJSON(ctx, a.Base+"/events?"+q.Encode(), &resp); err != nil {
			return nil, st, fmt.Errorf("kalshi events page %d: %w", page, err)
		}
		for _, ev := range resp.Events {
			for _, m := range ev.Markets {
				st.Seen++
				if cm, skip := normalize(ev, m, fees[ev.SeriesTicker]); skip != "" {
					st.Skip(skip)
				} else {
					out = append(out, cm)
					st.Kept++
				}
			}
		}
		if resp.Cursor == nil || *resp.Cursor == "" {
			break
		}
		if *resp.Cursor == cursor {
			return nil, st, fmt.Errorf("kalshi events page %d: cursor did not advance", page)
		}
		cursor = *resp.Cursor
	}
	return out, st, nil
}

func (a *Adapter) seriesIndex(ctx context.Context) (map[string]series, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.series != nil && a.Now().Sub(a.seriesAt) < SeriesTTL {
		return a.series, nil
	}
	var resp struct {
		Series []series `json:"series"`
	}
	if err := a.HTTP.GetJSON(ctx, a.Base+"/series", &resp); err != nil {
		return nil, err
	}
	a.series, a.seriesAt = make(map[string]series, len(resp.Series)), a.Now()
	for _, s := range resp.Series {
		a.series[s.Ticker] = s
	}
	return a.series, nil
}

// ParseMarket normalizes one market from captured /events JSON (the event object and one of its
// markets), for tests and tools that work offline. Fees are unknown without the series, so the market
// comes back untradable; matching doesn't need fees.
func ParseMarket(eventJSON, marketJSON []byte) (market.Market, string, error) {
	var ev event
	var m mkt
	if err := json.Unmarshal(eventJSON, &ev); err != nil {
		return market.Market{}, "", fmt.Errorf("kalshi event: %w", err)
	}
	if err := json.Unmarshal(marketJSON, &m); err != nil {
		return market.Market{}, "", fmt.Errorf("kalshi market: %w", err)
	}
	cm, skip := normalize(ev, m, series{})
	return cm, skip, nil
}

// normalize maps one Kalshi market into the canonical model, or returns why it was skipped.
func normalize(ev event, m mkt, s series) (market.Market, string) {
	switch {
	case m.MarketType != "binary":
		return market.Market{}, "not a binary market"
	case m.MVECollection != "":
		return market.Market{}, "multivariate combo"
	case m.Ticker == "":
		return market.Market{}, "missing ticker"
	}
	question := m.Title // deprecated by Kalshi but still the most complete sentence
	if question == "" {
		question = ev.Title
	}
	cm := market.Market{
		Venue:    "kalshi",
		ID:       m.Ticker,
		EventID:  ev.EventTicker,
		Event:    ev.Title,
		Question: question,
		Outcome:  m.YesSubTitle,
		Rules:    m.RulesPrimary,
		Category: cmp.Or(s.Category, ev.Category),
		Close:    market.ParseTime(m.CloseTime),
		URL:      "https://kalshi.com/markets/" + strings.ToLower(ev.SeriesTicker),
		BookRef:  m.Ticker,
		MinQty:   1,
		YesBid:   quote(m.YesBid, m.YesBidSize),
		YesAsk:   quote(m.YesAsk, m.YesAskSize),
	}
	// When will we know? occurrence → expected expiration. close_time is only a trading bound
	// (sports markets close days after the game to allow for rescheduling).
	for _, t := range []string{m.Occurrence, m.ExpectedExpiry} {
		if cm.Resolves = market.ParseTime(t); !cm.Resolves.IsZero() {
			break
		}
	}
	if m.Status != "active" {
		cm.Untradable = "status " + m.Status
	}
	feeType, mult := s.FeeType, s.FeeMultiplier
	if ev.FeeTypeOver != "" {
		feeType = ev.FeeTypeOver
	}
	if ev.FeeMultOver != nil {
		mult = ev.FeeMultOver
	}
	fee, why := feeCurve(feeType, mult)
	cm.Fee = fee
	if why != "" && cm.Untradable == "" {
		cm.Untradable = why
	}
	return cm, ""
}

// feeCurve maps Kalshi's fee schedule into a canonical curve. Taker fees on every quadratic variant are
// roundup(multiplier × 0.07 × C × P × (1−P)) to the cent; the variants differ only in maker fees, which a
// taker simulation never pays. Anything else ("flat", future types) can't be priced, so it can't be routed.
func feeCurve(feeType string, mult *float64) (market.FeeCurve, string) {
	switch feeType {
	case "quadratic", "quadratic_with_maker_fees", "quadratic_with_combo_maker_fees":
		m := 1.0
		if mult != nil {
			m = *mult
		}
		return market.FeeCurve{RatePPM: int64(math.Round(m * 70_000)), PExp: 1, QExp: 1, RoundTo: market.Cent}, ""
	case "":
		return market.FeeCurve{}, "fee schedule unknown"
	default:
		return market.FeeCurve{}, "unsupported fee type " + feeType
	}
}

// quote returns a top-of-book price only when that side actually has size: Kalshi's empty-side price
// sentinels are inconsistent (both 0.0000 and 1.0000 appear), sizes are not.
func quote(price, size string) market.Amount {
	if q, err := market.ParseQty(size); err != nil || q == 0 {
		return 0
	}
	p, err := market.ParseAmount(price)
	if err != nil {
		return 0
	}
	return p
}

// Books fetches order books 100 tickers per call. Kalshi publishes bids only: YES bids, and NO bids
// that are YES asks in disguise (a NO bid at q is an offer to sell YES at 1−q).
func (a *Adapter) Books(ctx context.Context, ms []market.Market) (map[string]market.Book, error) {
	out := map[string]market.Book{}
	var errs []error
	for start := 0; start < len(ms); start += 100 {
		chunk := ms[start:min(start+100, len(ms))]
		q := url.Values{}
		for _, m := range chunk {
			q.Add("tickers", m.BookRef) // must be repeated, not comma-joined
		}
		var resp struct {
			Books []struct {
				Ticker string `json:"ticker"`
				Book   struct {
					Yes [][2]string `json:"yes_dollars"`
					No  [][2]string `json:"no_dollars"`
				} `json:"orderbook_fp"`
			} `json:"orderbooks"`
		}
		if err := a.HTTP.GetJSON(ctx, a.Base+"/markets/orderbooks?"+q.Encode(), &resp); err != nil {
			errs = append(errs, fmt.Errorf("kalshi orderbooks %d-%d: %w", start, start+len(chunk), err))
			continue // one failed chunk must not cost the others
		}
		at := a.Now() // Kalshi books carry no timestamp; time of receipt is the honest bound
		for _, b := range resp.Books {
			book := market.Book{MarketKey: "kalshi:" + b.Ticker, AsOf: at}
			book.Bids = levels(b.Book.Yes, false)
			book.Asks = levels(b.Book.No, true)
			out[book.MarketKey] = book
		}
	}
	return out, errors.Join(errs...)
}

// levels parses [price, size] pairs, skipping malformed ones; complement turns NO bids into YES asks.
func levels(raw [][2]string, complement bool) []market.Level {
	out := make([]market.Level, 0, len(raw))
	for _, r := range raw {
		p, perr := market.ParseAmount(r[0])
		q, qerr := market.ParseQty(r[1])
		if perr != nil || qerr != nil {
			continue
		}
		if complement {
			p = market.Dollar - p
		}
		out = append(out, market.Level{Price: p, Qty: q})
	}
	return out
}
