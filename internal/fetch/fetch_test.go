package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetriesTransientFailuresThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer srv.Close()
	c := New(nil, 0)
	c.Backoff = time.Millisecond
	var v struct{ OK bool }
	if err := c.GetJSON(context.Background(), srv.URL, &v); err != nil || !v.OK || calls.Load() != 3 {
		t.Fatalf("err=%v v=%+v calls=%d", err, v, calls.Load())
	}
}

func TestDoesNotRetryPermanentFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"404":       func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"malformed": func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"markets": [`)) },
	} {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); h(w, r) }))
		c := New(nil, 0)
		c.Backoff = time.Millisecond
		err := c.GetJSON(context.Background(), srv.URL, &struct{}{})
		srv.Close()
		if err == nil || calls.Load() != 1 {
			t.Errorf("%s: err=%v calls=%d, want an error after exactly 1 call", name, err, calls.Load())
		}
		if name == "malformed" && !errors.Is(err, ErrMalformed) {
			t.Errorf("malformed: %v should wrap ErrMalformed", err)
		}
	}
}

func TestContextDeadlineBoundsSlowVenue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := New(nil, 0).GetJSON(ctx, srv.URL, &struct{}{})
	if err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("err=%v after %s; want a fast deadline error", err, time.Since(start))
	}
}

func TestRateLimitSpacesRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	defer srv.Close()
	c := New(nil, 20*time.Millisecond)
	start := time.Now()
	for range 4 {
		if err := c.GetJSON(context.Background(), srv.URL, &struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	if el := time.Since(start); el < 60*time.Millisecond {
		t.Errorf("4 requests took %s; want >= 60ms at a 20ms interval", el)
	}
}

func TestRecordThenReplay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"q":"` + r.URL.Query().Get("q") + `"}`))
	}))
	dir := t.TempDir()
	rec := New(Recorder{Dir: dir, Next: http.DefaultTransport}, 0)
	var v struct{ Q string }
	if err := rec.GetJSON(context.Background(), srv.URL+"/x?q=hello", &v); err != nil || v.Q != "hello" {
		t.Fatalf("record: %v %+v", err, v)
	}
	srv.Close() // replay must not need the network

	rep := New(Recorder{Dir: dir}, 0)
	v.Q = ""
	if err := rep.GetJSON(context.Background(), srv.URL+"/x?q=hello", &v); err != nil || v.Q != "hello" {
		t.Fatalf("replay: %v %+v", err, v)
	}
	var se *StatusError
	if err := rep.GetJSON(context.Background(), srv.URL+"/x?q=other", &v); !errors.As(err, &se) || se.Code != 404 {
		t.Fatalf("unrecorded request should be a 404, got %v", err)
	}
}
