package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/decisionlog"
	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/refdata"
	"github.com/hisefath/equinox/internal/venue"
)

// fakeVenue serves fixed markets and books, can be switched off, and counts calls (SEAM-6).
type fakeVenue struct {
	name  string
	ms    []market.Market
	books map[market.Key]market.Book
	down  atomic.Bool
	calls atomic.Int64
}

func (f *fakeVenue) Name() string { return f.name }
func (f *fakeVenue) Catalog(context.Context) ([]market.Market, venue.CatalogStats, error) {
	f.calls.Add(1)
	if f.down.Load() {
		return nil, venue.CatalogStats{}, errors.New("timed out")
	}
	return f.ms, venue.CatalogStats{Listed: int64(len(f.ms)), Considered: int64(len(f.ms))}, nil
}
func (f *fakeVenue) Books(_ context.Context, keys []market.Key) (map[market.Key]market.Book, error) {
	f.calls.Add(1)
	if f.down.Load() {
		return nil, errors.New("timed out")
	}
	out := map[market.Key]market.Book{}
	for _, k := range keys {
		out[k] = f.books[k]
	}
	return out, nil
}

var now = time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)

func mkt(venueName, id string, fee market.FeeTerms) market.Market {
	m, _ := market.NewMarket(market.Market{Key: market.Key(venueName + ":" + id), Venue: venueName,
		Question: "Will Candidate Lee win the 2026 Iowa Senate race?", Topic: "elections",
		RulesText: "Resolves Yes if Candidate Lee is certified winner of the 2026 Iowa US Senate general election.",
		Terms: market.Terms{Subject: "candidate lee iowa senate", Predicate: "win", Comparator: "=", Source: "certified result",
			WindowEnd: "2026-11-03"}, OpensAt: now.Add(-48 * time.Hour), ClosesAt: now.Add(30 * 24 * time.Hour),
		Volume24h: 500, Tradable: true, Fee: fee, MinQty: 1, ReceivedAt: now})
	return m
}

func book(k market.Key, ask market.Amount) market.Book {
	b, _ := market.NewBook(k, []market.Level{{Price: ask - 20_000, Qty: 2000}}, []market.Level{{Price: ask, Qty: 2000}}, now)
	return b
}

func setup(t *testing.T) (*Server, *fakeVenue, *fakeVenue, *ingest.Refresher, string) {
	t.Helper()
	fee := market.FeeTerms{RatePPM: 70_000, PExp: 1, QExp: 1, RoundTo: market.Cent}
	a, b := mkt("ashby", "IA", fee), mkt("birch", "IA", market.FeeTerms{RatePPM: 0, RoundTo: 10})
	va := &fakeVenue{name: "ashby", ms: []market.Market{a}, books: map[market.Key]market.Book{a.Key: book(a.Key, 580_000)}}
	vb := &fakeVenue{name: "birch", ms: []market.Market{b}, books: map[market.Key]market.Book{b.Key: book(b.Key, 585_000)}}
	ref := refdata.Set{Eligibility: map[string]refdata.Eligibility{
		"ashby": {Venue: "ashby", Regions: []string{"US", "non-US"}, Source: "test"},
		"birch": {Venue: "birch", Regions: []string{"non-US"}, Source: "test"}}}
	store := ingest.NewStore()
	clock := now
	r := ingest.NewRefresher([]venue.Venue{va, vb}, match.New("match-v1"), ref, store, ingest.Cadence{}, func() time.Time { return clock })
	if _, err := r.RefreshOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := store.Current()
	var id string
	for _, p := range snap.Pairs.Pairs {
		if p.Verdict == match.Equivalent {
			id = p.PairID
		}
	}
	if id == "" {
		t.Fatalf("fixture pair not equivalent: %+v", snap.Pairs.Pairs)
	}
	l, err := decisionlog.Open(filepath.Join(t.TempDir(), "d.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return New(store, l, ref, NewRateLimiter(1000, time.Minute, time.Now), "test"), va, vb, r, id
}

func post(s *Server, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/route", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

func logged(t *testing.T, s *Server) int {
	page, _, err := s.log.Page("", 100)
	if err != nil {
		t.Fatal(err)
	}
	return len(page)
}

// AE-006, AE-030, AE-015: a decision comes back, fees decide it, eligibility applies, and no venue is called.
func TestRouteEndpoint(t *testing.T) {
	s, va, vb, _, id := setup(t)
	before := va.calls.Load() + vb.calls.Load()
	w := post(s, `{"pair_id":"`+id+`","side":"yes","quantity":100,"customer_region":"non-US"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"venue": "birch"`) || !strings.Contains(w.Body.String(), "lowest_all_in") {
		t.Fatalf("non-US order: %d %s", w.Code, w.Body)
	}
	w = post(s, `{"pair_id":"`+id+`","side":"yes","quantity":100,"customer_region":"US"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"venue": "ashby"`) {
		t.Fatalf("US order should skip birch: %d %s", w.Code, w.Body)
	}
	if va.calls.Load()+vb.calls.Load() != before {
		t.Fatal("routing contacted a venue (INV-1)")
	}
	if logged(t, s) != 2 {
		t.Fatalf("want 2 logged decisions, got %d", logged(t, s))
	}
}

// AE-011, AE-031, AE-010: malformed orders are rejected and not logged; unknown pairs are refused and logged.
func TestRouteErrors(t *testing.T) {
	s, _, _, _, id := setup(t)
	cases := []struct {
		body string
		code int
	}{
		{`{"pair_id":"` + id + `","side":"yes","quantity":0,"customer_region":"US"}`, 422},
		{`{"pair_id":"` + id + `","side":"yes","quantity":5}`, 422},
		{`{"pair_id":"` + id + `","side":"maybe","quantity":5,"customer_region":"US"}`, 422},
		{`{"pair_id":"` + id + `","side":"yes","quantity":5,"limit":"1.5","customer_region":"US"}`, 422},
	}
	for _, c := range cases {
		if w := post(s, c.body); w.Code != c.code || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s -> %d %s", c.body, w.Code, w.Body)
		}
	}
	if n := logged(t, s); n != 0 {
		t.Fatalf("malformed orders were logged: %d", n)
	}
	if w := post(s, `{"pair_id":"nope","side":"yes","quantity":5,"customer_region":"US"}`); w.Code != 404 {
		t.Fatalf("unknown pair -> %d", w.Code)
	}
	if n := logged(t, s); n != 1 {
		t.Fatalf("refused order not logged: %d", n)
	}
}

// AE-016, AE-017: a venue that goes down keeps its last good data, is marked unhealthy, and ages out of decisions.
func TestVenueDownKeepsLastGoodData(t *testing.T) {
	s, _, vb, r, id := setup(t)
	vb.down.Store(true)
	if _, err := r.RefreshOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.store.Current()
	if snap.Health["birch"].State != ingest.Unhealthy || len(snap.Pairs.Pairs) == 0 {
		t.Fatalf("health %+v pairs %d", snap.Health["birch"], len(snap.Pairs.Pairs))
	}
	w := post(s, `{"pair_id":"`+id+`","side":"yes","quantity":100,"customer_region":"non-US"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"venue": "ashby"`) || !strings.Contains(w.Body.String(), "unhealthy") {
		t.Fatalf("decision with birch down: %d %s", w.Code, w.Body)
	}
}

func TestWarmingAndHealth(t *testing.T) {
	l, _ := decisionlog.Open(filepath.Join(t.TempDir(), "d.jsonl"))
	s := New(ingest.NewStore(), l, refdata.Set{}, NewRateLimiter(1, time.Minute, time.Now), "test")
	if w := post(s, `{"pair_id":"x","side":"yes","quantity":5,"customer_region":"US"}`); w.Code != 503 {
		t.Fatalf("warming -> %d", w.Code)
	}
	if w := post(s, `{"pair_id":"x","side":"yes","quantity":5,"customer_region":"US"}`); w.Code != 429 {
		t.Fatalf("rate limit -> %d", w.Code)
	}
	for _, path := range []string{"/health", "/version", "/decisions"} {
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK {
			t.Errorf("%s -> %d", path, w.Code)
		}
	}
}
