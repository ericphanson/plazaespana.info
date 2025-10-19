package filter

import (
	"fmt"
	"time"
)

// ParseEventDateTime parses Madrid API date format (DD/MM/YYYY) and optional time (HH:MM).
// Returns a time.Time in the given timezone.
func ParseEventDateTime(fecha, hora string, loc *time.Location) (time.Time, error) {
	// Madrid API uses DD/MM/YYYY format
	layout := "02/01/2006"
	if hora != "" {
		layout += " 15:04"
		fecha = fecha + " " + hora
	}

	t, err := time.ParseInLocation(layout, fecha, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing date/time: %w", err)
	}

	return t, nil
}

// IsInFuture returns true if t is after the reference time.
func IsInFuture(t, reference time.Time) bool {
	return t.After(reference)
}
