package fetch

import (
	"testing"
	"time"
)

func TestJSONEvent_ToCanonical(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	tests := []struct {
		name      string
		event     JSONEvent
		wantErr   bool
		checkFunc func(*testing.T, JSONEvent)
	}{
		{
			name: "valid event with all fields",
			event: JSONEvent{
				ID:          "12345",
				Title:       "Test Event",
				Description: "A test event description",
				StartTime:   "2025-10-25 19:00:00.0",
				EndTime:     "2025-10-25 23:59:00.0",
				Latitude:    40.42338,
				Longitude:   -3.71217,
				Location:    "Plaza de España",
				Link:        "http://example.com/event",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, event JSONEvent) {
				canonical, err := event.ToCanonical(loc)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if canonical.ID != "12345" {
					t.Errorf("ID = %q, want %q", canonical.ID, "12345")
				}
				if canonical.Title != "Test Event" {
					t.Errorf("Title = %q, want %q", canonical.Title, "Test Event")
				}
				if canonical.Description != "A test event description" {
					t.Errorf("Description = %q, want %q", canonical.Description, "A test event description")
				}
				if canonical.VenueName != "Plaza de España" {
					t.Errorf("VenueName = %q, want %q", canonical.VenueName, "Plaza de España")
				}
				if canonical.DetailsURL != "http://example.com/event" {
					t.Errorf("DetailsURL = %q, want %q", canonical.DetailsURL, "http://example.com/event")
				}
				if canonical.Latitude != 40.42338 {
					t.Errorf("Latitude = %f, want %f", canonical.Latitude, 40.42338)
				}
				if canonical.Longitude != -3.71217 {
					t.Errorf("Longitude = %f, want %f", canonical.Longitude, -3.71217)
				}

				// Check time parsing
				expectedStart := time.Date(2025, 10, 25, 19, 0, 0, 0, loc)
				if !canonical.StartTime.Equal(expectedStart) {
					t.Errorf("StartTime = %v, want %v", canonical.StartTime, expectedStart)
				}

				expectedEnd := time.Date(2025, 10, 25, 23, 59, 0, 0, loc)
				if !canonical.EndTime.Equal(expectedEnd) {
					t.Errorf("EndTime = %v, want %v", canonical.EndTime, expectedEnd)
				}

				// Check source tracking
				if len(canonical.Sources) != 1 || canonical.Sources[0] != "JSON" {
					t.Errorf("Sources = %v, want [JSON]", canonical.Sources)
				}
			},
		},
		{
			name: "missing ID should fail validation",
			event: JSONEvent{
				ID:        "",
				Title:     "Test Event",
				StartTime: "2025-10-25 19:00:00.0",
			},
			wantErr: true,
		},
		{
			name: "missing title should fail validation",
			event: JSONEvent{
				ID:        "12345",
				Title:     "",
				StartTime: "2025-10-25 19:00:00.0",
			},
			wantErr: true,
		},
		{
			name: "missing start time should fail",
			event: JSONEvent{
				ID:        "12345",
				Title:     "Test Event",
				StartTime: "",
			},
			wantErr: true,
		},
		{
			name: "invalid start time format should fail",
			event: JSONEvent{
				ID:        "12345",
				Title:     "Test Event",
				StartTime: "invalid-date",
			},
			wantErr: true,
		},
		{
			name: "invalid coordinates should fail validation",
			event: JSONEvent{
				ID:        "12345",
				Title:     "Test Event",
				StartTime: "2025-10-25 19:00:00.0",
				Latitude:  999.0,
				Longitude: -3.71217,
			},
			wantErr: true,
		},
		{
			name: "event with whitespace should be sanitized",
			event: JSONEvent{
				ID:        "  12345  ",
				Title:     "  Test Event  ",
				StartTime: "2025-10-25 19:00:00.0",
				Location:  "  Plaza de España  ",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, event JSONEvent) {
				canonical, err := event.ToCanonical(loc)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if canonical.ID != "12345" {
					t.Errorf("ID not trimmed: %q", canonical.ID)
				}
				if canonical.Title != "Test Event" {
					t.Errorf("Title not trimmed: %q", canonical.Title)
				}
				if canonical.VenueName != "Plaza de España" {
					t.Errorf("VenueName not trimmed: %q", canonical.VenueName)
				}
			},
		},
		{
			name: "event without end time should get default",
			event: JSONEvent{
				ID:        "12345",
				Title:     "Test Event",
				StartTime: "2025-10-25 19:00:00.0",
				EndTime:   "",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, event JSONEvent) {
				canonical, err := event.ToCanonical(loc)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// SanitizeEvent should add 2 hours to start time
				expectedEnd := canonical.StartTime.Add(2 * time.Hour)
				if !canonical.EndTime.Equal(expectedEnd) {
					t.Errorf("EndTime = %v, want %v (start + 2h)", canonical.EndTime, expectedEnd)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, err := tt.event.ToCanonical(loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToCanonical() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, tt.event)
			}

			// Additional check: all valid events should have non-zero times
			if !tt.wantErr && canonical.StartTime.IsZero() {
				t.Error("valid event has zero StartTime")
			}
		})
	}
}

func TestFetchJSON_FieldMapping(t *testing.T) {
	// This test uses the real fixture to verify field mapping
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Use file:// URL to load local fixture
	fixtureURL := getFixturePath(t, "madrid-events.json")
	result := client.FetchJSON(fixtureURL, loc)

	// Should have successfully parsed events
	if len(result.Events) == 0 {
		t.Error("expected some events to be parsed")
	}

	// Check that we got events with proper field mapping
	foundValidEvent := false
	for _, sourced := range result.Events {
		evt := sourced.Event
		if evt.ID != "" && evt.Title != "" && !evt.StartTime.IsZero() {
			foundValidEvent = true

			// Verify source tracking
			if len(evt.Sources) != 1 || evt.Sources[0] != "JSON" {
				t.Errorf("Event %s has wrong sources: %v", evt.ID, evt.Sources)
			}

			// Basic sanity checks
			if evt.Latitude != 0 && (evt.Latitude < -90 || evt.Latitude > 90) {
				t.Errorf("Event %s has invalid latitude: %f", evt.ID, evt.Latitude)
			}
			if evt.Longitude != 0 && (evt.Longitude < -180 || evt.Longitude > 180) {
				t.Errorf("Event %s has invalid longitude: %f", evt.ID, evt.Longitude)
			}

			break
		}
	}

	if !foundValidEvent {
		t.Error("no valid events found with ID, Title, and StartTime")
	}

	// Log parse statistics
	t.Logf("JSON parsing: %d events, %d errors", len(result.Events), len(result.Errors))
	if len(result.Errors) > 0 {
		t.Logf("First error: %v", result.Errors[0].Error)
	}
}

func TestFetchJSON_PartialFailure(t *testing.T) {
	// This test verifies that partial failures don't crash the entire parse
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(10*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Use file:// URL to load local fixture
	fixtureURL := getFixturePath(t, "madrid-events.json")
	result := client.FetchJSON(fixtureURL, loc)

	// We should have both successes and potentially some errors
	if len(result.Events) == 0 && len(result.Errors) == 0 {
		t.Error("expected either events or errors")
	}

	// Verify error recovery - errors should have source and context
	for i, parseErr := range result.Errors {
		if parseErr.Source != "JSON" {
			t.Errorf("Error %d has wrong source: %q", i, parseErr.Source)
		}
		if parseErr.RecoverType != "skipped" {
			t.Errorf("Error %d has wrong recover type: %q", i, parseErr.RecoverType)
		}
		if parseErr.Error == nil {
			t.Errorf("Error %d has nil error", i)
		}
	}

	// Log statistics
	totalAttempts := len(result.Events) + len(result.Errors)
	if totalAttempts == 0 {
		t.Log("No events attempted (might be empty fixture)")
	} else {
		successRate := float64(len(result.Events)) / float64(totalAttempts) * 100
		t.Logf("Success rate: %.1f%% (%d/%d)", successRate, len(result.Events), totalAttempts)
	}
}
