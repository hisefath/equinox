package polymarket

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/venue"
)

const (
	gamma      = "https://gamma-api.polymarket.com"
	clob       = "https://clob.polymarket.com"
	recordings = "../../../testdata/recordings/polymarket"
)

// recordedAt is the fixed clock for replays: the time the fixtures were recorded.
var recordedAt = time.Date(2026, 10, 11, 22, 30, 0, 0, time.UTC)

func fixture(t http.RoundTripper, perHost float64) *Adapter {
	a := New(fetch.New(t, perHost, 30*time.Second, 3, nil), gamma, clob, func() time.Time { return recordedAt })
	a.PageSize, a.MaxPages = 100, 2
	return a
}

func bookKeys(ms []market.Market) []market.Key {
	var keys []market.Key
	for i := 0; i < len(ms) && i < 20; i++ {
		keys = append(keys, ms[i].Key)
	}
	return keys
}

// TestRecord refreshes the fixtures through curl (Go binaries cannot reach the internet on the dev Mac).
// Run: EQUINOX_RECORD=1 go test ./internal/venues/polymarket -run TestRecord
func TestRecord(t *testing.T) {
	if os.Getenv("EQUINOX_RECORD") == "" {
		t.Skip("set EQUINOX_RECORD=1 to re-record live fixtures")
	}
	os.RemoveAll(recordings)
	a := fixture(fetch.NewRecorder(recordings, fetch.Record, fetch.Curl{}), 2)
	ms, _, err := a.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Books(context.Background(), bookKeys(ms)); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogReplay(t *testing.T) {
	a := fixture(fetch.NewRecorder(recordings, fetch.Replay, nil), 0)
	ms, st, err := a.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Listed < 200 || st.Listed != st.Considered+st.Inactive+st.TooShort+st.Unreadable || st.Considered != int64(len(ms)) {
		t.Fatalf("stats %+v, %d markets", st, len(ms))
	}
	parsed := 0
	for _, m := range ms {
		if ok, why := m.Active(); !ok || m.Venue != Name || !strings.HasPrefix(string(m.Key), "polymarket:") || m.BookRef == "" {
			t.Fatalf("bad market %s: active=%v %s", m.Key, ok, why)
		}
		if m.Fee.Known() && (m.Fee.RoundTo != 10 || m.Fee.PExp != 1 || m.Fee.QExp != 1) {
			t.Fatalf("%s fee %+v", m.Key, m.Fee)
		}
		if m.Terms.Predicate != market.Unknown {
			parsed++
		}
	}
	if parsed == 0 {
		t.Fatal("no question's terms were read")
	}
	tradable := 0
	for _, m := range ms {
		if m.Tradable {
			tradable++
		}
	}
	t.Logf("stats %+v, tradable %d, sample terms %+v", st, tradable, ms[0].Terms)
	keys := bookKeys(ms)
	books, err := a.Books(context.Background(), keys)
	if err != nil || len(books) != len(keys) {
		t.Fatalf("books: %d of %d, err %v", len(books), len(keys), err)
	}
	for k, b := range books {
		if b.Key != k || len(b.Bids)+len(b.Asks) == 0 {
			t.Fatalf("book %s: %+v", k, b)
		}
	}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const page = `{"next_cursor":"c2","markets":[
 {"id":"1","question":"Will Bitcoin be above $120,000 on October 20?","description":"Resolves Yes if ... If cancelled, 50-50.",
  "resolutionSource":"https://binance.com","startDate":"2026-10-01T00:00:00Z","endDate":"2026-10-20T16:00:00Z",
  "outcomes":"[\"Yes\", \"No\"]","clobTokenIds":"[\"t1y\", \"t1n\"]","volume24hr":1500.5,"active":true,"closed":false,
  "acceptingOrders":true,"enableOrderBook":true,"orderMinSize":5,"feesEnabled":true,"feeType":"crypto_fees",
  "feeSchedule":{"rate":0.07,"exponent":1},"events":[{"id":"e1"}]},
 {"id":"1","question":"duplicate","outcomes":"[\"Yes\", \"No\"]","clobTokenIds":"[\"a\", \"b\"]","volume24hr":10},
 {"id":"2","question":"49ers vs. Seahawks","description":"","startDate":"2026-10-01T00:00:00Z","endDate":"2026-10-11T20:25:00Z",
  "outcomes":"[\"49ers\", \"Seahawks\"]","clobTokenIds":"[\"t2a\", \"t2b\"]","volume24hr":900,"active":true,"closed":false,
  "acceptingOrders":true,"enableOrderBook":true,"orderMinSize":5,"feesEnabled":false,"sportsMarketType":"moneyline"},
 {"id":"3","question":"Bad tokens","outcomes":"[\"Yes\", \"No\"]","clobTokenIds":"[\"only one\"]","volume24hr":800},
 {"id":"4","question":"Will the Fed cut rates?","startDate":"2026-10-01T00:00:00Z","endDate":"2026-12-01T00:00:00Z",
  "outcomes":"[\"Yes\", \"No\"]","clobTokenIds":"[\"t4y\", \"t4n\"]","volume24hr":0,"active":true,"acceptingOrders":true,"enableOrderBook":true},
 {"id":"5","question":"never reached","outcomes":"[\"Yes\", \"No\"]","clobTokenIds":"[\"x\", \"y\"]","volume24hr":0}
]}`

func synthetic(calls *[]string) *Adapter {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		*calls = append(*calls, r.Method+" "+r.URL.Path)
		body := page
		if r.URL.Query().Get("after_cursor") != "" {
			body = `{"next_cursor":"","markets":[]}`
		}
		if r.URL.Path == "/books" {
			body = `[{"asset_id":"t1y","bids":[{"price":"0.40","size":"10.7"}],"asks":[{"price":"0.42","size":"3"},{"price":"0.41","size":"2"}]},
			         {"asset_id":"t2b","bids":[{"price":"0.30","size":"1"}],"asks":[]}]`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	return New(fetch.New(rt, 0, time.Second, 0, nil), "https://g.test", "https://c.test", func() time.Time { return recordedAt })
}

func TestCatalogStopsAtZeroVolumeAndSplitsOutcomes(t *testing.T) {
	var calls []string
	ms, st, err := synthetic(&calls).Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("paged past the first zero-volume market: %v", calls)
	}
	// rows: btc(1) + 49ers(2) considered; duplicate + bad tokens unreadable; fed inactive; id 5 never listed.
	if st != (venue.CatalogStats{Listed: 6, Considered: 3, Inactive: 1, TooShort: 0, Unreadable: 2}) {
		t.Fatalf("stats %+v", st)
	}
	btc := ms[0]
	if btc.Key != "polymarket:1" || btc.BookRef != "t1y" || btc.Topic != "crypto" || btc.MinQty != 5 || btc.EventID != "e1" || !btc.Tradable {
		t.Fatalf("btc %+v", btc)
	}
	if btc.Fee != (market.FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: 10}) {
		t.Fatalf("fee %+v", btc.Fee)
	}
	if tm := btc.Terms; tm.Subject != "bitcoin" || tm.Predicate != "above" || tm.Comparator != ">" || tm.Threshold != "120000" ||
		tm.Source != "https://binance.com" || tm.WindowEnd != "2026-10-20" || strings.Join(tm.EdgeRules, ",") != "cancellation,tie 50/50" {
		t.Fatalf("terms %+v", tm)
	}
	a, b := ms[1], ms[2]
	if a.Key != "polymarket:2:0" || b.Key != "polymarket:2:1" || b.BookRef != "t2b" || b.Outcome != "Seahawks" ||
		b.Terms.Subject != "seahawks" || b.Terms.Predicate != "win" || b.Topic != "sports" || b.Fee.RatePPM != 0 || !b.Fee.Known() {
		t.Fatalf("outcome rows %+v / %+v", a, b)
	}
	if b.Tradable || b.UntradableReason != "past its end" {
		t.Fatalf("game ended before recordedAt: tradable=%v %q", b.Tradable, b.UntradableReason)
	}
}

func TestBooksUseCatalogTokens(t *testing.T) {
	var calls []string
	a := synthetic(&calls)
	if _, _, err := a.Catalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	books, err := a.Books(context.Background(), []market.Key{"polymarket:1", "polymarket:2:1", "polymarket:unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 || calls[len(calls)-1] != "POST /books" {
		t.Fatalf("books %v calls %v", books, calls)
	}
	b := books["polymarket:1"]
	if b.Bids[0] != (market.Level{Price: 400_000, Qty: 10}) || b.Asks[0] != (market.Level{Price: 410_000, Qty: 2}) {
		t.Fatalf("book %+v", b)
	}
}

func TestTermsPatterns(t *testing.T) {
	for q, want := range map[string][4]string{
		"Will ETH hit $5k by December 31?":               {"eth", "above", ">=", "5000"},
		"Will the S&P 500 close below 6,000 on Friday?":  {"the s&p 500", "below", "<", "6000"},
		"Will Bitcoin be between $100,000 and $110,000?": {"bitcoin", "between", "range", "100000-110000"},
		"Will Jane Doe win the 2026 Senate race?":        {"jane doe", "win", "=", ""},
		"Bitcoin Up or Down - October 11, 6PM ET":        {"bitcoin up or down - october 11, 6pm et", market.Unknown, market.Unknown, ""},
	} {
		got := terms(rawMarket{Question: q}, "Yes", true)
		if [4]string{got.Subject, got.Predicate, got.Comparator, got.Threshold} != want {
			t.Errorf("%q: got %v want %v", q, [4]string{got.Subject, got.Predicate, got.Comparator, got.Threshold}, want)
		}
		if got.Source != market.Unknown {
			t.Errorf("%q: source %q, want unknown when the venue names none", q, got.Source)
		}
	}
}

func TestFeeTerms(t *testing.T) {
	for in, want := range map[string]market.FeeTerms{
		`{"feesEnabled":true,"feeSchedule":{"rate":0.03,"exponent":1}}`: {RatePPM: 30_000, PExp: 1, QExp: 1, RoundTo: 10},
		`{"feesEnabled":false,"feeSchedule":null}`:                      {RatePPM: 0, PExp: 1, QExp: 1, RoundTo: 10},
		`{"feesEnabled":true,"feeSchedule":null}`:                       {RatePPM: -1},
		`{"feesEnabled":true,"feeSchedule":{"rate":0.03,"exponent":2}}`: {RatePPM: -1},
	} {
		var r rawMarket
		if err := json.Unmarshal([]byte(in), &r); err != nil {
			t.Fatal(err)
		}
		if got := fee(r); got != want {
			t.Errorf("%s: got %+v want %+v", in, got, want)
		}
	}
	// R04 F8: 100 shares at $0.50, rate 0.07 -> $1.75; at $0.10 -> $0.63.
	f := market.FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: 10}
	if a, b := f.TakerFee(100, 500_000), f.TakerFee(100, 100_000); a != 1_750_000 || b != 630_000 {
		t.Fatalf("worked examples: %s, %s", a, b)
	}
}
