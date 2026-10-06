package polymarket

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
)

var now = time.Date(2026, 10, 6, 4, 0, 0, 0, time.UTC)

// Payloads trimmed from live Gamma/CLOB responses captured on 2026-10-06 (research/raw/).
const page1 = `{"markets":[
 {"id":"2589812","question":"Will there be no change in Fed interest rates after the October 2026 meeting?","groupItemTitle":"No change",
  "outcomes":"[\"Yes\", \"No\"]","clobTokenIds":"[\"111\", \"112\"]","endDate":"2026-10-29T03:59:00Z","active":true,"closed":false,
  "enableOrderBook":true,"acceptingOrders":true,"orderMinSize":5,"bestBid":0.8,"bestAsk":0.81,"feesEnabled":true,
  "feeSchedule":{"exponent":1,"rate":0.05,"takerOnly":true,"rebateRate":0.25},"description":"This market will resolve to Yes if...",
  "events":[{"id":"40001","slug":"fed-decision-in-october","title":"Fed decision in October?"}]},
 {"id":"4024681","question":"Ravens vs. Falcons","outcomes":["Ravens","Falcons"],"clobTokenIds":"[\"221\",\"222\"]",
  "endDate":"2026-10-19T00:20:00Z","gameStartTime":"2026-10-12 00:20:00+00","active":true,"closed":false,"enableOrderBook":true,
  "acceptingOrders":true,"orderMinSize":5,"bestBid":0.54,"bestAsk":0.55,"feesEnabled":false,"feeSchedule":null,
  "events":[{"id":"50001","slug":"nfl-bal-atl-2026-10-11","title":"Ravens vs. Falcons"}]}],
 "next_cursor":"c2"}`

const page2 = `{"markets":[
 {"id":"9","question":"Other","outcomes":"[\"Yes\",\"No\"]","clobTokenIds":"[\"9\",\"10\"]","active":true,"enableOrderBook":true,"negRiskOther":true},
 {"id":"10","question":"Placeholder","outcomes":"[\"Yes\",\"No\"]","clobTokenIds":"[\"11\",\"12\"]","active":false,"enableOrderBook":true},
 {"id":"11","question":"Three way?","outcomes":"[\"A\",\"B\",\"C\"]","clobTokenIds":"[\"1\",\"2\",\"3\"]","active":true,"enableOrderBook":true},
 {"id":"12","question":"No tokens?","outcomes":"[\"Yes\",\"No\"]","clobTokenIds":"","active":true,"enableOrderBook":true},
 {"id":"13","question":"XRP up or down Feb 24?","outcomes":"[\"Up\",\"Down\"]","clobTokenIds":"[\"31\",\"32\"]","endDate":"2026-02-24T17:00:00Z",
  "active":true,"enableOrderBook":true,"acceptingOrders":true},
 {"id":"14","question":"Disputed?","outcomes":"[\"Yes\",\"No\"]","clobTokenIds":"[\"41\",\"42\"]","endDate":"2026-12-01T00:00:00Z",
  "active":true,"enableOrderBook":true,"acceptingOrders":true,"umaResolutionStatus":"proposed"}],
 "next_cursor":""}`

func server(t *testing.T, books string) *Adapter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/markets/keyset" && r.URL.Query().Get("after_cursor") == "":
			w.Write([]byte(page1))
		case r.URL.Path == "/markets/keyset" && r.URL.Query().Get("after_cursor") == "c2":
			w.Write([]byte(page2))
		case r.URL.Path == "/books" && r.Method == http.MethodPost:
			io.Copy(io.Discard, r.Body)
			w.Write([]byte(books))
		default:
			http.Error(w, "unexpected "+r.URL.String(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	a := New(fetch.New(nil, 0), 0)
	a.Gamma, a.Clob, a.Now = srv.URL, srv.URL, func() time.Time { return now }
	return a
}

func TestMarketsNormalizesAndDecomposesNamedOutcomes(t *testing.T) {
	ms, st, err := server(t, "[]").Markets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Seen != 8 || st.Kept != 4 || st.Skipped["negative-risk 'Other' placeholder"] != 1 || st.Skipped["inactive or closed"] != 1 ||
		st.Skipped["not exactly two outcomes"] != 1 || st.Skipped["missing outcome token ids"] != 1 {
		t.Fatalf("stats = %+v", st)
	}
	byID := map[string]market.Market{}
	for _, m := range ms {
		byID[m.ID] = m
	}
	fed := byID["2589812"]
	if fed.Outcome != "No change" || fed.BookRef != "111" || fed.MinQty != 5 || fed.Event != "Fed decision in October?" || fed.YesAsk != 810_000 {
		t.Errorf("fed = %+v", fed)
	}
	if fed.Fee != (market.FeeCurve{RatePPM: 50_000, PExp: 1, QExp: 1, RoundTo: 10}) {
		t.Errorf("fee = %+v", fed.Fee)
	}
	// ["Ravens","Falcons"] becomes two propositions, each priced by its own token; the second is the mirror.
	rav, fal := byID["4024681:0"], byID["4024681:1"]
	if rav.Outcome != "Ravens" || rav.BookRef != "221" || fal.Outcome != "Falcons" || fal.BookRef != "222" {
		t.Errorf("decomposition: %+v / %+v", rav, fal)
	}
	if fal.YesBid != 450_000 || fal.YesAsk != 460_000 || rav.Fee.RatePPM != 0 {
		t.Errorf("mirrored quotes / fee-free: %v %v %+v", fal.YesBid, fal.YesAsk, rav.Fee)
	}
	if !rav.Resolves.Equal(time.Date(2026, 10, 12, 0, 20, 0, 0, time.UTC)) {
		t.Errorf("game start should be the resolution key: %v", rav.Resolves)
	}
	if byID["13:0"].Untradable != "past scheduled end, awaiting resolution" || byID["14"].Untradable != "resolution proposed" {
		t.Errorf("untradable reasons: %q %q", byID["13:0"].Untradable, byID["14"].Untradable)
	}
}

func TestBooksKeyedByAssetAndNormalized(t *testing.T) {
	books := `[
	 {"asset_id":"222","timestamp":"1791259305517","bids":[{"price":"0.44","size":"10"}],"asks":[{"price":"0.47","size":"5"},{"price":"0.46","size":"20.5"}]},
	 {"asset_id":"111","timestamp":"1791259305517","bids":[{"price":"0.79","size":"100"},{"price":"0.80","size":"50"}],"asks":[{"price":"0.99","size":"1"},{"price":"0.81","size":"20600.6"}]}]`
	a := server(t, books)
	ms := []market.Market{{Venue: "polymarket", ID: "2589812", BookRef: "111"}, {Venue: "polymarket", ID: "4024681:1", BookRef: "222"}}
	got, err := a.Books(context.Background(), ms)
	if err != nil {
		t.Fatal(err)
	}
	b := got["polymarket:2589812"].Normalize()
	if b.Bids[0] != (market.Level{Price: 800_000, Qty: 50}) || b.Asks[0] != (market.Level{Price: 810_000, Qty: 20600}) {
		t.Errorf("fed book = %+v", b)
	}
	if want := time.UnixMilli(1791259305517).UTC(); !b.AsOf.Equal(want) {
		t.Errorf("as_of = %v", b.AsOf)
	}
	if f := got["polymarket:4024681:1"].Normalize(); f.Asks[0].Price != 460_000 {
		t.Errorf("falcons book = %+v", f)
	}
}

func TestStringListAcceptsBothEncodings(t *testing.T) {
	for _, raw := range []string{`"[\"Yes\", \"No\"]"`, `["Yes","No"]`} {
		if got := stringList(json.RawMessage(raw)); len(got) != 2 || got[0] != "Yes" {
			t.Errorf("stringList(%s) = %v", raw, got)
		}
	}
	if got := stringList(json.RawMessage(`null`)); got != nil {
		t.Errorf("null = %v", got)
	}
}
