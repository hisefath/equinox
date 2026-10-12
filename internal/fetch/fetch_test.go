package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func reply(status int, body string, h ...string) *http.Response {
	hdr := http.Header{}
	for i := 0; i+1 < len(h); i += 2 {
		hdr.Set(h[i], h[i+1])
	}
	return &http.Response{StatusCode: status, Header: hdr, Body: io.NopCloser(strings.NewReader(body))}
}

func fast(t http.RoundTripper, retries int) *Client {
	c := New(t, 0, time.Second, retries, nil)
	c.backoff = time.Millisecond
	return c
}

func TestRetriesFiveHundredsThenSucceeds(t *testing.T) {
	var n atomic.Int32
	c := fast(rtFunc(func(r *http.Request) (*http.Response, error) {
		if n.Add(1) < 3 {
			return reply(503, "busy"), nil
		}
		return reply(200, `{"a":1}`), nil
	}), 3)
	var out struct{ A int }
	if err := c.GetJSON(context.Background(), "https://x.test/a", &out); err != nil || out.A != 1 || n.Load() != 3 {
		t.Fatalf("err=%v out=%+v calls=%d", err, out, n.Load())
	}
}

func TestRetriesAreBounded(t *testing.T) {
	var n atomic.Int32
	c := fast(rtFunc(func(r *http.Request) (*http.Response, error) {
		n.Add(1)
		return nil, errors.New("timeout")
	}), 3)
	if err := c.GetJSON(context.Background(), "https://x.test/a", &struct{}{}); err == nil || n.Load() != 4 {
		t.Fatalf("err=%v calls=%d, want error after 1+3 attempts", err, n.Load())
	}
}

func TestTimeoutIsRetried(t *testing.T) {
	var n atomic.Int32
	c := fast(rtFunc(func(r *http.Request) (*http.Response, error) {
		if n.Add(1) == 1 {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return reply(200, `{}`), nil
	}), 1)
	c.timeout = 20 * time.Millisecond
	if err := c.GetJSON(context.Background(), "https://x.test/a", &struct{}{}); err != nil || n.Load() != 2 {
		t.Fatalf("err=%v calls=%d", err, n.Load())
	}
}

func TestRefusals(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		resp *http.Response
		want ErrRefused
	}{
		{reply(429, "", "Retry-After", "7"), ErrRefused{429, 7 * time.Second}},
		{reply(429, "", "Retry-After", now.Add(90*time.Second).Format(http.TimeFormat)), ErrRefused{429, 90 * time.Second}},
		{reply(429, ""), ErrRefused{429, DefaultRefusedWait}},
		{reply(403, "<html>blocked</html>"), ErrRefused{403, 5 * time.Minute}},
	}
	for _, tc := range cases {
		var n atomic.Int32
		c := New(rtFunc(func(*http.Request) (*http.Response, error) { n.Add(1); return tc.resp, nil }), 0, time.Second, 3, func() time.Time { return now })
		err := c.GetJSON(context.Background(), "https://x.test/a", &struct{}{})
		var got ErrRefused
		if !errors.As(err, &got) || got != tc.want || n.Load() != 1 {
			t.Errorf("got %v (calls %d), want %v and no retry", err, n.Load(), tc.want)
		}
	}
}

func TestMalformedAndClientErrorsAreNotRetried(t *testing.T) {
	var n atomic.Int32
	c := fast(rtFunc(func(*http.Request) (*http.Response, error) { n.Add(1); return reply(200, "<html>"), nil }), 3)
	if err := c.GetJSON(context.Background(), "https://x.test/a", &struct{}{}); !errors.Is(err, ErrMalformed) || n.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, n.Load())
	}
	c = fast(rtFunc(func(*http.Request) (*http.Response, error) { return reply(404, strings.Repeat("x", 900)), nil }), 3)
	err := c.GetJSON(context.Background(), "https://x.test/a", &struct{}{})
	if err == nil || errors.Is(err, ErrMalformed) || len(err.Error()) > 600 {
		t.Fatalf("404 err=%v", err)
	}
}

func TestPacingPerHost(t *testing.T) {
	c := New(rtFunc(func(*http.Request) (*http.Response, error) { return reply(200, `{}`), nil }), 20, time.Second, 0, nil)
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := c.GetJSON(context.Background(), "https://a.test/x", &struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 90*time.Millisecond { // 3 requests at 20/s: 2 gaps of 50 ms
		t.Fatalf("3 paced requests took %s, want >= 100ms", d)
	}
	start = time.Now()
	if err := c.GetJSON(context.Background(), "https://b.test/x", &struct{}{}); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 40*time.Millisecond {
		t.Fatalf("other host waited %s; pacing must be per host", d)
	}
}

func TestPostJSONSendsBody(t *testing.T) {
	c := fast(rtFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || string(b) != `[{"token_id":"1"}]` || r.Header.Get("Content-Type") != "application/json" {
			return reply(400, "bad"), nil
		}
		return reply(200, `[]`), nil
	}), 0)
	var out []any
	if err := c.PostJSON(context.Background(), "https://x.test/books", []map[string]string{{"token_id": "1"}}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestRecorderRecordsThenReplays(t *testing.T) {
	dir := t.TempDir()
	var n atomic.Int32
	live := rtFunc(func(r *http.Request) (*http.Response, error) {
		n.Add(1)
		b, _ := io.ReadAll(r.Body)
		return reply(200, `{"echo":`+string(b)+`}`, "Content-Type", "application/json", "Set-Cookie", "x=1"), nil
	})
	rec := New(NewRecorder(dir, Record, live), 0, time.Second, 0, nil)
	var out struct{ Echo string }
	if err := rec.PostJSON(context.Background(), "https://x.test/b", "one", &out); err != nil || out.Echo != "one" {
		t.Fatalf("record: %v %+v", err, out)
	}
	play := New(NewRecorder(dir, Replay, nil), 0, time.Second, 0, nil)
	out.Echo = ""
	if err := play.PostJSON(context.Background(), "https://x.test/b", "one", &out); err != nil || out.Echo != "one" || n.Load() != 1 {
		t.Fatalf("replay: %v %+v live calls %d", err, out, n.Load())
	}
	if err := play.PostJSON(context.Background(), "https://x.test/b", "two", &out); err == nil {
		t.Fatal("a different body must miss the recording")
	}
	// Fill replays what exists without a live call and records only what is missing (D26).
	fill := New(NewRecorder(dir, Fill, live), 0, time.Second, 0, nil)
	if err := fill.PostJSON(context.Background(), "https://x.test/b", "one", &out); err != nil || out.Echo != "one" || n.Load() != 1 {
		t.Fatalf("fill replay: %v %+v live calls %d", err, out, n.Load())
	}
	if err := fill.PostJSON(context.Background(), "https://x.test/b", "two", &out); err != nil || out.Echo != "two" || n.Load() != 2 {
		t.Fatalf("fill miss: %v %+v live calls %d", err, out, n.Load())
	}
	if err := play.PostJSON(context.Background(), "https://x.test/b", "two", &out); err != nil || out.Echo != "two" {
		t.Fatalf("filled recording should replay: %v", err)
	}
	r, _ := http.NewRequest("POST", "https://x.test/b", strings.NewReader(`"one"`))
	resp, err := NewRecorder(dir, Replay, nil).RoundTrip(r)
	if err != nil || resp.Header.Get("Set-Cookie") != "" || resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("saved headers: %v %v", err, resp.Header)
	}
}
