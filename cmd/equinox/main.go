// Command equinox ingests Kalshi and Polymarket, detects equivalent markets and simulates routing.
//
//	equinox scan   [flags]                 fetch, match, print matched pairs
//	equinox route  [flags] -pair N -side yes -qty 100
//	equinox serve  [flags] -addr :8080     background ingestion + JSON API
//
// Data source flags (all commands): -record DIR saves raw API responses while running live;
// -replay DIR runs offline from such a recording, with the clock set to when it was recorded.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/route"
	"github.com/hisefath/equinox/internal/venues/kalshi"
	"github.com/hisefath/equinox/internal/venues/polymarket"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "scan":
		err = scanCmd(args)
	case "route":
		err = routeCmd(args)
	case "serve":
		err = serveCmd(args)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "equinox:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: equinox scan|route|serve [flags]   (equinox <cmd> -h for flags)")
	os.Exit(2)
}

// config holds the flags every command shares.
type config struct {
	record, replay string
	kalshiPages    int
	polyPages      int
	timeout        time.Duration
	minTier        string
	verbose        bool
	clock          func() time.Time
}

func (c *config) register(fs *flag.FlagSet) {
	fs.StringVar(&c.record, "record", "", "save raw API responses under this directory while running live")
	fs.StringVar(&c.replay, "replay", "", "run offline from a directory written by -record")
	fs.IntVar(&c.kalshiPages, "kalshi-pages", 30, "Kalshi /events pages to crawl (200 events each)")
	fs.IntVar(&c.polyPages, "poly-pages", 30, "Polymarket /markets pages to crawl (100 most-traded markets each)")
	fs.DurationVar(&c.timeout, "timeout", 90*time.Second, "per-venue deadline for one refresh")
	fs.StringVar(&c.minTier, "min-tier", match.Equivalent, "lowest match tier that may be routed: equivalent or review")
	fs.BoolVar(&c.verbose, "v", false, "debug logging")
}

// setup builds the venue adapters and the clock for live, recording or replay mode.
func (c *config) setup() ([]ingest.Venue, error) {
	level := slog.LevelInfo
	if c.verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	var rt http.RoundTripper = http.DefaultTransport
	c.clock = time.Now
	switch {
	case c.replay != "":
		rt = fetch.Recorder{Dir: c.replay}
		at, err := readRecordedAt(c.replay)
		if err != nil {
			return nil, err
		}
		c.clock = func() time.Time { return at }
		slog.Info("replaying recording", "dir", c.replay, "recorded_at", at.Format(time.RFC3339))
	case c.record != "":
		rt = fetch.Recorder{Dir: c.record, Next: http.DefaultTransport}
		if err := writeRecordedAt(c.record, time.Now()); err != nil {
			return nil, err
		}
	}
	k := kalshi.New(fetch.New(rt, 100*time.Millisecond), c.kalshiPages) // 10 req/s, half Kalshi's basic tier
	k.Now = c.clock
	p := polymarket.New(fetch.New(rt, 50*time.Millisecond), c.polyPages) // 20 req/s, well under Gamma's 30
	p.Now = c.clock
	return []ingest.Venue{k, p}, nil
}

// pipeline runs one full ingestion: markets from every venue, matching, then books for matched markets.
func (c *config) pipeline(ctx context.Context, store *ingest.Store, venues []ingest.Venue) match.Result {
	store.RefreshMarkets(ctx, venues, c.timeout)
	snap := store.Snapshot()
	start := time.Now()
	res := match.Match(snap.AllMarkets(), match.Options{})
	slog.Info("matched", "markets", res.Markets, "candidates", res.Compared, "pairs", len(res.Pairs),
		"vetoed", res.Vetoed, "took", time.Since(start).Round(time.Millisecond))
	var ms []market.Market
	for _, p := range res.Pairs {
		ms = append(ms, p.A, p.B)
	}
	store.RefreshBooks(ctx, venues, ms, c.timeout)
	return res
}

func scanCmd(args []string) error {
	var c config
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	c.register(fs)
	out := fs.String("out", "", "also write the full result (pairs, evidence, near misses) as JSON to this file")
	showMisses := fs.Int("near-misses", 10, "print this many high-similarity pairs that a veto rejected")
	fs.Parse(args)
	venues, err := c.setup()
	if err != nil {
		return err
	}
	store := ingest.NewStore()
	res := c.pipeline(context.Background(), store, venues)
	printHealth(store.Snapshot())
	printPairs(res, store.Snapshot())
	if *showMisses > 0 && len(res.NearMisses) > 0 {
		fmt.Printf("\nNear misses: similar text, rejected by a veto (top %d of %d)\n", min(*showMisses, len(res.NearMisses)), len(res.NearMisses))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, r := range res.NearMisses[:min(*showMisses, len(res.NearMisses))] {
			fmt.Fprintf(w, "  %.2f\t%s\t%s\tveto: %s\n", r.Score, clip(r.AText, 55), clip(r.BText, 55), r.Veto)
		}
		w.Flush()
	}
	if *out != "" {
		return writeJSON(*out, res)
	}
	return nil
}

func routeCmd(args []string) error {
	var c config
	fs := flag.NewFlagSet("route", flag.ExitOnError)
	c.register(fs)
	pairArg := fs.String("pair", "1", "pair to route: its number in the scan listing, or its id")
	side := fs.String("side", "yes", "outcome to buy: yes or no")
	qty := fs.Int64("qty", 100, "contracts to buy")
	limit := fs.String("limit", "", "worst acceptable price per contract, e.g. 0.55 (default: none)")
	split := fs.Bool("split", false, "allow splitting the order across venues")
	maxAge := fs.Duration("max-age", 30*time.Second, "books older than this are not trusted")
	logPath := fs.String("log", "data/decisions.jsonl", "append every decision here (JSON lines)")
	fs.Parse(args)

	order := route.Order{Side: market.Side(strings.ToLower(*side)), Qty: *qty}
	if *limit != "" {
		l, err := market.ParseAmount(*limit)
		if err != nil {
			return fmt.Errorf("-limit: %w", err)
		}
		order.Limit = l
	}
	venues, err := c.setup()
	if err != nil {
		return err
	}
	store := ingest.NewStore()
	res := c.pipeline(context.Background(), store, venues)
	pair, err := pickPair(res.Pairs, *pairArg)
	if err != nil {
		return err
	}
	if pair.Tier != match.Equivalent && c.minTier != match.Review {
		return fmt.Errorf("pair %s is only %q; pass -min-tier review to route it anyway", pair.ID, pair.Tier)
	}
	// Ingestion is finished: from here on the decision reads only the in-memory snapshot.
	d := route.Route(order, quotesFor(pair, store.Snapshot()), route.Policy{MaxBookAge: *maxAge, Split: *split}, c.clock())

	fmt.Printf("Pair %s (%s, score %.2f)\n", pair.ID, pair.Tier, pair.Score)
	fmt.Printf("  %s: %s\n  %s: %s\n", pair.A.Venue, label(pair.A), pair.B.Venue, label(pair.B))
	for _, e := range pair.Evidence {
		fmt.Println("  evidence:", e)
	}
	for _, cv := range pair.Caveats {
		fmt.Println("  caveat:  ", cv)
	}
	fmt.Printf("\nDecision %s: %s\n", d.ID, strings.ToUpper(d.Status))
	for _, line := range d.Explanation {
		fmt.Println("  -", line)
	}
	return logDecision(*logPath, pair, d)
}

// quotesFor assembles the router's input for a pair from a snapshot. A market without a book still
// gets a quote: the router then records why it was excluded instead of the market silently vanishing.
func quotesFor(p match.Pair, snap *ingest.Snapshot) []route.Quote {
	var qs []route.Quote
	for _, m := range []market.Market{p.A, p.B} {
		qs = append(qs, route.Quote{Market: m, Book: snap.Books[m.Key()], Unhealthy: snap.Unhealthy(m.Venue)})
	}
	return qs
}

func pickPair(pairs []match.Pair, arg string) (match.Pair, error) {
	if n, err := strconv.Atoi(arg); err == nil {
		if n < 1 || n > len(pairs) {
			return match.Pair{}, fmt.Errorf("pair %d out of range (1..%d)", n, len(pairs))
		}
		return pairs[n-1], nil
	}
	for _, p := range pairs {
		if p.ID == arg {
			return p, nil
		}
	}
	return match.Pair{}, fmt.Errorf("no pair %q in this snapshot", arg)
}

// logDecision appends the decision, with the match it relied on, as one JSON line: the audit trail.
func logDecision(path string, p match.Pair, d route.Decision) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	rec := struct {
		Pair     string         `json:"pair"`
		Tier     string         `json:"tier"`
		Score    float64        `json:"score"`
		Evidence []string       `json:"match_evidence"`
		Caveats  []string       `json:"match_caveats,omitempty"`
		Decision route.Decision `json:"decision"`
	}{p.ID, p.Tier, p.Score, p.Evidence, p.Caveats, d}
	slog.Info("routing decision", "id", d.ID, "pair", p.ID, "status", d.Status, "filled", d.Filled, "all_in", d.AllIn.String())
	return json.NewEncoder(f).Encode(rec)
}

func printHealth(snap *ingest.Snapshot) {
	fmt.Println("Venues")
	for _, v := range []string{"kalshi", "polymarket"} {
		h := snap.Health[v]
		status := "ok"
		if !h.OK {
			status = "FAILED: " + h.Error
		}
		fmt.Printf("  %-10s %s; %d markets kept of %d seen; skipped %v; %d bad books\n", v, status, h.Stats.Kept, h.Stats.Seen, h.Stats.Skipped, h.BookErrors)
	}
}

func printPairs(res match.Result, snap *ingest.Snapshot) {
	fmt.Printf("\nMatched pairs: %d (markets compared: %v, candidates scored: %d)\n", len(res.Pairs), res.Markets, res.Compared)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  #\ttier\tscore\tA\tB\tA ask\tB ask")
	for i, p := range res.Pairs {
		fmt.Fprintf(w, "  %d\t%s\t%.2f\t%s\t%s\t%s\t%s\n", i+1, p.Tier, p.Score, clip(label(p.A), 50), clip(label(p.B), 50),
			bestAsk(snap.Books[p.A.Key()]), bestAsk(snap.Books[p.B.Key()]))
	}
	w.Flush()
}

func bestAsk(b market.Book) string {
	if len(b.Asks) == 0 {
		return "-"
	}
	return b.Asks[0].Price.String()
}

func label(m market.Market) string {
	if m.Outcome != "" && !strings.Contains(m.Question, m.Outcome) {
		return m.Question + " [" + m.Outcome + "]"
	}
	return m.Question
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

const recordedAtFile = "recorded_at.txt"

func writeRecordedAt(dir string, t time.Time) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, recordedAtFile), []byte(t.UTC().Format(time.RFC3339Nano)), 0o644)
}

func readRecordedAt(dir string) (time.Time, error) {
	b, err := os.ReadFile(filepath.Join(dir, recordedAtFile))
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, fmt.Errorf("%s is not a recording (no %s)", dir, recordedAtFile)
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b)))
}
