// Package fetch is the only HTTP client: per-host pacing, timeouts, bounded retries and honoured waits (lld.md BE-5).
package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// ErrMalformed marks a response body that is not the JSON the caller expected.
var ErrMalformed = errors.New("malformed response")

// ErrRefused is a venue refusing requests: 429 with its Retry-After, or 403 (a block) with 5 minutes (FR-025).
type ErrRefused struct {
	Status int
	Wait   time.Duration
}

func (e ErrRefused) Error() string {
	return fmt.Sprintf("refused with HTTP %d, wait %s", e.Status, e.Wait)
}

// DefaultRefusedWait applies to a 429 without Retry-After (the US-regulated venue sends none, R01 F4).
// ponytail: fixed 1 minute; make it per-venue if a venue publishes a different cooldown.
const DefaultRefusedWait = time.Minute

// BlockedWait applies to a 403 (lld.md FM-3).
const BlockedWait = 5 * time.Minute

// Client paces, times out and retries requests. Safe for concurrent use.
type Client struct {
	t       http.RoundTripper
	gap     time.Duration // minimum spacing between requests to one host
	timeout time.Duration
	retries int
	now     func() time.Time
	backoff time.Duration // first retry delay; doubles each retry

	mu   sync.Mutex
	next map[string]time.Time
}

// New returns a client sending at most perHost requests per second to each host (0 or less: unpaced), giving each
// attempt timeout, and retrying timeouts, transport errors and 5xx up to retries more times with backoff and jitter.
func New(t http.RoundTripper, perHost float64, timeout time.Duration, retries int, now func() time.Time) *Client {
	if t == nil {
		t = http.DefaultTransport
	}
	if now == nil {
		now = time.Now
	}
	var gap time.Duration
	if perHost > 0 {
		gap = time.Duration(float64(time.Second) / perHost)
	}
	return &Client{t: t, gap: gap, timeout: timeout, retries: retries, now: now, backoff: 500 * time.Millisecond, next: map[string]time.Time{}}
}

// GetJSON fetches url and decodes its JSON body into out.
func (c *Client) GetJSON(ctx context.Context, url string, out any) error {
	return c.do(ctx, http.MethodGet, url, nil, out)
}

// PostJSON posts body as JSON to url and decodes the JSON response into out. Used only for reads, so it is retried.
func (c *Client) PostJSON(ctx context.Context, url string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, url, b, out)
}

func (c *Client) do(ctx context.Context, method, rawURL string, body []byte, out any) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	var last error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			d := c.backoff << (attempt - 1)
			d += time.Duration(rand.Int64N(int64(d)/2 + 1)) // jitter up to +50%
			if err := sleep(ctx, d); err != nil {
				return err
			}
		}
		if err := c.pace(ctx, u.Host); err != nil {
			return err
		}
		retry, err := c.once(ctx, method, rawURL, body, out)
		if err == nil || !retry {
			return err
		}
		last = err
	}
	return last
}

// once makes one attempt; retry says whether the error is worth another.
func (c *Client) once(ctx context.Context, method, rawURL string, body []byte, out any) (retry bool, err error) {
	actx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		actx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(actx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.t.RoundTrip(req)
	if err != nil {
		return ctx.Err() == nil, fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ctx.Err() == nil, fmt.Errorf("%s %s: read body: %w", method, rawURL, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return false, ErrRefused{Status: 429, Wait: retryAfter(resp.Header.Get("Retry-After"), c.now())}
	case resp.StatusCode == http.StatusForbidden:
		return false, ErrRefused{Status: 403, Wait: BlockedWait}
	case resp.StatusCode >= 500:
		return true, fmt.Errorf("%s %s: HTTP %d: %s", method, rawURL, resp.StatusCode, trunc(data))
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return false, fmt.Errorf("%s %s: HTTP %d: %s", method, rawURL, resp.StatusCode, trunc(data))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("%w: %s %s: %v", ErrMalformed, method, rawURL, err)
	}
	return false, nil
}

// pace waits until host may receive another request and books the slot after it.
func (c *Client) pace(ctx context.Context, host string) error {
	if c.gap == 0 {
		return nil
	}
	c.mu.Lock()
	now := c.now()
	at := c.next[host]
	if at.Before(now) {
		at = now
	}
	c.next[host] = at.Add(c.gap)
	c.mu.Unlock()
	return sleep(ctx, at.Sub(now))
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func retryAfter(v string, now time.Time) time.Duration {
	if s, err := strconv.Atoi(v); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil && t.After(now) {
		return t.Sub(now)
	}
	return DefaultRefusedWait
}

// trunc keeps venue error bodies to 500 characters (lld.md §8 error policy).
func trunc(b []byte) string {
	if len(b) > 500 {
		b = b[:500]
	}
	return string(b)
}
