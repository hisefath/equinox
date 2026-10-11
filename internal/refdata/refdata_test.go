package refdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDir(t *testing.T, elig, verdicts, overrides string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"eligibility.json": elig, "verdicts.jsonl": verdicts, "overrides.jsonl": overrides} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const okElig = `[{"venue":"kalshi","regions":["US","non-US"],"source":"R06","checked_on":"2026-10-11"},
{"venue":"polymarket","regions":["non-US"],"source":"R06","checked_on":"2026-10-11"}]`

func TestLoadSeedFiles(t *testing.T) {
	s, err := Load("../../refdata", []string{"kalshi", "polymarket"})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Eligible("kalshi", "US"); !ok {
		t.Error("kalshi should be open to US")
	}
	if ok, why := s.Eligible("polymarket", "US"); ok || why != "not open to US customers" {
		t.Errorf("polymarket US: %v %q", ok, why)
	}
	if ok, _ := s.Eligible("polymarket", "non-US"); !ok {
		t.Error("polymarket should be open to non-US")
	}
	if ok, _ := s.Eligible("gemini", "US"); ok {
		t.Error("unknown venue must not be eligible")
	}
	if s.RulesVersion == "" {
		t.Error("rules version empty")
	}
}

func TestLoadValid(t *testing.T) {
	v := `{"pair_id":"p1","fingerprint_a":"fa","fingerprint_b":"fb","verdict":"related","differing_part":"source","reason":"r","model":"m","prompt_version":"1","decided_at":"2026-10-11T00:00:00Z"}

`
	o := `{"pair_id":"p1","fingerprint_a":"fa","fingerprint_b":"fb","verdict":"equivalent","decided_at":"2026-10-11T00:00:00Z"}`
	s, err := Load(writeDir(t, okElig, v, o), []string{"kalshi", "polymarket"})
	if err != nil {
		t.Fatal(err)
	}
	k := PK{"p1", "fa", "fb"}
	if s.Verdicts[k].DifferingPart != "source" || s.Overrides[k].Verdict != "equivalent" {
		t.Errorf("got %+v %+v", s.Verdicts[k], s.Overrides[k])
	}
}

func TestLoadRejects(t *testing.T) {
	good := `{"pair_id":"p1","fingerprint_a":"fa","fingerprint_b":"fb","verdict":"equivalent","reason":"r","model":"m","prompt_version":"1","decided_at":"2026-10-11T00:00:00Z"}`
	cases := []struct{ name, elig, v, o, want string }{
		{"missing venue row", `[{"venue":"kalshi","regions":["US"],"source":"s","checked_on":"d"}]`, "", "", "polymarket"},
		{"bad region", strings.Replace(okElig, `"non-US"]`, `"EU"]`, 1), "", "", "region"},
		{"no source", strings.Replace(okElig, `"source":"R06","checked_on":"2026-10-11"}]`, `"source":"","checked_on":"x"}]`, 1), "", "", "source"},
		{"empty regions", strings.Replace(okElig, `["non-US"]`, `[]`, 1), "", "", "regions"},
		{"bad verdict", okElig, strings.Replace(good, `"equivalent"`, `"maybe"`, 1), "", "verdicts.jsonl:1"},
		{"related without part", okElig, strings.Replace(good, `"equivalent"`, `"related"`, 1), "", "differing_part"},
		{"part without related", okElig, strings.Replace(good, `"reason"`, `"differing_part":"x","reason"`, 1), "", "differing_part"},
		{"no model", okElig, strings.Replace(good, `"model":"m"`, `"model":""`, 1), "", "model"},
		{"duplicate verdict", okElig, good + "\n" + good, "", "verdicts.jsonl:2"},
		{"not json", okElig, "{", "", "verdicts.jsonl:1"},
		{"bad override", okElig, "", `{"pair_id":"p","fingerprint_a":"a","fingerprint_b":"b","verdict":"related","decided_at":"2026-10-11T00:00:00Z"}`, "overrides.jsonl:1"},
	}
	for _, c := range cases {
		_, err := Load(writeDir(t, c.elig, c.v, c.o), []string{"kalshi", "polymarket"})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want mention of %q", c.name, err, c.want)
		}
	}
	if _, err := Load(t.TempDir(), []string{"kalshi"}); err == nil {
		t.Error("missing files must fail")
	}
}
