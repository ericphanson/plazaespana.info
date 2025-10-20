package event

import (
	"errors"
	"testing"
	"time"
)

// TestCanonicalEvent_Creation verifies that CanonicalEvent can be created
// with all fields populated correctly.
func TestCanonicalEvent_Creation(t *testing.T) {
	madrid := time.FixedZone("Europe/Madrid", 3600) // CET (UTC+1)
	startTime := time.Date(2025, 10, 20, 19, 0, 0, 0, madrid)
	endTime := time.Date(2025, 10, 20, 22, 0, 0, 0, madrid)

	event := CanonicalEvent{
		ID:          "event-123",
		Title:       "Test Event",
		Description: "A test event description",
		StartTime:   startTime,
		EndTime:     endTime,
		Latitude:    40.42338,
		Longitude:   -3.71217,
		VenueName:   "Plaza de España",
		Address:     "Plaza de España, Madrid",
		DetailsURL:  "https://example.com/event-123",
		Sources:     []string{"JSON"},
	}

	// Verify all fields were set correctly
	if event.ID != "event-123" {
		t.Errorf("Expected ID 'event-123', got '%s'", event.ID)
	}
	if event.Title != "Test Event" {
		t.Errorf("Expected Title 'Test Event', got '%s'", event.Title)
	}
	if event.Description != "A test event description" {
		t.Errorf("Expected Description 'A test event description', got '%s'", event.Description)
	}
	if !event.StartTime.Equal(startTime) {
		t.Errorf("Expected StartTime %v, got %v", startTime, event.StartTime)
	}
	if !event.EndTime.Equal(endTime) {
		t.Errorf("Expected EndTime %v, got %v", endTime, event.EndTime)
	}
	if event.Latitude != 40.42338 {
		t.Errorf("Expected Latitude 40.42338, got %f", event.Latitude)
	}
	if event.Longitude != -3.71217 {
		t.Errorf("Expected Longitude -3.71217, got %f", event.Longitude)
	}
	if event.VenueName != "Plaza de España" {
		t.Errorf("Expected VenueName 'Plaza de España', got '%s'", event.VenueName)
	}
	if event.Address != "Plaza de España, Madrid" {
		t.Errorf("Expected Address 'Plaza de España, Madrid', got '%s'", event.Address)
	}
	if event.DetailsURL != "https://example.com/event-123" {
		t.Errorf("Expected DetailsURL 'https://example.com/event-123', got '%s'", event.DetailsURL)
	}
	if len(event.Sources) != 1 || event.Sources[0] != "JSON" {
		t.Errorf("Expected Sources ['JSON'], got %v", event.Sources)
	}
}

// TestSourcedEvent_Tracking verifies that SourcedEvent correctly tracks
// which source an event came from.
func TestSourcedEvent_Tracking(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		eventID        string
		expectedSource string
	}{
		{
			name:           "JSON source",
			source:         "JSON",
			eventID:        "json-event-1",
			expectedSource: "JSON",
		},
		{
			name:           "XML source",
			source:         "XML",
			eventID:        "xml-event-1",
			expectedSource: "XML",
		},
		{
			name:           "CSV source",
			source:         "CSV",
			eventID:        "csv-event-1",
			expectedSource: "CSV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical := CanonicalEvent{
				ID:      tt.eventID,
				Title:   "Test Event",
				Sources: []string{tt.source},
			}

			sourced := SourcedEvent{
				Event:  canonical,
				Source: tt.source,
			}

			// Verify source is tracked correctly
			if sourced.Source != tt.expectedSource {
				t.Errorf("Expected source '%s', got '%s'", tt.expectedSource, sourced.Source)
			}

			// Verify event data is preserved
			if sourced.Event.ID != tt.eventID {
				t.Errorf("Expected event ID '%s', got '%s'", tt.eventID, sourced.Event.ID)
			}

			// Verify sources array matches source field
			if len(sourced.Event.Sources) != 1 || sourced.Event.Sources[0] != tt.source {
				t.Errorf("Expected Sources ['%s'], got %v", tt.source, sourced.Event.Sources)
			}
		})
	}
}

// TestParseResult_Creation verifies that ParseResult can track both
// successful events and parsing errors.
func TestParseResult_Creation(t *testing.T) {
	// Create a successful event
	successEvent := SourcedEvent{
		Event: CanonicalEvent{
			ID:      "success-1",
			Title:   "Successful Event",
			Sources: []string{"JSON"},
		},
		Source: "JSON",
	}

	// Create a parse error
	parseErr := ParseError{
		Source:      "JSON",
		Index:       5,
		RawData:     "ID=bad-event Title=Malformed",
		Error:       errors.New("invalid date format"),
		RecoverType: "skipped",
	}

	// Create ParseResult with both
	result := ParseResult{
		Events: []SourcedEvent{successEvent},
		Errors: []ParseError{parseErr},
	}

	// Verify successful event tracking
	if len(result.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Event.ID != "success-1" {
		t.Errorf("Expected event ID 'success-1', got '%s'", result.Events[0].Event.ID)
	}

	// Verify error tracking
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Source != "JSON" {
		t.Errorf("Expected error source 'JSON', got '%s'", result.Errors[0].Source)
	}
	if result.Errors[0].Index != 5 {
		t.Errorf("Expected error index 5, got %d", result.Errors[0].Index)
	}
	if result.Errors[0].RecoverType != "skipped" {
		t.Errorf("Expected recover type 'skipped', got '%s'", result.Errors[0].RecoverType)
	}
}

// TestParseError_Fields verifies that ParseError captures all necessary
// context for debugging failed parses.
func TestParseError_Fields(t *testing.T) {
	err := ParseError{
		Source:      "XML",
		Index:       42,
		RawData:     "ID-EVENTO=12345 TITULO=Bad Date",
		Error:       errors.New("time parsing failed"),
		RecoverType: "partial",
	}

	if err.Source != "XML" {
		t.Errorf("Expected Source 'XML', got '%s'", err.Source)
	}
	if err.Index != 42 {
		t.Errorf("Expected Index 42, got %d", err.Index)
	}
	if err.RawData != "ID-EVENTO=12345 TITULO=Bad Date" {
		t.Errorf("Expected specific RawData, got '%s'", err.RawData)
	}
	if err.Error == nil {
		t.Error("Expected Error to be set")
	}
	if err.Error.Error() != "time parsing failed" {
		t.Errorf("Expected error message 'time parsing failed', got '%s'", err.Error.Error())
	}
	if err.RecoverType != "partial" {
		t.Errorf("Expected RecoverType 'partial', got '%s'", err.RecoverType)
	}
}
