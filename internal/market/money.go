package market

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Amount is a fixed-point US-dollar amount in micro-dollars: 1 unit = $0.000001.
//
// Every binary contract pays $1.00 (= Dollar) if it resolves in the holder's favour, so a price is also
// an Amount. Kalshi's fixed-point fields carry up to 6 decimal places and Polymarket settles in USDC (6
// decimals), so micro-dollars represent both exactly. Integers instead of float64 matter for
// determinism: the Go spec lets the compiler fuse x*y+z into one FMA instruction (it does on arm64, not
// on amd64), so float arithmetic can differ by architecture, and a near-tie could then route
// differently on a Mac than on an x86 Cloud Run host.
type Amount int64

const (
	Micro  Amount = 1
	Cent   Amount = 10_000
	Dollar Amount = 1_000_000
)

const decimals = 6

// ParseAmount parses a decimal dollar string ("0.5230", "0.022", "1") exactly.
// More than 6 significant decimal places is rejected rather than silently rounded.
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" && frac == "" {
		return 0, fmt.Errorf("amount %q: empty", s)
	}
	if !digits(whole) || !digits(frac) {
		return 0, fmt.Errorf("amount %q: not a decimal number", s)
	}
	if len(frac) > decimals {
		if strings.TrimRight(frac[decimals:], "0") != "" {
			return 0, fmt.Errorf("amount %q: more than %d decimal places", s, decimals)
		}
		frac = frac[:decimals]
	}
	frac += strings.Repeat("0", decimals-len(frac))
	if whole == "" {
		whole = "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount %q: %w", s, err)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount %q: %w", s, err)
	}
	if w > (math.MaxInt64-f)/int64(Dollar) {
		return 0, fmt.Errorf("amount %q: out of range", s)
	}
	a := Amount(w)*Dollar + Amount(f)
	if neg {
		a = -a
	}
	return a, nil
}

func digits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// String renders dollars with as many decimals as needed (at least 2): "0.52", "0.0175", "1.716".
func (a Amount) String() string {
	sign := ""
	if a < 0 {
		sign, a = "-", -a
	}
	frac := strings.TrimRight(fmt.Sprintf("%06d", a%Dollar), "0")
	if len(frac) < 2 {
		frac += strings.Repeat("0", 2-len(frac))
	}
	return fmt.Sprintf("%s%d.%s", sign, a/Dollar, frac)
}

// MarshalText makes JSON output human-readable dollars instead of raw pips.
func (a Amount) MarshalText() ([]byte, error) { return []byte(a.String()), nil }

// UnmarshalText is the inverse of MarshalText.
func (a *Amount) UnmarshalText(b []byte) error {
	v, err := ParseAmount(string(b))
	*a = v
	return err
}

// ParseQty parses a contract/share size such as "595.01" and floors it to whole contracts.
// Flooring is deliberately conservative: the router must never count on liquidity that isn't there.
func ParseQty(s string) (int64, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(s), ".")
	if whole == "" {
		whole = "0"
	}
	if !digits(frac) {
		return 0, fmt.Errorf("qty %q: not a decimal number", s)
	}
	n, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("qty %q: %w", s, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("qty %q: negative", s)
	}
	return n, nil
}

// FeeCurve describes a venue's taker fee as data, so the router can price fees without knowing which
// venue it is looking at. For q contracts bought at price p (in dollars):
//
//	fee = q × Rate × p^PExp × (1−p)^QExp
//
// computed exactly and rounded up to the micro-dollar per execution; the total for one order on one
// venue is then rounded up to a multiple of RoundTo. Both venues publish this shape:
//
//	Kalshi:     roundup(0.07 × multiplier × C × P × (1−P)) to the cent  → PExp=QExp=1, RoundTo=1¢
//	Polymarket: C × rate × (p(1−p))^exponent, to 5 decimals          → PExp=QExp=exponent, RoundTo=$0.00001
//
// A zero Rate means no fee. Rounding up where a venue doesn't document the direction is deliberate:
// a fee estimate should never flatter a venue.
type FeeCurve struct {
	RatePPM int64  `json:"rate_ppm"` // rate in parts per million (0.07 = 70_000)
	PExp    int    `json:"p_exp"`
	QExp    int    `json:"q_exp"`
	RoundTo Amount `json:"round_to"` // 0 or 1 = no extra rounding beyond the micro-dollar
}

// Fee returns the unrounded-per-order taker fee for qty contracts at price p, exact to the micro-dollar
// and rounded up (a fee estimate should never flatter a venue).
func (c FeeCurve) Fee(p Amount, qty int64) Amount {
	if c.RatePPM <= 0 || qty <= 0 || p <= 0 || p >= Dollar {
		return 0
	}
	// fee_micros = qty × rate_ppm/1e6 × (P/1e6)^a × ((1e6−P)/1e6)^b × 1e6
	num := big.NewInt(qty)
	num.Mul(num, big.NewInt(c.RatePPM))
	num.Mul(num, pow(int64(p), c.PExp))
	num.Mul(num, pow(int64(Dollar-p), c.QExp))
	num.Mul(num, big.NewInt(int64(Dollar)))
	den := pow(int64(Dollar), c.PExp+c.QExp)
	den.Mul(den, big.NewInt(1_000_000))
	return Amount(ceilDiv(num, den))
}

// Round applies the venue's per-order rounding to a summed fee.
func (c FeeCurve) Round(fee Amount) Amount {
	if r := c.RoundTo; r > 1 && fee%r != 0 {
		return (fee/r + 1) * r
	}
	return fee
}

func pow(b int64, e int) *big.Int {
	return new(big.Int).Exp(big.NewInt(b), big.NewInt(int64(e)), nil)
}

func ceilDiv(num, den *big.Int) int64 {
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q.Int64()
}

// ParseTime parses the timestamp shapes venues actually send (RFC 3339 with or without fractional
// seconds, Polymarket's "2006-01-02 15:04:05+00", or a bare date). A value it can't read becomes the
// zero time, which callers treat as "unknown" rather than failing a whole page over one bad field.
func ParseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05-07", "2006-01-02 15:04:05Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
