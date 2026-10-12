// Command equinox serves the page and API, and runs the offline jobs: refresh, route, measurement and AI
// adjudication (lld.md BE-12). Venue data comes from recorded real responses (-replay, the default) or live
// (-live), per D17.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/hisefath/equinox/internal/adjudicate"
	"github.com/hisefath/equinox/internal/decisionlog"
	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/measure"
	"github.com/hisefath/equinox/internal/refdata"
	"github.com/hisefath/equinox/internal/venue"
	"github.com/hisefath/equinox/internal/venues/kalshi"
	"github.com/hisefath/equinox/internal/venues/polymarket"
	"github.com/hisefath/equinox/internal/web"
)

// RulesVersion names the matching rules; labels drawn under another version are refused (INV-9).
const RulesVersion = "match-v1"

const usage = `equinox — cross-venue prediction-market matching and routing simulation

Commands:
  serve                 run the page and API (EP-1..EP-6) with a background refresher
  refresh               fetch both venues once, match, print counts and the top pairs
  route                 decide one hypothetical order and log it
  decisions             print the newest logged decisions
  sample                draw a seeded measurement sample and write blind cards
  import-labels         validate a labels file for a sample and store it
  score                 score a sample's labels: precision, recall, kappa
  export-adjudication   write cards for undecided pairs for the offline AI
  import-verdicts       validate the AI's verdict file and add it to refdata/verdicts.jsonl

Data source flags (every command): -replay DIR (default testdata/live-20261011) | -live | -record DIR
Run "equinox <command> -h" for a command's flags.
`

type common struct {
	replay, record, refdir, datadir string
	live, fill                      bool
	kalshiPages, polyPages          int
}

func (c *common) register(fs *flag.FlagSet) {
	fs.StringVar(&c.replay, "replay", "testdata/live-20261011", "answer venue requests from recorded real responses in DIR")
	fs.BoolVar(&c.live, "live", false, "call the venues' public APIs live")
	fs.StringVar(&c.record, "record", "", "call the venues live and record every response into DIR")
	fs.BoolVar(&c.fill, "fill", false, "with -replay: fetch only requests missing from the recording, live, and add them to it")
	fs.StringVar(&c.refdir, "refdata", "refdata", "reviewed reference data directory")
	fs.StringVar(&c.datadir, "data", "data", "directory for the decision log")
	fs.IntVar(&c.kalshiPages, "kalshi-pages", 0, "page cap for the US-regulated venue (0 = all)")
	fs.IntVar(&c.polyPages, "poly-pages", 0, "page cap for the offshore venue (0 = until 24h volume reaches zero)")
}

func (c *common) venues() []venue.Venue {
	now := time.Now
	client := func(name string) *fetch.Client {
		// Live calls go through the system curl: this Mac's firewall blocks newly built binaries but allows curl.
		var t http.RoundTripper = fetch.Curl{}
		if os.Getenv("EQUINOX_NATIVE_HTTP") == "1" {
			t = http.DefaultTransport
		}
		switch {
		case c.record != "":
			t = fetch.NewRecorder(filepath.Join(c.record, name), fetch.Record, t)
		case c.fill:
			t = fetch.NewRecorder(filepath.Join(c.replay, name), fetch.Fill, t)
		case !c.live:
			t = fetch.NewRecorder(filepath.Join(c.replay, name), fetch.Replay, nil)
		}
		rate := 2.0 // NFR-003: at most 2 requests per second per venue
		if !c.live && c.record == "" && !c.fill {
			rate = 10_000 // replay never leaves the machine, so no pacing
		}
		return fetch.New(t, rate, 30*time.Second, 3, now)
	}
	k := kalshi.New(client(kalshi.Name), "https://api.elections.kalshi.com/trade-api/v2", now)
	k.MaxPages = c.kalshiPages
	p := polymarket.New(client(polymarket.Name), "https://gamma-api.polymarket.com", "https://clob.polymarket.com", now)
	p.MaxPages = c.polyPages
	return []venue.Venue{k, p}
}

func (c *common) setup() (refdata.Set, *ingest.Refresher, *ingest.Store) {
	vs := c.venues()
	names := []string{}
	for _, v := range vs {
		names = append(names, v.Name())
	}
	ref, err := refdata.Load(c.refdir, names)
	if err != nil {
		log.Fatalf("reference data: %v", err) // FM-6: never route on guessed eligibility
	}
	store := ingest.NewStore()
	return ref, ingest.NewRefresher(vs, match.New(RulesVersion), ref, store, ingest.Cadence{}, time.Now), store
}

func version() string {
	if v := os.Getenv("EQUINOX_VERSION"); v != "" {
		return v
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				return s.Value[:7]
			}
		}
	}
	return "dev"
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	var c common
	c.register(fs)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	switch cmd {
	case "serve":
		addr := fs.String("addr", ":8080", "listen address")
		fs.Parse(args)
		serve(ctx, &c, *addr)
	case "refresh":
		top := fs.Int("top", 10, "pairs to print")
		fs.Parse(args)
		_, r, _ := c.setup()
		snap := mustRefresh(ctx, r)
		printSnapshot(snap, *top)
	case "route":
		pair := fs.String("pair", "", "pair id (from refresh or /pairs)")
		side := fs.String("side", "yes", "yes or no")
		qty := fs.Int64("qty", 100, "contracts")
		limit := fs.String("limit", "", "optional price limit, 0-1")
		region := fs.String("region", "non-US", "customer region: US or non-US")
		goal := fs.String("goal", "best_price", "best_price (same bet only) or higher_payout (may pick a riskier related bet, D26)")
		fs.Parse(args)
		ref, r, store := c.setup()
		mustRefresh(ctx, r)
		routeOnce(&c, ref, store, *pair, *side, *qty, *limit, *region, *goal)
	case "decisions":
		n := fs.Int("n", 10, "how many")
		fs.Parse(args)
		l := openLog(&c)
		page, _, err := l.Page("", *n)
		check(err)
		printJSON(page)
	case "sample":
		seed := fs.Uint64("seed", 20261011, "random seed (recorded in the sample)")
		n := fs.Int("n", 100, "equivalent pairs to sample")
		fs.Parse(args)
		_, r, _ := c.setup()
		snap := mustRefresh(ctx, r)
		s := measure.Draw(snap.Pairs, snap.Markets, *seed, *n, RulesVersion, time.Now())
		path := filepath.Join(c.refdir, "samples", s.SampleID+".json")
		writeJSONFile(path, s)
		cards := filepath.Join(c.datadir, "cards-"+s.SampleID+".jsonl")
		writeLines(cards, s.Cards)
		fmt.Printf("sample %s: %d precision pairs, %d recall-pool pairs\nsample: %s\nblind cards for raters: %s\n",
			s.SampleID, len(s.PrecisionPairs), len(s.RecallPool), path, cards)
	case "import-labels":
		sp := fs.String("sample", "", "sample file (refdata/samples/<id>.json)")
		in := fs.String("labels", "", "labels file (JSON lines: pair_id, rater, label)")
		fs.Parse(args)
		s := readSample(*sp)
		f, err := os.Open(*in)
		check(err)
		labels, err := measure.ImportLabels(s, f)
		check(err)
		out := filepath.Join(c.refdir, "labels", s.SampleID+".jsonl")
		appendLines(out, labels)
		fmt.Printf("imported %d labels into %s\n", len(labels), out)
	case "score":
		sp := fs.String("sample", "", "sample file")
		fs.Parse(args)
		s := readSample(*sp)
		f, err := os.Open(filepath.Join(c.refdir, "labels", s.SampleID+".jsonl"))
		check(err)
		labels, err := measure.ImportLabels(s, f)
		check(err)
		rep, err := measure.Score(s, labels, RulesVersion)
		check(err)
		printJSON(rep)
	case "export-adjudication":
		out := fs.String("out", "data/adjudication-cards.jsonl", "where to write the AI cards")
		fs.Parse(args)
		_, r, _ := c.setup()
		snap := mustRefresh(ctx, r)
		cards := adjudicate.Export(snap.Pairs, snap.Markets)
		writeLines(*out, cards)
		fmt.Printf("%d undecided pairs written to %s\n", len(cards), *out)
	case "import-verdicts":
		in := fs.String("in", "", "the AI's verdict file (JSON lines)")
		fs.Parse(args)
		_, r, _ := c.setup()
		snap := mustRefresh(ctx, r)
		f, err := os.Open(*in)
		check(err)
		vs, err := adjudicate.Import(f, snap.Pairs)
		check(err)
		appendLines(filepath.Join(c.refdir, "verdicts.jsonl"), vs)
		fmt.Printf("imported %d verdicts; review refdata/verdicts.jsonl in a merge request before deploying\n", len(vs))
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
}

func serve(ctx context.Context, c *common, addr string) {
	ref, r, store := c.setup()
	go func() {
		if err := r.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("refresher stopped: %v", err)
		}
	}()
	s := web.New(store, openLog(c), ref, web.NewRateLimiter(30, time.Minute, time.Now), version())
	srv := &http.Server{Addr: addr, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sh)
	}()
	log.Printf("equinox %s listening on %s (venue data: %s)", version(), addr, c.source())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (c *common) source() string {
	switch {
	case c.record != "":
		return "live, recording to " + c.record
	case c.live:
		return "live"
	}
	return "recorded responses in " + c.replay
}

func mustRefresh(ctx context.Context, r *ingest.Refresher) *ingest.Snapshot {
	snap, err := r.RefreshOnce(ctx)
	check(err)
	return snap
}

func routeOnce(c *common, ref refdata.Set, store *ingest.Store, pairID, side string, qty int64, limit, region, goal string) {
	l := openLog(c)
	s := web.New(store, l, ref, web.NewRateLimiter(1000, time.Minute, time.Now), version())
	body := map[string]any{"pair_id": pairID, "side": side, "quantity": qty, "customer_region": region, "goal": goal}
	if limit != "" {
		body["limit"] = limit
	}
	rec := &recorder{header: http.Header{}}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/route", bytesReader(b))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	os.Stdout.Write(rec.body)
	if rec.status >= 400 {
		os.Exit(1)
	}
}

func printSnapshot(s *ingest.Snapshot, top int) {
	for _, name := range []string{kalshi.Name, polymarket.Name} {
		h, rep := s.Health[name], s.Reports[name]
		fmt.Printf("%-11s %-9s listed %d · considered %d · inactive %d · too short %d · unreadable %d\n",
			name, h.State, rep.Listed, rep.Considered, rep.Inactive, rep.TooShort, rep.Unreadable)
		if h.LastError != "" {
			fmt.Printf("            last error: %s\n", h.LastError)
		}
	}
	var eq, rel, und int
	for _, p := range s.Pairs.Pairs {
		switch p.Verdict {
		case match.Equivalent:
			eq++
		case match.Related:
			rel++
		default:
			und++
		}
	}
	fmt.Printf("pairs: %d equivalent · %d related, not routable · %d undecided (for offline AI) · %d near misses\n\n",
		eq, rel, und, len(s.Pairs.NearMisses))
	shown := 0
	for _, p := range s.Pairs.Pairs {
		if p.Verdict != match.Equivalent || shown == top {
			continue
		}
		shown++
		fmt.Printf("%s  %s\n   ⇄ %s\n   evidence: %v\n", p.PairID, s.Markets[p.AKey].Question, s.Markets[p.BKey].Question, p.Evidence)
		if len(p.Caveats) > 0 {
			fmt.Printf("   caveats: %v\n", p.Caveats)
		}
	}
	for i, n := range s.Pairs.NearMisses {
		if i == 5 {
			break
		}
		fmt.Printf("near miss: %s ⇄ %s — %s\n", s.Markets[n.Pair.AKey].Question, s.Markets[n.Pair.BKey].Question, n.Reason)
	}
	pps := s.PayoutPairs()
	fmt.Printf("\nhigher-payout pairs (different bets, routable only with -goal higher_payout): %d\n", len(pps))
	for i, pv := range pps {
		if i == 40 {
			break
		}
		_, hasA := s.Books[pv.AKey]
		_, hasB := s.Books[pv.BKey]
		fmt.Printf("%s %4d  %s (%s) ⇄ %s (%s)  books: %v/%v\n", pv.PairID, pv.Score, s.Markets[pv.AKey].Question, pv.A, s.Markets[pv.BKey].Question, pv.B, hasA, hasB)
	}
}

func openLog(c *common) *decisionlog.Log {
	check(os.MkdirAll(c.datadir, 0o755))
	l, err := decisionlog.Open(filepath.Join(c.datadir, "decisions.jsonl"))
	check(err)
	return l
}

func readSample(path string) measure.Sample {
	var s measure.Sample
	b, err := os.ReadFile(path)
	check(err)
	check(json.Unmarshal(b, &s))
	return s
}

func writeJSONFile(path string, v any) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	b, err := json.MarshalIndent(v, "", "  ")
	check(err)
	check(os.WriteFile(path, append(b, '\n'), 0o644))
}

func writeLines[T any](path string, items []T) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	check(err)
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, it := range items {
		check(enc.Encode(it))
	}
}

func appendLines[T any](path string, items []T) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	check(err)
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, it := range items {
		check(enc.Encode(it))
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
