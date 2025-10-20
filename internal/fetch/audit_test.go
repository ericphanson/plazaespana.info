package fetch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRequestAuditor_Record(t *testing.T) {
	auditor := NewRequestAuditor()

	// Record several requests
	records := []RequestRecord{
		{
			URL:        "https://example.com/api",
			Timestamp:  time.Now(),
			CacheHit:   false,
			StatusCode: 200,
			DelayMs:    0,
		},
		{
			URL:        "https://example.com/api",
			Timestamp:  time.Now(),
			CacheHit:   true,
			StatusCode: 200,
			DelayMs:    0,
		},
		{
			URL:         "https://other.com/data",
			Timestamp:   time.Now(),
			CacheHit:    false,
			StatusCode:  429,
			DelayMs:     2000,
			RateLimited: true,
		},
	}

	for _, r := range records {
		auditor.Record(r)
	}

	// Verify records
	got := auditor.Records()
	if len(got) != 3 {
		t.Fatalf("Records count = %d, want 3", len(got))
	}

	// Check first record
	if got[0].URL != "https://example.com/api" {
		t.Errorf("Record[0].URL = %q", got[0].URL)
	}
	if got[0].CacheHit {
		t.Errorf("Record[0].CacheHit = true, want false")
	}

	// Check second record (cache hit)
	if !got[1].CacheHit {
		t.Errorf("Record[1].CacheHit = false, want true")
	}

	// Check third record (rate limited)
	if !got[2].RateLimited {
		t.Errorf("Record[2].RateLimited = false, want true")
	}
	if got[2].StatusCode != 429 {
		t.Errorf("Record[2].StatusCode = %d, want 429", got[2].StatusCode)
	}
}

func TestRequestAuditor_Export(t *testing.T) {
	auditor := NewRequestAuditor()

	// Add test records
	timestamp := time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC)
	auditor.Record(RequestRecord{
		URL:        "https://example.com/api",
		Timestamp:  timestamp,
		CacheHit:   false,
		StatusCode: 200,
		DelayMs:    0,
	})
	auditor.Record(RequestRecord{
		URL:        "https://example.com/api",
		Timestamp:  timestamp.Add(5 * time.Second),
		CacheHit:   true,
		StatusCode: 200,
		DelayMs:    0,
	})

	// Export to file
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "audit.json")

	if err := auditor.Export(auditPath); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read and verify file
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var records []RequestRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Exported records count = %d, want 2", len(records))
	}

	// Verify first record
	if records[0].URL != "https://example.com/api" {
		t.Errorf("Record[0].URL = %q", records[0].URL)
	}
	if records[0].CacheHit {
		t.Errorf("Record[0].CacheHit = true, want false")
	}

	// Verify second record (cache hit)
	if !records[1].CacheHit {
		t.Errorf("Record[1].CacheHit = false, want true")
	}
}

func TestRequestAuditor_ConcurrentAccess(t *testing.T) {
	auditor := NewRequestAuditor()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			auditor.Record(RequestRecord{
				URL:       "https://example.com/api",
				Timestamp: time.Now(),
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all records were added
	records := auditor.Records()
	if len(records) != 10 {
		t.Errorf("Records count = %d, want 10", len(records))
	}
}

func TestRequestAuditor_ErrorRecord(t *testing.T) {
	auditor := NewRequestAuditor()

	auditor.Record(RequestRecord{
		URL:       "https://example.com/api",
		Timestamp: time.Now(),
		Error:     "connection timeout",
	})

	records := auditor.Records()
	if len(records) != 1 {
		t.Fatalf("Records count = %d, want 1", len(records))
	}

	if records[0].Error != "connection timeout" {
		t.Errorf("Record[0].Error = %q, want 'connection timeout'", records[0].Error)
	}
}
