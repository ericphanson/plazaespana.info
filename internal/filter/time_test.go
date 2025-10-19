package filter

import (
	"testing"
	"time"
)

func TestParseEventDateTime(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("Failed to load Europe/Madrid timezone: %v", err)
	}

	tests := []struct {
		name        string
		fecha       string
		hora        string
		expectError bool
		expectedDay int
	}{
		{
			name:        "Valid date with time",
			fecha:       "15/11/2025",
			hora:        "19:30",
			expectError: false,
			expectedDay: 15,
		},
		{
			name:        "Valid date without time (all-day)",
			fecha:       "20/11/2025",
			hora:        "",
			expectError: false,
			expectedDay: 20,
		},
		{
			name:        "Invalid date format",
			fecha:       "2025-11-15",
			hora:        "19:30",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseEventDateTime(tt.fecha, tt.hora, loc)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Day() != tt.expectedDay {
				t.Errorf("Expected day %d, got %d", tt.expectedDay, result.Day())
			}

			if result.Location() != loc {
				t.Errorf("Expected Europe/Madrid timezone, got %s", result.Location())
			}
		})
	}
}

func TestIsInFuture(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Madrid")
	now := time.Now().In(loc)

	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	if !IsInFuture(futureTime, now) {
		t.Error("Expected future time to be in future")
	}

	if IsInFuture(pastTime, now) {
		t.Error("Expected past time to not be in future")
	}
}
