package filter

import (
	"time"

	"github.com/ericphanson/plazaespana.info/internal/event"
)

// FilterCityEvents filters city events by GPS radius, category, and time.
//
// GPS filtering: Events are included if their distance from (centerLat, centerLon)
// is <= radiusKM kilometers.
//
// Category filtering: If categories is empty or nil, all events are included.
// Otherwise, only events whose Category field matches one of the specified categories
// are included.
//
// Time filtering: Events are excluded if their EndDate is more than (pastWeeks * 7)
// days in the past. If pastWeeks is 0, only events that haven't ended yet are included.
// All time comparisons use Europe/Madrid timezone.
func FilterCityEvents(
	events []event.CityEvent,
	centerLat, centerLon, radiusKM float64,
	categories []string,
	pastWeeks int,
) []event.CityEvent {
	if len(events) == 0 {
		return []event.CityEvent{}
	}

	// Load Madrid timezone for time comparisons
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		// Fallback to UTC if Madrid timezone unavailable
		madrid = time.UTC
	}

	now := time.Now().In(madrid)
	cutoffTime := now.Add(-time.Duration(pastWeeks) * 7 * 24 * time.Hour)

	// Build category lookup map for O(1) category checking
	categoryMap := make(map[string]bool)
	if len(categories) > 0 {
		for _, cat := range categories {
			categoryMap[cat] = true
		}
	}

	filtered := make([]event.CityEvent, 0, len(events))

	for _, e := range events {
		// GPS filtering: check if within radius
		if !WithinRadius(centerLat, centerLon, e.Latitude, e.Longitude, radiusKM) {
			continue
		}

		// Category filtering: if categories specified, event must match
		if len(categoryMap) > 0 && !categoryMap[e.Category] {
			continue
		}

		// Time filtering: exclude events that ended before cutoff time
		if e.EndDate.Before(cutoffTime) {
			continue
		}

		filtered = append(filtered, e)
	}

	return filtered
}
