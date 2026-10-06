package kalshi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
)

// Payloads trimmed from live responses captured on 2026-10-06 (research/raw/).
const seriesJSON = `{"series":[
 {"ticker":"KXFEDDECISION","category":"Economics","fee_type":"quadratic_with_maker_fees","fee_multiplier":1},
 {"ticker":"KXMLBGAME","category":"Sports","fee_type":"quadratic_with_maker_fees","fee_multiplier":0.5},
 {"ticker":"KXODD","category":"Exotics","fee_type":"flat","fee_multiplier":1}]}`

const page1 = `{"cursor":"page2","events":[{"event_ticker":"KXFEDDECISION-26OCT","series_ticker":"KXFEDDECISION","title":"Fed decision in Oct 2026?",
 "markets":[
  {"ticker":"KXFEDDECISION-26OCT-H0","market_type":"binary","title":"Will the Federal Reserve Hike rates by 0bps at their October 2026 meeting?",
   "yes_sub_title":"Fed maintains rate","status":"active","close_time":"2026-10-28T17:59:00Z","expected_expiration_time":"2026-10-28T18:05:00Z",
   "occurrence_datetime":null,"yes_bid_dollars":"0.7900","yes_ask_dollars":"0.8000","yes_bid_size_fp":"24154.29","yes_ask_size_fp":"49649.18",
   "rules_primary":"If the Federal Reserve does a Hike of 0bps on October 28, 2026, then the market resolves to Yes."},
  {"ticker":"KXFEDDECISION-26OCT-C26","market_type":"binary","title":"Will the Federal Reserve Cut rates by >25bps?","yes_sub_title":"Cut >25bps",
   "status":"finalized","close_time":"2026-10-28T17:59:00Z","yes_bid_dollars":"0.0000","yes_ask_dollars":"1.0000","yes_bid_size_fp":"0.00","yes_ask_size_fp":"0.00"},
  {"ticker":"KXFEDDECISION-26OCT-SCALAR","market_type":"scalar","title":"Rate level","status":"active"},
  {"ticker":"KXMVECROSS-1","market_type":"binary","title":"combo","status":"active","mve_collection_ticker":"KXMVECROSSCATEGORY-R"},
  {"ticker":"KXFEDDECISION-26OCT-BAD","market_type":"binary","title":"odd timestamp","status":"active","close_time":"soon"}]}]}`

const page2 = `{"cursor":"","events":[
 {"event_ticker":"KXMLBGAME-26OCT07TBNYY","series_ticker":"KXMLBGAME","title":"Tampa Bay vs New York Y (Oct 7)","fee_multiplier_override":1,
  "markets":[{"ticker":"KXMLBGAME-26OCT07TBNYY-TB","market_type":"binary","title":"Tampa Bay vs New York Y Winner?","yes_sub_title":"Tampa Bay","status":"active"}]},
 {"event_ticker":"KXODD-1","series_ticker":"KXODD","title":"Odd fees","markets":[{"ticker":"KXODD-1-A","market_type":"binary","title":"Odd?","status":"active"}]}]}`

func server(t *testing.T, routes map[string]string) *Adapter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if c := r.URL.Query().Get("cursor"); c != "" {
			key += "?cursor=" + c
		}
		body, ok := routes[key]
		if !ok {
			http.Error(w, `{"error":"no route `+key+`"}`, http.StatusInternalServerError)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c := fetch.New(nil, 0)
	c.Backoff = time.Millisecond
	a := New(c, 0)
	a.Base = srv.URL
	return a
}

func TestMarketsNormalizesAndCountsSkips(t *testing.T) {
	a := server(t, map[string]string{"/series": seriesJSON, "/events": page1, "/events?cursor=page2": page2})
	ms, st, err := a.Markets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Seen != 7 || st.Kept != 5 || st.Skipped["not a binary market"] != 1 || st.Skipped["multivariate combo"] != 1 {
		t.Fatalf("stats = %+v", st)
	}
	byID := map[string]market.Market{}
	for _, m := range ms {
		byID[m.ID] = m
	}

	h0 := byID["KXFEDDECISION-26OCT-H0"]
	if h0.Event != "Fed decision in Oct 2026?" || h0.Outcome != "Fed maintains rate" || h0.Untradable != "" || h0.Category != "Economics" {
		t.Errorf("H0 = %+v", h0)
	}
	if h0.YesBid != 790_000 || h0.YesAsk != 800_000 || !h0.Resolves.Equal(time.Date(2026, 10, 28, 18, 5, 0, 0, time.UTC)) {
		t.Errorf("H0 quotes/resolves = %v %v %v", h0.YesBid, h0.YesAsk, h0.Resolves)
	}
	if h0.Fee != (market.FeeCurve{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: market.Cent}) {
		t.Errorf("H0 fee = %+v", h0.Fee)
	}
	// Empty sides: sizes are zero, so the sentinel prices (0.0000 / 1.0000) must not become quotes.
	if c26 := byID["KXFEDDECISION-26OCT-C26"]; c26.YesBid != 0 || c26.YesAsk != 0 || c26.Untradable != "status finalized" {
		t.Errorf("C26 = %+v", c26)
	}
	// One unparseable timestamp leaves that field unknown; it does not fail the page.
	if bad := byID["KXFEDDECISION-26OCT-BAD"]; !bad.Close.IsZero() {
		t.Errorf("bad close = %v", bad.Close)
	}
	// Event-level fee override (x1) beats the series multiplier (x0.5).
	if tb := byID["KXMLBGAME-26OCT07TBNYY-TB"]; tb.Fee.RatePPM != 70_000 {
		t.Errorf("override ignored: %+v", tb.Fee)
	}
	if odd := byID["KXODD-1-A"]; odd.Untradable != "unsupported fee type flat" {
		t.Errorf("flat fee should make the market untradable: %+v", odd)
	}
}

func TestMarketsFailsWholeRefreshOnVenueError(t *testing.T) {
	a := server(t, map[string]string{"/series": seriesJSON}) // /events -> 500
	if _, _, err := a.Markets(context.Background()); err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("want a 500 error, got %v", err)
	}
	a = server(t, map[string]string{"/series": `{"series": [`})
	if _, _, err := a.Markets(context.Background()); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("want malformed JSON error, got %v", err)
	}
}

func TestBooksDeriveYesAsksFromNoBids(t *testing.T) {
	a := server(t, map[string]string{"/markets/orderbooks": `{"orderbooks":[{"ticker":"T1","orderbook_fp":{
		"yes_dollars":[["0.4100","10.00"],["0.4200","103.00"]],
		"no_dollars":[["0.5600","5.50"],["0.5700","72612.49"],["bad","1"]]}}]}`})
	at := time.Date(2026, 10, 6, 4, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return at }
	books, err := a.Books(context.Background(), []market.Market{{Venue: "kalshi", ID: "T1", BookRef: "T1"}})
	if err != nil {
		t.Fatal(err)
	}
	b := books["kalshi:T1"].Normalize()
	// Best YES bid 0.42; best NO bid 0.57 -> best YES ask 0.43 (size 72612, floored); malformed level dropped.
	if b.Bids[0] != (market.Level{Price: 420_000, Qty: 103}) || b.Asks[0] != (market.Level{Price: 430_000, Qty: 72612}) || len(b.Asks) != 2 {
		t.Fatalf("book = %+v", b)
	}
	if !b.AsOf.Equal(at) || b.Validate() != nil {
		t.Errorf("as_of=%v validate=%v", b.AsOf, b.Validate())
	}
}
