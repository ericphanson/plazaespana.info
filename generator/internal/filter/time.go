package filter

import (
	"fmt"
	"time"
)

// ParseEventDateTime parses Madrid API date formats and returns a time.Time in the given timezone.
// Supports multiple formats:
// - DD/MM/YYYY (JSON/XML API format)
// - DD/MM/YYYY HH:MM (with optional time)
// - YYYY-MM-DD HH:MM:SS.S (CSV format with timestamp)
func ParseEventDateTime(fecha, hora string, loc *time.Location) (time.Time, error) {
	// Try different date formats in order of likelihood
	formats := []struct {
		layout  string
		useHora bool // whether to append hora field
	}{
		{"2006-01-02 15:04:05.0", false}, // CSV format with fractional seconds (already has time)
		{"2006-01-02 15:04:05", false},   // CSV format without fractional seconds (already has time)
		{"02/01/2006 15:04", true},       // DD/MM/YYYY format - can have hora appended
		{"02/01/2006", true},             // DD/MM/YYYY without time (can append hora)
	}

	var lastErr error
	for _, fmt := range formats {
		dateStr := fecha
		if fmt.useHora && hora != "" {
			dateStr = fecha + " " + hora
		}

		t, err := time.ParseInLocation(fmt.layout, dateStr, loc)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("parsing date/time: %w", lastErr)
}

// IsInFuture returns true if t is after the reference time.
func IsInFuture(t, reference time.Time) bool {
	return t.After(reference)
}
