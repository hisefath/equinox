package measure

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
)

func set() (match.PairSet, map[market.Key]market.Market) {
	ms := map[market.Key]market.Market{}
	var ps match.PairSet
	for i := 0; i < 240; i++ {
		a, b := market.Key("a:"+itoa(i)), market.Key("b:"+itoa(i))
		ms[a], ms[b] = market.Market{Key: a, Question: "Q" + itoa(i)}, market.Market{Key: b, Question: "Q" + itoa(i)}
		v := match.Equivalent
		if i%4 == 0 {
			v = match.Related
		}
		ps.Pairs = append(ps.Pairs, match.Pair{PairID: "p" + itoa(i), AKey: a, BKey: b, Verdict: v, Score: 900})
	}
	return ps, ms
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Repeat(" ", 0) + string(rune('0'+i/100)) + string(rune('0'+i/10%10)) + string(rune('0'+i%10)))
}

// AE-023: the same seed draws the same 100 pairs.
func TestSameSeedSameSample(t *testing.T) {
	ps, ms := set()
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	a, b := Draw(ps, ms, 20261011, 100, "v1", at), Draw(ps, ms, 20261011, 100, "v1", at)
	if len(a.PrecisionPairs) != 100 || !reflect.DeepEqual(a.PrecisionPairs, b.PrecisionPairs) {
		t.Fatalf("draws differ or wrong size: %d", len(a.PrecisionPairs))
	}
	if c := Draw(ps, ms, 7, 100, "v1", at); reflect.DeepEqual(a.PrecisionPairs, c.PrecisionPairs) {
		t.Fatal("different seed drew the same sample")
	}
}

// AE-024 / INV-8: a card has no verdict, score or evidence field.
func TestCardHasNoVerdict(t *testing.T) {
	for _, f := range reflect.VisibleFields(reflect.TypeOf(Card{})) {
		n := strings.ToLower(f.Name)
		if strings.Contains(n, "verdict") || strings.Contains(n, "score") || strings.Contains(n, "evidence") {
			t.Fatalf("Card exposes %s", f.Name)
		}
	}
}

// AE-025, AE-026, AE-027, AE-028, INV-9.
func TestScore(t *testing.T) {
	s := Sample{SampleID: "s", RulesVersion: "v1"}
	var labels []Label
	for i := 0; i < 100; i++ {
		id := "p" + itoa(i)
		s.PrecisionPairs = append(s.PrecisionPairs, id)
		l := "equivalent"
		if i < 8 {
			l = "not_equivalent"
		}
		labels = append(labels, Label{PairID: id, Rater: "ai", Label: l, RulesVersion: "v1"})
		if i < 30 {
			labels = append(labels, Label{PairID: id, Rater: "human", Label: l, RulesVersion: "v1"})
		}
	}
	for i := 0; i < 23; i++ { // 23 true pairs the system missed -> recall 92/(92+23) = 0.8
		id := "r" + itoa(i)
		s.RecallPool = append(s.RecallPool, id)
		labels = append(labels, Label{PairID: id, Rater: "ai", Label: "equivalent", RulesVersion: "v1"})
	}
	r, err := Score(s, labels, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if r.PrecisionTrue != 92 || r.PrecisionN != 100 || r.PrecisionLow >= 0.92 || r.PrecisionHigh <= 0.92 {
		t.Fatalf("precision %+v", r)
	}
	if r.Recall != 0.8 || r.KappaN != 30 || r.Kappa != 1 || r.Warning != "" {
		t.Fatalf("report %+v", r)
	}
	if _, err := Score(s, labels, "v2"); err == nil {
		t.Fatal("rules mismatch accepted")
	}
	low := append([]Label{}, labels...)
	for i := 8; i < 14; i++ {
		low = append(low, Label{PairID: "p" + itoa(i), Rater: "human", Label: "not_equivalent", RulesVersion: "v1"})
	}
	if r, _ := Score(s, low, "v1"); r.Warning == "" {
		t.Fatalf("no warning below 0.90: %v", r.Precision)
	}
}
