// Package fetch is the one HTTP path every venue adapter uses: per-request timeouts via context,
// client-side rate limiting, bounded retries with exponential backoff, and record/replay of raw
// responses so tests and demos can run on real captured venue data without a network.
package fetch

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client fetches JSON politely and survives transient failures.
type Client struct {
	HTTP     *http.Client
	Interval time.Duration // minimum gap between requests (client-side rate limit); 0 = none
	Retries  int           // extra attempts after the first, for network errors, 429 and 5xx
	Backoff  time.Duration // first retry delay; doubles each attempt unless Retry-After says otherwise

	mu   sync.Mutex
	next time.Time
}

// New returns a client with sane defaults for public read-only APIs.
func New(rt http.RoundTripper, interval time.Duration) *Client {
	return &Client{HTTP: &http.Client{Transport: rt, Timeout: 20 * time.Second}, Interval: interval, Retries: 3, Backoff: 250 * time.Millisecond}
}

// ErrMalformed marks a response that arrived but could not be decoded. Retrying won't help.
var ErrMalformed = errors.New("malformed JSON")

// StatusError is a non-2xx response that was not worth (or no longer worth) retrying.
type StatusError struct {
	URL  string
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GET %s: HTTP %d: %s", e.URL, e.Code, e.Body)
}

// GetJSON GETs url and decodes the JSON body into v.
func (c *Client) GetJSON(ctx context.Context, url string, v any) error {
	return c.do(ctx, http.MethodGet, url, nil, v)
}

// PostJSON POSTs body as JSON and decodes the response into v (used for batch reads).
func (c *Client) PostJSON(ctx context.Context, url string, body, v any) error {
	return c.do(ctx, http.MethodPost, url, body, v)
}

func (c *Client) do(ctx context.Context, method, url string, body, v any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	delay := c.Backoff
	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx); err != nil {
			return err
		}
		retryAfter, err := c.once(ctx, method, url, payload, v)
		if err == nil || attempt >= c.Retries || !retryable(err) || ctx.Err() != nil {
			return err
		}
		if retryAfter > 0 {
			delay = retryAfter
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %v)", ctx.Err(), err)
		case <-time.After(delay):
		}
		delay *= 2
	}
}

func (c *Client) once(ctx context.Context, method, url string, payload []byte, v any) (time.Duration, error) {
	var rd io.Reader
	if payload != nil {
		rd = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode/100 != 2 {
		ra, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		return time.Duration(ra) * time.Second, &StatusError{URL: url, Code: resp.StatusCode, Body: truncate(string(b), 200)}
	}
	if err := json.Unmarshal(b, v); err != nil {
		return 0, fmt.Errorf("%s %s: %w: %v", method, url, ErrMalformed, err)
	}
	return 0, nil
}

func retryable(err error) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Code == http.StatusTooManyRequests || se.Code >= 500
	}
	return !errors.Is(err, ErrMalformed)
}

// wait enforces Interval between request starts across goroutines sharing the client.
func (c *Client) wait(ctx context.Context) error {
	if c.Interval <= 0 {
		return nil
	}
	c.mu.Lock()
	now := time.Now()
	at := c.next
	if at.Before(now) {
		at = now
	}
	c.next = at.Add(c.Interval)
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(at.Sub(now)):
		return nil
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// Recorder is an http.RoundTripper that saves every response body, gzipped, under Dir (record mode)
// or serves requests only from Dir (replay mode, when Next is nil). A missing recording is a 404, which
// the adapters treat like any other venue failure.
type Recorder struct {
	Dir  string
	Next http.RoundTripper // nil = replay only
}

func (r Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	path := filepath.Join(r.Dir, FileFor(req))
	if r.Next == nil {
		b, err := readGzip(path)
		code := http.StatusOK
		if err != nil {
			b, code = []byte(`{"error":"not recorded"}`), http.StatusNotFound
		}
		return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(b)), Request: req}, nil
	}
	resp, err := r.Next.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return resp, err
	}
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if err := writeGzip(path, b); err != nil {
		return nil, fmt.Errorf("recording %s: %w", path, err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, nil
}

func writeGzip(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	zw.Write(b)
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func readGzip(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(zr)
}

var unsafe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// FileFor names the recording for a request: readable prefix plus a hash of method, URL and body.
func FileFor(req *http.Request) string {
	h := sha256.New()
	h.Write([]byte(req.Method + " " + req.URL.String()))
	if req.GetBody != nil {
		if b, err := req.GetBody(); err == nil {
			_, _ = io.Copy(h, b)
		}
	}
	name := strings.Trim(unsafe.ReplaceAllString(req.URL.Host+req.URL.Path, "_"), "_")
	if len(name) > 80 {
		name = name[:80]
	}
	return name + "_" + hex.EncodeToString(h.Sum(nil))[:12] + ".json.gz"
}
