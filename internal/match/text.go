package match

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hisefath/equinox/internal/market"
)

// features is everything the matcher extracts from one market. Text similarity finds candidates;
// the structured fields (numbers, comparator shape, dates, scopes, sources, outcome identity) decide.
type features struct {
	m        market.Market
	tokens   map[string]float64 // term frequencies; outcome tokens count double
	outcome  []string           // entity tokens of the outcome label ("" for numbers/dates/comparators)
	teams    []string           // team ids named by the outcome label (or the question if no outcome)
	leagues  []string
	nums     []string // canonical numbers from question+outcome, excluding years and dates
	shape    string   // comparator class, see shapeOf
	dates    []int    // day-of-year of explicit "Oct 31" style dates
	months   []int    // months mentioned without a day
	years    []int    // years not part of an explicit date
	scopes   map[string]string
	ranks    []string // "#2", "2nd place", "second-most": a rank is part of the proposition
	mods     []string
	context  map[string]float64 // tokens other than the outcome label's: what is being asked
	subject  []string           // question+outcome tokens, team names removed: candidates for "distinctive"
	sources  []string
	resolves time.Time
}

// ---------------------------------------------------------------------------------------------
// Vocabularies. These small tables are the matcher's domain knowledge and its main maintenance cost.
// Every entry exists because a real Kalshi/Polymarket pair needed it (see research/overlap_census.md).

// synonyms rewrite phrases to one canonical form before tokenizing. Longest phrases first.
var synonyms = [][2]string{
	{"year over year", "yoy"}, {"month over month", "mom"},
	{"federal reserve", "fed"}, {"fomc", "fed"},
	{"basis points", "bp"}, {"basis point", "bp"}, {"bps", "bp"},
	{"interest rates", "rate"}, {"interest rate", "rate"},
	{"no change", "hold"}, {"maintains rate", "hold"}, {"maintain rates", "hold"}, {"unchanged", "hold"},
	{"united states", "us"}, {"u s", "us"}, {"usa", "us"},
	{"pro baseball", "mlb"}, {"pro football", "nfl"}, {"pro basketball", "nba"}, {"pro hockey", "nhl"},
	{"world series", "mlb championship"}, {"super bowl", "nfl championship"},
	{"nba finals", "nba championship"}, {"stanley cup", "nhl championship"},
}

// words maps single tokens to a canonical token (applied after stemming).
var words = map[string]string{
	"decrease": "cut", "lower": "cut", "reduce": "cut",
	"increase": "hike", "raise": "hike",
	"maintain": "hold", "pause": "hold",
	"democratic": "democrat", "dem": "democrat", "democratics": "democrat",
	"gop": "republican",
	"btc": "bitcoin", "eth": "ethereum",
	"percent": "pct",
	"leave":   "out", "leaves": "out", "exit": "out", "exits": "out", "ousted": "out", "removed": "out",
}

var stop = toSet("will the be a an of in on at by to for and or is are after before than this that with their there who what which how many much does do did it its as from end any be")

// units and comparator words carry no identity; they are dropped from outcome entity tokens.
var generic = stemSet("bp pct point game run goal vote view million billion thousand yes no over under above below exactly least most more less fewer between plus up down hold cut hike rate price score win wins winner party out")

var ordinals = map[string]string{"second": "2", "runner up": "2", "third": "3", "fourth": "4"}

var monthNames = map[int]string{1: "jan", 2: "feb", 3: "mar", 4: "apr", 5: "may", 6: "jun", 7: "jul", 8: "aug", 9: "sep", 10: "oct", 11: "nov", 12: "dec"}

var months = map[string]int{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4, "may": 5,
	"jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9,
	"oct": 10, "october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12,
}

// scopeGroups are mutually exclusive qualifiers: two markets that name different values in the same
// group are about different things (Senate vs House, ALCS vs World Series, core CPI YoY vs MoM).
// Values are matched as whole phrases; the outcome label is consulted first, then the question.
var scopeGroups = map[string][][2]string{
	"rate action": {
		{"cut", "cut"}, {"cuts", "cut"}, {"decrease", "cut"}, {"decreases", "cut"}, {"lower", "cut"}, {"lowers", "cut"},
		{"hike", "hike"}, {"hikes", "hike"}, {"increase", "hike"}, {"increases", "hike"}, {"raise", "hike"}, {"raises", "hike"},
		{"no change", "hold"}, {"maintain", "hold"}, {"maintains", "hold"}, {"hold", "hold"}, {"holds", "hold"}, {"unchanged", "hold"},
	},
	"office": {
		{"vice presidential", "vice president"}, {"vice president", "vice president"},
		{"prime minister", "prime minister"}, {"defense minister", "defense minister"}, {"minister of defense", "defense minister"},
		{"national security minister", "security minister"}, {"minister of national security", "security minister"}, {"foreign minister", "foreign minister"},
		{"finance minister", "finance minister"}, {"chancellor", "chancellor"}, {"speaker", "speaker"},
		{"senate", "senate"}, {"house", "house"}, {"governor", "governor"}, {"gubernatorial", "governor"},
		{"presidential", "president"}, {"president", "president"}, {"mayor", "mayor"}, {"mayoral", "mayor"},
	},
	"predicate": {
		{"relegated", "relegate"}, {"relegation", "relegate"}, {"matchup", "matchup"},
		{"finish", "finish"}, {"finishes", "finish"}, {"sold", "sale"}, {"sale", "sale"}, {"charged", "charged"}, {"indicted", "charged"},
		{"sworn in", "take office"}, {"leave", "leave"}, {"leaves", "leave"}, {"out as", "leave"}, {"out by", "leave"}, {"resign", "leave"}, {"removed", "leave"},
		{"caucus", "primary"}, {"primary", "primary"},
		{"nominee", "nominee"}, {"nomination", "nominee"}, {"nominated", "nominee"},
		{"appear on", "appear"}, {"candidate list", "appear"}, {"ballot", "appear"}, {"participate", "appear"}, {"run for", "appear"},
		{"qualify", "advance"}, {"qualifies", "advance"}, {"advance", "advance"}, {"advances", "advance"}, {"reach the", "advance"}, {"make the playoffs", "advance"},
		{"win", "win"}, {"wins", "win"}, {"winner", "win"}, {"champion", "win"},
		{"next", "next"}, // last: "win the next election" is about winning; "be the next PM" is not leaving office
	},
	"competition": {
		{"wnba", "wnba"}, {"nba", "nba"}, {"pro basketball", "nba"}, {"nfl", "nfl"}, {"pro football", "nfl"},
		{"mlb", "mlb"}, {"pro baseball", "mlb"}, {"nhl", "nhl"}, {"pro hockey", "nhl"}, {"mls", "mls"}, {"nwsl", "nwsl"},
		{"ncaa", "ncaa"}, {"college football", "ncaa"}, {"college basketball", "ncaa"}, {"march madness", "ncaa"},
		{"horizon league", "ncaa"}, {"big ten", "ncaa"}, {"big 12", "ncaa"}, {"big east", "ncaa"}, {"mountain west", "ncaa"},
		{"conference tournament", "ncaa"},
		{"champions league", "ucl"}, {"europa league", "uel"}, {"conference league", "uecl"}, {"premier league", "epl"},
		{"la liga", "laliga"}, {"serie a", "seriea"}, {"bundesliga", "bundesliga"}, {"ligue 1", "ligue1"},
		{"concacaf", "concacaf"}, {"nations league", "nations league"}, {"world cup", "world cup"}, {"eurovision", "eurovision"},
		{"australian open", "ausopen"}, {"french open", "rg"}, {"roland garros", "rg"}, {"wimbledon", "wimbledon"}, {"us open", "usopen"},
		{"formula 1", "f1"}, {"grand prix", "f1"}, {"ufc", "ufc"}, {"nascar", "nascar"}, {"oscars", "oscars"}, {"grammy", "grammys"}, {"emmy", "emmys"},
	},
	"segment": {
		{"first half", "h1"}, {"1 st half", "h1"}, {"second half", "h2"}, {"2 nd half", "h2"}, {"halftime", "h1"},
		{"first quarter", "q1"}, {"1 st quarter", "q1"}, {"first period", "p1"}, {"1 st period", "p1"},
		{"first inning", "i1"}, {"1 st inning", "i1"}, {"first 5 innings", "f5"}, {"full game", "full"},
	},
	"stage": {
		{"alcs", "al"}, {"american league", "al"}, {"nlcs", "nl"}, {"national league", "nl"},
		{"division series", "ds"}, {"alds", "ds"}, {"nlds", "ds"}, {"wild card", "wildcard"},
		{"world series", "title"}, {"pro baseball championship", "title"}, {"super bowl", "title"},
		{"pro football championship", "title"}, {"nba finals", "title"}, {"stanley cup", "title"},
		{"afc", "afc"}, {"nfc", "nfc"}, {"conference finals", "conference"},
	},
	"indicator": {{"cpi", "cpi"}, {"pce", "pce"}, {"ppi", "ppi"}, {"gdp", "gdp"}, {"unemployment", "unemployment"}, {"payrolls", "payrolls"}, {"jobless claims", "claims"}},
	"measure":   {{"yoy", "yoy"}, {"year over year", "yoy"}, {"mom", "mom"}, {"month over month", "mom"}},
	"direction": {{"up", "up"}, {"down", "down"}},
}

// modifiers must appear on both sides or neither: "core" CPI is not CPI, a fantasy D/ST ranking is not
// the Super Bowl, "any state" is not Texas, the Women's World Cup is not the World Cup.
var modifiers = []string{"core", "fantasy", "women", "any", "esports", "vice", "announce", "announces", "confirm", "confirms", "final", "finals", "meeting"}

// negated outcome labels flip a market's polarity ("Neither", "No IPO by Dec 31"); such a market is
// only comparable with another negated one.
var negations = []string{"neither", "none", "no one", "nobody", "no", "not"}

// ordering phrases make a proposition relative ("the next leader out", "first to drop out"), which is
// never the same bet as an absolute one ("Erdoğan leaves office before 2027").
var ordering = []string{"next to", "next leader", "first to", "be first", "be the first", "last to", "first this list"}

// oracles are resolution sources known to make look-alike markets settle differently (a BRTI average
// vs a Binance candle; Central Park vs LaGuardia). Unknown sources don't veto; different known ones do.
var oracles = [][2]string{
	{"binance", "binance"}, {"coinbase", "coinbase"}, {"kraken", "kraken"}, {"bitstamp", "bitstamp"},
	{"cf benchmarks", "cf-benchmarks"}, {"brti", "cf-benchmarks"}, {"chainlink", "chainlink"}, {"pyth", "pyth"},
	{"laguardia", "klga"}, {"klga", "klga"}, {"central park", "knyc"}, {"clinyc", "knyc"},
}

// teams lists the four major US leagues: league|id|Kalshi-style city labels|nicknames. Kalshi avoids
// league trademarks and writes cities ("Milwaukee", "New York Y"); Polymarket writes nicknames
// ("Brewers"). Ambiguous cities resolve to every team they could mean; nicknames to exactly one per league.
var teams = parseTeams(`
mlb|ari|arizona|diamondbacks,d-backs
mlb|atl|atlanta|braves
mlb|bal|baltimore|orioles
mlb|bos|boston|red sox
mlb|chc|chicago c,chicago|cubs
mlb|cws|chicago ws,chicago|white sox
mlb|cin|cincinnati|reds
mlb|cle|cleveland|guardians
mlb|col|colorado|rockies
mlb|det|detroit|tigers
mlb|hou|houston|astros
mlb|kc|kansas city|royals
mlb|laa|los angeles a,los angeles|angels
mlb|lad|los angeles d,los angeles|dodgers
mlb|mia|miami|marlins
mlb|mil|milwaukee|brewers
mlb|min|minnesota|twins
mlb|nym|new york m,new york|mets
mlb|nyy|new york y,new york|yankees
mlb|ath|athletics,sacramento,oakland|athletics,a's
mlb|phi|philadelphia|phillies
mlb|pit|pittsburgh|pirates
mlb|sd|san diego|padres
mlb|sf|san francisco|giants
mlb|sea|seattle|mariners
mlb|stl|st louis|cardinals
mlb|tb|tampa bay|rays
mlb|tex|texas|rangers
mlb|tor|toronto|blue jays
mlb|wsh|washington|nationals
nfl|ari|arizona|cardinals
nfl|atl|atlanta|falcons
nfl|bal|baltimore|ravens
nfl|buf|buffalo|bills
nfl|car|carolina|panthers
nfl|chi|chicago|bears
nfl|cin|cincinnati|bengals
nfl|cle|cleveland|browns
nfl|dal|dallas|cowboys
nfl|den|denver|broncos
nfl|det|detroit|lions
nfl|gb|green bay|packers
nfl|hou|houston|texans
nfl|ind|indianapolis|colts
nfl|jax|jacksonville|jaguars
nfl|kc|kansas city|chiefs
nfl|lv|las vegas|raiders
nfl|lac|los angeles c,los angeles|chargers
nfl|lar|los angeles r,los angeles|rams
nfl|mia|miami|dolphins
nfl|min|minnesota|vikings
nfl|ne|new england|patriots
nfl|no|new orleans|saints
nfl|nyg|new york g,new york|giants
nfl|nyj|new york j,new york|jets
nfl|phi|philadelphia|eagles
nfl|pit|pittsburgh|steelers
nfl|sf|san francisco|49ers
nfl|sea|seattle|seahawks
nfl|tb|tampa bay|buccaneers,bucs
nfl|ten|tennessee|titans
nfl|wsh|washington|commanders
nba|atl|atlanta|hawks
nba|bos|boston|celtics
nba|bkn|brooklyn|nets
nba|cha|charlotte|hornets
nba|chi|chicago|bulls
nba|cle|cleveland|cavaliers,cavs
nba|dal|dallas|mavericks,mavs
nba|den|denver|nuggets
nba|det|detroit|pistons
nba|gsw|golden state|warriors
nba|hou|houston|rockets
nba|ind|indiana|pacers
nba|lac|los angeles c,la clippers,los angeles|clippers
nba|lal|los angeles l,los angeles|lakers
nba|mem|memphis|grizzlies
nba|mia|miami|heat
nba|mil|milwaukee|bucks
nba|min|minnesota|timberwolves,wolves
nba|nop|new orleans|pelicans
nba|nyk|new york|knicks
nba|okc|oklahoma city|thunder
nba|orl|orlando|magic
nba|phi|philadelphia|76ers,sixers
nba|phx|phoenix|suns
nba|por|portland|trail blazers,blazers
nba|sac|sacramento|kings
nba|sas|san antonio|spurs
nba|tor|toronto|raptors
nba|uta|utah|jazz
nba|was|washington|wizards
nhl|ana|anaheim|ducks
nhl|bos|boston|bruins
nhl|buf|buffalo|sabres
nhl|cgy|calgary|flames
nhl|car|carolina|hurricanes
nhl|chi|chicago|blackhawks
nhl|col|colorado|avalanche
nhl|cbj|columbus|blue jackets
nhl|dal|dallas|stars
nhl|det|detroit|red wings
nhl|edm|edmonton|oilers
nhl|fla|florida|panthers
nhl|lak|los angeles|kings
nhl|min|minnesota|wild
nhl|mtl|montreal|canadiens
nhl|nsh|nashville|predators
nhl|njd|new jersey|devils
nhl|nyi|new york i,new york|islanders
nhl|nyr|new york r,new york|rangers
nhl|ott|ottawa|senators
nhl|phi|philadelphia|flyers
nhl|pit|pittsburgh|penguins
nhl|sjs|san jose|sharks
nhl|sea|seattle|kraken
nhl|stl|st louis|blues
nhl|tbl|tampa bay|lightning
nhl|tor|toronto|maple leafs
nhl|uta|utah|mammoth
nhl|van|vancouver|canucks
nhl|vgk|vegas|golden knights
nhl|wsh|washington|capitals
nhl|wpg|winnipeg|jets
`)

var leagueWords = map[string]string{
	"mlb": "mlb", "baseball": "mlb", "alcs": "mlb", "nlcs": "mlb", "alds": "mlb", "nlds": "mlb",
	"nfl": "nfl", "football": "nfl", "nba": "nba", "basketball": "nba", "nhl": "nhl", "hockey": "nhl",
}

type teamAlias struct {
	phrase string
	ids    []string
	nick   bool // a nickname pins the league; a city label doesn't
}

// parseTeams builds phrase -> team ids. Nicknames shared across leagues (Giants, Cardinals, Rangers,
// Panthers, Kings, Jets) map to several ids, one per league.
func parseTeams(table string) []teamAlias {
	idx := map[string]*teamAlias{}
	var order []string
	add := func(phrase, id string, nick bool) {
		phrase = strings.TrimSpace(normalize(phrase))
		a, ok := idx[phrase]
		if !ok {
			a = &teamAlias{phrase: phrase, nick: nick}
			idx[phrase] = a
			order = append(order, phrase)
		}
		if !slices.Contains(a.ids, id) {
			a.ids = append(a.ids, id)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(table), "\n") {
		f := strings.Split(line, "|")
		id := f[0] + ":" + f[1]
		for _, c := range strings.Split(f[2], ",") {
			add(c, id, false)
		}
		for _, n := range strings.Split(f[3], ",") {
			add(n, id, true)
		}
	}
	// Longest phrases first, so "new york y" wins over "new york".
	slices.SortStableFunc(order, func(a, b string) int { return len(b) - len(a) })
	out := make([]teamAlias, len(order))
	for i, p := range order {
		out[i] = *idx[p]
	}
	return out
}

// ---------------------------------------------------------------------------------------------

var (
	nonWord = regexp.MustCompile(`[^\p{L}\p{N}.%$#+-]+`)
	// Venues disagree on accents ("Erdoğan"/"Erdogan"); dropping them instead would split names into
	// fragments that collide ("Hernández" and "Fernández" both ending in "ndez").
	folder = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i", "ı", "i", "ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ø", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u", "ñ", "n", "ç", "c", "ğ", "g", "ş", "s", "ș", "s", "ț", "t", "ł", "l", "ž", "z", "š", "s", "č", "c", "ć", "c", "ř", "r", "ý", "y", "ß", "ss", "æ", "ae", "œ", "oe")
	dateRe    = regexp.MustCompile(`\b(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|sept|september|oct|october|nov|november|dec|december)\.? (\d{1,2})(?:st|nd|rd|th)?\b(?: (\d{4}))?`)
	monthRe   = regexp.MustCompile(`\b(jan|january|feb|february|mar|march|apr|april|jun|june|jul|july|aug|august|sep|sept|september|oct|october|nov|november|dec|december)\b`)
	yearRe    = regexp.MustCompile(`\b(20\d\d)\b`)
	rankRe    = regexp.MustCompile(`\btop (\d+|half|ranked)\b|#(\d+)\b|\b(\d+) (?:st|nd|rd|th) (?:place|seed|most|ranked)\b|\b(second|third|fourth|runner up) (?:place|seed|most|ranked)?`)
	isoDateRe = regexp.MustCompile(`\b(20\d\d)-(\d\d)-(\d\d)\b`)
	compound  = regexp.MustCompile(`\b([a-z]+)-?0*(\d+)\b`) // "mi-07" -> "mi7", "q3", "u-3"
	numberRe  = regexp.MustCompile(`(?:^|[^a-z0-9.])[#$]?(\d[\d,]*(?:\.\d+)?)(?:\s*(k|m|b|million|billion|thousand)\b)?`)
	rangeRe   = regexp.MustCompile(`\d(?:\.\d+)?%?\s*(?:-|to)\s*\$?\d|\bbetween \$?\d[\d,.]*%? (?:and|-) \$?\d`)
	plusRe    = regexp.MustCompile(`\d\+`)
	thousands = regexp.MustCompile(`(\d),(\d{3})\b`)
	season    = regexp.MustCompile(`\b(20\d\d)-(\d\d)\b`)
	digitWord = regexp.MustCompile(`(\d)([a-z])`)
	bareNumRe = regexp.MustCompile(`^[$#]?\d[\d,]*(?:\.\d+)?\s*(%|k|m|million|billion)?$`)
)

// normalize lowercases, folds the punctuation venues use differently and collapses whitespace.
// The result is space-separated words, padded with one space on each side for phrase matching.
func normalize(s string) string {
	s = folder.Replace(strings.ToLower(s))
	s = strings.NewReplacer(
		"ㅤ", " ", " ", " ", "’", "'", "‘", "'", "“", " ", "”", " ",
		"–", "-", "—", "-", "−", "-", "≥", " at least ", "≤", " at most ", ">=", " at least ", "<=", " at most ",
		">", " above ", "<", " below ", "↑", " reach ", "↓", " dip ",
		"&", " and ", "'s", "", "o/u", " ou ", "u.s.", "us",
	).Replace(s)
	// Keep "-" only where a digit touches it ("mi-07", "6-9m", "2.5-3.0"); between words it separates
	// ("year-over-year"), and before a number it is a sign we don't need ("Spread -7.5").
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' && !(i > 0 && i+1 < len(s) && isAlnum(s[i-1]) && isAlnum(s[i+1]) && (isDigit(s[i-1]) || isDigit(s[i+1]))) {
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(c)
	}
	s = b.String()
	for thousands.MatchString(s) { // "1,000,000" -> "1000000" before commas become separators
		s = thousands.ReplaceAllString(s, "$1$2")
	}
	// ISO dates become "oct 6 2026" so the one date parser reads them, and before the season rule
	// below can mistake "2026-10" for a season.
	s = isoDateRe.ReplaceAllStringFunc(s, func(d string) string {
		m := isoDateRe.FindStringSubmatch(d)
		mo, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		if mo < 1 || mo > 12 {
			return d
		}
		return fmt.Sprintf(" %s %d %s ", monthNames[mo], day, m[1])
	})
	s = season.ReplaceAllString(s, "$1 20$2")  // "2026-27" season -> both years, not a range
	s = digitWord.ReplaceAllString(s, "$1 $2") // "25bps" -> "25 bps", "84k" -> "84 k"
	s = nonWord.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(" "+strings.Join(strings.Fields(s), " ")+" ", " white house ", " whitehouse ")
	return s
}

func isAlnum(c byte) bool { return isDigit(c) || (c >= 'a' && c <= 'z') || c >= 0x80 }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// extract computes the features of one market.
func extract(m market.Market) features {
	f := features{m: m, tokens: map[string]float64{}, scopes: map[string]string{}, resolves: m.Resolves}
	if f.resolves.IsZero() {
		f.resolves = m.Close
	}
	qo := normalize(m.Question + " " + m.Outcome)
	out := normalize(m.Outcome)
	all := normalize(m.Event + " " + m.Question + " " + m.Outcome)

	// Dates first: their day numbers and years must not be mistaken for thresholds.
	rest := qo
	for _, d := range dateRe.FindAllStringSubmatch(qo, -1) {
		day, _ := strconv.Atoi(d[2])
		if day >= 1 && day <= 31 {
			f.dates = append(f.dates, dayOfYear(months[d[1]], day))
		}
		rest = strings.Replace(rest, d[0], " ", 1)
	}
	for _, mo := range monthRe.FindAllString(rest, -1) {
		f.months = append(f.months, months[mo])
	}
	for _, y := range yearRe.FindAllString(rest, -1) {
		n, _ := strconv.Atoi(y)
		f.years = append(f.years, n)
	}
	rest = yearRe.ReplaceAllString(rest, " ")
	rest = compound.ReplaceAllString(rest, " ") // district/quarter ids are names, not thresholds
	for _, n := range numberRe.FindAllStringSubmatch(rest, -1) {
		f.nums = append(f.nums, canonicalNumber(n[1], n[2]))
	}
	// Shape is read with dates taken out: "2026-10-06" must not look like a "6-1" range.
	f.shape = shapeOf(dateRe.ReplaceAllString(qo, " $1 "), out)

	eq := normalize(m.Event + " " + m.Question)
	for group, vals := range scopeGroups {
		if v := scopeOf(out, vals); v != "" && group != "competition" {
			f.scopes[group] = v
		} else if v := scopeOf(eq, vals); v != "" {
			f.scopes[group] = v
		}
	}
	for _, mod := range modifiers {
		if strings.Contains(all, " "+mod+" ") {
			f.mods = append(f.mods, mod)
		}
	}
	if strings.Contains(m.Outcome, " / ") || strings.Contains(out, " or ") {
		f.mods = append(f.mods, "either-or outcome") // "Retires / No Team" pays on either
	}
	for _, n := range negations {
		if strings.HasPrefix(applySynonyms(out), " "+n+" ") { // after synonyms: "No change" is "hold"
			f.mods = append(f.mods, "negated outcome")
			break
		}
	}
	for _, o := range ordering {
		if strings.Contains(all, " "+o+" ") {
			f.mods = append(f.mods, "ordering")
			break
		}
	}
	for _, r := range rankRe.FindAllStringSubmatch(qo, -1) {
		if r[1] == "ranked" {
			f.ranks = append(f.ranks, "1")
		} else if r[1] != "" {
			f.ranks = append(f.ranks, "top"+r[1])
			continue
		}
		f.ranks = append(f.ranks, cmp.Or(r[2], r[3], ordinals[r[4]]))
	}
	rules := strings.ToLower(m.Rules)
	for _, o := range oracles {
		if strings.Contains(rules, o[0]) && !slices.Contains(f.sources, o[1]) {
			f.sources = append(f.sources, o[1])
		}
	}

	teamText := out
	if strings.TrimSpace(out) == "" {
		teamText = qo
	}
	f.teams, _, _ = teamsIn(teamText)
	allTeams, nickLeagues, _ := teamsIn(all)
	f.leagues = nickLeagues
	for _, t := range strings.Fields(all) {
		if l, ok := leagueWords[t]; ok && !slices.Contains(f.leagues, l) {
			f.leagues = append(f.leagues, l)
		}
	}

	// Tokens: explicit dates keep their month and year but lose the day (the date check compares days
	// structurally; "Oct 16" vs "October 15" should not cost similarity for "before Oct 16" vs "by Oct 15").
	for _, t := range tokenize(dateRe.ReplaceAllString(all, " $1 $3 ")) {
		f.tokens[t]++
	}
	_, _, qoNoTeams := teamsIn(qo)
	for _, t := range tokenize(dateRe.ReplaceAllString(qoNoTeams, " $1 ")) {
		if !isNumeric(t) && !isMonthName(t) && !generic[t] && !vocabulary[t] {
			f.subject = append(f.subject, t)
		}
	}
	_, _, outNoTeams := teamsIn(out)
	for _, t := range tokenize(dateRe.ReplaceAllString(outNoTeams, " $1 $3 ")) {
		f.tokens[t]++ // the outcome label is what separates siblings in one event
		if !generic[t] && !isNumeric(t) && months[t] == 0 && !isMonthName(t) {
			f.outcome = append(f.outcome, t)
		}
	}
	outTokens := map[string]bool{}
	for _, t := range tokenize(dateRe.ReplaceAllString(out, " $1 $3 ")) {
		outTokens[t] = true
	}
	f.context = map[string]float64{}
	for t, n := range f.tokens {
		if !outTokens[t] {
			f.context[t] = n
		}
	}
	for _, id := range allTeams {
		f.tokens["team:"+id]++
	}
	for _, id := range f.teams {
		f.tokens["team:"+id] += 2
	}
	for _, s := range [][]string{f.nums, f.outcome, f.teams, f.leagues, f.sources, f.mods, f.ranks} {
		slices.Sort(s)
	}
	f.nums, f.outcome, f.ranks = slices.Compact(f.nums), slices.Compact(f.outcome), slices.Compact(f.ranks)
	return f
}

func applySynonyms(norm string) string {
	for _, s := range synonyms {
		norm = strings.ReplaceAll(norm, " "+s[0]+" ", " "+s[1]+" ")
	}
	return norm
}

// tokenize turns normalized text into canonical tokens: synonyms, a light plural stem, stopwords out.
func tokenize(norm string) []string {
	norm = compound.ReplaceAllStringFunc(norm, func(c string) string {
		m := compound.FindStringSubmatch(c)
		return m[1] + m[2]
	})
	norm = applySynonyms(norm)
	var out []string
	for _, t := range strings.Fields(norm) {
		t = strings.Trim(t, ".#$+")
		if t == "" || stop[t] {
			continue
		}
		if isNumeric(strings.TrimSuffix(t, "%")) {
			out = append(out, canonicalNumber(strings.TrimSuffix(t, "%"), ""))
			continue
		}
		t = strings.TrimSuffix(t, "%")
		if w, ok := words[t]; ok {
			t = w
		}
		if mo := months[t]; mo != 0 {
			t = monthNames[mo]
		}
		t = stem(t)
		if len(t) < 2 || stop[t] {
			continue
		}
		out = append(out, t)
	}
	return out
}

// shapeOf classifies the comparator a market's YES condition uses. Order matters: "or above" must be
// read as ≥ before "above" is read as >.
func shapeOf(qo, outcome string) string {
	qo = applySynonyms(qo)
	has := func(phrases ...string) bool {
		for _, p := range phrases {
			if strings.Contains(qo, " "+p+" ") {
				return true
			}
		}
		return false
	}
	switch {
	case has("or above", "or more", "or higher", "at least", "or greater", "no less than") || plusRe.MatchString(qo):
		return "≥"
	case has("or below", "or less", "or fewer", "or lower", "at most", "no more than"):
		return "≤"
	case rangeRe.MatchString(qo):
		return "range"
	case has("reach", "hit", "dip", "dip to", "touch"):
		return "touch"
	case has("above", "over", "more than", "greater than", "higher than", "exceed", "exceeds"):
		return ">"
	case has("below", "under", "less than", "fewer than", "lower than"):
		return "<"
	case has("exactly") || bareNumRe.MatchString(strings.TrimSpace(outcome)):
		return "="
	}
	return ""
}

// stem strips common English inflections so "dissenting"/"dissent" and "released"/"release" meet. It
// is deliberately crude; it only has to map both venues' wording to the same token.
func stem(t string) string {
	if isMonthName(t) || strings.Contains(t, ":") {
		return t
	}
	for _, suf := range []string{"ing", "ed", "es", "s", "e"} {
		if len(t)-len(suf) >= 3 && strings.HasSuffix(t, suf) && !strings.HasSuffix(t, "ss") {
			return t[:len(t)-len(suf)]
		}
	}
	return t
}

func scopeOf(norm string, vals [][2]string) string {
	for _, v := range vals {
		if strings.Contains(norm, " "+v[0]+" ") {
			return v[1]
		}
	}
	return ""
}

// teamsIn returns the team ids named in text, the leagues pinned by nicknames, and the text with the
// team names removed. Nicknames are read first; a city then adds its teams only if no nickname already
// identified one of them, so "New York Mets" is the Mets, while a bare "Tampa Bay" could be any of
// three teams.
func teamsIn(norm string) (ids, leagues []string, rest string) {
	for _, nickPass := range []bool{true, false} {
		for _, a := range teams {
			if a.nick != nickPass || !strings.Contains(norm, " "+a.phrase+" ") {
				continue
			}
			norm = strings.ReplaceAll(norm, " "+a.phrase+" ", " ")
			if !a.nick && slices.ContainsFunc(a.ids, func(id string) bool { return slices.Contains(ids, id) }) {
				continue
			}
			for _, id := range a.ids {
				if !slices.Contains(ids, id) {
					ids = append(ids, id)
				}
				if l, _, _ := strings.Cut(id, ":"); a.nick && !slices.Contains(leagues, l) {
					leagues = append(leagues, l)
				}
			}
		}
	}
	slices.Sort(ids)
	return ids, leagues, norm
}

func canonicalNumber(s, scale string) string {
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return s
	}
	switch scale {
	case "k", "thousand":
		v *= 1e3
	case "m", "million":
		v *= 1e6
	case "b", "billion":
		v *= 1e9
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	return err == nil
}

func dayOfYear(month, day int) int {
	return time.Date(2001, time.Month(month), day, 0, 0, 0, 0, time.UTC).YearDay() // non-leap reference year
}

func isMonthName(t string) bool {
	for _, n := range monthNames {
		if n == t {
			return true
		}
	}
	return false
}

// vocabulary holds the words the matcher already reasons about structurally (predicates, scopes,
// modifiers). They describe what is asked, never who it is about, so they are never "distinctive".
var vocabulary = func() map[string]bool {
	v := map[string]bool{}
	add := func(phrase string) {
		for _, t := range tokenize(normalize(phrase)) {
			v[t] = true
		}
	}
	for _, vals := range scopeGroups {
		for _, x := range vals {
			add(x[0])
		}
	}
	for _, list := range [][]string{modifiers, ordering, negations} {
		for _, x := range list {
			add(x)
		}
	}
	return v
}()

func stemSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		out[stem(w)] = true
	}
	return out
}

func toSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		out[w] = true
	}
	return out
}
