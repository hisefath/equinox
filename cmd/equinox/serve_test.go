package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
)

var t0 = time.Date(2026, 10, 6, 4, 46, 45, 0, time.UTC)

type staticVenue struct {
	name  string
	ms    []market.Market
	books map[string]market.Book
}

func (v staticVenue) Name() string { return v.name }
func (v staticVenue) Markets(context.Context) ([]market.Market, ingest.Stats, error) {
	return v.ms, ingest.Stats{Seen: len(v.ms), Kept: len(v.ms)}, nil
}
func (v staticVenue) Books(context.Context, []market.Market) (map[string]market.Book, error) {
	return v.books, nil
}

// fixture builds a store with one equivalent pair (a:1 ~ b:1) and one rejected pair (a:2 ~ b:2).
func fixture(t *testing.T) (*config, http.Handler, string) {
	t.Helper()
	mk := func(v, id string) market.Market { return market.Market{Venue: v, ID: id, Question: "q"} }
	book := func(ask string) market.Book {
		p, _ := market.ParseAmount(ask)
		return market.Book{Asks: []market.Level{{Price: p, Qty: 1000}}, Bids: []market.Level{{Price: p - 10_000, Qty: 1000}}, AsOf: t0}
	}
	a := staticVenue{"a", []market.Market{mk("a", "1"), mk("a", "2")}, map[string]market.Book{"a:1": book("0.50"), "a:2": book("0.50")}}
	b := staticVenue{"b", []market.Market{mk("b", "1"), mk("b", "2")}, map[string]market.Book{"b:1": book("0.48"), "b:2": book("0.48")}}
	store := ingest.NewStore()
	venues := []ingest.Venue{a, b}
	store.RefreshMarkets(context.Background(), venues, time.Second)
	store.RefreshBooks(context.Background(), venues, append(a.ms, b.ms...), time.Second)
	res := &match.Result{Pairs: []match.Pair{
		{ID: "a:1~b:1", Tier: match.Equivalent, A: a.ms[0], B: b.ms[0]},
		{ID: "a:2~b:2", Tier: match.Rejected, A: a.ms[1], B: b.ms[1]},
	}}
	var cur atomic.Pointer[match.Result]
	cur.Store(res)
	c := &config{clock: func() time.Time { return t0 }, minTier: match.Equivalent}
	log := filepath.Join(t.TempDir(), "decisions.jsonl")
	return c, c.handler(store, &cur, log), log
}

func get(t *testing.T, h http.Handler, url string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestRouteEndpoint(t *testing.T) {
	_, h, log := fixture(t)
	code, body := get(t, h, "/route?pair=1&side=yes&qty=100&max_age=1m")
	if code != http.StatusOK {
		t.Fatalf("code %d: %v", code, body)
	}
	d := body["decision"].(map[string]any)
	if d["status"] != "filled" || d["allocations"].([]any)[0].(map[string]any)["venue"] != "b" {
		t.Errorf("decision = %v", d)
	}
	lines, _ := os.ReadFile(log)
	if strings.Count(string(lines), "\n") != 1 || !strings.Contains(string(lines), `"pair":"a:1~b:1"`) {
		t.Errorf("decision log = %s", lines)
	}

	for url, want := range map[string]int{
		"/route?pair=2&qty=10":         http.StatusConflict,   // rejected by review
		"/route?pair=9":                http.StatusNotFound,   // no such pair
		"/route?pair=1&qty=abc":        http.StatusBadRequest, // not a number
		"/route?pair=1&qty=0":          http.StatusBadRequest,
		"/route?pair=1&side=maybe":     http.StatusBadRequest,
		"/route?pair=1&limit=0":        http.StatusBadRequest, // 0 is not "no limit"
		"/route?pair=1&max_age=0s":     http.StatusBadRequest, // must not disable staleness
		"/route?pair=1&max_age=banana": http.StatusBadRequest,
	} {
		if code, body := get(t, h, url); code != want {
			t.Errorf("%s: code %d, want %d (%v)", url, code, want, body)
		}
	}
	if lines, _ := os.ReadFile(log); strings.Count(string(lines), "\n") != 1 {
		t.Errorf("invalid requests must not reach the decision log:\n%s", lines)
	}
}

func TestRoutableGate(t *testing.T) {
	eq := match.Pair{ID: "x", Tier: match.Equivalent}
	reviewed := match.Pair{ID: "y", Tier: match.Equivalent, Reviewed: "analyst"}
	rv := match.Pair{ID: "z", Tier: match.Review}
	rej := match.Pair{ID: "w", Tier: match.Rejected}
	for _, c := range []struct {
		cfg  config
		p    match.Pair
		pass bool
	}{
		{config{minTier: match.Equivalent}, eq, true},
		{config{minTier: match.Equivalent}, rv, false},
		{config{minTier: match.Review}, rv, true},
		{config{minTier: match.Review}, rej, false},
		{config{minTier: match.Equivalent, requireReview: true}, eq, false},
		{config{minTier: match.Equivalent, requireReview: true}, reviewed, true},
	} {
		if err := c.cfg.routable(c.p); (err == nil) != c.pass {
			t.Errorf("routable(%+v, tier %s, reviewed %q) = %v", c.cfg, c.p.Tier, c.p.Reviewed, err)
		}
	}
	pairs := []match.Pair{eq, rv}
	if p, err := pickPair(pairs, "2"); err != nil || p.ID != "z" {
		t.Errorf("pickPair by number: %v %v", p.ID, err)
	}
	if p, err := pickPair(pairs, "x"); err != nil || p.ID != "x" {
		t.Errorf("pickPair by id: %v %v", p.ID, err)
	}
	if _, err := pickPair(pairs, "3"); err == nil {
		t.Error("out-of-range pair accepted")
	}
}
