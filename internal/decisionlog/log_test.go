package decisionlog

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hisefath/equinox/internal/route"
)

// TestLogSurvivesRestart: 40 decisions appended, the log reopened, all 40 read back newest first (INV-6, NFR-006).
func TestLogSurvivesRestart(t *testing.T) {
	p := filepath.Join(t.TempDir(), "decisions.jsonl")
	l, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if err := l.Append(route.Decision{DecisionID: fmt.Sprint(i), Outcome: route.Filled}); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()
	l, _ = Open(p)
	var got []route.Decision
	cur := ""
	for {
		page, next, err := l.Page(cur, 25)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, page...)
		if next == "" {
			break
		}
		cur = next
	}
	if len(got) != 40 || got[0].DecisionID != "39" || got[39].DecisionID != "0" {
		t.Fatalf("read %d decisions, first %q", len(got), got[0].DecisionID)
	}
	if _, _, err := l.Page("!!", 5); err == nil {
		t.Fatal("bad cursor accepted")
	}
}
