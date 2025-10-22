package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ericphanson/madrid-events/internal/fetch"
)

// getFixturePath returns the absolute file:// URL for a fixture file
func getFixturePath(filename string) string {
	// Try multiple possible paths since go test working directory can vary
	possiblePaths := []string{
		filepath.Join("testdata", "fixtures", filename),             // From project root
		filepath.Join("..", "..", "testdata", "fixtures", filename), // From package dir
	}

	for _, relPath := range possiblePaths {
		absPath, err := filepath.Abs(relPath)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return "file://" + absPath
		}
	}

	// Fallback: return first path even if it doesn't exist (will fail in test with clear error)
	absPath, _ := filepath.Abs(possiblePaths[0])
	return "file://" + absPath
}

func TestPipeline_FetchAll_Sequential(t *testing.T) {
	// This test uses real fixtures to verify sequential fetching works
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		getFixturePath("madrid-events.json"),
		getFixturePath("madrid-events.xml"),
		getFixturePath("madrid-events.csv"),
		client,
		loc,
	)

	result := pipeline.FetchAll()

	// All three sources should have events
	if len(result.JSONEvents) == 0 {
		t.Error("expected JSON events, got 0")
	}
	if len(result.XMLEvents) == 0 {
		t.Error("expected XML events, got 0")
	}
	if len(result.CSVEvents) == 0 {
		t.Error("expected CSV events, got 0")
	}

	t.Logf("Fetched JSON: %d events, %d errors", len(result.JSONEvents), len(result.JSONErrors))
	t.Logf("Fetched XML: %d events, %d errors", len(result.XMLEvents), len(result.XMLErrors))
	t.Logf("Fetched CSV: %d events, %d errors", len(result.CSVEvents), len(result.CSVErrors))
}

func TestPipeline_FetchAll_ErrorIsolation(t *testing.T) {
	// This test verifies that JSON failure doesn't prevent CSV/XML from working
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		"file:///nonexistent/json.json", // Will fail
		getFixturePath("madrid-events.xml"),
		getFixturePath("madrid-events.csv"),
		client,
		loc,
	)

	result := pipeline.FetchAll()

	// JSON should fail
	if len(result.JSONEvents) > 0 {
		t.Error("expected JSON to fail, got events")
	}
	if len(result.JSONErrors) == 0 {
		t.Error("expected JSON errors, got none")
	}

	// But XML and CSV should still succeed
	if len(result.XMLEvents) == 0 {
		t.Error("expected XML events despite JSON failure")
	}
	if len(result.CSVEvents) == 0 {
		t.Error("expected CSV events despite JSON failure")
	}

	t.Logf("JSON failed as expected: %d errors", len(result.JSONErrors))
	t.Logf("XML succeeded: %d events", len(result.XMLEvents))
	t.Logf("CSV succeeded: %d events", len(result.CSVEvents))
}

func TestPipeline_Merge_Deduplication(t *testing.T) {
	// This test uses real fixtures to verify deduplication works
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		getFixturePath("madrid-events.json"),
		getFixturePath("madrid-events.xml"),
		getFixturePath("madrid-events.csv"),
		client,
		loc,
	)

	result := pipeline.FetchAll()
	merged := pipeline.Merge(result)

	// Merged count should be <= sum of individual counts (deduplication)
	totalBeforeMerge := len(result.JSONEvents) + len(result.XMLEvents) + len(result.CSVEvents)
	if len(merged) > totalBeforeMerge {
		t.Errorf("merged count (%d) should not exceed sum (%d)", len(merged), totalBeforeMerge)
	}

	// Should have significant deduplication (same events in all 3 sources)
	if len(merged) >= totalBeforeMerge {
		t.Errorf("expected deduplication: %d merged vs %d total", len(merged), totalBeforeMerge)
	}

	t.Logf("Before merge: %d total events (JSON:%d + XML:%d + CSV:%d)",
		totalBeforeMerge, len(result.JSONEvents), len(result.XMLEvents), len(result.CSVEvents))
	t.Logf("After merge: %d unique events", len(merged))
	t.Logf("Deduplication: removed %d duplicates", totalBeforeMerge-len(merged))
}

func TestPipeline_Merge_SourceTracking(t *testing.T) {
	// This test verifies that events found in multiple sources track all sources
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		getFixturePath("madrid-events.json"),
		getFixturePath("madrid-events.xml"),
		getFixturePath("madrid-events.csv"),
		client,
		loc,
	)

	result := pipeline.FetchAll()
	merged := pipeline.Merge(result)

	// Check that some events have multiple sources
	foundMultiSource := false
	sourceCounts := make(map[int]int) // Count events by number of sources

	for _, evt := range merged {
		numSources := len(evt.Sources)
		sourceCounts[numSources]++

		if numSources > 1 {
			foundMultiSource = true
			t.Logf("Event %s found in %d sources: %v", evt.ID, numSources, evt.Sources)
		}

		// Validate source values
		for _, src := range evt.Sources {
			if src != "JSON" && src != "XML" && src != "CSV" {
				t.Errorf("Event %s has invalid source: %q", evt.ID, src)
			}
		}
	}

	if !foundMultiSource {
		t.Error("expected some events to be found in multiple sources")
	}

	// Log source distribution
	t.Logf("Source distribution:")
	for numSources, count := range sourceCounts {
		t.Logf("  %d source(s): %d events", numSources, count)
	}
}

func TestPipeline_Merge_HandlesFailures(t *testing.T) {
	// This test verifies that merge works even when some sources fail
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		"file:///nonexistent/json.json",     // Will fail
		getFixturePath("madrid-events.xml"), // Will succeed
		"file:///nonexistent/csv.csv",       // Will fail
		client,
		loc,
	)

	result := pipeline.FetchAll()
	merged := pipeline.Merge(result)

	// Should still get merged events from successful source (XML)
	if len(merged) == 0 {
		t.Error("expected merged events from successful source (XML)")
	}

	// All events should be from XML only
	for _, evt := range merged {
		if len(evt.Sources) != 1 || evt.Sources[0] != "XML" {
			t.Errorf("Event %s should only have XML source, got: %v", evt.ID, evt.Sources)
		}
	}

	t.Logf("Merged %d events from XML despite JSON and CSV failures", len(merged))
}

func TestPipeline_Merge_EmptyResult(t *testing.T) {
	// This test verifies that merge handles empty results gracefully
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		"file:///nonexistent/json.json",
		"file:///nonexistent/xml.xml",
		"file:///nonexistent/csv.csv",
		client,
		loc,
	)

	result := pipeline.FetchAll()
	merged := pipeline.Merge(result)

	// Should return empty slice, not nil
	if merged == nil {
		t.Error("expected empty slice, got nil")
	}

	if len(merged) != 0 {
		t.Errorf("expected 0 events, got %d", len(merged))
	}

	t.Log("Merge handled all-failures case correctly (empty result)")
}

func TestPipeline_Merge_DeduplicatesSources(t *testing.T) {
	// This test specifically verifies that duplicate sources are removed
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := fetch.DefaultDevelopmentConfig()
	client, err := fetch.NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	pipeline := NewPipeline(
		getFixturePath("madrid-events.json"),
		getFixturePath("madrid-events.xml"),
		getFixturePath("madrid-events.csv"),
		client,
		loc,
	)

	result := pipeline.FetchAll()
	merged := pipeline.Merge(result)

	// Verify no event has duplicate sources
	for _, evt := range merged {
		// Check for duplicates in Sources
		seen := make(map[string]bool)
		for _, src := range evt.Sources {
			if seen[src] {
				t.Errorf("Event %s has duplicate source %q: %v", evt.ID, src, evt.Sources)
			}
			seen[src] = true
		}

		// Verify Sources slice length matches unique count
		if len(evt.Sources) != len(seen) {
			t.Errorf("Event %s has %d sources but only %d unique: %v",
				evt.ID, len(evt.Sources), len(seen), evt.Sources)
		}
	}

	t.Logf("Verified %d events have no duplicate sources", len(merged))
}
