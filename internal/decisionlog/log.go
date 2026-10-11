// Package decisionlog appends each decided or refused order as one JSON line, synced to disk before the answer is
// returned (HLD INV-6), and reads pages back newest first (lld.md BE-9).
package decisionlog

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/hisefath/equinox/internal/route"
)

// Log is an append-only file of decisions.
type Log struct {
	mu   sync.Mutex
	f    *os.File
	path string
}

// Open opens (or creates) the log file for appending.
func Open(path string) (*Log, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Log{f: f, path: path}, nil
}

// Append writes one decision as a whole line and syncs it; only then may the caller answer (INV-6).
func (l *Log) Append(d route.Decision) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.f.Write(append(b, '\n')); err != nil {
		return err
	}
	return l.f.Sync()
}

// Close closes the file.
func (l *Log) Close() error { return l.f.Close() }

// Page returns up to limit decisions, newest first, starting after cursor, and the next cursor ("" when done).
func (l *Log) Page(cursor string, limit int) ([]route.Decision, string, error) {
	skip := 0
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("bad cursor")
		}
		if skip, err = strconv.Atoi(string(raw)); err != nil || skip < 0 {
			return nil, "", fmt.Errorf("bad cursor")
		}
	}
	l.mu.Lock()
	f, err := os.Open(l.path)
	l.mu.Unlock()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	var lines [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		lines = append(lines, append([]byte(nil), sc.Bytes()...))
	}
	if err := sc.Err(); err != nil {
		return nil, "", err
	}
	var out []route.Decision
	i := len(lines) - 1 - skip
	for ; i >= 0 && len(out) < limit; i-- {
		var d route.Decision
		if err := json.Unmarshal(lines[i], &d); err != nil {
			return nil, "", fmt.Errorf("log line %d: %w", i+1, err)
		}
		out = append(out, d)
	}
	next := ""
	if i >= 0 {
		next = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(skip + len(out))))
	}
	return out, next, nil
}
