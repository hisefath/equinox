package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

type fakeVenue struct {
	name    string
	markets []market.Market
	err     error
	block   chan struct{} // if set, Markets waits on it or the context
	book    market.Book
	bookErr error
}

func (f *fakeVenue) Name() string { return f.name }

func (f *fakeVenue) Markets(ctx context.Context) ([]market.Market, Stats, error) {
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return nil, Stats{}, ctx.Err()
		}
	}
	return f.markets, Stats{Seen: len(f.markets), Kept: len(f.markets)}, f.err
}

func (f *fakeVenue) Books(ctx context.Context, ms []market.Market) (map[string]market.Book, error) {
	out := map[string]market.Book{}
	for _, m := range ms {
		out[m.Key()] = f.book
	}
	return out, f.bookErr
}

func mk(venue, id string) market.Market { return market.Market{Venue: venue, ID: id} }

func TestFailingVenueKeepsLastGoodDataAndDoesNotAffectOthers(t *testing.T) {
	s := NewStore()
	a := &fakeVenue{name: "a", markets: []market.Market{mk("a", "1")}}
	b := &fakeVenue{name: "b", markets: []market.Market{mk("b", "1"), mk("b", "2")}}
	s.RefreshMarkets(context.Background(), []Venue{a, b}, time.Second)

	b.err, b.markets = errors.New("HTTP 503"), nil
	a.markets = append(a.markets, mk("a", "2"))
	s.RefreshMarkets(context.Background(), []Venue{a, b}, time.Second)

	snap := s.Snapshot()
	if len(snap.Markets["a"]) != 2 {
		t.Errorf("healthy venue should update: %v", snap.Markets["a"])
	}
	if len(snap.Markets["b"]) != 2 {
		t.Errorf("failed venue should keep last good markets: %v", snap.Markets["b"])
	}
	if snap.Unhealthy("a") != "" || snap.Unhealthy("b") == "" || snap.Unhealthy("never") == "" {
		t.Errorf("health: a=%q b=%q", snap.Unhealthy("a"), snap.Unhealthy("b"))
	}
}

func TestSlowVenueIsCutOffAndReadersNeverBlock(t *testing.T) {
	s := NewStore()
	slow := &fakeVenue{name: "slow", block: make(chan struct{})}
	fast := &fakeVenue{name: "fast", markets: []market.Market{mk("fast", "1")}}
	done := make(chan struct{})
	go func() {
		s.RefreshMarkets(context.Background(), []Venue{slow, fast}, 200*time.Millisecond)
		close(done)
	}()

	// While a refresh is in flight, Snapshot must return immediately.
	start := time.Now()
	for range 1000 {
		_ = s.Snapshot()
	}
	if el := time.Since(start); el > 50*time.Millisecond {
		t.Errorf("1000 snapshot reads took %s during a refresh", el)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("refresh did not respect its timeout")
	}
	snap := s.Snapshot()
	if snap.Unhealthy("slow") == "" || len(snap.Markets["fast"]) != 1 {
		t.Errorf("slow venue should be unhealthy and fast venue ingested: %+v", snap.Health)
	}
}

func TestRefreshBooksDropsInvalidAndKeepsPrevious(t *testing.T) {
	now := time.Now()
	good := market.Book{Asks: []market.Level{{Price: 510_000, Qty: 10}}, Bids: []market.Level{{Price: 490_000, Qty: 10}}, AsOf: now}
	v := &fakeVenue{name: "v", book: good}
	m := mk("v", "1")
	s := NewStore()
	s.RefreshBooks(context.Background(), []Venue{v}, []market.Market{m}, time.Second)
	if b, ok := s.Snapshot().Books[m.Key()]; !ok || b.Asks[0].Price != 510_000 {
		t.Fatalf("book not stored: %+v", b)
	}

	// A crossed book is rejected; the previous good book stays (and will age out in the router).
	v.book = market.Book{Asks: []market.Level{{Price: 400_000, Qty: 1}}, Bids: []market.Level{{Price: 450_000, Qty: 1}}, AsOf: now}
	s.RefreshBooks(context.Background(), []Venue{v}, []market.Market{m}, time.Second)
	snap := s.Snapshot()
	if snap.Books[m.Key()].Asks[0].Price != 510_000 || snap.Health["v"].BookErrors != 1 {
		t.Errorf("want previous book kept and 1 book error, got %+v / %+v", snap.Books[m.Key()], snap.Health["v"])
	}
}
