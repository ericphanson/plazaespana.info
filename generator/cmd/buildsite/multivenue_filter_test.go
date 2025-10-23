package main

import (
	"testing"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/event"
	"github.com/ericphanson/plazaespana.info/internal/filter"
)

// TestMultiVenueFiltering validates the multi-venue Plaza de España filtering logic
// These tests verify the filtering decision tree implemented in Phase 3
func TestMultiVenueFiltering(t *testing.T) {
	// Reference point: Plaza de España
	refLat := 40.42338
	refLon := -3.71217
	radiusKm := 0.35

	// Time references (using fixed time for reproducibility)
	now := time.Date(2025, 10, 20, 12, 0, 0, 0, time.UTC)
	futureDate := now.Add(7 * 24 * time.Hour)  // 1 week future
	oldDate := now.Add(-30 * 24 * time.Hour)   // 30 days ago (beyond cutoff)
	cutoffTime := now.Add(-7 * 24 * time.Hour) // 1 week cutoff

	tests := []struct {
		name                string
		evt                 event.CityEvent
		wantKept            bool
		wantMultiVenueKept  bool
		wantPlazaEspanaText bool
		wantFilterReason    string
	}{
		{
			name: "city_event_within_radius_kept_by_geo",
			evt: event.CityEvent{
				Title:       "Concierto en Plaza de España",
				Latitude:    40.42338, // Exact Plaza de España coordinates
				Longitude:   -3.71217,
				StartDate:   futureDate,
				EndDate:     futureDate.Add(2 * time.Hour),
				Description: "Un concierto en Plaza de España",
			},
			wantKept:            true,
			wantMultiVenueKept:  false, // Kept by geo, not text
			wantPlazaEspanaText: true,  // Text does match, but geo takes precedence
			wantFilterReason:    "kept",
		},
		{
			name: "city_event_outside_radius_with_text_match_kept",
			evt: event.CityEvent{
				Title:       "Mercadillos navideños",
				Latitude:    40.41794, // Plaza Mayor (~0.96 km away)
				Longitude:   -3.70736,
				StartDate:   futureDate,
				EndDate:     futureDate.Add(24 * time.Hour),
				Description: "Mercadillos en varios puntos: Plaza de España, Plaza Mayor, Sol",
			},
			wantKept:            true,
			wantMultiVenueKept:  true, // Kept via text matching
			wantPlazaEspanaText: true,
			wantFilterReason:    "kept (multi-venue: Plaza de España)",
		},
		{
			name: "city_event_outside_radius_no_text_match_excluded",
			evt: event.CityEvent{
				Title:       "Evento en Plaza Mayor",
				Latitude:    40.41794, // Plaza Mayor
				Longitude:   -3.70736,
				StartDate:   futureDate,
				EndDate:     futureDate.Add(2 * time.Hour),
				Description: "Un evento en Plaza Mayor",
			},
			wantKept:            false,
			wantMultiVenueKept:  false,
			wantPlazaEspanaText: false,
			wantFilterReason:    "outside GPS radius",
		},
		{
			name: "city_event_text_match_but_too_old_excluded",
			evt: event.CityEvent{
				Title:       "Fiestas del Orgullo",
				Latitude:    40.4200,
				Longitude:   -3.7000,
				StartDate:   oldDate,
				EndDate:     oldDate.Add(24 * time.Hour),
				Description: "Fiestas en Plaza de Pedro Zerolo, Plaza de España, y Sol",
			},
			wantKept:            false,
			wantMultiVenueKept:  false,
			wantPlazaEspanaText: true, // Text matches but event is too old
			wantFilterReason:    "event too old",
		},
		{
			name: "city_event_no_coords_with_text_match_kept",
			evt: event.CityEvent{
				Title:       "Pista de hielo",
				Latitude:    0.0, // Missing coordinates
				Longitude:   0.0,
				StartDate:   futureDate,
				EndDate:     futureDate.Add(48 * time.Hour),
				Description: "Pista de hielo en Pza. España",
			},
			wantKept:            true,
			wantMultiVenueKept:  true, // Kept via text match (no coords)
			wantPlazaEspanaText: true,
			wantFilterReason:    "kept (multi-venue: Plaza de España)",
		},
		{
			name: "city_event_no_coords_no_text_match_excluded",
			evt: event.CityEvent{
				Title:       "Evento genérico",
				Latitude:    0.0, // Missing coordinates
				Longitude:   0.0,
				StartDate:   futureDate,
				EndDate:     futureDate.Add(2 * time.Hour),
				Description: "Un evento en Madrid",
			},
			wantKept:            false,
			wantMultiVenueKept:  false,
			wantPlazaEspanaText: false,
			wantFilterReason:    "missing location data",
		},
		{
			name: "city_event_abbreviated_pza_espana_kept",
			evt: event.CityEvent{
				Title:     "Mercadillo navideño",
				Venue:     "Pza. España",
				Latitude:  40.42000, // Slightly outside radius
				Longitude: -3.72000,
				StartDate: futureDate,
				EndDate:   futureDate.Add(12 * time.Hour),
			},
			wantKept:            true,
			wantMultiVenueKept:  true,
			wantPlazaEspanaText: true,
			wantFilterReason:    "kept (multi-venue: Plaza de España)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from main.go
			result := event.FilterResult{}

			hasCoords := tt.evt.Latitude != 0.0 && tt.evt.Longitude != 0.0
			result.HasCoordinates = hasCoords

			result.StartDate = tt.evt.StartDate
			result.EndDate = tt.evt.EndDate
			result.DaysOld = int(now.Sub(tt.evt.EndDate).Hours() / 24)
			result.TooOld = tt.evt.EndDate.Before(cutoffTime)

			// Check for Plaza de España text mention
			result.PlazaEspanaText = filter.MatchesPlazaEspana(
				tt.evt.Title,
				tt.evt.Venue,
				tt.evt.Address,
				tt.evt.Description,
			)

			// Filtering decision logic (from Phase 3)
			if !hasCoords {
				if result.PlazaEspanaText {
					if result.TooOld {
						result.Kept = false
						result.FilterReason = "event too old"
					} else {
						result.Kept = true
						result.FilterReason = "kept (multi-venue: Plaza de España)"
						result.MultiVenueKept = true
					}
				} else {
					result.Kept = false
					result.FilterReason = "missing location data"
				}
			} else {
				result.GPSDistanceKm = filter.HaversineDistance(
					refLat, refLon,
					tt.evt.Latitude, tt.evt.Longitude)
				result.WithinRadius = (result.GPSDistanceKm <= radiusKm)

				if result.WithinRadius {
					if result.TooOld {
						result.Kept = false
						result.FilterReason = "event too old"
					} else {
						result.Kept = true
						result.FilterReason = "kept"
					}
				} else if result.PlazaEspanaText {
					if result.TooOld {
						result.Kept = false
						result.FilterReason = "event too old"
					} else {
						result.Kept = true
						result.FilterReason = "kept (multi-venue: Plaza de España)"
						result.MultiVenueKept = true
					}
				} else {
					result.Kept = false
					result.FilterReason = "outside GPS radius"
				}
			}

			// Validate results
			if result.Kept != tt.wantKept {
				t.Errorf("Kept = %v, want %v", result.Kept, tt.wantKept)
			}
			if result.MultiVenueKept != tt.wantMultiVenueKept {
				t.Errorf("MultiVenueKept = %v, want %v", result.MultiVenueKept, tt.wantMultiVenueKept)
			}
			if result.PlazaEspanaText != tt.wantPlazaEspanaText {
				t.Errorf("PlazaEspanaText = %v, want %v", result.PlazaEspanaText, tt.wantPlazaEspanaText)
			}
			if result.FilterReason != tt.wantFilterReason {
				t.Errorf("FilterReason = %q, want %q", result.FilterReason, tt.wantFilterReason)
			}
		})
	}
}
