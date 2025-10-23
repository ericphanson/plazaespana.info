package validate

import (
	"strings"
	"testing"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/event"
)

func TestValidateEvent_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		event   event.CulturalEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid event with all required fields",
			event: event.CulturalEvent{
				ID:        "12345",
				Title:     "Concert at Plaza de España",
				StartTime: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			event: event.CulturalEvent{
				Title:     "Concert",
				StartTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "missing ID",
		},
		{
			name: "missing Title",
			event: event.CulturalEvent{
				ID:        "12345",
				StartTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "missing title",
		},
		{
			name: "missing StartTime",
			event: event.CulturalEvent{
				ID:    "12345",
				Title: "Concert",
			},
			wantErr: true,
			errMsg:  "missing start time",
		},
		{
			name: "multiple missing fields",
			event: event.CulturalEvent{
				Description: "A great event",
			},
			wantErr: true,
			errMsg:  "missing ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEvent(tt.event)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error message should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidateEvent_CoordinateBounds(t *testing.T) {
	baseEvent := event.CulturalEvent{
		ID:        "12345",
		Title:     "Concert",
		StartTime: time.Now(),
	}

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid coordinates (Madrid)",
			latitude:  40.42338,
			longitude: -3.71217,
			wantErr:   false,
		},
		{
			name:      "valid coordinates (North Pole)",
			latitude:  90.0,
			longitude: 0.0,
			wantErr:   false,
		},
		{
			name:      "valid coordinates (South Pole)",
			latitude:  -90.0,
			longitude: 0.0,
			wantErr:   false,
		},
		{
			name:      "valid coordinates (Date Line)",
			latitude:  0.0,
			longitude: 180.0,
			wantErr:   false,
		},
		{
			name:      "valid coordinates (Prime Meridian)",
			latitude:  0.0,
			longitude: -180.0,
			wantErr:   false,
		},
		{
			name:      "zero coordinates (no location)",
			latitude:  0.0,
			longitude: 0.0,
			wantErr:   false, // Zero coords are allowed (events without location)
		},
		{
			name:      "latitude too high",
			latitude:  91.0,
			longitude: 0.0,
			wantErr:   true,
			errMsg:    "invalid latitude: 91.00000",
		},
		{
			name:      "latitude too low",
			latitude:  -91.0,
			longitude: 0.0,
			wantErr:   true,
			errMsg:    "invalid latitude: -91.00000",
		},
		{
			name:      "longitude too high",
			latitude:  40.0,
			longitude: 181.0,
			wantErr:   true,
			errMsg:    "invalid longitude: 181.00000",
		},
		{
			name:      "longitude too low",
			latitude:  40.0,
			longitude: -181.0,
			wantErr:   true,
			errMsg:    "invalid longitude: -181.00000",
		},
		{
			name:      "both coordinates invalid",
			latitude:  100.0,
			longitude: 200.0,
			wantErr:   true,
			errMsg:    "invalid latitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := baseEvent
			evt.Latitude = tt.latitude
			evt.Longitude = tt.longitude

			err := ValidateEvent(evt)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error message should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestSanitizeEvent_Whitespace(t *testing.T) {
	evt := event.CulturalEvent{
		ID:        "  12345  ",
		Title:     "\tConcert at Plaza\n",
		VenueName: "  Teatro Real  ",
	}

	SanitizeEvent(&evt)

	if evt.ID != "12345" {
		t.Errorf("ID whitespace not trimmed: got %q", evt.ID)
	}
	if evt.Title != "Concert at Plaza" {
		t.Errorf("Title whitespace not trimmed: got %q", evt.Title)
	}
	if evt.VenueName != "Teatro Real" {
		t.Errorf("VenueName whitespace not trimmed: got %q", evt.VenueName)
	}
}

func TestSanitizeEvent_DefaultEndTime(t *testing.T) {
	tests := []struct {
		name          string
		startTime     time.Time
		endTime       time.Time
		expectedDiff  time.Duration
		expectChanged bool
	}{
		{
			name:          "missing end time gets default",
			startTime:     time.Date(2025, 10, 20, 18, 0, 0, 0, time.UTC),
			endTime:       time.Time{}, // Zero value
			expectedDiff:  2 * time.Hour,
			expectChanged: true,
		},
		{
			name:          "existing end time preserved",
			startTime:     time.Date(2025, 10, 20, 18, 0, 0, 0, time.UTC),
			endTime:       time.Date(2025, 10, 20, 21, 0, 0, 0, time.UTC),
			expectedDiff:  3 * time.Hour,
			expectChanged: false,
		},
		{
			name:          "zero start time no change",
			startTime:     time.Time{},
			endTime:       time.Time{},
			expectedDiff:  0,
			expectChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.CulturalEvent{
				ID:        "12345",
				Title:     "Concert",
				StartTime: tt.startTime,
				EndTime:   tt.endTime,
			}

			SanitizeEvent(&evt)

			if tt.expectChanged && evt.EndTime.IsZero() {
				t.Error("EndTime should have been set but is still zero")
			}

			if !tt.expectChanged && tt.endTime != evt.EndTime {
				t.Errorf("EndTime changed unexpectedly: got %v, want %v", evt.EndTime, tt.endTime)
			}

			if tt.expectChanged && !evt.StartTime.IsZero() {
				diff := evt.EndTime.Sub(evt.StartTime)
				if diff != tt.expectedDiff {
					t.Errorf("EndTime diff = %v, want %v", diff, tt.expectedDiff)
				}
			}
		})
	}
}

func TestSanitizeEvent_DeduplicateSources(t *testing.T) {
	evt := event.CulturalEvent{
		ID:      "12345",
		Title:   "Concert",
		Sources: []string{"JSON", "XML", "JSON", "CSV", "XML"},
	}

	SanitizeEvent(&evt)

	// Check for duplicates
	seen := make(map[string]bool)
	for _, source := range evt.Sources {
		if seen[source] {
			t.Errorf("duplicate source found: %s", source)
		}
		seen[source] = true
	}

	// Should have exactly 3 unique sources
	if len(evt.Sources) != 3 {
		t.Errorf("expected 3 unique sources, got %d: %v", len(evt.Sources), evt.Sources)
	}
}
