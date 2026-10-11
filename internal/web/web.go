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
	data := map[string]any{"Version": s.version, "Snap": snap, "Warming": !ok}
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
<link href="https://fonts.googleapis.com/css2?family=Instrument+Serif&family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{--bg:#000;--panel:#110e08;--line:#2a2722;--text:#fff;--muted:#9b978f;--neon:#ccff00;--ink:#110e08;--up:#ccff00;--down:#ff5a3c}
*{box-sizing:border-box}html{background:var(--bg)}body{margin:0;background:var(--bg);color:var(--text);font:15px/1.5 Inter,system-ui,sans-serif;-webkit-font-smoothing:antialiased}
a{color:var(--text)}code{font:12px ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--muted)}
.wrap{max-width:1180px;margin:0 auto;padding:0 20px}
nav{position:sticky;top:0;z-index:5;background:rgba(0,0,0,.85);backdrop-filter:blur(8px);border-bottom:1px solid var(--line)}
nav .wrap{display:flex;align-items:center;gap:20px;height:64px}
.brand{font:500 20px Inter,sans-serif;letter-spacing:-.02em;text-decoration:none;display:flex;align-items:center;gap:8px}
.brand i{width:18px;height:18px;border-radius:50%;background:conic-gradient(var(--neon) 0 50%,#fff 0)}
nav .links{display:flex;gap:18px;margin-left:auto;font-size:14px}nav .links a{text-decoration:none;color:var(--muted)}nav .links a:hover{color:var(--text)}
.pill{display:inline-block;white-space:nowrap;border:0;border-radius:36px;padding:10px 20px;font:500 14px Inter,sans-serif;cursor:pointer;text-decoration:none}
.pill.neon{background:var(--neon);color:var(--ink)}.pill.neon:hover{filter:brightness(.92)}.pill.ghost{background:transparent;color:var(--text);border:1px solid var(--text)}
.banner{background:var(--neon);color:var(--ink);text-align:center;font-size:13px;padding:8px 16px}
h1,h2,.serif{font-family:"Instrument Serif",Georgia,serif;font-weight:400;letter-spacing:-.01em}
.hero{text-align:center;padding-top:72px;padding-bottom:40px}.hero h1{font-size:clamp(48px,9vw,104px);line-height:1;margin:0 0 20px}
.hero p{color:var(--muted);max-width:620px;margin:0 auto 28px;font-size:17px}
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:1px;background:var(--line);border:1px solid var(--line);border-radius:16px;overflow:hidden;margin:0 0 48px}
.stat{background:var(--bg);padding:20px}.stat b{display:block;font:400 40px "Instrument Serif",serif}.stat span{color:var(--muted);font-size:13px}
.grid{display:grid;grid-template-columns:minmax(0,1fr) 360px;gap:40px;align-items:start;padding-bottom:64px}
h2{font-size:36px;margin:0 0 6px}.sub{color:var(--muted);font-size:14px;margin:0 0 20px}
.row{display:flex;gap:16px;align-items:flex-start;padding:18px 0;border-bottom:1px solid var(--line)}
.row .q{flex:1;min-width:0}.row .q div{margin-bottom:6px}
.tag{display:inline-block;font-size:11px;font-weight:600;letter-spacing:.04em;text-transform:uppercase;border-radius:6px;padding:2px 7px;margin-right:6px;vertical-align:1px}
.tag.k{background:#1d2a00;color:var(--neon)}.tag.p{background:#1f1f1f;color:#fff}.tag.bad{background:#2b0f0a;color:var(--down)}
.chips{margin-top:8px}.chip{display:inline-block;font-size:12px;color:var(--muted);border:1px solid var(--line);border-radius:36px;padding:2px 10px;margin:0 6px 6px 0}
.chip.warn{border-color:#5a4a00;color:#e8d36a}
.ticket{position:sticky;top:88px;background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:24px}
.ticket h3{font:500 18px Inter,sans-serif;margin:0 0 18px}
.ticket label{display:flex;justify-content:space-between;align-items:center;gap:12px;padding:12px 0;border-bottom:1px solid var(--line);font-size:14px}
.ticket select,.ticket input{background:#000;color:var(--text);border:1px solid var(--line);border-radius:8px;padding:8px 10px;font:14px Inter,sans-serif;max-width:190px;width:100%}
.ticket input{text-align:right}.ticket .pill{width:100%;margin-top:20px;padding:14px}
.seg{display:flex;background:#000;border-radius:36px;padding:4px;margin-bottom:8px}.seg input{display:none}
.seg label{flex:1;justify-content:center;border:0;padding:8px;border-radius:36px;cursor:pointer;color:var(--muted)}
.seg input:checked+label{background:var(--neon);color:var(--ink);font-weight:600}
.health{display:flex;gap:10px;flex-wrap:wrap;margin-top:18px;font-size:13px;color:var(--muted)}
.dot{display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px;background:var(--up)}.dot.off{background:var(--down)}
.note{color:var(--muted);font-size:12px;margin-top:14px}
footer{border-top:1px solid var(--line);color:var(--muted);font-size:13px;padding:28px 0 48px}footer a{color:var(--muted);margin-right:16px}
.big{font:400 clamp(56px,10vw,96px)/1 "Instrument Serif",serif;margin:8px 0}.up{color:var(--up)}.down{color:var(--down)}
.card{background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:24px;margin-bottom:24px}
table{width:100%;border-collapse:collapse;font-size:14px}th{text-align:left;color:var(--muted);font-weight:500;font-size:12px;text-transform:uppercase;letter-spacing:.04em;padding:10px 8px;border-bottom:1px solid var(--line)}
td{padding:14px 8px;border-bottom:1px solid var(--line);vertical-align:top}
@media(max-width:860px){.grid{grid-template-columns:1fr}.ticket{position:static}.stats{grid-template-columns:repeat(2,1fr)}nav .links a.hide{display:none}}
</style></head><body>
<div class="banner">Simulation only: no real orders are placed. Prices come from the venues' public data.</div>
<nav><div class="wrap"><a class="brand" href="/"><i></i>Equinox</a><div class="links"><a class="hide" href="/#pairs">Pairs</a><a class="hide" href="/#rejected">Rejected</a><a href="/decisions">Decisions</a><a class="hide" href="/health">Health</a></div><a class="pill neon" href="/#ticket">Route an order</a></div></nav>{{end}}
{{define "foot"}}<footer><div class="wrap"><a href="/health">Health</a><a href="/decisions">Decision log</a><a href="/pairs">Pairs API</a><a href="/version">Version {{.Version}}</a><p>Equinox is a feasibility prototype: it matches equivalent prediction markets across Kalshi and Polymarket and simulates where an order should go, explaining every decision.</p></div></footer></body></html>{{end}}
{{define "index"}}{{template "head" .}}
{{if .Warming}}<section class="hero wrap"><h1>Warming up</h1><p>Still collecting market data. Refresh in a minute.</p></section>{{else}}
<section class="hero wrap"><h1>Same bet.<br>Best venue.</h1>
<p>Equinox finds prediction markets on Kalshi and Polymarket that pay out in exactly the same situations, then routes a hypothetical order to the cheapest venue you're allowed to use, and shows its reasoning.</p>
<a class="pill neon" href="#ticket">Route an order</a></section>
<div class="wrap"><div class="stats">
{{range $name, $r := .Snap.Reports}}<div class="stat"><b>{{num $r.Considered}}</b><span>active markets on {{venue $name}} ({{num $r.Listed}} listed)</span></div>{{end}}
<div class="stat"><b class="up">{{len .Pairs}}</b><span>equivalent pairs, routable</span></div>
<div class="stat"><b class="down">{{num .NearCount}}</b><span>look-alikes rejected, never routed</span></div></div>
<div class="grid"><main>
<section id="pairs"><h2>Equivalent pairs</h2><p class="sub">Every settlement part agrees: same outcome, threshold, statistic and edge rules. Only payout timing may differ.</p>
{{range .Pairs}}<div class="row"><div class="q"><div><span class="tag k">Kalshi</span>{{.AQuestion}}</div><div><span class="tag p">Poly</span>{{.BQuestion}}</div>
<div class="chips">{{range .Evidence}}<span class="chip">{{.}}</span>{{end}}{{range .Caveats}}<span class="chip warn">{{.}}</span>{{end}}</div></div><code>{{.PairID}}</code></div>{{end}}</section>
<section id="rejected" style="margin-top:56px"><h2>Closest rejected look-alikes</h2><p class="sub">They look like the same bet but could pay differently, so they are never routed. Top {{len .Near}} of {{num .NearCount}}.</p>
{{range .Near}}<div class="row"><div class="q"><div><span class="tag k">Kalshi</span>{{.A}}</div><div><span class="tag p">Poly</span>{{.B}}</div></div><span class="tag bad">{{.Reason}}</span></div>{{end}}</section>
</main>
<aside class="ticket" id="ticket"><h3>Route a hypothetical order</h3>
<form method="post" action="/route">
<div class="seg"><input type="radio" name="side" id="s-yes" value="yes" checked><label for="s-yes">Buy YES</label><input type="radio" name="side" id="s-no" value="no"><label for="s-no">Buy NO</label></div>
<label>Market<select name="pair_id">{{range .Pairs}}<option value="{{.PairID}}">{{.BQuestion}}</option>{{end}}</select></label>
<label>Contracts<input name="quantity" type="number" min="1" value="500"></label>
<label>Limit price<input name="limit" placeholder="optional, 0–1" inputmode="decimal"></label>
<label>Customer<select name="customer_region"><option value="non-US">Outside the US</option><option value="US">In the US</option></select></label>
<button class="pill neon">Route order</button></form>
<div class="health">{{range $name, $h := .Snap.Health}}<span><span class="dot{{if ne $h.State "healthy"}} off{{end}}"></span>{{venue $name}} {{$h.State}}</span>{{end}}</div>
<p class="note">Snapshot {{.Snap.AsOf.Format "Jan 2, 15:04:05 UTC"}}. US customers are never routed to Polymarket.</p></aside>
</div></div>{{end}}{{template "foot" .}}{{end}}
{{define "decision"}}{{template "head" .}}<div class="wrap" style="padding:56px 20px 64px;max-width:900px">
{{with .Problem}}<p class="sub">Order not routed</p><div class="big down">{{.Title}}</div><p>{{.Detail}}</p>{{range .Errors}}<p class="sub">{{.Pointer}}: {{.Detail}}</p>{{end}}{{end}}
{{with .D}}<p class="sub">Decision <code>{{.DecisionID}}</code> · {{.Outcome}}</p>
<p>{{$.B}}</p>
{{if .Venue}}<div class="big">{{money .AllIn}}</div>
<p style="font-size:18px">Buy <b>{{.FilledQty}}</b> {{.Order.Side}} on <b>{{venue .Venue}}</b> at avg {{money .AvgPrice}} + {{money .Fees}} fees{{if .UnfilledQty}} · <span class="down">{{.UnfilledQty}} unfilled</span>{{end}}</p>
<p>{{if .Saving}}<span class="up">▲ {{money .Saving}} saved</span> vs the next-best venue · {{end}}rule: <b>{{.DecidingRule}}</b></p>{{end}}
<div class="card" style="margin-top:32px"><table><tr><th>Venue</th><th>Eligible</th><th>Filled</th><th>All-in</th><th>Why</th></tr>{{range .Venues}}<tr><td><b>{{venue .Venue}}</b><br><code>{{.MarketKey}}</code></td><td>{{if .Eligible}}<span class="up">yes</span>{{else}}<span class="down">no</span>{{end}}</td><td>{{.Filled}}</td><td>{{money .AllIn}}</td><td>{{.Reason}}</td></tr>{{end}}</table></div>
<div class="card"><p class="sub" style="margin:0 0 8px">Why these are the same bet</p><div class="chips">{{range .Evidence}}<span class="chip">{{.}}</span>{{end}}{{range .Caveats}}<span class="chip warn">{{.}}</span>{{end}}</div>
<p class="note">Kalshi: {{$.A}}<br>Polymarket: {{$.B}}</p></div>{{end}}
<a class="pill ghost" href="/#ticket">Route another order</a></div>{{template "foot" .}}{{end}}`))
