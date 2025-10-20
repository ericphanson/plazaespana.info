package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
)

func TestSaveAuditJSON(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "audit-events.json")

	// Create test events
	buildTime := time.Date(2025, 10, 20, 14, 0, 0, 0, time.UTC)
	duration := 2500 * time.Millisecond

	culturalEvents := []event.CulturalEvent{
		{
			ID:        "event1",
			Title:     "Test Event 1",
			StartTime: buildTime,
			Distrito:  "CENTRO",
			FilterResult: event.FilterResult{
				Kept:         true,
				FilterReason: "kept",
			},
		},
		{
			ID:        "event2",
			Title:     "Test Event 2",
			StartTime: buildTime,
			Distrito:  "VICALVARO",
			FilterResult: event.FilterResult{
				Kept:         false,
				FilterReason: "outside target distrito",
			},
		},
	}

	cityEvents := []event.CityEvent{
		{
			ID:        "city1",
			Title:     "City Event 1",
			StartDate: buildTime,
			FilterResult: event.FilterResult{
				Kept:         true,
				FilterReason: "kept",
			},
		},
	}

	// Save audit JSON
	err := SaveAuditJSON(culturalEvents, cityEvents, auditPath, buildTime, duration)
	if err != nil {
		t.Fatalf("SaveAuditJSON failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		t.Fatal("Audit file was not created")
	}

	// Read and parse JSON
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	var audit AuditFile
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatalf("Failed to parse audit JSON: %v", err)
	}

	// Verify structure
	if audit.BuildDuration != 2.5 {
		t.Errorf("BuildDuration = %v, want 2.5", audit.BuildDuration)
	}

	if audit.TotalEvents != 3 {
		t.Errorf("TotalEvents = %d, want 3", audit.TotalEvents)
	}

	// Verify cultural events pipeline
	if audit.CulturalEvents.Total != 2 {
		t.Errorf("CulturalEvents.Total = %d, want 2", audit.CulturalEvents.Total)
	}
	if audit.CulturalEvents.Kept != 1 {
		t.Errorf("CulturalEvents.Kept = %d, want 1", audit.CulturalEvents.Kept)
	}
	if audit.CulturalEvents.Filtered != 1 {
		t.Errorf("CulturalEvents.Filtered = %d, want 1", audit.CulturalEvents.Filtered)
	}

	// Verify filter breakdown
	if audit.CulturalEvents.FilterBreakdown["kept"] != 1 {
		t.Errorf("FilterBreakdown[kept] = %d, want 1", audit.CulturalEvents.FilterBreakdown["kept"])
	}
	if audit.CulturalEvents.FilterBreakdown["outside target distrito"] != 1 {
		t.Errorf("FilterBreakdown[outside target distrito] = %d, want 1", audit.CulturalEvents.FilterBreakdown["outside target distrito"])
	}

	// Verify city events pipeline
	if audit.CityEvents.Total != 1 {
		t.Errorf("CityEvents.Total = %d, want 1", audit.CityEvents.Total)
	}
	if audit.CityEvents.Kept != 1 {
		t.Errorf("CityEvents.Kept = %d, want 1", audit.CityEvents.Kept)
	}
}

func TestProcessCulturalEvents(t *testing.T) {
	events := []event.CulturalEvent{
		{
			ID: "1",
			FilterResult: event.FilterResult{
				Kept:         true,
				FilterReason: "kept",
			},
		},
		{
			ID: "2",
			FilterResult: event.FilterResult{
				Kept:         false,
				FilterReason: "outside target distrito",
			},
		},
		{
			ID: "3",
			FilterResult: event.FilterResult{
				Kept:         false,
				FilterReason: "outside target distrito",
			},
		},
		{
			ID: "4",
			FilterResult: event.FilterResult{
				Kept:         false,
				FilterReason: "event too old",
			},
		},
	}

	pipeline := processCulturalEvents(events)

	if pipeline.Total != 4 {
		t.Errorf("Total = %d, want 4", pipeline.Total)
	}
	if pipeline.Kept != 1 {
		t.Errorf("Kept = %d, want 1", pipeline.Kept)
	}
	if pipeline.Filtered != 3 {
		t.Errorf("Filtered = %d, want 3", pipeline.Filtered)
	}

	// Check filter breakdown
	if pipeline.FilterBreakdown["kept"] != 1 {
		t.Errorf("FilterBreakdown[kept] = %d, want 1", pipeline.FilterBreakdown["kept"])
	}
	if pipeline.FilterBreakdown["outside target distrito"] != 2 {
		t.Errorf("FilterBreakdown[outside target distrito] = %d, want 2", pipeline.FilterBreakdown["outside target distrito"])
	}
	if pipeline.FilterBreakdown["event too old"] != 1 {
		t.Errorf("FilterBreakdown[event too old] = %d, want 1", pipeline.FilterBreakdown["event too old"])
	}
}

func TestProcessCityEvents(t *testing.T) {
	events := []event.CityEvent{
		{
			ID: "1",
			FilterResult: event.FilterResult{
				Kept:         true,
				FilterReason: "kept",
			},
		},
		{
			ID: "2",
			FilterResult: event.FilterResult{
				Kept:         false,
				FilterReason: "outside GPS radius",
			},
		},
	}

	pipeline := processCityEvents(events)

	if pipeline.Total != 2 {
		t.Errorf("Total = %d, want 2", pipeline.Total)
	}
	if pipeline.Kept != 1 {
		t.Errorf("Kept = %d, want 1", pipeline.Kept)
	}
	if pipeline.Filtered != 1 {
		t.Errorf("Filtered = %d, want 1", pipeline.Filtered)
	}

	// Check filter breakdown
	if pipeline.FilterBreakdown["kept"] != 1 {
		t.Errorf("FilterBreakdown[kept] = %d, want 1", pipeline.FilterBreakdown["kept"])
	}
	if pipeline.FilterBreakdown["outside GPS radius"] != 1 {
		t.Errorf("FilterBreakdown[outside GPS radius] = %d, want 1", pipeline.FilterBreakdown["outside GPS radius"])
	}
}
