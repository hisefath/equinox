// Package polymarket adapts Polymarket's public Gamma (metadata) and CLOB (order book) APIs to the
// canonical market model.
//
// Polymarket-specific concerns live only here: JSON-in-a-string fields, outcome tokens, the
// worst-first book ordering, negative-risk placeholders, and the feeSchedule curve.
package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
)

const (
	GammaBase = "https://gamma-api.polymarket.com"
	ClobBase  = "https://clob.polymarket.com"
)

// Adapter implements ingest.Venue for Polymarket.
type Adapter struct {
	Gamma, Clob string
	HTTP        *fetch.Client
	MaxPages    int              // keyset pages of 100 markets, most active (24h volume) first; 0 = no cap
	Now         func() time.Time // injectable for tests; markets past their end date are not tradable
}

// New returns an adapter. Polymarket lists a very large number of open markets (research estimated
// 70-85k; a verification crawl estimated ~257k), so ingestion is scoped to the most actively traded
// ones: liquid markets are where routing between venues matters.
func New(c *fetch.Client, maxPages int) *Adapter {
	return &Adapter{Gamma: GammaBase, Clob: ClobBase, HTTP: c, MaxPages: maxPages, Now: time.Now}
}

func (a *Adapter) Name() string { return "polymarket" }

type gammaMarket struct {
	ID              string          `json:"id"`
	Question        string          `json:"question"`
	Description     string          `json:"description"`
	GroupItemTitle  string          `json:"groupItemTitle"`
	Outcomes        json.RawMessage `json:"outcomes"`     // a JSON array encoded inside a string
	ClobTokenIDs    json.RawMessage `json:"clobTokenIds"` // ditto
	PositionIDs     json.RawMessage `json:"positionIds"`  // a real array (protocol v2)
	Version         string          `json:"version"`
	EndDate         string          `json:"endDate"`
	GameStartTime   string          `json:"gameStartTime"` // sports only, "2026-10-11 12:45:00+00"
	Active          bool            `json:"active"`
	Closed          bool            `json:"closed"`
	EnableOrderBook bool            `json:"enableOrderBook"`
	AcceptingOrders bool            `json:"acceptingOrders"`
	UMAStatus       string          `json:"umaResolutionStatus"`
	NegRiskOther    bool            `json:"negRiskOther"`
	OrderMinSize    float64         `json:"orderMinSize"`
	BestBid         *float64        `json:"bestBid"`
	BestAsk         *float64        `json:"bestAsk"`
	FeesEnabled     bool            `json:"feesEnabled"`
	FeeSchedule     *struct {
		Rate     float64 `json:"rate"`
		Exponent int     `json:"exponent"`
	} `json:"feeSchedule"`
	Events []gammaEvent `json:"events"`
}

type gammaEvent struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// ParseMarket normalizes one captured Gamma market (and its parent event, which markets nested inside
// an /events response don't repeat), for tests and tools that work offline.
func ParseMarket(eventJSON, marketJSON []byte, now time.Time) ([]market.Market, string, error) {
	var m gammaMarket
	if err := json.Unmarshal(marketJSON, &m); err != nil {
		return nil, "", fmt.Errorf("polymarket market: %w", err)
	}
	if len(m.Events) == 0 && len(eventJSON) > 0 {
		var ev gammaEvent
		if err := json.Unmarshal(eventJSON, &ev); err != nil {
			return nil, "", fmt.Errorf("polymarket event: %w", err)
		}
		m.Events = []gammaEvent{ev}
	}
	cms, skip := normalize(m, now)
	return cms, skip, nil
}

// Markets pages /markets/keyset ordered by 24h volume.
func (a *Adapter) Markets(ctx context.Context) ([]market.Market, ingest.Stats, error) {
	var (
		st     ingest.Stats
		out    []market.Market
		cursor string
		now    = a.Now()
	)
	for page := 0; a.MaxPages == 0 || page < a.MaxPages; page++ {
		q := url.Values{"closed": {"false"}, "limit": {"100"}, "order": {"volume24hr"}, "ascending": {"false"}}
		if cursor != "" {
			q.Set("after_cursor", cursor)
		}
		var resp struct {
			Markets    []gammaMarket `json:"markets"`
			NextCursor string        `json:"next_cursor"`
		}
		if err := a.HTTP.GetJSON(ctx, a.Gamma+"/markets/keyset?"+q.Encode(), &resp); err != nil {
			return nil, st, fmt.Errorf("polymarket markets page %d: %w", page, err)
		}
		for _, m := range resp.Markets {
			st.Seen++
			cms, skip := normalize(m, now)
			if skip != "" {
				st.Skip(skip)
				continue
			}
			out = append(out, cms...)
			st.Kept++
		}
		if resp.NextCursor == "" || len(resp.Markets) == 0 {
			break
		}
		if resp.NextCursor == cursor {
			return nil, st, fmt.Errorf("polymarket markets page %d: cursor did not advance", page)
		}
		cursor = resp.NextCursor
	}
	return out, st, nil
}

// normalize maps one Gamma market into one or two canonical binary markets, or says why it was skipped.
//
// A ["Yes","No"] market is one proposition. A two-outcome market with named outcomes (["Rays","Yankees"],
// ["Over","Under"]) is two propositions, "Rays" and "Yankees", each priced by its own outcome token's
// book. That makes "Kalshi: will the Rays win?" a plain YES-to-YES match, with no polarity logic
// anywhere downstream.
func normalize(m gammaMarket, now time.Time) ([]market.Market, string) {
	switch {
	case !m.Active || m.Closed:
		return nil, "inactive or closed"
	case !m.EnableOrderBook:
		return nil, "no order book"
	case m.NegRiskOther:
		return nil, "negative-risk 'Other' placeholder"
	case m.Question == "":
		return nil, "missing question"
	}
	outcomes, tokens := stringList(m.Outcomes), stringList(m.ClobTokenIDs)
	if m.Version == "v2" {
		tokens = stringList(m.PositionIDs)
	}
	switch {
	case len(outcomes) != 2:
		return nil, "not exactly two outcomes"
	case len(tokens) != 2 || tokens[0] == "" || tokens[1] == "":
		return nil, "missing outcome token ids"
	}
	end := market.ParseTime(m.EndDate)

	base := market.Market{
		Venue:   "polymarket",
		ID:      m.ID,
		EventID: m.ID,
		Rules:   m.Description,
		MinQty:  int64(math.Ceil(m.OrderMinSize)),
		Fee:     feeCurve(m),
	}
	base.Close, base.Resolves = end, end
	if t := market.ParseTime(m.GameStartTime); !t.IsZero() {
		base.Resolves = t
	}
	if len(m.Events) > 0 {
		base.EventID, base.Event = m.Events[0].ID, m.Events[0].Title
		base.URL = "https://polymarket.com/event/" + m.Events[0].Slug
	}
	switch {
	case !m.AcceptingOrders:
		base.Untradable = "not accepting orders"
	case m.UMAStatus != "":
		base.Untradable = "resolution " + m.UMAStatus
	case !end.IsZero() && end.Before(now):
		base.Untradable = "past scheduled end, awaiting resolution"
	}
	bid, ask := price(m.BestBid), price(m.BestAsk)

	if strings.EqualFold(outcomes[0], "yes") && strings.EqualFold(outcomes[1], "no") {
		yes := base
		yes.Question, yes.Outcome, yes.BookRef, yes.YesBid, yes.YesAsk = m.Question, m.GroupItemTitle, tokens[0], bid, ask
		return []market.Market{yes}, ""
	}
	out := make([]market.Market, 2)
	for i := range 2 {
		cm := base
		cm.ID = m.ID + ":" + strconv.Itoa(i)
		cm.Question, cm.Outcome, cm.BookRef = m.Question, outcomes[i], tokens[i]
		if i == 0 {
			cm.YesBid, cm.YesAsk = bid, ask
		} else if bid > 0 || ask > 0 { // outcome 1 is the mirror of outcome 0
			cm.YesBid, cm.YesAsk = complement(ask), complement(bid)
		}
		out[i] = cm
	}
	return out, ""
}

// feeCurve reads the market's own feeSchedule (the docs say to: category tables lag, e.g. older sports
// markets still charge 0.03). fee = C × rate × (p(1−p))^exponent, taker only, 5-decimal rounding.
// stringList decodes Gamma's list fields, which arrive either as a JSON array or as a JSON array encoded
// inside a string (outcomes, clobTokenIds). Anything else yields nil.
func stringList(raw json.RawMessage) []string {
	var out []string
	if json.Unmarshal(raw, &out) == nil {
		return out
	}
	var inner string
	if json.Unmarshal(raw, &inner) == nil && json.Unmarshal([]byte(inner), &out) == nil {
		return out
	}
	return nil
}

func feeCurve(m gammaMarket) market.FeeCurve {
	if !m.FeesEnabled || m.FeeSchedule == nil || m.FeeSchedule.Rate <= 0 {
		return market.FeeCurve{}
	}
	e := m.FeeSchedule.Exponent
	return market.FeeCurve{RatePPM: int64(math.Round(m.FeeSchedule.Rate * 1e6)), PExp: e, QExp: e, RoundTo: 10}
}

func price(f *float64) market.Amount {
	if f == nil || *f <= 0 || *f >= 1 {
		return 0
	}
	return market.Amount(math.Round(*f * float64(market.Dollar)))
}

func complement(p market.Amount) market.Amount {
	if p == 0 {
		return 0
	}
	return market.Dollar - p
}

type clobBook struct {
	AssetID string `json:"asset_id"`
	Bids    []struct {
		Price string `json:"price"`
		Size  string `json:"size"`
	} `json:"bids"`
	Asks []struct {
		Price string `json:"price"`
		Size  string `json:"size"`
	} `json:"asks"`
}

// Books fetches outcome-token books 500 per POST /books call. Results come back in arbitrary order and
// silently omit tokens without a book, so they are keyed by asset id.
func (a *Adapter) Books(ctx context.Context, ms []market.Market) (map[string]market.Book, error) {
	byToken := map[string][]market.Market{}
	var tokens []string
	for _, m := range ms {
		if _, seen := byToken[m.BookRef]; !seen {
			tokens = append(tokens, m.BookRef)
		}
		byToken[m.BookRef] = append(byToken[m.BookRef], m)
	}
	out := map[string]market.Book{}
	var errs []error
	for start := 0; start < len(tokens); start += 500 {
		var req []map[string]string
		for _, t := range tokens[start:min(start+500, len(tokens))] {
			req = append(req, map[string]string{"token_id": t})
		}
		var resp []clobBook
		if err := a.HTTP.PostJSON(ctx, a.Clob+"/books", req, &resp); err != nil {
			errs = append(errs, fmt.Errorf("polymarket books %d-%d: %w", start, min(start+500, len(tokens)), err))
			continue // one failed chunk must not cost the others
		}
		// The book's own timestamp is when it last changed, not when it was read: a quiet but valid
		// book can be minutes old. Staleness is about our copy, so stamp it with receipt time.
		at := a.Now()
		for _, b := range resp {
			ms, ok := byToken[b.AssetID]
			if !ok {
				continue
			}
			book := market.Book{AsOf: at}
			for _, l := range b.Bids {
				book.Bids = appendLevel(book.Bids, l.Price, l.Size)
			}
			for _, l := range b.Asks {
				book.Asks = appendLevel(book.Asks, l.Price, l.Size)
			}
			for _, m := range ms {
				book.MarketKey = m.Key()
				out[m.Key()] = book
			}
		}
	}
	return out, errors.Join(errs...)
}

func appendLevel(ls []market.Level, price, size string) []market.Level {
	p, perr := market.ParseAmount(price)
	q, qerr := market.ParseQty(size)
	if perr != nil || qerr != nil {
		return ls
	}
	return append(ls, market.Level{Price: p, Qty: q})
}
