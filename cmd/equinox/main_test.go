package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/route"
)

// TestPipelineOnRecordedSnapshot runs the whole system offline on the committed live recording:
// ingest both venues, match, fetch books, route. It is the end-to-end check that the pieces fit.
func TestPipelineOnRecordedSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("matches ~60k recorded markets; run without -short")
	}
	c := config{replay: "../../testdata/snapshot", kalshiPages: 30, polyPages: 30, timeout: time.Minute}
	venues, err := c.setup()
	if err != nil {
		t.Fatal(err)
	}
	store := ingest.NewStore()
	res := c.pipeline(context.Background(), store, venues)
	snap := store.Snapshot()
	if snap.Unhealthy("kalshi") != "" || snap.Unhealthy("polymarket") != "" {
		t.Fatalf("venues unhealthy: %+v", snap.Health)
	}
	if len(res.Pairs) < 100 {
		t.Fatalf("only %d pairs matched", len(res.Pairs))
	}
	var routed int
	for _, p := range res.Pairs[:20] {
		if p.Tier != match.Equivalent {
			continue
		}
		order := route.Order{Side: "yes", Qty: 100}
		policy := route.Policy{MaxBookAge: 5 * time.Minute, Split: true}
		d1 := route.Route(order, quotesFor(p, snap), policy, c.clock())
		d2 := route.Route(order, quotesFor(p, snap), policy, c.clock())
		b1, _ := json.Marshal(d1)
		b2, _ := json.Marshal(d2)
		if string(b1) != string(b2) {
			t.Fatalf("pair %s: routing is not deterministic", p.ID)
		}
		if d1.Status != route.Rejected {
			routed++
		}
	}
	if routed == 0 {
		t.Fatal("no top pair could be routed on the recorded books")
	}
}
