package kalshi

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
)

const (
	live       = "https://api.elections.kalshi.com/trade-api/v2"
	recordings = "../../../testdata/recordings/kalshi"
)

// recordedAt is the fixed clock for replays: the time the fixtures were recorded.
var recordedAt = time.Date(2026, 10, 11, 22, 30, 0, 0, time.UTC)

// fixture builds the adapter exactly as the recording ran, so every URL matches a saved file.
func fixture(t http.RoundTripper, perHost float64) *Adapter {
	a := New(fetch.New(t, perHost, 30*time.Second, 3, nil), live, func() time.Time { return recordedAt })
	a.PageSize, a.MaxPages = 1000, 2
	return a
}

// TestRecord refreshes the fixtures through curl (Go binaries cannot reach the internet on the dev Mac).
// Run: EQUINOX_RECORD=1 go test ./internal/venues/kalshi -run TestRecord
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

func bookKeys(ms []market.Market) []market.Key {
	var keys []market.Key
	for i := 0; i < len(ms) && i < 20; i++ {
		keys = append(keys, ms[i].Key)
	}
	return keys
}

func TestCatalogReplay(t *testing.T) {
	a := fixture(fetch.NewRecorder(recordings, fetch.Replay, nil), 0)
	ms, st, err := a.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Listed != 2000 || st.Listed != st.Considered+st.Inactive+st.TooShort+st.Unreadable || st.Considered != int64(len(ms)) || len(ms) == 0 {
		t.Fatalf("stats %+v, %d markets", st, len(ms))
	}
	for _, m := range ms {
		if ok, why := m.Active(); !ok || m.Venue != Name || !strings.HasPrefix(string(m.Key), "kalshi:") || m.BookRef == "" {
			t.Fatalf("bad market %s: active=%v %s", m.Key, ok, why)
		}
		if m.Tradable && (!m.Fee.Known() || m.Fee.RoundTo != market.Cent || m.Fee.PExp != 1 || m.Fee.QExp != 1) {
			t.Fatalf("%s fee %+v", m.Key, m.Fee)
		}
		if m.Terms.Comparator == ">" && m.Terms.Threshold == "" {
			t.Fatalf("%s: comparator without threshold", m.Key)
		}
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
		if b.Key != k || !b.AsOf.Equal(recordedAt) {
			t.Fatalf("book %s: %+v", k, b)
		}
	}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func serve(routes map[string]string) http.RoundTripper {
	return rtFunc(func(r *http.Request) (*http.Response, error) {
		best := ""
		for prefix := range routes { // longest matching prefix wins
			if strings.HasPrefix(r.URL.Path, prefix) && len(prefix) > len(best) {
				best = prefix
			}
		}
		if best != "" {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(routes[best]))}, nil
		}
		return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
}

const page = `{"cursor":"","markets":[
 {"ticker":"KXBTC-1-T100","event_ticker":"KXBTC-1","title":"Bitcoin above 100,000?","yes_sub_title":"$100,000 or above","status":"active",
  "open_time":"2026-10-01T00:00:00Z","close_time":"2026-10-20T00:00:00Z","expected_expiration_time":"2026-10-20T01:00:00Z",
  "volume_24h_fp":"10.50","rules_primary":"If the price is above 100000, Yes.","rules_secondary":"If cancelled, fair price.",
  "strike_type":"greater","floor_strike":100000},
 {"ticker":"KXBTC-1-T100","event_ticker":"KXBTC-1","title":"duplicate","status":"active","open_time":"2026-10-01T00:00:00Z","close_time":"2026-10-20T00:00:00Z","volume_24h_fp":"1"},
 {"ticker":"KXBTC-1-QUIET","event_ticker":"KXBTC-1","title":"quiet","status":"active","open_time":"2026-10-01T00:00:00Z","close_time":"2026-10-20T00:00:00Z","volume_24h_fp":"0.00"},
 {"ticker":"KXBTC-1-SHORT","event_ticker":"KXBTC-1","title":"short","status":"active","open_time":"2026-10-11T00:00:00Z","close_time":"2026-10-11T00:15:00Z","volume_24h_fp":"3"},
 {"ticker":"KXBTC-1-BAD","event_ticker":"KXBTC-1","title":"bad","status":"active","open_time":"not a time"},
 {"ticker":"KXGAME-1-A","event_ticker":"KXGAME-1","title":"Team A wins","yes_sub_title":"Team A","status":"closed",
  "open_time":"2026-10-01T00:00:00Z","close_time":"2026-10-20T00:00:00Z","volume_24h_fp":"2","strike_type":"structured"}
]}`

func synthetic() *Adapter {
	return New(fetch.New(serve(map[string]string{
		"/v2/markets/KXBTC-1-T100/orderbook": `{"orderbook_fp":{"yes_dollars":[["0.4000","10.00"],["0.4100","5.50"]],"no_dollars":[["0.5500","7.00"]]}}`,
		"/v2/markets":                        page,
		"/v2/series/KXBTC":                   `{"series":{"category":"Crypto","fee_type":"quadratic","fee_multiplier":0.5,"settlement_sources":[{"name":"CF Benchmarks"}]}}`,
		"/v2/series/KXGAME":                  `{"series":{"category":"Sports","fee_type":"quadratic","fee_multiplier":1}}`,
	}), 0, time.Second, 0, nil), "https://k.test/v2", func() time.Time { return recordedAt })
}

func TestCatalogParsesAndCounts(t *testing.T) {
	ms, st, err := synthetic().Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := (struct{ L, C, I, S, U int64 }{6, 2, 1, 1, 2}); st.Listed != want.L || st.Considered != want.C || st.Inactive != want.I || st.TooShort != want.S || st.Unreadable != want.U {
		t.Fatalf("stats %+v", st)
	}
	btc := ms[0]
	if btc.Key != "kalshi:KXBTC-1-T100" || btc.Topic != "crypto" || btc.Volume24h != 11 || !btc.Tradable {
		t.Fatalf("btc %+v", btc)
	}
	if btc.Fee != (market.FeeTerms{RatePPM: 35_000, PExp: 1, QExp: 1, RoundTo: market.Cent}) {
		t.Fatalf("fee %+v", btc.Fee)
	}
	tm := btc.Terms
	if tm.Predicate != "above" || tm.Comparator != ">" || tm.Threshold != "100000" || tm.Source != "cf benchmarks" ||
		tm.WindowEnd != "2026-10-20" || strings.Join(tm.EdgeRules, ",") != "cancellation,fair price" {
		t.Fatalf("terms %+v", tm)
	}
	game := ms[1]
	if game.Tradable || game.UntradableReason != "status closed" || game.Terms.Predicate != "win" || game.Terms.Subject != "team a" {
		t.Fatalf("game %+v", game)
	}
}

func TestBooksFlipNoBidsIntoYesAsks(t *testing.T) {
	books, err := synthetic().Books(context.Background(), []market.Key{"kalshi:KXBTC-1-T100", "polymarket:x"})
	if err != nil {
		t.Fatal(err)
	}
	b := books["kalshi:KXBTC-1-T100"]
	if len(books) != 1 || len(b.Bids) != 2 || b.Bids[0] != (market.Level{Price: 410_000, Qty: 5}) ||
		len(b.Asks) != 1 || b.Asks[0] != (market.Level{Price: 450_000, Qty: 7}) || b.Crossed {
		t.Fatalf("book %+v", b)
	}
}

func TestFeeTerms(t *testing.T) {
	for _, tc := range []struct {
		typ, mult string
		want      market.FeeTerms
	}{
		{"quadratic", "1", market.FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: market.Cent}},
		{"quadratic_with_maker_fees", "0.5", market.FeeTerms{RatePPM: 35_000, PExp: 1, QExp: 1, RoundTo: market.Cent}},
		{"quadratic", "0", market.FeeTerms{RatePPM: 0, PExp: 1, QExp: 1, RoundTo: market.Cent}},
		{"flat", "1", market.FeeTerms{RatePPM: -1}},
		{"quadratic", "abc", market.FeeTerms{RatePPM: -1}},
	} {
		got := fee(&series{FeeType: tc.typ, FeeMultiplier: json.Number(tc.mult)})
		if got != tc.want {
			t.Errorf("%s x%s: got %+v want %+v", tc.typ, tc.mult, got, tc.want)
		}
	}
	// 0.07 x 100 x 0.50 x 0.50 = $1.75 (R04 I2 curve).
	if f := fee(&series{FeeType: "quadratic", FeeMultiplier: "1"}).TakerFee(100, 500_000); f != 1_750_000 {
		t.Fatalf("fee at 50c = %s", f)
	}
}
