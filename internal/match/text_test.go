package match

import (
	"slices"
	"strings"
	"testing"

	"github.com/hisefath/equinox/internal/market"
)

func TestShape(t *testing.T) {
	for _, c := range []struct{ q, out, want string }{
		{"Bitcoin price on Oct 9, 2026?", "$84,000 or above", "≥"},
		{"Will the price of Bitcoin be above $84,000 on October 9?", "84,000", ">"},
		{"Will Bitcoin reach $95,000 in October?", "↑ 95,000", "touch"},
		{"Will four or more people dissent?", "4+", "≥"},
		{"How many dissenting votes?", "4", "="},
		{"Will Core CPI YoY be 2.5% in September?", "2.5%", "="},
		{"CPI core year-over-year in Sep 2026?", "Exactly 2.5%", "="}, // "year over year" is not "over"
		{"Will US GDP growth be between 2.5% and 3.0%?", "2.5–3.0%", "range"},
		{"Will the #1 global Netflix show have between 6 and 9 million views?", "6-9M", "range"},
		{"Ravens vs. Falcons: O/U 43.5", "Under", "<"},
		{"Will Democrats win the House in 2026?", "Democratic Party", ""},
		{"Will there be a game between the Ravens and Falcons?", "", ""}, // "between" without numbers
	} {
		if got := shapeOf(normalize(c.q+" "+c.out), normalize(c.out)); got != c.want {
			t.Errorf("shapeOf(%q, %q) = %q, want %q", c.q, c.out, got, c.want)
		}
	}
}

func TestExtractNumbersDatesYears(t *testing.T) {
	f := extract(market.Market{Question: "Will Friedrich Merz leave before Nov 1, 2026?", Outcome: "Before Nov 1, 2026"})
	if len(f.nums) != 0 || len(f.years) != 0 || !slices.Contains(f.dates, 305) {
		t.Errorf("date parts must not become thresholds or years: nums=%v years=%v dates=%v", f.nums, f.years, f.dates)
	}
	f = extract(market.Market{Question: "Will the Democratic Party win the MI-07 House seat in 2026?", Outcome: "William Lawrence (D)"})
	if len(f.nums) != 0 || !slices.Equal(f.years, []int{2026}) || f.tokens["mi7"] == 0 {
		t.Errorf("district ids are names, not thresholds: nums=%v years=%v", f.nums, f.years)
	}
	f = extract(market.Market{Question: "Will the Fed decrease interest rates by 25 bps after the October 2026 meeting?", Outcome: "25bps decrease"})
	if !slices.Equal(f.nums, []string{"25"}) || !slices.Equal(f.months, []int{10}) || f.scopes["rate action"] != "cut" {
		t.Errorf("nums=%v months=%v scopes=%v", f.nums, f.months, f.scopes)
	}
	f = extract(market.Market{Question: "Will the #1 Show on Netflix have at least 6 million views?", Outcome: "At least 6 million"})
	if !slices.Equal(f.nums, []string{"1", "6000000"}) {
		t.Errorf("scaled numbers: %v", f.nums)
	}
}

func TestTeamsAndLeagues(t *testing.T) {
	km := market.Market{Event: "BAL Ravens vs ATL Falcons", Question: "Baltimore wins", Outcome: "Baltimore"}
	pm := market.Market{Event: "Ravens vs. Falcons", Question: "Ravens vs. Falcons", Outcome: "Falcons"}
	k, p := extract(km), extract(pm)
	if !slices.Contains(k.teams, "nfl:bal") || !slices.Equal(p.teams, []string{"nfl:atl"}) || !slices.Equal(k.leagues, []string{"nfl"}) {
		t.Fatalf("k.teams=%v k.leagues=%v p.teams=%v", k.teams, k.leagues, p.teams)
	}
	if v := NewCorpus([]market.Market{km, pm}).Explain(km, pm, Options{}); !strings.HasPrefix(v.Veto, "team") {
		t.Errorf("Baltimore must not match the Falcons outcome: %+v", v)
	}
	ym := market.Market{Question: "Will New York Y win the 2026 Pro Baseball Championship?", Outcome: "New York Y"}
	mm := market.Market{Question: "Will the New York Mets win the 2026 World Series?", Outcome: "New York Mets"}
	if y := extract(ym); !slices.Equal(y.teams, []string{"mlb:nyy"}) {
		t.Errorf("New York Y = %v", y.teams)
	}
	if v := NewCorpus([]market.Market{ym, mm}).Explain(ym, mm, Options{}); !strings.HasPrefix(v.Veto, "team") {
		t.Errorf("Yankees vs Mets must be told apart: %+v", v)
	}
}

func TestOneToOneAndDeterminism(t *testing.T) {
	ms := []market.Market{
		{Venue: "a", ID: "1", Question: "Will the Fed cut rates by 25 bps at the October 2026 meeting?", Outcome: "Cut 25bps"},
		{Venue: "a", ID: "2", Question: "Will the Fed hike rates by 25 bps at the October 2026 meeting?", Outcome: "Hike 25bps"},
		{Venue: "b", ID: "x", Question: "Fed decreases rates by 25 bps after October 2026 meeting?", Outcome: "25 bps decrease"},
		{Venue: "b", ID: "y", Question: "Fed increases rates by 25 bps after October 2026 meeting?", Outcome: "25 bps increase"},
		{Venue: "b", ID: "z", Question: "Fed decreases rates by 50+ bps after October 2026 meeting?", Outcome: "50+ bps decrease"},
	}
	r1 := Match(ms, Options{MaxDF: 1})
	r2 := Match([]market.Market{ms[4], ms[2], ms[0], ms[3], ms[1]}, Options{MaxDF: 1})
	var ids []string
	for _, p := range r1.Pairs {
		ids = append(ids, p.ID)
	}
	if !slices.Equal(ids, []string{"a:1~b:x", "a:2~b:y"}) {
		t.Fatalf("pairs = %v (vetoes %v)", ids, r1.Vetoed)
	}
	for i := range r1.Pairs {
		if r1.Pairs[i].ID != r2.Pairs[i].ID || r1.Pairs[i].Score != r2.Pairs[i].Score {
			t.Fatal("input order changed the result")
		}
	}
}

// Live false positives found in the first end-to-end run, each now rejected for the right reason.
func TestLiveFalsePositivesAreVetoed(t *testing.T) {
	for _, c := range []struct {
		a, b market.Market
		veto string
	}{
		{market.Market{Event: "2027 FIFA Women's World Cup Champion", Question: "Will Spain win the 2027 FIFA Women's World Cup?", Outcome: "Spain"},
			market.Market{Event: "Eurovision 2027 Participants", Question: "Will Spain participate in Eurovision 2027?", Outcome: "Spain"}, "modifier"},
		{market.Market{Event: "2028 U.S. Vice-Presidential Election", Question: "Who will win the 2028 United States vice presidential election?", Outcome: "Raphael Warnock"},
			market.Market{Event: "Presidential Election Winner 2028", Question: "Will Raphael Warnock win the 2028 US Presidential Election?", Outcome: "Raphael Warnock"}, "modifier"},
		{market.Market{Event: "Top Fantasy D/ST", Question: "Will DAL Cowboys D/ST be the #1 ranked fantasy DST in the 2026 season?", Outcome: "DAL Cowboys D/ST"},
			market.Market{Event: "Pro Football: 2027 Champion", Question: "Will the Dallas Cowboys win the 2027 NFL league championship?", Outcome: "Dallas Cowboys"}, "rank"},
		{market.Market{Event: "NHL Playoff Qualifiers", Question: "Will the Nashville Predators qualify for the playoffs in the 2026-27 season?", Outcome: "Nashville Predators"},
			market.Market{Event: "MLS: 2026 Eastern Conference Champion", Question: "Will Nashville SC win the 2026 MLS Eastern Conference?", Outcome: "Nashville SC"}, ""},
		{market.Market{Event: "Los Angeles mayoral election: total votes", Question: "Will the total vote count for all participants in the 2026 Los Angeles mayoral election be above 1.0M?", Outcome: "Above 1.0M"},
			market.Market{Event: "Los Angeles Mayoral Election", Question: "Will Karen Bass win the 2026 Los Angeles mayoral election?", Outcome: "Karen Bass"}, "threshold"},
		{market.Market{Event: "Data center moratorium", Question: "Will any state enact data center moratorium legislation before Jan 1, 2027?", Outcome: "Yes"},
			market.Market{Event: "Texas enacts data center moratorium by...?", Question: "Will Texas enact a data center moratorium by December 31, 2026?", Outcome: "December 31, 2026"}, "modifier"},
		{market.Market{Event: "2027 French Presidential Election candidates", Question: "Will Nicolas Dupont-Aignan appear on the official candidate list?", Outcome: "Nicolas Dupont-Aignan"},
			market.Market{Event: "Next French Presidential Election", Question: "Will Nicolas Dupont-Aignan win the 2027 French presidential election?", Outcome: "Nicolas Dupont-Aignan"}, "scope (predicate)"},
		{market.Market{Event: "Australian Open Men's Singles Champion", Question: "Australian Open Men's Singles: Carlos Alcaraz wins", Outcome: "Carlos Alcaraz"},
			market.Market{Event: "Japan Open Tennis Championships", Question: "Game Spread: Carlos Alcaraz (-3.5) vs Jiri Lehecka (+3.5)", Outcome: "Carlos Alcaraz"}, ""},
		{market.Market{Event: "Western Conference #2 Seed", Question: "Western Conference #2 Seed: Phoenix", Outcome: "Phoenix"},
			market.Market{Event: "NBA: 2027 Western Conference Champion", Question: "Will the Phoenix Suns be the 2027 NBA Western Conference Champion?", Outcome: "Phoenix Suns"}, "rank"},
		{market.Market{Event: "Which leaders will leave office before 2027?", Question: "Will Recep Tayyip Erdoğan leave President of Turkey before Jan 1, 2027?", Outcome: "Recep Tayyip Erdoğan"},
			market.Market{Event: "Next leader out of power before 2027?", Question: "Will Recep Tayyip Erdoğan be the next leader out before 2027?", Outcome: "Recep Tayyip Erdoğan"}, "modifier"},
		{market.Market{Event: "EPL Relegation", Question: "Will Tottenham be relegated from EPL in 2026-27?", Outcome: "Tottenham"},
			market.Market{Event: "EPL: 2027 Champion", Question: "Will Tottenham win the 2026-27 English Premier League?", Outcome: "Tottenham"}, "scope (predicate)"},
		{market.Market{Event: "2028 Iowa Republican caucus winner", Question: "Will Ivanka Trump win the 2028 Iowa Republican caucus?", Outcome: "Ivanka Trump"},
			market.Market{Event: "Presidential Election Winner 2028", Question: "Will Ivanka Trump win the 2028 US Presidential Election?", Outcome: "Ivanka Trump"}, "scope (predicate)"},
	} {
		v := NewCorpus([]market.Market{c.a, c.b}).Explain(c.a, c.b, Options{})
		if v.Veto == "" || !strings.HasPrefix(v.Veto, c.veto) {
			t.Errorf("%q ~ %q: veto %q, want prefix %q (score %.2f)", c.a.Question, c.b.Question, v.Veto, c.veto, v.Score)
		}
	}
}
