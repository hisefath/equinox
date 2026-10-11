// Package venue states the one contract every venue adapter fulfils (lld.md BE-2, SEAM-6), plus small parsing
// helpers both adapters share.
package venue

import (
	"context"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/hisefath/equinox/internal/market"
)

// Venue is one prediction-market venue behind an adapter.
type Venue interface {
	Name() string
	Catalog(ctx context.Context) ([]market.Market, CatalogStats, error)
	Books(ctx context.Context, keys []market.Key) (map[market.Key]market.Book, error)
}

// CatalogStats counts one catalog read: Listed = Considered + Inactive + TooShort + Unreadable (ck_refresh_counts).
type CatalogStats struct {
	Listed     int64 `json:"listed"`
	Considered int64 `json:"considered"`
	Inactive   int64 `json:"inactive"`
	TooShort   int64 `json:"too_short"`
	Unreadable int64 `json:"unreadable"`
}

// Count files one parsed market into the stats and reports whether it is active (kept).
func (s *CatalogStats) Count(m market.Market) bool {
	s.Listed++
	ok, why := m.Active()
	switch {
	case ok:
		s.Considered++
	case why == "too_short":
		s.TooShort++
	default:
		s.Inactive++
	}
	return ok
}

// Unread files one record that could not be parsed, or a duplicate (FR-024, FM-8).
func (s *CatalogStats) Unread() { s.Listed++; s.Unreadable++ }

// edgeWords maps phrases in settlement rules to the carve-out they signal (Terms.EdgeRules).
// ponytail: keyword scan; the AI path (FR-038) handles carve-outs worded otherwise.
var edgeWords = []struct{ phrase, rule string }{
	{"50-50", "tie 50/50"}, {"50/50", "tie 50/50"},
	{"fair price", "fair price"},
	{"postpone", "postponement"},
	{"cancel", "cancellation"},
	{"overtime", "overtime"},
	{"death", "death"}, {"dies", "death"},
	{"early close", "early close"}, {"close early", "early close"},
}

// EdgeRules returns the sorted, de-duplicated carve-outs the rules text mentions.
func EdgeRules(rules string) []string {
	low := strings.ToLower(rules)
	seen := map[string]bool{}
	var out []string
	for _, w := range edgeWords {
		if strings.Contains(low, w.phrase) && !seen[w.rule] {
			seen[w.rule] = true
			out = append(out, w.rule)
		}
	}
	slices.Sort(out)
	return out
}

// Contracts parses a venue count ("12.50", 12.5) into whole contracts, rounding up so any trade counts as volume.
// Not money: volume only gates the active filter.
func Contracts(s string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || f < 0 || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, &strconv.NumError{Func: "Contracts", Num: s, Err: strconv.ErrSyntax}
	}
	return int64(math.Ceil(f)), nil
}

// Topic maps a venue category word to the system's own topic labels (A-6).
func Topic(category string) string {
	c := strings.ToLower(category)
	switch {
	case strings.Contains(c, "elect") || strings.Contains(c, "politic"):
		return "elections"
	case strings.Contains(c, "econom") || strings.Contains(c, "financ") || strings.Contains(c, "fed"):
		return "economics"
	case strings.Contains(c, "sport"):
		return "sports"
	case strings.Contains(c, "crypto"):
		return "crypto"
	}
	return "other"
}

// Normalize lowercases text and collapses whitespace, for Terms.Subject.
func Normalize(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

// Qty parses a book size into whole contracts, rounding down: a fractional remainder cannot be bought whole.
func Qty(s string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || f < 0 || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, &strconv.NumError{Func: "Qty", Num: s, Err: strconv.ErrSyntax}
	}
	return int64(math.Floor(f)), nil
}
