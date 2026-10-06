package match

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Rejected is the tier of a proposed pair that a reviewer ruled out.
const Rejected = "rejected"

// PairReview is a verdict on one proposed pair. A file of these is the reviewed mapping table: the matcher
// proposes, reviews decide. Measured live precision of unreviewed "equivalent" pairs is ~0.87 (see
// docs/TEST_RESULTS.md), which is why production routing should require a review.
type PairReview struct {
	Pair    string `json:"pair"`    // Pair.ID
	Verdict string `json:"verdict"` // Equivalent or "not_equivalent"
	Source  string `json:"source"`  // who decided: a person, or an audit process
	Note    string `json:"note,omitempty"`
}

// LoadReviews reads a reviews file. A missing file is not an error: there is simply nothing reviewed.
func LoadReviews(path string) ([]PairReview, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || path == "" {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rs []PairReview
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return rs, nil
}

// ApplyReviews promotes pairs a reviewer confirmed (even from the review tier) and blocks pairs a
// reviewer rejected. Unreviewed pairs keep the matcher's tier and an empty Reviewed field.
func ApplyReviews(pairs []Pair, rs []PairReview) []Pair {
	byPair := map[string]PairReview{}
	for _, r := range rs {
		byPair[r.Pair] = r
	}
	out := make([]Pair, len(pairs))
	for i, p := range pairs {
		if r, ok := byPair[p.ID]; ok {
			p.Reviewed = r.Source
			switch r.Verdict {
			case Equivalent:
				p.Tier = Equivalent
				p.Evidence = append(p.Evidence, "confirmed by review ("+r.Source+")")
			default:
				p.Tier = Rejected
				p.Caveats = append(p.Caveats, "rejected by review ("+r.Source+"): "+r.Note)
			}
		}
		out[i] = p
	}
	return out
}
