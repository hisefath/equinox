package fetch

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// RecordMode picks what a recorder does with a request.
type RecordMode int

const (
	Replay RecordMode = iota // answer from dir; a missing recording is an error, nothing leaves the machine
	Record                   // pass to next, save the response to dir, return it
	Fill                     // replay when a recording exists; otherwise pass to next and save it (adds to a recording)
)

// recording is one saved exchange; files are gzip JSON named by Key (D17).
type recording struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Status int         `json:"status"`
	Header http.Header `json:"header"`
	Body   string      `json:"body"`
}

// Key names a recording: hex SHA-256 of method, URL and body, so a POST with another body is another file.
func Key(method, url string, body []byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s %s\n", method, url)
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

type recorder struct {
	dir  string
	mode RecordMode
	next http.RoundTripper
}

// NewRecorder returns a transport that records responses from next into dir, or replays them from dir (SEAM-2).
func NewRecorder(dir string, mode RecordMode, next http.RoundTripper) http.RoundTripper {
	return &recorder{dir: dir, mode: mode, next: next}
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
		body = b
		req.Body = io.NopCloser(bytes.NewReader(b))
	}
	path := filepath.Join(r.dir, Key(req.Method, req.URL.String(), body)+".json.gz")
	if r.mode == Fill {
		if rec, err := load(path); err == nil {
			return rec.response(req), nil
		}
	}
	if r.mode == Replay {
		rec, err := load(path)
		if err != nil {
			return nil, fmt.Errorf("no recording for %s %s: %w", req.Method, req.URL, err)
		}
		return rec.response(req), nil
	}
	resp, err := r.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	rec := recording{Method: req.Method, URL: req.URL.String(), Status: resp.StatusCode, Header: http.Header{}, Body: string(data)}
	for _, k := range []string{"Content-Type", "Retry-After"} { // nothing else: no cookies or tracing ids in fixtures
		if v := resp.Header.Get(k); v != "" {
			rec.Header.Set(k, v)
		}
	}
	if err := save(path, rec); err != nil {
		return nil, err
	}
	return rec.response(req), nil
}

func (rec recording) response(req *http.Request) *http.Response {
	return &http.Response{
		Status: strconv.Itoa(rec.Status) + " " + http.StatusText(rec.Status), StatusCode: rec.Status,
		Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: rec.Header.Clone(),
		Body: io.NopCloser(strings.NewReader(rec.Body)), ContentLength: int64(len(rec.Body)), Request: req,
	}
}

func load(path string) (recording, error) {
	f, err := os.Open(path)
	if err != nil {
		return recording{}, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return recording{}, err
	}
	var rec recording
	err = json.NewDecoder(zr).Decode(&rec)
	return rec, err
}

func save(path string, rec recording) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(zw)
	enc.SetIndent("", " ")
	if err := enc.Encode(rec); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// Curl is a transport that runs the system curl. It exists for recording fixtures on a host whose firewall blocks
// newly built binaries from the internet (AGENTS.md); curl verifies TLS as usual. Not for production use.
type Curl struct{}

func (Curl) RoundTrip(req *http.Request) (*http.Response, error) {
	args := []string{"-sS", "--http1.1", "--max-time", "60", "-X", req.Method, "-D", "-", req.URL.String()}
	var stdin io.Reader
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		stdin = bytes.NewReader(b)
		args = append(args, "--data-binary", "@-")
	}
	for k := range req.Header {
		args = append(args, "-H", k+": "+req.Header.Get(k))
	}
	cmd := exec.CommandContext(req.Context(), "curl", args...)
	cmd.Stdin = stdin
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("curl %s: %w", req.URL, err)
	}
	// Skip interim header blocks (100 Continue) and keep the last one before the body.
	for {
		head, rest, ok := bytes.Cut(out, []byte("\r\n\r\n"))
		if !ok {
			return nil, fmt.Errorf("curl %s: no header block", req.URL)
		}
		resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(append(head, "\r\n\r\n"...))), req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 200 || !bytes.HasPrefix(rest, []byte("HTTP/")) {
			resp.Body = io.NopCloser(bytes.NewReader(rest))
			resp.ContentLength = int64(len(rest))
			resp.Header.Del("Content-Encoding")
			return resp, nil
		}
		out = rest
	}
}
