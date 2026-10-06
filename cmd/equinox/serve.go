package main

import (
	"cmp"
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/route"
)

// serveCmd runs ingestion in the background and answers requests from the latest snapshot. A routing
// request never waits on a venue: it reads an immutable snapshot that a background loop replaces.
func serveCmd(args []string) error {
	var c config
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	c.register(fs)
	addr := fs.String("addr", ":"+cmp.Or(os.Getenv("PORT"), "8080"), "listen address ($PORT on Cloud Run)")
	marketsEvery := fs.Duration("markets-every", 5*time.Minute, "how often to re-crawl markets and re-match")
	booksEvery := fs.Duration("books-every", 10*time.Second, "how often to refresh books of matched markets")
	logPath := fs.String("log", "data/decisions.jsonl", "append every decision here (JSON lines)")
	fs.Parse(args)
	venues, err := c.setup()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	store := ingest.NewStore()
	var current atomic.Pointer[match.Result]
	go func() {
		for ctx.Err() == nil {
			res := c.pipeline(ctx, store, venues)
			current.Store(&res)
			if c.replay != "" {
				return // a recording doesn't change
			}
			next := time.Now().Add(*marketsEvery)
			for time.Now().Before(next) && sleep(ctx, *booksEvery) {
				var ms []market.Market
				for _, p := range res.Pairs {
					ms = append(ms, p.A, p.B)
				}
				store.RefreshBooks(ctx, venues, ms, c.timeout)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		snap := store.Snapshot()
		pairs := 0
		if res := current.Load(); res != nil {
			pairs = len(res.Pairs)
		}
		reply(w, http.StatusOK, map[string]any{"ready": current.Load() != nil, "pairs": pairs, "venues": snap.Health})
	})
	mux.HandleFunc("GET /pairs", func(w http.ResponseWriter, r *http.Request) {
		res := current.Load()
		if res == nil {
			reply(w, http.StatusServiceUnavailable, map[string]string{"error": "warming up: first ingestion still running"})
			return
		}
		tier := r.URL.Query().Get("tier")
		out := []match.Pair{}
		for _, p := range res.Pairs {
			if tier == "" || p.Tier == tier {
				out = append(out, p)
			}
		}
		reply(w, http.StatusOK, out)
	})
	mux.HandleFunc("GET /near-misses", func(w http.ResponseWriter, r *http.Request) {
		if res := current.Load(); res != nil {
			reply(w, http.StatusOK, res.NearMisses)
			return
		}
		reply(w, http.StatusServiceUnavailable, map[string]string{"error": "warming up"})
	})
	mux.HandleFunc("GET /route", func(w http.ResponseWriter, r *http.Request) {
		res := current.Load()
		if res == nil {
			reply(w, http.StatusServiceUnavailable, map[string]string{"error": "warming up"})
			return
		}
		q := r.URL.Query()
		pair, err := pickPair(res.Pairs, cmp.Or(q.Get("pair"), "1"))
		if err != nil {
			reply(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if err := c.routable(pair); err != nil {
			reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		order := route.Order{Side: market.Side(strings.ToLower(cmp.Or(q.Get("side"), "yes")))}
		order.Qty, _ = strconv.ParseInt(cmp.Or(q.Get("qty"), "100"), 10, 64)
		if l := q.Get("limit"); l != "" {
			if order.Limit, err = market.ParseAmount(l); err != nil {
				reply(w, http.StatusBadRequest, map[string]string{"error": "limit: " + err.Error()})
				return
			}
		}
		maxAge, err := time.ParseDuration(cmp.Or(q.Get("max_age"), "30s"))
		if err != nil {
			reply(w, http.StatusBadRequest, map[string]string{"error": "max_age: " + err.Error()})
			return
		}
		policy := route.Policy{MaxBookAge: maxAge, Split: q.Get("split") == "true"}
		d := route.Route(order, quotesFor(pair, store.Snapshot()), policy, c.clock())
		if err := logDecision(*logPath, pair, d); err != nil {
			slog.Error("decision log", "err", err)
		}
		reply(w, http.StatusOK, map[string]any{"pair": pair, "decision": d})
	})

	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	slog.Info("serving", "addr", *addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	enc.Encode(v)
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
