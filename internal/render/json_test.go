package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONRenderer_Render(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "events.json")

	renderer := NewJSONRenderer()

	events := []JSONEvent{
		{
			ID:         "JSON-001",
			Title:      "JSON Event",
			StartTime:  time.Date(2025, 11, 15, 19, 30, 0, 0, time.UTC).Format(time.RFC3339),
			VenueName:  "Test Venue",
			DetailsURL: "https://example.com/event",
		},
	}

	err := renderer.Render(events, outputPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify output file exists and is valid JSON
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	var loaded []JSONEvent
	if err := json.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(loaded))
	}

	if loaded[0].ID != "JSON-001" {
		t.Errorf("Expected ID 'JSON-001', got '%s'", loaded[0].ID)
	}
}
