package filter

import (
	"testing"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/event"
)

func TestFilterCityEvents_GPSRadius(t *testing.T) {
	// Plaza de España coordinates
	centerLat := 40.42338
	centerLon := -3.71217

	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("failed to load Madrid timezone: %v", err)
	}

	now := time.Now().In(madrid)
	events := []event.CityEvent{
		{
			ID:        "1",
			Title:     "Near Plaza de España",
			Latitude:  40.42338, // Exactly at center
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "2",
			Title:     "Within 0.35 km",
			Latitude:  40.42600, // ~300m away
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "3",
			Title:     "Far away",
			Latitude:  40.43000, // ~750m away
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
	}

	filtered := FilterCityEvents(events, centerLat, centerLon, 0.35, nil, 0)

	// Should include events 1 and 2, but not 3
	if len(filtered) != 2 {
		t.Errorf("expected 2 events within radius, got %d", len(filtered))
	}

	// Check that the correct events were included
	foundNear := false
	foundWithin := false
	for _, e := range filtered {
		if e.ID == "1" {
			foundNear = true
		}
		if e.ID == "2" {
			foundWithin = true
		}
		if e.ID == "3" {
			t.Errorf("event 3 should be filtered out (too far)")
		}
	}

	if !foundNear {
		t.Error("event 1 (at center) should be included")
	}
	if !foundWithin {
		t.Error("event 2 (within 0.35km) should be included")
	}
}

func TestFilterCityEvents_Category(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("failed to load Madrid timezone: %v", err)
	}

	now := time.Now().In(madrid)
	events := []event.CityEvent{
		{
			ID:        "1",
			Title:     "City Event",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "2",
			Title:     "Gaming Event",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Gaming",
		},
		{
			ID:        "3",
			Title:     "Other Event",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Other",
		},
	}

	// Test: filter by specific category
	filtered := FilterCityEvents(events, 40.42338, -3.71217, 1.0, []string{"Eventos de ciudad"}, 0)
	if len(filtered) != 1 {
		t.Errorf("expected 1 event with category 'Eventos de ciudad', got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].ID != "1" {
		t.Errorf("expected event 1, got event %s", filtered[0].ID)
	}

	// Test: filter by multiple categories
	filtered = FilterCityEvents(events, 40.42338, -3.71217, 1.0, []string{"Eventos de ciudad", "Gaming"}, 0)
	if len(filtered) != 2 {
		t.Errorf("expected 2 events with specified categories, got %d", len(filtered))
	}

	// Test: empty categories slice should include all events
	filtered = FilterCityEvents(events, 40.42338, -3.71217, 1.0, nil, 0)
	if len(filtered) != 3 {
		t.Errorf("expected 3 events when categories is nil, got %d", len(filtered))
	}

	filtered = FilterCityEvents(events, 40.42338, -3.71217, 1.0, []string{}, 0)
	if len(filtered) != 3 {
		t.Errorf("expected 3 events when categories is empty, got %d", len(filtered))
	}
}

func TestFilterCityEvents_TimeFiltering(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("failed to load Madrid timezone: %v", err)
	}

	now := time.Now().In(madrid)
	events := []event.CityEvent{
		{
			ID:        "1",
			Title:     "Future Event",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "2",
			Title:     "Recent Past (1 week ago)",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(-8 * 24 * time.Hour),
			EndDate:   now.Add(-7 * 24 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "3",
			Title:     "Old Event (3 weeks ago)",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(-22 * 24 * time.Hour),
			EndDate:   now.Add(-21 * 24 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "4",
			Title:     "Currently Ongoing",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(-24 * time.Hour),
			EndDate:   now.Add(24 * time.Hour),
			Category:  "Eventos de ciudad",
		},
	}

	// Test: pastWeeks = 0 should exclude past events
	filtered := FilterCityEvents(events, 40.42338, -3.71217, 1.0, nil, 0)
	if len(filtered) != 2 {
		t.Errorf("expected 2 events (future + ongoing), got %d", len(filtered))
	}
	for _, e := range filtered {
		if e.ID == "2" || e.ID == "3" {
			t.Errorf("event %s (past) should be filtered out when pastWeeks=0", e.ID)
		}
	}

	// Test: pastWeeks = 2 should include event 2 but not event 3
	filtered = FilterCityEvents(events, 40.42338, -3.71217, 1.0, nil, 2)
	if len(filtered) != 3 {
		t.Errorf("expected 3 events (future + ongoing + recent past), got %d", len(filtered))
	}
	foundRecent := false
	for _, e := range filtered {
		if e.ID == "2" {
			foundRecent = true
		}
		if e.ID == "3" {
			t.Errorf("event 3 (3 weeks old) should be filtered out when pastWeeks=2")
		}
	}
	if !foundRecent {
		t.Error("event 2 (1 week old) should be included when pastWeeks=2")
	}

	// Test: pastWeeks = 4 should include all events
	filtered = FilterCityEvents(events, 40.42338, -3.71217, 1.0, nil, 4)
	if len(filtered) != 4 {
		t.Errorf("expected 4 events when pastWeeks=4, got %d", len(filtered))
	}
}

func TestFilterCityEvents_CombinedFilters(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("failed to load Madrid timezone: %v", err)
	}

	now := time.Now().In(madrid)
	events := []event.CityEvent{
		{
			ID:        "1",
			Title:     "Near, correct category, future",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "2",
			Title:     "Far, correct category, future",
			Latitude:  40.43000, // ~750m away
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Eventos de ciudad",
		},
		{
			ID:        "3",
			Title:     "Near, wrong category, future",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(24 * time.Hour),
			EndDate:   now.Add(48 * time.Hour),
			Category:  "Other",
		},
		{
			ID:        "4",
			Title:     "Near, correct category, past",
			Latitude:  40.42338,
			Longitude: -3.71217,
			StartDate: now.Add(-48 * time.Hour),
			EndDate:   now.Add(-24 * time.Hour),
			Category:  "Eventos de ciudad",
		},
	}

	// Apply all filters: radius 0.35km, category "Eventos de ciudad", no past events
	filtered := FilterCityEvents(events, 40.42338, -3.71217, 0.35, []string{"Eventos de ciudad"}, 0)

	// Only event 1 should pass all filters
	if len(filtered) != 1 {
		t.Errorf("expected 1 event passing all filters, got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].ID != "1" {
		t.Errorf("expected event 1, got event %s", filtered[0].ID)
	}
}

func TestFilterCityEvents_EmptyInput(t *testing.T) {
	// Test with empty events slice
	filtered := FilterCityEvents([]event.CityEvent{}, 40.42338, -3.71217, 0.35, nil, 0)
	if len(filtered) != 0 {
		t.Errorf("expected 0 events from empty input, got %d", len(filtered))
	}

	// Test with nil events slice
	filtered = FilterCityEvents(nil, 40.42338, -3.71217, 0.35, nil, 0)
	if len(filtered) != 0 {
		t.Errorf("expected 0 events from nil input, got %d", len(filtered))
	}
}
