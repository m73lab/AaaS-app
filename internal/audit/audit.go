package audit

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// Record is a PII-free audit entry. It deliberately contains only metadata
// (categories and counts), never the sensitive value nor the token.
type Record struct {
	TS        time.Time     `json:"ts"`
	Session   string        `json:"session"`
	Tenant    string        `json:"tenant"`
	Action    string        `json:"action"` // "forward" | "blocked" | "error"
	Entities  int           `json:"entities"`
	Categories map[string]int `json:"categories"`
	Error     string        `json:"error,omitempty"`
}

// Auditor appends tamper-evident-free, PII-free audit records.
type Auditor struct {
	mu sync.Mutex
	f  *os.File
}

// New creates an auditor. When path is empty, records go to stderr.
func New(path string) (*Auditor, error) {
	if path == "" {
		return &Auditor{f: os.Stderr}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &Auditor{f: f}, nil
}

// Log writes a record as a single JSON line.
func (a *Auditor) Log(r Record) {
	if r.Categories == nil {
		r.Categories = map[string]int{}
	}
	if r.TS.IsZero() {
		r.TS = time.Now().UTC()
	}
	b, err := json.Marshal(r)
	if err != nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, err := a.f.Write(append(b, '\n')); err != nil {
		log.Printf("audit: write failed: %v", err)
	}
}

// Close releases the backing file.
func (a *Auditor) Close() error {
	return a.f.Close()
}
