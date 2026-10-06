package match

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/fetch"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/venues/kalshi"
	"github.com/hisefath/equinox/internal/venues/polymarket"
)

// labelled is one hand-labelled cross-venue pair from the live census (research/overlap_census.md),
// stored with the raw venue JSON so that the real adapters normalize it.
type labelled struct {
	Label        string          `json:"label"`
	Topic        string          `json:"topic"`
	NegativeType string          `json:"negative_type"`
	Reason       string          `json:"reason"`
	YesIndex     int             `json:"polymarket_yes_outcome_index"`
	KEvent       json.RawMessage `json:"kalshi_event"`
	KMarket      json.RawMessage `json:"kalshi_market"`
	PEvent       json.RawMessage `json:"polymarket_event"`
	PMarket      json.RawMessage `json:"polymarket_market"`

	k, p market.Market
}

func loadLabelled(t *testing.T) []labelled {
	t.Helper()
	b, err := os.ReadFile("testdata/labelled_pairs.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Snapshot string     `json:"snapshot_utc"`
		Pairs    []labelled `json:"pairs"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		t.Fatal(err)
	}
	snap := time.Date(2026, 10, 6, 3, 55, 0, 0, time.UTC)
	for i := range file.Pairs {
		lp := &file.Pairs[i]
		k, skip, err := kalshi.ParseMarket(lp.KEvent, lp.KMarket)
		if err != nil || skip != "" {
			t.Fatalf("pair %d kalshi: %v %s", i, err, skip)
		}
		ps, skip, err := polymarket.ParseMarket(lp.PEvent, lp.PMarket, snap)
		if err != nil || skip != "" {
			t.Fatalf("pair %d polymarket: %v %s", i, err, skip)
		}
		lp.k, lp.p = k, ps[0]
		if len(ps) == 2 {
			lp.p = ps[lp.YesIndex]
		}
	}
	return file.Pairs
}

var (
	snapOnce sync.Once
	snap     []market.Market
)

// snapshotMarkets replays the committed live recording (testdata/snapshot) through the real adapters, so
// term statistics in these tests are the ones a production run sees: ~60k markets, not ~110.
func snapshotMarkets(t *testing.T) []market.Market {
	t.Helper()
	snapOnce.Do(func() {
		dir := filepath.Join("..", "..", "testdata", "snapshot")
		b, err := os.ReadFile(filepath.Join(dir, "recorded_at.txt"))
		if err != nil {
			return
		}
		at, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b)))
		clock := func() time.Time { return at }
		rt := fetch.Recorder{Dir: dir}
		k, p := kalshi.New(fetch.New(rt, 0), 30), polymarket.New(fetch.New(rt, 0), 30)
		k.Now, p.Now = clock, clock
		km, _, kerr := k.Markets(context.Background())
		pm, _, perr := p.Markets(context.Background())
		if kerr == nil && perr == nil {
			snap = append(km, pm...)
		}
	})
	if len(snap) == 0 {
		t.Fatal("testdata/snapshot could not be replayed")
	}
	return snap
}

func corpusOf(pairs []labelled, background []market.Market) []market.Market {
	seen := map[string]bool{}
	var out []market.Market
	for _, lp := range pairs {
		for _, m := range []market.Market{lp.k, lp.p} {
			if !seen[m.Key()] {
				seen[m.Key()] = true
				out = append(out, m)
			}
		}
	}
	for _, m := range background {
		if !seen[m.Key()] {
			seen[m.Key()] = true
			out = append(out, m)
		}
	}
	return out
}

// TestLabelledPairs is the matcher's accuracy gate: each hand-labelled pair is judged on its own, and
// precision on "equivalent" verdicts must stay perfect, because a false match routes money into a
// different bet while a miss only forgoes a price improvement.
func TestLabelledPairs(t *testing.T) {
	pairs := loadLabelled(t)
	corpus := NewCorpus(corpusOf(pairs, snapshotMarkets(t)))
	var tp, fp, fn, tn int
	var report strings.Builder
	byTopic := map[string]*[4]int{} // tp, fp, fn, tn
	for _, lp := range pairs {
		v := corpus.Explain(lp.k, lp.p, Options{})
		got := v.Tier == Equivalent
		want := lp.Label == "equivalent"
		mark := "ok  "
		if byTopic[lp.Topic] == nil {
			byTopic[lp.Topic] = &[4]int{}
		}
		k := byTopic[lp.Topic]
		switch {
		case got && want:
			tp++
			k[0]++
		case got && !want:
			fp++
			k[1]++
			mark = "FP  "
		case !got && want:
			fn++
			k[2]++
			mark = "miss"
		default:
			tn++
			k[3]++
		}
		why := v.Veto
		if why == "" {
			why = fmt.Sprintf("tier=%q", v.Tier)
		}
		fmt.Fprintf(&report, "%s %-13s %-28s %.2f %.2f  %-40.40s | %-40.40s | %s\n", mark, lp.Topic, cmp.Or(lp.NegativeType, "equivalent"),
			v.Score, v.Context, label(lp.k), label(lp.p), why)
	}
	precision := float64(tp) / float64(max(tp+fp, 1))
	recall := float64(tp) / float64(max(tp+fn, 1))
	fmt.Fprintf(&report, "\nby topic (tp fp fn tn):\n")
	for _, topic := range slices.Sorted(maps.Keys(byTopic)) {
		k := byTopic[topic]
		fmt.Fprintf(&report, "  %-13s %2d %2d %2d %2d   precision %s  recall %s\n", topic, k[0], k[1], k[2], k[3], ratio(k[0], k[0]+k[1]), ratio(k[0], k[0]+k[2]))
	}
	t.Logf("labelled pairs: %d (tp=%d fp=%d fn=%d tn=%d) precision=%.3f recall=%.3f\n%s",
		len(pairs), tp, fp, fn, tn, precision, recall, report.String())
	if fp > 0 {
		t.Errorf("precision %.3f: %d labelled non-equivalent pairs were judged equivalent", precision, fp)
	}
	if recall < 0.75 {
		t.Errorf("recall %.3f below 0.75", recall)
	}
}

// TestLabelledPairsInContext runs the full pipeline (blocking, vetoes, one-to-one assignment) over all
// labelled markets at once, so siblings compete: the right Fed bucket must beat its neighbours.
func TestLabelledPairsInContext(t *testing.T) {
	if testing.Short() {
		t.Skip("matches ~60k markets; run without -short")
	}
	pairs := loadLabelled(t)
	res := Match(corpusOf(pairs, snapshotMarkets(t)), Options{})
	found := map[string]string{}
	for _, p := range res.Pairs {
		found[p.ID] = p.Tier
	}
	var tp, fp, fn int
	for _, lp := range pairs {
		tier := found[lp.k.Key()+"~"+lp.p.Key()]
		switch {
		case lp.Label == "equivalent" && tier == Equivalent:
			tp++
		case lp.Label == "equivalent":
			fn++
		case tier == Equivalent:
			fp++
			t.Errorf("non-equivalent pair matched: %s ~ %s (%s)", lp.k.Key(), lp.p.Key(), lp.NegativeType)
		}
	}
	t.Logf("in context (%d markets): %d pairs proposed, labelled tp=%d fp=%d fn=%d, vetoes %v", res.Markets["kalshi"]+res.Markets["polymarket"], len(res.Pairs), tp, fp, fn, res.Vetoed)
}

func ratio(a, b int) string {
	if b == 0 {
		return "  -  "
	}
	return fmt.Sprintf("%.3f", float64(a)/float64(b))
}
