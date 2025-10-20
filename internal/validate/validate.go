package validate

import (
	"fmt"
	"strings"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
)

// ValidateEvent checks if cultural event has required fields.
// Returns error if critical data is missing or invalid.
func ValidateEvent(evt event.CulturalEvent) error {
	var issues []string

	// Required fields
	if evt.ID == "" {
		issues = append(issues, "missing ID")
	}
	if evt.Title == "" {
		issues = append(issues, "missing title")
	}
	if evt.StartTime.IsZero() {
		issues = append(issues, "missing start time")
	}

	// Coordinate sanity checks
	if evt.Latitude != 0 || evt.Longitude != 0 {
		if evt.Latitude < -90 || evt.Latitude > 90 {
			issues = append(issues, fmt.Sprintf("invalid latitude: %.5f", evt.Latitude))
		}
		if evt.Longitude < -180 || evt.Longitude > 180 {
			issues = append(issues, fmt.Sprintf("invalid longitude: %.5f", evt.Longitude))
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(issues, ", "))
	}
	return nil
}

// SanitizeEvent fixes common data quality issues.
func SanitizeEvent(evt *event.CulturalEvent) {
	// Trim whitespace
	evt.ID = strings.TrimSpace(evt.ID)
	evt.Title = strings.TrimSpace(evt.Title)
	evt.VenueName = strings.TrimSpace(evt.VenueName)

	// Fix end time if missing (use start time)
	if evt.EndTime.IsZero() && !evt.StartTime.IsZero() {
		evt.EndTime = evt.StartTime.Add(2 * time.Hour) // Default 2hr event
	}

	// Deduplicate sources
	evt.Sources = uniqueStrings(evt.Sources)
}

// uniqueStrings removes duplicates from a string slice.
func uniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
