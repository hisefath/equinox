// Package adjudicate exports pairs the rules left undecided as AI cards and imports the AI's verdict file after
// validating every line and fingerprint (lld.md BE-14, D10, D16). The AI runs outside the product.
package adjudicate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/measure"
	"github.com/hisefath/equinox/internal/refdata"
)

// Export returns one card per undecided pair (FR-038).
func Export(ps match.PairSet, markets map[market.Key]market.Market) []measure.Card {
	var out []measure.Card
	for _, p := range ps.Pairs {
		if p.Verdict != match.Undecided {
			continue
		}
		s := measure.Draw(match.PairSet{Pairs: []match.Pair{{PairID: p.PairID, AKey: p.AKey, BKey: p.BKey, Verdict: match.Equivalent,
			FingerprintA: p.FingerprintA, FingerprintB: p.FingerprintB}}}, markets, 1, 1, "", p.MatchedAt)
		out = append(out, s.Cards...)
	}
	return out
}

// Import reads JSON-lines verdicts and rejects the whole file on any bad line or a fingerprint that no longer
// matches the current pair (FR-039).
func Import(r io.Reader, ps match.PairSet) ([]refdata.Verdict, error) {
	cur := map[string]match.Pair{}
	for _, p := range ps.Pairs {
		cur[p.PairID] = p
	}
	var out []refdata.Verdict
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for line := 1; sc.Scan(); line++ {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var v refdata.Verdict
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		p, ok := cur[v.PairID]
		switch {
		case !ok:
			return nil, fmt.Errorf("line %d: unknown pair %s", line, v.PairID)
		case p.FingerprintA != v.FingerprintA || p.FingerprintB != v.FingerprintB:
			return nil, fmt.Errorf("line %d: pair %s rules changed since export", line, v.PairID)
		case v.Verdict != match.Equivalent && v.Verdict != match.Related:
			return nil, fmt.Errorf("line %d: verdict must be equivalent or related", line)
		case v.Reason == "" || v.Model == "" || v.PromptVersion == "":
			return nil, fmt.Errorf("line %d: reason, model and prompt_version are required", line)
		}
		out = append(out, v)
	}
	return out, sc.Err()
}
