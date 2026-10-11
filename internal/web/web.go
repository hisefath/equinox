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

var pageTmpl = template.Must(template.New("page").Funcs(template.FuncMap{"money": func(a any) string {
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
}}).Parse(`{{define "head"}}<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Equinox</title>
<style>body{font:14px/1.45 system-ui,sans-serif;margin:24px;max-width:1100px;color:#222}table{border-collapse:collapse;width:100%;margin:8px 0 20px}td,th{border-bottom:1px solid #ddd;padding:4px 6px;text-align:left;vertical-align:top}th{background:#f5f5f5}.bad{color:#b00}.ok{color:#070}code{font-size:12px}form label{display:inline-block;margin:4px 12px 4px 0}</style></head><body>
<h1>Equinox <small style="font-weight:normal;color:#777">cross-venue prediction-market matching and routing · {{.Version}}</small></h1>{{end}}
{{define "index"}}{{template "head" .}}
{{if .Warming}}<p class="bad">Still collecting market data. Refresh in a minute.</p>{{else}}
<h2>Venue health <small>snapshot {{.Snap.AsOf.Format "2006-01-02 15:04:05 UTC"}}</small></h2>
<table><tr><th>Venue</th><th>State</th><th>Last good data</th><th>Last error</th></tr>
{{range $name, $h := .Snap.Health}}<tr><td>{{$name}}</td><td class="{{if eq $h.State "healthy"}}ok{{else}}bad{{end}}">{{$h.State}}</td><td>{{$h.LastGoodAt.Format "15:04:05 UTC"}}</td><td>{{$h.LastError}}</td></tr>{{end}}</table>
<h2>Route a hypothetical order</h2>
<form method="post" action="/route"><label>Pair <select name="pair_id">{{range .Pairs}}<option value="{{.PairID}}">{{.AQuestion}} ⇄ {{.BQuestion}}</option>{{end}}</select></label><br>
<label>Side <select name="side"><option>yes</option><option>no</option></select></label>
<label>Quantity <input name="quantity" type="number" min="1" value="100"></label>
<label>Limit (0-1, optional) <input name="limit" size="6"></label>
<label>Customer region <select name="customer_region"><option>US</option><option>non-US</option></select></label>
<button>Route</button></form>
<h2>Equivalent pairs ({{len .Pairs}} shown)</h2>
<table><tr><th>Market A</th><th>Market B</th><th>Evidence</th><th>Caveats</th></tr>
{{range .Pairs}}<tr><td>{{.AQuestion}}<br><code>{{.AKey}}</code></td><td>{{.BQuestion}}<br><code>{{.BKey}}</code></td><td>{{range .Evidence}}{{.}}<br>{{end}}</td><td>{{range .Caveats}}{{.}}<br>{{end}}</td></tr>{{end}}</table>
<h2>Closest rejected look-alikes</h2>
<table><tr><th>Market A</th><th>Market B</th><th>Why rejected</th></tr>{{range .Near}}<tr><td>{{.A}}</td><td>{{.B}}</td><td>{{.Reason}}</td></tr>{{end}}</table>
{{end}}<p><a href="/health">health</a> · <a href="/decisions">decision log</a> · <a href="/pairs">pairs (JSON)</a> · <a href="/version">version</a></p></body></html>{{end}}
{{define "decision"}}{{template "head" .}}
{{with .Problem}}<h2 class="bad">{{.Title}}</h2><p>{{.Detail}}</p>{{range .Errors}}<p>{{.Pointer}}: {{.Detail}}</p>{{end}}{{end}}
{{with .D}}<h2>Decision <code>{{.DecisionID}}</code>: {{.Outcome}}</h2>
<p>{{$.A}} ⇄ {{$.B}}</p>
{{if .Venue}}<p>Route <b>{{.FilledQty}}</b> to <b>{{.Venue}}</b> ({{.MarketKey}}) at avg {{money .AvgPrice}}; fees {{money .Fees}}; all-in {{money .AllIn}}{{if .UnfilledQty}}; {{.UnfilledQty}} unfilled{{end}}.</p>
<p>Deciding rule: <b>{{.DecidingRule}}</b>{{if .Saving}}, saving {{money .Saving}} against the next-best venue{{end}}.</p>{{end}}
<table><tr><th>Venue</th><th>Market</th><th>Eligible</th><th>Filled</th><th>All-in</th><th>Reason</th></tr>{{range .Venues}}<tr><td>{{.Venue}}</td><td><code>{{.MarketKey}}</code></td><td>{{.Eligible}}</td><td>{{.Filled}}</td><td>{{money .AllIn}}</td><td>{{.Reason}}</td></tr>{{end}}</table>
<p><b>Match evidence:</b> {{range .Evidence}}{{.}}; {{end}}</p><p><b>Caveats:</b> {{range .Caveats}}{{.}}; {{end}}</p>{{end}}
<p><a href="/">back</a></p></body></html>{{end}}`))
