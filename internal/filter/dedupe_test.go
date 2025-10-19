package filter

import (
	"testing"

	"github.com/yourusername/madrid-events/internal/fetch"
)

func TestDeduplicateByID(t *testing.T) {
	events := []fetch.RawEvent{
		{IDEvento: "EVT-001", Titulo: "First"},
		{IDEvento: "EVT-002", Titulo: "Second"},
		{IDEvento: "EVT-001", Titulo: "Duplicate First"},
		{IDEvento: "EVT-003", Titulo: "Third"},
		{IDEvento: "EVT-002", Titulo: "Duplicate Second"},
	}

	result := DeduplicateByID(events)

	if len(result) != 3 {
		t.Fatalf("Expected 3 unique events, got %d", len(result))
	}

	seen := make(map[string]bool)
	for _, event := range result {
		if seen[event.IDEvento] {
			t.Errorf("Duplicate ID in result: %s", event.IDEvento)
		}
		seen[event.IDEvento] = true
	}

	// Verify we kept the first occurrence
	if result[0].Titulo != "First" {
		t.Errorf("Expected 'First', got '%s'", result[0].Titulo)
	}
}

func TestDeduplicateByID_Empty(t *testing.T) {
	result := DeduplicateByID([]fetch.RawEvent{})
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d events", len(result))
	}
}
