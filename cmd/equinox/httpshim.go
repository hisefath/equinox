package main

import (
	"bytes"
	"io"
	"net/http"
)

// recorder captures a handler's response so the route command can reuse the exact EP-3 code path.
type recorder struct {
	header http.Header
	status int
	body   []byte
}

func (r *recorder) Header() http.Header { return r.header }
func (r *recorder) WriteHeader(s int)   { r.status = s }
func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = 200
	}
	r.body = append(r.body, b...)
	return len(b), nil
}

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
