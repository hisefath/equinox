// Package web serves the page and the JSON endpoints EP-1..EP-6 (lld.md BE-11). Routing reads only the current
// snapshot (HLD INV-1); every decided or refused order is logged before the answer (INV-6).
package web

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hisefath/equinox/internal/decisionlog"
	"github.com/hisefath/equinox/internal/ingest"
	"github.com/hisefath/equinox/internal/market"
	"github.com/hisefath/equinox/internal/match"
	"github.com/hisefath/equinox/internal/refdata"
	"github.com/hisefath/equinox/internal/route"
)

// Server holds what the handlers read.
type Server struct {
	store   *ingest.Store
	log     *decisionlog.Log
	ref     refdata.Set
	limiter *RateLimiter
	version string
	started time.Time
	policy  route.Policy
}

// New wires a server. version is the running build's identifier (FR-043).
func New(s *ingest.Store, l *decisionlog.Log, ref refdata.Set, lim *RateLimiter, version string) *Server {
	return &Server{store: s, log: l, ref: ref, limiter: lim, version: version, started: time.Now().UTC()}
}

// Handler returns the mux for EP-1..EP-6.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.Index)
	mux.HandleFunc("GET /pairs", s.Pairs)
	mux.HandleFunc("POST /route", s.Route)
	mux.HandleFunc("GET /decisions", s.Decisions)
	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("GET /version", s.Version)
	return mux
}

// Problem is an RFC 9457 problem details body.
type Problem struct {
	Type          string             `json:"type"`
	Title         string             `json:"title"`
	Status        int                `json:"status"`
	Detail        string             `json:"detail,omitempty"`
	Instance      string             `json:"instance,omitempty"`
	Errors        []route.FieldError `json:"errors,omitempty"`
	PairID        string             `json:"pair_id,omitempty"`
	Verdict       string             `json:"verdict,omitempty"`
	DifferingPart string             `json:"differing_part,omitempty"`
	RetryAfter    int                `json:"retry_after_seconds,omitempty"`
}

var titles = map[string]string{
	"/problems/validation":               "Some fields are invalid",
	"/problems/unknown-pair":             "No such pair",
	"/problems/pair-not-routable":        "This pair is not the same bet",
	"/problems/rules-changed":            "The rules changed since matching",
	"/problems/rate-limited":             "Too many requests",
	"/problems/warming-up":               "Still collecting market data",
	"/problems/decision-log-unavailable": "Decisions can't be recorded right now",
	"/problems/internal":                 "Something went wrong",
}

func problem(typ string, status int, detail string) Problem {
	return Problem{Type: typ, Title: titles[typ], Status: status, Detail: detail}
}

func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-Id"); id != "" && len(id) <= 64 {
		return id
	}
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func wantsHTML(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") ||
		(strings.Contains(r.Header.Get("Accept"), "text/html") && !strings.Contains(r.Header.Get("Accept"), "application/json"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, p Problem) {
	p.Instance = "/requests/" + w.Header().Get("X-Request-Id")
	if p.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(p.RetryAfter))
	}
	if wantsHTML(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(p.Status)
		pageTmpl.ExecuteTemplate(w, "decision", map[string]any{"Problem": p, "Version": s.version})
		return
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

// Index renders the page: venue health, equivalent pairs, near misses and the order form (FR-042).
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	snap, ok := s.store.Current()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	data := map[string]any{"Version": s.version, "Snap": snap, "Warming": !ok, "Selected": r.URL.Query().Get("pair")}
	if ok {
		data["Pairs"] = s.pairViews(snap, match.Equivalent, 50)
		var near []map[string]any
		for i, n := range snap.Pairs.NearMisses {
			if i == 20 {
				break
			}
			near = append(near, map[string]any{"A": snap.Markets[n.Pair.AKey].Question, "B": snap.Markets[n.Pair.BKey].Question, "Reason": n.Reason})
		}
		data["Near"] = near
		data["NearCount"] = len(snap.Pairs.NearMisses)
	}
	pageTmpl.ExecuteTemplate(w, "index", data)
}

// PairView is one pair as the page and EP-2 show it.
type PairView struct {
	match.Pair
	AQuestion string `json:"a_question"`
	BQuestion string `json:"b_question"`
}

func (s *Server) pairViews(snap *ingest.Snapshot, kind string, limit int) []PairView {
	var out []PairView
	for _, p := range snap.Pairs.Pairs {
		if p.Verdict != kind {
			continue
		}
		out = append(out, PairView{Pair: p, AQuestion: question(snap, p.AKey), BQuestion: question(snap, p.BKey)})
		if limit > 0 && len(out) == limit {
			break
		}
	}
	return out
}

func question(snap *ingest.Snapshot, k market.Key) string {
	m := snap.Markets[k]
	if m.Outcome != "" && m.Outcome != "Yes" {
		return m.Question + " — " + m.Outcome
	}
	return m.Question
}

// Pairs serves EP-2: one page of pairs of one kind, highest score first.
func (s *Server) Pairs(w http.ResponseWriter, r *http.Request) {
	snap, ok := s.store.Current()
	if !ok {
		s.fail(w, r, Problem{Type: "/problems/warming-up", Title: titles["/problems/warming-up"], Status: 503, RetryAfter: 60})
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = match.Equivalent
	}
	limit, err := intParam(r, "limit", 50, 200)
	if err != nil || (kind != match.Equivalent && kind != match.Related && kind != match.Undecided) {
		p := problem("/problems/validation", 422, "bad kind or limit")
		p.Errors = []route.FieldError{{Pointer: "#/kind", Detail: "kind is equivalent, related or undecided; limit is 1-200"}}
		s.fail(w, r, p)
		return
	}
	all := s.pairViews(snap, kind, 0)
	start := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		raw, err := base64.RawURLEncoding.DecodeString(c)
		parts := strings.SplitN(string(raw), "|", 2)
		if err != nil || len(parts) != 2 || parts[0] != snap.AsOf.Format(time.RFC3339Nano) {
			p := problem("/problems/validation", 422, "cursor is from another snapshot; restart the list")
			p.Errors = []route.FieldError{{Pointer: "#/cursor", Detail: "stale or invalid cursor"}}
			s.fail(w, r, p)
			return
		}
		start, _ = strconv.Atoi(parts[1])
	}
	end := min(start+limit, len(all))
	if start > len(all) {
		start = len(all)
	}
	var next *string
	if end < len(all) {
		c := base64.RawURLEncoding.EncodeToString([]byte(snap.AsOf.Format(time.RFC3339Nano) + "|" + strconv.Itoa(end)))
		next = &c
	}
	writeJSON(w, 200, map[string]any{"snapshot_as_of": snap.AsOf, "data": nonNil(all[start:end]), "next_cursor": next})
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func intParam(r *http.Request, name string, def, max int) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("bad %s", name)
	}
	return n, nil
}

// routeRequest is EP-3's body (JSON or form).
type routeRequest struct {
	PairID         string  `json:"pair_id"`
	Side           string  `json:"side"`
	Quantity       int64   `json:"quantity"`
	Limit          *string `json:"limit"`
	CustomerRegion string  `json:"customer_region"`
}

// Route serves EP-3: validate, read the snapshot (no venue calls), decide, log, answer (SEQ-2).
func (s *Server) Route(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	w.Header().Set("X-Request-Id", rid)
	if wait, ok := s.limiter.Allow(clientIP(r)); !ok {
		s.fail(w, r, Problem{Type: "/problems/rate-limited", Title: titles["/problems/rate-limited"], Status: 429, RetryAfter: wait})
		return
	}
	var req routeRequest
	var fieldErrs route.FieldErrors
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		r.ParseForm()
		req.PairID, req.Side, req.CustomerRegion = r.PostForm.Get("pair_id"), r.PostForm.Get("side"), r.PostForm.Get("customer_region")
		if q, err := strconv.ParseInt(r.PostForm.Get("quantity"), 10, 64); err == nil {
			req.Quantity = q
		}
		if l := strings.TrimSpace(r.PostForm.Get("limit")); l != "" {
			req.Limit = &l
		}
	} else if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		fieldErrs = append(fieldErrs, route.FieldError{Pointer: "#", Detail: "body must be JSON with pair_id, side, quantity, customer_region"})
	}
	order := route.Order{PairID: req.PairID, Side: market.Side(req.Side), Quantity: req.Quantity, CustomerRegion: req.CustomerRegion}
	if req.Limit != nil {
		lim, err := market.ParseAmount(*req.Limit)
		if err != nil || lim <= 0 || lim > market.Dollar {
			fieldErrs = append(fieldErrs, route.FieldError{Pointer: "#/limit", Detail: "limit is a price between 0 and 1, up to 6 decimals"})
		} else {
			order.Limit = &lim
		}
	}
	if err := order.Validate(); err != nil {
		var fe route.FieldErrors
		if errors.As(err, &fe) {
			fieldErrs = append(fieldErrs, fe...)
		}
	}
	if len(fieldErrs) > 0 { // malformed: rejected, nothing logged (FR-016)
		p := problem("/problems/validation", 422, "the order is malformed")
		p.Errors = fieldErrs
		s.fail(w, r, p)
		return
	}
	snap, ok := s.store.Current()
	if !ok {
		s.fail(w, r, Problem{Type: "/problems/warming-up", Title: titles["/problems/warming-up"], Status: 503, RetryAfter: 60})
		return
	}
	pair, err := snap.Pairs.Routable(order.PairID, snap.Fingerprints())
	if err != nil {
		var p Problem
		switch {
		case errors.Is(err, match.ErrUnknownPair):
			p = problem("/problems/unknown-pair", 404, err.Error())
		case errors.Is(err, match.ErrRulesChanged):
			p = problem("/problems/rules-changed", 409, err.Error())
		default:
			p = problem("/problems/pair-not-routable", 409, err.Error())
			p.Verdict, p.DifferingPart = pair.Verdict, pair.DifferingPart
		}
		p.PairID = order.PairID
		refused := route.Decision{RequestID: rid, Order: order, SnapshotAsOf: snap.AsOf, Outcome: route.Refused,
			ProblemType: p.Type, Venues: []route.VenueResult{}, Evidence: nonNil(pair.Evidence), Caveats: nonNil(pair.Caveats)}
		refused.DecisionID = route.Route(order, nil, s.policy, snap.AsOf).DecisionID
		refused.LoggedAt = time.Now().UTC()
		if err := s.log.Append(refused); err != nil {
			s.fail(w, r, problem("/problems/decision-log-unavailable", 503, "the decision log could not be written"))
			return
		}
		s.fail(w, r, p)
		return
	}
	quotes := []route.Quote{s.quote(snap, pair.AKey, order.CustomerRegion), s.quote(snap, pair.BKey, order.CustomerRegion)}
	d := route.Route(order, quotes, s.policy, snap.AsOf)
	d.RequestID, d.LoggedAt = rid, time.Now().UTC()
	d.Evidence, d.Caveats = nonNil(pair.Evidence), nonNil(pair.Caveats)
	if err := s.log.Append(d); err != nil { // INV-6: no answer without a log line
		s.fail(w, r, problem("/problems/decision-log-unavailable", 503, "the decision log could not be written"))
		return
	}
	if wantsHTML(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pageTmpl.ExecuteTemplate(w, "decision", map[string]any{"D": d, "Version": s.version,
			"A": question(snap, pair.AKey), "B": question(snap, pair.BKey)})
		return
	}
	writeJSON(w, 200, d)
}

// quote builds the venue-neutral facts routing reads for one market (FR-014).
func (s *Server) quote(snap *ingest.Snapshot, k market.Key, region string) route.Quote {
	m := snap.Markets[k]
	h := snap.Health[m.Venue]
	elig, _ := s.ref.Eligible(m.Venue, region)
	q := route.Quote{Venue: m.Venue, Key: k, Book: snap.Books[k], Fee: m.Fee, MinQty: m.MinQty,
		Healthy: h.State == ingest.Healthy, Tradable: m.Tradable, UntradableReason: m.UntradableReason, Eligible: elig}
	if !q.Healthy {
		q.HealthReason = "venue " + h.State
		if h.LastError != "" {
			q.HealthReason += ": " + h.LastError
		}
	}
	return q
}

// Decisions serves EP-4: logged decisions, newest first.
func (s *Server) Decisions(w http.ResponseWriter, r *http.Request) {
	limit, err := intParam(r, "limit", 25, 100)
	if err == nil {
		var page []route.Decision
		var next string
		if page, next, err = s.log.Page(r.URL.Query().Get("cursor"), limit); err == nil {
			var nc *string
			if next != "" {
				nc = &next
			}
			writeJSON(w, 200, map[string]any{"data": nonNil(page), "next_cursor": nc})
			return
		}
	}
	p := problem("/problems/validation", 422, err.Error())
	p.Errors = []route.FieldError{{Pointer: "#/cursor", Detail: "invalid cursor or limit (1-100)"}}
	s.fail(w, r, p)
}

// Health serves EP-5: venue health, refresh counts, eligibility, log status and version (FR-003, FR-026).
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	type venueHealth struct {
		ingest.Health
		ingest.Report
		Regions []string `json:"regions"`
	}
	out := map[string]any{"snapshot_as_of": nil, "undecided_pairs": 0, "decision_log": "ok", "version": s.version}
	var venues []venueHealth
	if snap, ok := s.store.Current(); ok {
		out["snapshot_as_of"], out["undecided_pairs"] = snap.AsOf, snap.UndecidedPairs
		for _, name := range sortedKeys(snap.Health) {
			venues = append(venues, venueHealth{Health: snap.Health[name], Report: snap.Reports[name], Regions: s.ref.Eligibility[name].Regions})
		}
	}
	out["venues"] = nonNil(venues)
	writeJSON(w, 200, out)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// Version serves EP-6.
func (s *Server) Version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"version": s.version, "started_at": s.started})
}

func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimiter allows N requests per window per client (HLD TH-1). ponytail: in-memory map, fine for one instance.
type RateLimiter struct {
	mu     sync.Mutex
	n      int
	window time.Duration
	seen   map[string][]time.Time
	now    func() time.Time
}

// NewRateLimiter allows n requests per window per client.
func NewRateLimiter(n int, window time.Duration, now func() time.Time) *RateLimiter {
	return &RateLimiter{n: n, window: window, seen: map[string][]time.Time{}, now: now}
}

// Allow records a request and reports whether it is within the limit, with seconds to wait if not.
func (l *RateLimiter) Allow(client string) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	keep := l.seen[client][:0]
	for _, t := range l.seen[client] {
		if now.Sub(t) < l.window {
			keep = append(keep, t)
		}
	}
	if len(keep) >= l.n {
		l.seen[client] = keep
		return int(l.window.Seconds() - now.Sub(keep[0]).Seconds() + 1), false
	}
	l.seen[client] = append(keep, now)
	return 0, true
}

var pageTmpl = template.Must(template.New("page").Funcs(template.FuncMap{"num": func(n any) string {
	d := fmt.Sprint(n)
	for i := len(d) - 3; i > 0 && d[i-1] != '-'; i -= 3 {
		d = d[:i] + "," + d[i:]
	}
	return d
}, "venue": func(v string) string {
	switch v {
	case "kalshi":
		return "Kalshi"
	case "polymarket":
		return "Polymarket"
	}
	return v
}, "rule": func(r string) string {
	return map[string]string{route.RuleOnlyEligible: "it is the only venue this customer may use",
		route.RuleMostFilled: "it fills the most contracts", route.RuleLowestAllIn: "lowest total cost after fees",
		route.RuleTieBreak: "both cost the same; a fixed order breaks the tie"}[r]
}, "money": func(a any) string {
	switch v := a.(type) {
	case market.Amount:
		return "$" + v.String()
	case *market.Amount:
		if v == nil {
			return "—"
		}
		return "$" + v.String()
	}
	return fmt.Sprint(a)
}}).Parse(`{{define "head"}}<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Equinox</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Instrument+Serif&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#000;--panel:#14110b;--panel2:#1c1913;--line:#33302a;--text:#fff;--muted:#cdc9c1;--faint:#9d998f;--neon:#ccff00;--ink:#110e08;--down:#ff6b4f;--warn:#f0dc78}
*{box-sizing:border-box}html{background:var(--bg);scroll-behavior:smooth}body{margin:0;background:var(--bg);color:var(--text);font:17px/1.6 Inter,system-ui,sans-serif;-webkit-font-smoothing:antialiased}
a{color:var(--text)}code{font:13px ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--faint)}
.wrap{max-width:1180px;margin:0 auto;padding-left:20px;padding-right:20px}
nav{position:sticky;top:0;z-index:5;background:rgba(0,0,0,.9);backdrop-filter:blur(8px);border-bottom:1px solid var(--line)}
nav .wrap{display:flex;align-items:center;gap:20px;height:64px}
.brand{font:600 20px Inter,sans-serif;letter-spacing:-.02em;text-decoration:none;display:flex;align-items:center;gap:8px}
.brand i{width:18px;height:18px;border-radius:50%;background:conic-gradient(var(--neon) 0 50%,#fff 0)}
nav .links{display:flex;gap:22px;margin-left:auto;font-size:15px}nav .links a{text-decoration:none;color:var(--muted)}nav .links a:hover{color:var(--text)}
.pill{display:inline-block;white-space:nowrap;border:0;border-radius:36px;padding:12px 22px;font:600 16px Inter,sans-serif;cursor:pointer;text-decoration:none}
.pill.neon{background:var(--neon);color:var(--ink)}.pill.neon:hover{filter:brightness(.92)}
.pill.ghost{background:transparent;color:var(--text);border:1px solid #6b675f}.pill.ghost:hover{border-color:var(--text)}
.pill.small{padding:8px 16px;font-size:15px}
.banner{background:var(--neon);color:var(--ink);text-align:center;font-size:14px;font-weight:500;padding:8px 16px}
.serif{font-family:"Instrument Serif",Georgia,serif;font-weight:400;letter-spacing:-.01em}
.hero{text-align:center;padding-top:64px;padding-bottom:40px}.hero h1{font:400 clamp(52px,9vw,96px)/1 "Instrument Serif",Georgia,serif;margin:0 0 20px}
.hero p.lead{color:var(--muted);max-width:640px;margin:0 auto;font-size:19px}
.steps{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin:40px 0 24px;text-align:left}
.step{display:block;background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:20px 22px;text-decoration:none}
.step:hover{border-color:#6b675f}.step .num{color:var(--ink);font-size:14px}.step b{display:block;font-size:18px;margin:6px 0 4px}.step span{color:var(--muted);font-size:15px}
.num{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border-radius:50%;background:var(--neon);color:var(--ink);font-weight:700;font-size:14px}
.stats{display:flex;flex-wrap:wrap;gap:8px 28px;justify-content:center;color:var(--muted);font-size:15px;margin-bottom:56px}.stats b{color:var(--text)}
.grid{display:grid;grid-template-columns:minmax(0,1fr) 380px;gap:40px;align-items:start;padding-bottom:64px}
.sh{display:flex;align-items:center;gap:12px;margin:0 0 6px}.sh h2{font:700 28px/1.2 Inter,sans-serif;letter-spacing:-.02em;margin:0}
.sub{color:var(--muted);font-size:16px;margin:0 0 20px}
.card{background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:20px 22px;margin-bottom:14px}
.card.sel{border-color:var(--neon);box-shadow:0 0 0 1px var(--neon) inset}
.mk{display:flex;gap:10px;align-items:baseline;margin-bottom:8px;font-size:17px;font-weight:500}
.tag{flex:none;display:inline-block;font-size:12px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;border-radius:6px;padding:2px 8px;min-width:88px;text-align:center}
.tag.k{background:#243300;color:var(--neon)}.tag.p{background:#2a2a2a;color:#fff}.tag.bad{background:#3a140c;color:var(--down);min-width:0}
.why{color:var(--faint);font-size:14px;margin:10px 0 0}.why b{color:var(--muted);font-weight:600}
.cardfoot{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-top:14px}
details{margin-top:28px}summary{cursor:pointer;color:var(--muted);font-size:16px;padding:12px 0}summary:hover{color:var(--text)}
.rej{padding:14px 0;border-bottom:1px solid var(--line)}.rej .mk{font-size:16px;font-weight:400}
.ticket{background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:24px}
.field{padding:14px 0;border-bottom:1px solid var(--line)}.field label{display:block;font-weight:600;font-size:15px;margin-bottom:6px}
.field small{display:block;color:var(--faint);font-size:13px;margin-top:6px}
.ticket select,.ticket input{width:100%;background:#000;color:var(--text);border:1px solid #4a463f;border-radius:10px;padding:11px 12px;font:16px Inter,sans-serif}
.ticket select:focus,.ticket input:focus{outline:2px solid var(--neon);outline-offset:1px}
.ticket .pill{width:100%;margin-top:20px;padding:15px;font-size:17px}
.seg{display:flex;background:#000;border:1px solid #4a463f;border-radius:36px;padding:4px}.seg input{position:absolute;opacity:0;pointer-events:none}
.seg label{flex:1;text-align:center;margin:0;padding:9px;border-radius:36px;cursor:pointer;color:var(--muted);font-weight:600}
.seg input:checked+label{background:var(--neon);color:var(--ink)}.seg input:focus-visible+label{outline:2px solid #fff}
.health{display:flex;gap:16px;flex-wrap:wrap;margin-top:18px;font-size:14px;color:var(--muted)}
.dot{display:inline-block;width:9px;height:9px;border-radius:50%;margin-right:6px;background:var(--neon)}.dot.off{background:var(--down)}
.note{color:var(--faint);font-size:14px;margin-top:12px}
footer{border-top:1px solid var(--line);color:var(--faint);font-size:14px;padding:28px 0 48px}footer a{color:var(--muted);margin-right:18px}
.big{font:400 clamp(56px,10vw,92px)/1 "Instrument Serif",Georgia,serif;margin:6px 0 10px}.up{color:var(--neon)}.down{color:var(--down)}
.answer{font:700 clamp(30px,5vw,44px)/1.15 Inter,sans-serif;letter-spacing:-.02em;margin:10px 0 0}
.venues{display:grid;grid-template-columns:1fr 1fr;gap:14px;margin:28px 0}
.venue{background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:20px 22px}.venue.win{border-color:var(--neon);box-shadow:0 0 0 1px var(--neon) inset}
.venue h3{margin:0 0 2px;font-size:19px;display:flex;justify-content:space-between;align-items:center}.venue .cost{font:400 40px/1.1 "Instrument Serif",serif;margin:8px 0}
.venue dl{display:grid;grid-template-columns:auto 1fr;gap:4px 14px;margin:10px 0 0;font-size:15px}.venue dt{color:var(--faint)}.venue dd{margin:0}
.badge{font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:.04em;border-radius:6px;padding:3px 8px;background:var(--neon);color:var(--ink)}
.badge.no{background:#3a140c;color:var(--down)}.badge.lose{background:#2a2a2a;color:var(--muted)}
.actions{display:flex;flex-wrap:wrap;gap:12px;margin-top:28px}.actions form{margin:0}
.chips{margin-top:6px}.chip{display:inline-block;font-size:14px;color:var(--muted);border:1px solid var(--line);border-radius:36px;padding:3px 12px;margin:0 6px 8px 0}.chip.warn{border-color:#6a5a10;color:var(--warn)}
@media(min-width:901px) and (min-height:880px){.ticket{position:sticky;top:84px}}@media(max-width:900px){.grid{grid-template-columns:1fr}.steps{grid-template-columns:1fr}.venues{grid-template-columns:1fr}nav .links a.hide{display:none}}
</style></head><body>
<div class="banner">Simulation only: no real orders are placed. Prices come from the venues' public data.</div>
<nav><div class="wrap"><a class="brand" href="/"><i></i>Equinox</a><div class="links"><a class="hide" href="/#pairs">Pairs</a><a href="/decisions">Decision log</a><a class="hide" href="/health">Health</a></div><a class="pill neon small" href="/#ticket">Route an order</a></div></nav>{{end}}
{{define "foot"}}<footer><div class="wrap"><a href="/health">Health</a><a href="/decisions">Decision log</a><a href="/pairs">Pairs API</a><a href="/version">Version {{.Version}}</a><p>Equinox is a feasibility prototype: it matches equivalent prediction markets across Kalshi and Polymarket and simulates where an order should go, explaining every decision.</p></div></footer></body></html>{{end}}
{{define "index"}}{{template "head" .}}
{{if .Warming}}<section class="hero wrap"><h1>Warming up</h1><p class="lead">Still collecting market data. Refresh in a minute.</p></section>{{else}}
<section class="hero wrap"><h1>Same bet.<br>Best venue.</h1>
<p class="lead">Equinox finds markets on Kalshi and Polymarket that pay out in exactly the same situations, then sends a hypothetical order to the cheapest venue the customer is allowed to use, and explains why.</p>
<div class="steps">
<a class="step" href="#pairs"><span class="num">1</span><b>Pick a matched pair</b><span>{{len .Pairs}} pairs pay the same on both venues, with the evidence shown.</span></a>
<a class="step" href="#ticket"><span class="num">2</span><b>Place a test order</b><span>Choose YES or NO, how many contracts, and where the customer lives.</span></a>
<div class="step"><span class="num">3</span><b>See where it goes and why</b><span>Cost on each venue after fees, which one wins, and the rule that decided it.</span></div></div>
<p class="stats">{{range $name, $r := .Snap.Reports}}<span><b>{{num $r.Considered}}</b> active markets on {{venue $name}}</span>{{end}}<span><b class="up">{{len .Pairs}}</b> equivalent pairs</span><span><b>{{num .NearCount}}</b> look-alikes rejected</span></p></section>
<div class="wrap"><div class="grid"><main id="pairs">
<div class="sh"><span class="num">1</span><h2>Pick a matched pair</h2></div>
<p class="sub">Each pair agrees on every part that decides the payout: the outcome, threshold, statistic and edge rules. Only the payout date may differ.</p>
{{range .Pairs}}<div class="card{{if eq .PairID $.Selected}} sel{{end}}" id="p-{{.PairID}}">
<div class="mk"><span class="tag k">Kalshi</span><span>{{.AQuestion}}</span></div>
<div class="mk"><span class="tag p">Polymarket</span><span>{{.BQuestion}}</span></div>
<p class="why"><b>Agrees on:</b> {{range $i, $e := .Evidence}}{{if $i}} · {{end}}{{$e}}{{end}}{{if .Caveats}}<br><b>Worth checking:</b> {{range $i, $c := .Caveats}}{{if $i}} · {{end}}{{$c}}{{end}}{{end}}</p>
<div class="cardfoot"><code>{{.PairID}}</code><a class="pill {{if eq .PairID $.Selected}}neon{{else}}ghost{{end}} small" href="/?pair={{.PairID}}#ticket">{{if eq .PairID $.Selected}}Selected ✓{{else}}Route this pair →{{end}}</a></div></div>{{end}}
<details id="rejected"><summary>Show the closest rejected look-alikes (top {{len .Near}} of {{num .NearCount}}). They look the same but could pay differently, so they are never routed.</summary>
{{range .Near}}<div class="rej"><div class="mk"><span class="tag k">Kalshi</span><span>{{.A}}</span></div><div class="mk"><span class="tag p">Polymarket</span><span>{{.B}}</span></div><span class="tag bad">{{.Reason}}</span></div>{{end}}</details>
</main>
<aside class="ticket" id="ticket"><div class="sh"><span class="num">2</span><h2 style="font-size:22px">Place a test order</h2></div>
<form method="post" action="/route">
<div class="field"><label for="pair">Market</label><select id="pair" name="pair_id">{{range .Pairs}}<option value="{{.PairID}}"{{if eq .PairID $.Selected}} selected{{end}}>{{.BQuestion}}</option>{{end}}</select><small>Tip: press "Route this pair" on any card to fill this in.</small></div>
<div class="field"><label>Side</label><div class="seg"><input type="radio" name="side" id="s-yes" value="yes" checked><label for="s-yes">Buy YES</label><input type="radio" name="side" id="s-no" value="no"><label for="s-no">Buy NO</label></div></div>
<div class="field"><label for="qty">Contracts</label><input id="qty" name="quantity" type="number" min="1" value="500"><small>Each contract pays $1 if your side wins.</small></div>
<div class="field"><label for="cust">Customer lives</label><select id="cust" name="customer_region"><option value="non-US">Outside the US</option><option value="US">In the US</option></select><small>US customers can only use Kalshi.</small></div>
<div class="field" style="border:0"><label for="lim">Max price per contract <span style="color:var(--faint);font-weight:400">(optional)</span></label><input id="lim" name="limit" placeholder="e.g. 0.05" inputmode="decimal"></div>
<button class="pill neon">Route order →</button></form>
<div class="health">{{range $name, $h := .Snap.Health}}<span><span class="dot{{if ne $h.State "healthy"}} off{{end}}"></span>{{venue $name}} {{$h.State}}</span>{{end}}</div>
<p class="note">Prices as of {{.Snap.AsOf.Format "Jan 2, 15:04 UTC"}}. Routing never calls a venue: it reads this snapshot only.</p></aside>
</div></div>{{end}}{{template "foot" .}}{{end}}
{{define "decision"}}{{template "head" .}}<div class="wrap" style="padding-top:48px;padding-bottom:64px;max-width:920px">
<div class="sh"><span class="num">3</span><span style="color:var(--muted)">Where the order goes, and why</span></div>
{{with .Problem}}<p class="answer down">Not routed: {{.Title}}</p><p class="sub" style="margin-top:12px">{{.Detail}}</p>{{range .Errors}}<p class="sub">{{.Pointer}}: {{.Detail}}</p>{{end}}{{end}}
{{with .D}}<p class="sub" style="margin:14px 0 0">{{$.B}}</p>
{{if .Venue}}<p class="answer">Send it to <span class="up">{{venue .Venue}}</span></p>
<div class="big">{{money .AllIn}}</div>
<p style="font-size:19px;margin:0">for {{.FilledQty}} {{.Order.Side}} contracts at {{money .AvgPrice}} each, plus {{money .Fees}} in fees{{if .UnfilledQty}} · <span class="down">{{.UnfilledQty}} could not be filled</span>{{end}}.</p>
<p style="font-size:18px;color:var(--muted)">Why: <b style="color:var(--text)">{{rule .DecidingRule}}</b>{{if .Saving}}{{if ne (money .Saving) "$0"}} · <span class="up">saves {{money .Saving}}</span> against the other venue{{end}}{{end}}.</p>{{else}}<p class="answer down">Not routed: {{.Outcome}}</p>{{end}}
<div class="venues">{{range .Venues}}<div class="venue{{if eq .Venue $.D.Venue}} win{{end}}"><h3>{{venue .Venue}} {{if eq .Venue $.D.Venue}}<span class="badge">Chosen</span>{{else if not .Eligible}}<span class="badge no">Not allowed</span>{{else}}<span class="badge lose">Not chosen</span>{{end}}</h3>
<div class="cost">{{money .AllIn}}</div><dl><dt>Filled</dt><dd>{{.Filled}} contracts</dd><dt>Reason</dt><dd>{{.Reason}}</dd><dt>Market</dt><dd><code>{{.MarketKey}}</code></dd></dl></div>{{end}}</div>
<div class="card"><p style="margin:0 0 6px;font-weight:600">Why these two markets are the same bet</p><div class="chips">{{range .Evidence}}<span class="chip">{{.}}</span>{{end}}{{range .Caveats}}<span class="chip warn">{{.}}</span>{{end}}</div>
<p class="note">Kalshi: {{$.A}}<br>Polymarket: {{$.B}}<br>Decision <code>{{.DecisionID}}</code> is saved in the <a href="/decisions">decision log</a>.</p></div>
<div class="actions"><form method="post" action="/route"><input type="hidden" name="pair_id" value="{{.Order.PairID}}"><input type="hidden" name="side" value="{{.Order.Side}}"><input type="hidden" name="quantity" value="{{.Order.Quantity}}"><input type="hidden" name="customer_region" value="{{if eq .Order.CustomerRegion "US"}}non-US{{else}}US{{end}}"><button class="pill neon">Try it for a customer {{if eq .Order.CustomerRegion "US"}}outside the US{{else}}in the US{{end}} →</button></form>
<a class="pill ghost" href="/?pair={{.Order.PairID}}#ticket">Change the order</a><a class="pill ghost" href="/#pairs">Pick another pair</a></div>{{end}}
{{if .Problem}}<div class="actions"><a class="pill ghost" href="/#ticket">Back to the order</a></div>{{end}}
</div>{{template "foot" .}}{{end}}`))
