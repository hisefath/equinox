// Package refdata loads and validates the reviewed reference files (eligibility, AI verdicts, human overrides)
// into one immutable Set (lld.md BE-10). A bad file stops the service, naming the file and line (FM-6).
package refdata

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Eligibility is one venue's row of refdata/eligibility.json (lld.md eligibility).
type Eligibility struct {
	Venue     string   `json:"venue"`
	Regions   []string `json:"regions"`
	Source    string   `json:"source"`
	CheckedOn string   `json:"checked_on"`
}

// PK is the primary key of a verdict or override: a pair id plus both rules fingerprints at decision time.
type PK struct{ PairID, FingerprintA, FingerprintB string }

// Verdict is one AI adjudication line of refdata/verdicts.jsonl (lld.md verdicts).
type Verdict struct {
	PairID        string    `json:"pair_id"`
	FingerprintA  string    `json:"fingerprint_a"`
	FingerprintB  string    `json:"fingerprint_b"`
	Verdict       string    `json:"verdict"`
	DifferingPart string    `json:"differing_part,omitempty"`
	Reason        string    `json:"reason"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"prompt_version"`
	DecidedAt     time.Time `json:"decided_at"`
}

// Override is one person's accept or reject line of refdata/overrides.jsonl (lld.md overrides).
type Override struct {
	PairID       string    `json:"pair_id"`
	FingerprintA string    `json:"fingerprint_a"`
	FingerprintB string    `json:"fingerprint_b"`
	Verdict      string    `json:"verdict"`
	Note         string    `json:"note,omitempty"`
	DecidedAt    time.Time `json:"decided_at"`
}

// Set is the loaded reference data. RulesVersion is a hash of the three files' bytes.
type Set struct {
	Eligibility  map[string]Eligibility
	Verdicts     map[PK]Verdict
	Overrides    map[PK]Override
	RulesVersion string
}

// Regions a customer may be in (ck_eligibility_regions).
var Regions = []string{"US", "non-US"}

// Load reads eligibility.json, verdicts.jsonl and overrides.jsonl from dir and requires an eligibility row for
// every venue in venues (HLD L-5).
func Load(dir string, venues []string) (Set, error) {
	s := Set{Eligibility: map[string]Eligibility{}, Verdicts: map[PK]Verdict{}, Overrides: map[PK]Override{}}
	h := sha256.New()
	read := func(name string) ([]byte, error) {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("refdata: %w", err)
		}
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(b))
		h.Write(b)
		return b, nil
	}

	b, err := read("eligibility.json")
	if err != nil {
		return Set{}, err
	}
	var rows []Eligibility
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rows); err != nil {
		return Set{}, fmt.Errorf("refdata: eligibility.json: %w", err)
	}
	for i, r := range rows {
		if err := checkEligibility(r); err != nil {
			return Set{}, fmt.Errorf("refdata: eligibility.json row %d: %w", i+1, err)
		}
		if _, dup := s.Eligibility[r.Venue]; dup {
			return Set{}, fmt.Errorf("refdata: eligibility.json row %d: duplicate venue %q", i+1, r.Venue)
		}
		s.Eligibility[r.Venue] = r
	}
	for _, v := range venues {
		if _, ok := s.Eligibility[v]; !ok {
			return Set{}, fmt.Errorf("refdata: eligibility.json: no row for venue %q", v)
		}
	}

	if b, err = read("verdicts.jsonl"); err != nil {
		return Set{}, err
	}
	err = eachLine(b, "verdicts.jsonl", func(v Verdict) error {
		k := PK{v.PairID, v.FingerprintA, v.FingerprintB}
		if _, dup := s.Verdicts[k]; dup {
			return errors.New("duplicate pair_id and fingerprints")
		}
		if err := checkVerdict(v); err != nil {
			return err
		}
		s.Verdicts[k] = v
		return nil
	})
	if err != nil {
		return Set{}, err
	}

	if b, err = read("overrides.jsonl"); err != nil {
		return Set{}, err
	}
	err = eachLine(b, "overrides.jsonl", func(o Override) error {
		k := PK{o.PairID, o.FingerprintA, o.FingerprintB}
		if _, dup := s.Overrides[k]; dup {
			return errors.New("duplicate pair_id and fingerprints")
		}
		if o.PairID == "" || o.FingerprintA == "" || o.FingerprintB == "" {
			return errors.New("pair_id and both fingerprints are required")
		}
		if o.Verdict != "equivalent" && o.Verdict != "not_equivalent" {
			return fmt.Errorf("verdict %q is not equivalent or not_equivalent", o.Verdict)
		}
		s.Overrides[k] = o
		return nil
	})
	if err != nil {
		return Set{}, err
	}
	s.RulesVersion = hex.EncodeToString(h.Sum(nil))[:12]
	return s, nil
}

func checkEligibility(r Eligibility) error {
	if r.Venue == "" {
		return errors.New("venue is empty")
	}
	if len(r.Regions) == 0 {
		return errors.New("regions is empty")
	}
	for _, g := range r.Regions {
		if !slices.Contains(Regions, g) {
			return fmt.Errorf("region %q is not one of %v", g, Regions)
		}
	}
	if r.Source == "" {
		return errors.New("source is empty")
	}
	return nil
}

func checkVerdict(v Verdict) error {
	switch {
	case v.PairID == "" || v.FingerprintA == "" || v.FingerprintB == "":
		return errors.New("pair_id and both fingerprints are required")
	case v.Verdict != "equivalent" && v.Verdict != "related":
		return fmt.Errorf("verdict %q is not equivalent or related", v.Verdict)
	case (v.Verdict == "related") != (v.DifferingPart != ""):
		return errors.New("differing_part must be set exactly when verdict is related")
	case v.Reason == "":
		return errors.New("reason is empty")
	case v.Model == "":
		return errors.New("model is empty")
	case v.PromptVersion == "":
		return errors.New("prompt_version is empty")
	}
	return nil
}

// eachLine decodes every non-blank JSON line of a .jsonl file into T and calls fn; errors name file and line.
func eachLine[T any](b []byte, name string, fn func(T) error) error {
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(nil, 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var v T
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&v); err != nil {
			return fmt.Errorf("refdata: %s:%d: %w", name, n, err)
		}
		if err := fn(v); err != nil {
			return fmt.Errorf("refdata: %s:%d: %w", name, n, err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("refdata: %s: %w", name, err)
	}
	return nil
}

// Eligible reports whether customers in region may use venue, with the reason text (FR-036).
func (s Set) Eligible(venue, region string) (bool, string) {
	r, ok := s.Eligibility[venue]
	if !ok {
		return false, "no eligibility data for this venue"
	}
	if slices.Contains(r.Regions, region) {
		return true, fmt.Sprintf("open to %s customers", region)
	}
	return false, fmt.Sprintf("not open to %s customers", region)
}
