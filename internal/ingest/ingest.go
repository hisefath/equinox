// Package ingest refreshes both venues in the background and publishes each result as one immutable snapshot that
// routing reads without ever contacting a venue (lld.md BE-6, D14, HLD INV-1).
package ingest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/refdata"
	"github.com/hisefath/equinox/internal/venue"
)

// Venue health states (SM-2).
const (
	Warming    = "warming"
	Healthy    = "healthy"
	BackingOff = "backing_off"
	Unhealthy  = "unhealthy"
)

// Health is one venue's row (lld.md venue_health).
type Health struct {
	Venue      string    `json:"venue"`
	State      string    `json:"state"`
	LastGoodAt time.Time `json:"last_good_at,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
	WaitUntil  time.Time `json:"wait_until,omitempty"`
}

// Report is one venue's latest refresh counts (lld.md refresh_reports).
type Report struct {
	Venue string `json:"-"`
	venue.CatalogStats
	Duration   time.Duration `json:"duration"`
	FinishedAt time.Time     `json:"finished_at"`
}

// Snapshot is everything routing may read. It is never mutated after Publish.
type Snapshot struct {
	AsOf           time.Time                    `json:"as_of"`
	Markets        map[market.Key]market.Market `json:"-"`
	Books          map[market.Key]market.Book   `json:"-"`
	Pairs          match.PairSet                `json:"-"`
	Health         map[string]Health            `json:"health"`
	Reports        map[string]Report            `json:"reports"`
	UndecidedPairs int                          `json:"undecided_pairs"`
	byVenue        map[string][]market.Market
}

// Fingerprints returns each market's current rules fingerprint, for PairSet.Routable (FR-027).
func (s *Snapshot) Fingerprints() map[market.Key]string {
	out := make(map[market.Key]string, len(s.Markets))
	for k, m := range s.Markets {
		out[k] = m.RulesFingerprint
	}
	return out
}

// Store holds the current snapshot behind an atomic pointer: readers never block and never see half a refresh.
type Store struct{ cur atomic.Pointer[Snapshot] }

// NewStore returns an empty (warming) store.
func NewStore() *Store { return &Store{} }

// Current returns the latest snapshot; false while warming.
func (s *Store) Current() (*Snapshot, bool) {
	p := s.cur.Load()
	return p, p != nil
}

// Cadence sets how often each kind of data refreshes (HLD §10 Time).
type Cadence struct {
	Catalog time.Duration // full catalog + match; default 15 min
	Books   time.Duration // books of matched markets; default 30 s
	Timeout time.Duration // per venue per refresh; default 10 min
}

// Refresher runs refreshes and publishes snapshots (SEQ-1).
type Refresher struct {
	venues  []venue.Venue
	matcher *match.Matcher
	ref     refdata.Set
	store   *Store
	cfg     Cadence
	now     func() time.Time
	mu      sync.Mutex // one refresh at a time
}

// NewRefresher wires a refresher; now is the clock seam (SEAM-1).
func NewRefresher(vs []venue.Venue, m *match.Matcher, ref refdata.Set, s *Store, cfg Cadence, now func() time.Time) *Refresher {
	if cfg.Catalog <= 0 {
		cfg.Catalog = 15 * time.Minute
	}
	if cfg.Books <= 0 {
		cfg.Books = 30 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Minute
	}
	vs = slices.Clone(vs)
	slices.SortFunc(vs, func(a, b venue.Venue) int { return compare(a.Name(), b.Name()) })
	return &Refresher{venues: vs, matcher: m, ref: ref, store: s, cfg: cfg, now: now}
}

func compare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// Run refreshes the catalog on one timer and matched books on another until ctx ends.
func (r *Refresher) Run(ctx context.Context) error {
	r.RefreshOnce(ctx)
	cat := time.NewTicker(r.cfg.Catalog)
	books := time.NewTicker(r.cfg.Books)
	defer cat.Stop()
	defer books.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cat.C:
			r.RefreshOnce(ctx)
		case <-books.C:
			r.RefreshBooks(ctx)
		}
	}
}

type venueResult struct {
	markets []market.Market
	stats   venue.CatalogStats
	err     error
	took    time.Duration
}

// RefreshOnce fetches both catalogs in parallel, matches, fetches matched books and publishes one snapshot. A
// failing venue keeps its previous data and is marked unhealthy or backing off (FR-021, FR-025).
func (r *Refresher) RefreshOnce(ctx context.Context) (*Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	prev, _ := r.store.Current()
	results := make([]venueResult, len(r.venues))
	var wg sync.WaitGroup
	for i, v := range r.venues {
		if h, ok := health(prev, v.Name()); ok && h.State == BackingOff && r.now().Before(h.WaitUntil) {
			results[i].err = errWaiting
			continue
		}
		wg.Add(1)
		go func(i int, v venue.Venue) {
			defer wg.Done()
			c, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
			defer cancel()
			start := time.Now()
			ms, st, err := v.Catalog(c)
			results[i] = venueResult{ms, st, err, time.Since(start)}
		}(i, v)
	}
	wg.Wait()

	at := r.now().UTC()
	next := &Snapshot{AsOf: at, Markets: map[market.Key]market.Market{}, Books: map[market.Key]market.Book{},
		Health: map[string]Health{}, Reports: map[string]Report{}, byVenue: map[string][]market.Market{}}
	var anyGood bool
	for i, v := range r.venues {
		name, res := v.Name(), results[i]
		h, _ := health(prev, name)
		h.Venue = name
		if h.State == "" {
			h.State = Warming
		}
		if res.err != nil {
			h = failed(h, res.err, at)
			if prev != nil { // keep last good data for this venue
				next.byVenue[name] = prev.byVenue[name]
				if rep, ok := prev.Reports[name]; ok {
					next.Reports[name] = rep
				}
			}
		} else {
			anyGood = true
			h.State, h.LastGoodAt, h.LastError, h.WaitUntil = Healthy, at, "", time.Time{}
			next.byVenue[name] = res.markets
			next.Reports[name] = Report{Venue: name, CatalogStats: res.stats, Duration: res.took, FinishedAt: at}
		}
		next.Health[name] = h
		for _, m := range next.byVenue[name] {
			next.Markets[m.Key] = m
		}
	}
	if !anyGood && prev == nil {
		return nil, errors.New("no venue returned data yet")
	}
	names := make([]string, 0, len(r.venues))
	for _, v := range r.venues {
		names = append(names, v.Name())
	}
	if len(names) >= 2 {
		next.Pairs = r.matcher.Match(next.byVenue[names[0]], next.byVenue[names[1]], r.ref)
	}
	for _, p := range next.Pairs.Pairs {
		if p.Verdict == "undecided" {
			next.UndecidedPairs++
		}
	}
	r.fetchBooks(ctx, next, prev)
	r.store.cur.Store(next)
	return next, nil
}

// RefreshBooks re-reads books for equivalent pairs and publishes a copy of the current snapshot with them.
func (r *Refresher) RefreshBooks(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.store.Current()
	if !ok {
		return
	}
	next := *cur
	next.Books = map[market.Key]market.Book{}
	next.Health = map[string]Health{}
	for k, v := range cur.Health {
		next.Health[k] = v
	}
	r.fetchBooks(ctx, &next, cur)
	r.store.cur.Store(&next)
}

func (r *Refresher) fetchBooks(ctx context.Context, next, prev *Snapshot) {
	want := map[string][]market.Key{}
	for _, p := range next.Pairs.Pairs {
		if p.Verdict != "equivalent" {
			continue
		}
		for _, k := range []market.Key{p.AKey, p.BKey} {
			if m, ok := next.Markets[k]; ok {
				want[m.Venue] = append(want[m.Venue], k)
			}
		}
	}
	for _, v := range r.venues {
		name := v.Name()
		keys := want[name]
		if len(keys) == 0 {
			continue
		}
		if h := next.Health[name]; h.State == BackingOff && r.now().Before(h.WaitUntil) {
			copyBooks(next, prev, keys)
			continue
		}
		c, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
		books, err := v.Books(c, keys)
		cancel()
		if err != nil {
			next.Health[name] = failed(next.Health[name], err, r.now().UTC())
		}
		for k, b := range books {
			next.Books[k] = b
		}
		copyBooks(next, prev, keys) // fill anything missing from the last good books; they age out by freshness (FR-022)
	}
}

func copyBooks(next, prev *Snapshot, keys []market.Key) {
	if prev == nil {
		return
	}
	for _, k := range keys {
		if _, ok := next.Books[k]; !ok {
			if b, ok := prev.Books[k]; ok {
				next.Books[k] = b
			}
		}
	}
}

var errWaiting = errors.New("waiting out a refusal")

func failed(h Health, err error, at time.Time) Health {
	var ref fetch.ErrRefused
	switch {
	case errors.Is(err, errWaiting):
		return h
	case errors.As(err, &ref):
		h.State, h.WaitUntil, h.LastError = BackingOff, at.Add(ref.Wait), err.Error()
	default:
		h.State, h.LastError = Unhealthy, truncate(err.Error(), 500)
	}
	return h
}

func health(s *Snapshot, venue string) (Health, bool) {
	if s == nil {
		return Health{}, false
	}
	h, ok := s.Health[venue]
	return h, ok
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
