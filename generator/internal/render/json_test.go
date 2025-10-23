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

	// Create sample cultural and city events
	culturalEvents := []JSONEvent{
		{
			ID:         "CULTURAL-001",
			Title:      "Cultural Event 1",
			StartTime:  time.Date(2025, 11, 15, 19, 30, 0, 0, time.UTC).Format(time.RFC3339),
			VenueName:  "Cultural Venue",
			DetailsURL: "https://example.com/cultural",
		},
		{
			ID:         "CULTURAL-002",
			Title:      "Cultural Event 2",
			StartTime:  time.Date(2025, 11, 16, 20, 0, 0, 0, time.UTC).Format(time.RFC3339),
			VenueName:  "Another Venue",
			DetailsURL: "https://example.com/cultural2",
		},
	}

	cityEvents := []JSONEvent{
		{
			ID:         "CITY-001",
			Title:      "City Event 1",
			StartTime:  time.Date(2025, 11, 17, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
			VenueName:  "City Venue",
			DetailsURL: "https://example.com/city",
		},
	}

	updateTime := time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC)

	err := renderer.Render(culturalEvents, cityEvents, updateTime, outputPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify output file exists and is valid JSON
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify structure
	if len(output.CulturalEvents) != 2 {
		t.Errorf("Expected 2 cultural events, got %d", len(output.CulturalEvents))
	}

	if len(output.CityEvents) != 1 {
		t.Errorf("Expected 1 city event, got %d", len(output.CityEvents))
	}

	// Verify meta information
	if output.Meta.TotalCultural != 2 {
		t.Errorf("Expected TotalCultural=2, got %d", output.Meta.TotalCultural)
	}

	if output.Meta.TotalCity != 1 {
		t.Errorf("Expected TotalCity=1, got %d", output.Meta.TotalCity)
	}

	if output.Meta.UpdateTime == "" {
		t.Error("Expected UpdateTime to be set")
	}

	// Verify event content
	if output.CulturalEvents[0].ID != "CULTURAL-001" {
		t.Errorf("Expected first cultural event ID 'CULTURAL-001', got '%s'", output.CulturalEvents[0].ID)
	}

	if output.CityEvents[0].ID != "CITY-001" {
		t.Errorf("Expected first city event ID 'CITY-001', got '%s'", output.CityEvents[0].ID)
	}
}

func TestJSONRenderer_RenderEmptyEvents(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "events.json")

	renderer := NewJSONRenderer()

	// Test with empty arrays
	culturalEvents := []JSONEvent{}
	cityEvents := []JSONEvent{}
	updateTime := time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC)

	err := renderer.Render(culturalEvents, cityEvents, updateTime, outputPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify output
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify empty arrays (not null)
	if output.CulturalEvents == nil {
		t.Error("CulturalEvents should be empty array, not null")
	}

	if output.CityEvents == nil {
		t.Error("CityEvents should be empty array, not null")
	}

	if output.Meta.TotalCultural != 0 {
		t.Errorf("Expected TotalCultural=0, got %d", output.Meta.TotalCultural)
	}

	if output.Meta.TotalCity != 0 {
		t.Errorf("Expected TotalCity=0, got %d", output.Meta.TotalCity)
	}
}
