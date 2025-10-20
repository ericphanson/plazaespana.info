package fetch

import (
	"os"
	"testing"
	"time"
)

func TestFetchCSV_FieldMapping(t *testing.T) {
	// Use file:// URL to load fixture
	fixtureURL := "file:///workspace/testdata/fixtures/madrid-events.csv"

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchCSV(fixtureURL, loc)

	// Should have at least some events from the fixture
	if len(result.Events) == 0 {
		if len(result.Errors) > 0 {
			t.Fatalf("FetchCSV failed: %v", result.Errors[0].Error)
		}
		t.Fatal("Expected at least one event from fixture")
	}

	// Validate first event has proper field mapping
	firstEvent := result.Events[0].Event
	if firstEvent.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if firstEvent.Title == "" {
		t.Error("Expected non-empty Title")
	}
	if firstEvent.StartTime.IsZero() {
		t.Error("Expected non-zero StartTime")
	}

	// Check that source is properly tagged
	if result.Events[0].Source != "CSV" {
		t.Errorf("Expected source 'CSV', got '%s'", result.Events[0].Source)
	}

	// Validate coordinates are parsed (at least for some events)
	hasCoordinates := false
	for _, evt := range result.Events {
		if evt.Event.Latitude != 0 && evt.Event.Longitude != 0 {
			hasCoordinates = true
			// Madrid coordinates sanity check
			if evt.Event.Latitude < 40 || evt.Event.Latitude > 41 {
				t.Errorf("Latitude out of Madrid range: %f", evt.Event.Latitude)
			}
			if evt.Event.Longitude > -3 || evt.Event.Longitude < -4 {
				t.Errorf("Longitude out of Madrid range: %f", evt.Event.Longitude)
			}
			break
		}
	}
	if !hasCoordinates {
		t.Error("Expected at least one event with coordinates")
	}
}

func TestFetchCSV_EncodingConversion(t *testing.T) {
	// Use file:// URL to load fixture (contains Windows-1252 encoded characters)
	fixtureURL := "file:///workspace/testdata/fixtures/madrid-events.csv"

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchCSV(fixtureURL, loc)

	if len(result.Events) == 0 {
		if len(result.Errors) > 0 {
			t.Fatalf("FetchCSV failed: %v", result.Errors[0].Error)
		}
		t.Fatal("Expected at least one event from fixture")
	}

	// Check for proper UTF-8 decoding by looking for Spanish characters
	// The fixture contains "ó" in various fields (años, descripción, etc.)
	hasSpanishChars := false
	for _, evt := range result.Events {
		// Check title and description for Spanish characters
		if containsSpanishChars(evt.Event.Title) || containsSpanishChars(evt.Event.Description) {
			hasSpanishChars = true
			break
		}
	}

	if !hasSpanishChars {
		t.Error("Expected to find Spanish characters (Windows-1252 → UTF-8 conversion)")
	}
}

func TestFetchCSV_PartialFailure(t *testing.T) {
	// Create CSV with one valid event and one invalid event (missing required fields)
	csvData := `ID-EVENTO;TITULO;FECHA;FECHA-FIN;HORA;NOMBRE-INSTALACION;LATITUD;LONGITUD
VALID-001;Valid Event;2025-11-25 00:00:00.0;2025-11-25 23:59:00.0;17:30;Valid Venue;40.423;-3.712
INVALID-001;;2025-11-26 00:00:00.0;2025-11-26 23:59:00.0;18:00;Invalid Venue;40.42;-3.71
VALID-002;Another Valid Event;2025-11-27 00:00:00.0;2025-11-27 23:59:00.0;19:00;Another Venue;40.43;-3.70`

	// Write to temp file
	tmpFile := t.TempDir() + "/partial-fail.csv"
	err := writeFile(tmpFile, []byte(csvData))
	if err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchCSV("file://"+tmpFile, loc)

	// Should have 2 valid events
	if len(result.Events) != 2 {
		t.Errorf("Expected 2 valid events, got %d", len(result.Events))
	}

	// Should have 1 error (the invalid event)
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
		for i, err := range result.Errors {
			t.Logf("Error %d: %v", i, err.Error)
		}
	}

	// Verify the valid events are correct
	if len(result.Events) >= 2 {
		if result.Events[0].Event.ID != "VALID-001" {
			t.Errorf("Expected first event ID 'VALID-001', got '%s'", result.Events[0].Event.ID)
		}
		if result.Events[1].Event.ID != "VALID-002" {
			t.Errorf("Expected second event ID 'VALID-002', got '%s'", result.Events[1].Event.ID)
		}
	}

	// Verify error has proper metadata
	if len(result.Errors) > 0 {
		err := result.Errors[0]
		if err.Source != "CSV" {
			t.Errorf("Expected error source 'CSV', got '%s'", err.Source)
		}
		if err.RecoverType != "skipped" {
			t.Errorf("Expected recover type 'skipped', got '%s'", err.RecoverType)
		}
		if err.Index != 1 {
			t.Errorf("Expected error at index 1 (second row), got %d", err.Index)
		}
	}
}

// Helper functions

func containsSpanishChars(s string) bool {
	// Check for common Spanish characters that would be in Windows-1252
	spanishChars := []rune{'á', 'é', 'í', 'ó', 'ú', 'ñ', 'Á', 'É', 'Í', 'Ó', 'Ú', 'Ñ', 'ü', 'Ü'}
	for _, char := range s {
		for _, spanish := range spanishChars {
			if char == spanish {
				return true
			}
		}
	}
	return false
}

func writeFile(path string, data []byte) error {
	// Simple file write helper
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}
