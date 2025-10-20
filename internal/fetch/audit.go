package fetch

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// RequestRecord captures details of an HTTP request.
type RequestRecord struct {
	URL         string    `json:"url"`
	Timestamp   time.Time `json:"timestamp"`
	CacheHit    bool      `json:"cache_hit"`
	StatusCode  int       `json:"status_code,omitempty"`
	DelayMs     int64     `json:"delay_ms"`
	RateLimited bool      `json:"rate_limited"`
	Error       string    `json:"error,omitempty"`
}

// RequestAuditor collects request records.
type RequestAuditor struct {
	records []RequestRecord
	mu      sync.Mutex
}

// NewRequestAuditor creates an empty auditor.
func NewRequestAuditor() *RequestAuditor {
	return &RequestAuditor{
		records: make([]RequestRecord, 0),
	}
}

// Record adds a request to the audit trail.
func (a *RequestAuditor) Record(r RequestRecord) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = append(a.records, r)
}

// Export writes the audit trail to a JSON file.
func (a *RequestAuditor) Export(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := json.MarshalIndent(a.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling audit: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing audit file: %w", err)
	}

	return nil
}

// Records returns a copy of all records.
func (a *RequestAuditor) Records() []RequestRecord {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make([]RequestRecord, len(a.records))
	copy(result, a.records)
	return result
}
