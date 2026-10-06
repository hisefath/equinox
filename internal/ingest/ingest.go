// Package ingest pulls markets and order books from every venue concurrently and publishes them as
// immutable snapshots.
//
// Readers (the matcher, the router, HTTP handlers) call Store.Snapshot, which is a single atomic load:
// it never waits on a network call or on a writer. Refreshes build a new snapshot from the previous one
// and swap it in, so a failed or slow venue leaves its last good data in place, marked with its health.
package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

// Venue is the port every venue adapter implements. Adding a venue means implementing this and
// registering it; matching and routing are untouched.
type Venue interface {
	Name() string
	Markets(ctx context.Context) ([]market.Market, Stats, error)
	// Books fetches order books for markets of this venue, keyed by market.Key(). A market missing
	// from the result had no book; err is for failures of the whole call.
	Books(ctx context.Context, ms []market.Market) (map[string]market.Book, error)
}

// Stats counts what an adapter kept and dropped while normalizing, so data problems are visible
// instead of silently swallowed.
type Stats struct {
	Seen    int            `json:"seen"`
	Kept    int            `json:"kept"`
	Skipped map[string]int `json:"skipped,omitempty"` // reason -> count
}

// Skip records one dropped record.
func (s *Stats) Skip(reason string) {
	if s.Skipped == nil {
		s.Skipped = map[string]int{}
	}
	s.Skipped[reason]++
}

// Health is a venue's ingestion status.
type Health struct {
	OK          bool      `json:"ok"`
	Error       string    `json:"error,omitempty"`
	LastAttempt time.Time `json:"last_attempt"`
	LastSuccess time.Time `json:"last_success,omitempty"`
	Stats       Stats     `json:"stats"`
	BookErrors  int       `json:"book_errors"` // failed book fetches in the last refresh
}

// Snapshot is an immutable view of everything ingested so far. Do not modify it.
type Snapshot struct {
	Markets map[string][]market.Market `json:"markets"` // venue -> markets
	Books   map[string]market.Book     `json:"books"`   // market key -> book
	Health  map[string]Health          `json:"health"`  // venue -> health
}

// Store holds the current snapshot.
type Store struct {
	cur     atomic.Pointer[Snapshot]
	writeMu sync.Mutex // serializes writers only; readers never take it
}

// NewStore returns an empty store.
func NewStore() *Store {
	s := &Store{}
	s.cur.Store(&Snapshot{Markets: map[string][]market.Market{}, Books: map[string]market.Book{}, Health: map[string]Health{}})
	return s
}

// Snapshot returns the current snapshot without blocking.
func (s *Store) Snapshot() *Snapshot { return s.cur.Load() }

func (s *Store) update(f func(next *Snapshot)) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	old := s.cur.Load()
	next := &Snapshot{Markets: maps.Clone(old.Markets), Books: maps.Clone(old.Books), Health: maps.Clone(old.Health)}
	f(next)
	s.cur.Store(next)
}

// RefreshMarkets fetches every venue's market list in parallel, each bounded by timeout. A venue that
// fails keeps its previous markets and is marked unhealthy; the others are unaffected.
func (s *Store) RefreshMarkets(ctx context.Context, venues []Venue, timeout time.Duration) {
	type result struct {
		ms    []market.Market
		stats Stats
		err   error
		at    time.Time
	}
	results := make([]result, len(venues))
	var wg sync.WaitGroup
	for i, v := range venues {
		wg.Go(func() {
			vctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			start := time.Now()
			ms, st, err := v.Markets(vctx)
			results[i] = result{ms, st, err, start}
			slog.Info("markets refreshed", "venue", v.Name(), "kept", st.Kept, "seen", st.Seen, "skipped", st.Skipped,
				"took", time.Since(start).Round(time.Millisecond), "err", err)
		})
	}
	wg.Wait()
	s.update(func(next *Snapshot) {
		for i, v := range venues {
			r, h := results[i], next.Health[v.Name()]
			h.LastAttempt, h.Stats = r.at, r.stats
			if r.err != nil {
				h.OK, h.Error = false, r.err.Error()
			} else {
				h.OK, h.Error, h.LastSuccess = true, "", r.at
				next.Markets[v.Name()] = r.ms
			}
			next.Health[v.Name()] = h
		}
	})
}

// RefreshBooks fetches books for the given markets, all venues in parallel, each bounded by timeout.
// Invalid books (crossed, empty) are dropped and counted. A failed or dropped book keeps its previous
// version, whose age then grows until the router's staleness check excludes it: the intended degradation.
func (s *Store) RefreshBooks(ctx context.Context, venues []Venue, ms []market.Market, timeout time.Duration) {
	byVenue := map[string][]market.Market{}
	for _, m := range ms {
		byVenue[m.Venue] = append(byVenue[m.Venue], m)
	}
	type result struct {
		books map[string]market.Book
		bad   int
	}
	results := make([]result, len(venues))
	var wg sync.WaitGroup
	for i, v := range venues {
		want := byVenue[v.Name()]
		if len(want) == 0 {
			continue
		}
		wg.Go(func() {
			vctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			start := time.Now()
			got, err := v.Books(vctx, want)
			r := result{books: map[string]market.Book{}}
			if err != nil {
				r.bad = len(want)
				slog.Warn("books fetch failed", "venue", v.Name(), "markets", len(want), "err", err)
			}
			for _, m := range want {
				b, ok := got[m.Key()]
				if !ok {
					if err == nil {
						r.bad++
					}
					continue
				}
				b = b.Normalize()
				if verr := b.Validate(); verr != nil {
					r.bad++
					slog.Debug("book rejected", "market", m.Key(), "reason", verr)
					continue
				}
				r.books[m.Key()] = b
			}
			results[i] = r
			slog.Info("books refreshed", "venue", v.Name(), "requested", len(want), "ok", len(r.books), "bad", r.bad,
				"took", time.Since(start).Round(time.Millisecond))
		})
	}
	wg.Wait()
	s.update(func(next *Snapshot) {
		for i, v := range venues {
			maps.Copy(next.Books, results[i].books)
			if len(byVenue[v.Name()]) > 0 {
				h := next.Health[v.Name()]
				h.BookErrors = results[i].bad
				next.Health[v.Name()] = h
			}
		}
	})
}

// Unhealthy explains why a venue's data shouldn't be traded on, or returns "" if it's fine.
func (s *Snapshot) Unhealthy(venue string) string {
	h, ok := s.Health[venue]
	switch {
	case !ok:
		return "venue never ingested"
	case !h.OK:
		return fmt.Sprintf("last refresh failed at %s: %s", h.LastAttempt.UTC().Format(time.RFC3339), h.Error)
	}
	return ""
}

// AllMarkets flattens the snapshot's markets across venues, in venue-name order so that everything
// downstream sees the same sequence on every run.
func (s *Snapshot) AllMarkets() []market.Market {
	var out []market.Market
	for _, v := range slices.Sorted(maps.Keys(s.Markets)) {
		out = append(out, s.Markets[v]...)
	}
	return out
}
