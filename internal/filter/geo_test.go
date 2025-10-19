package filter

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "Same point",
			lat1:      40.42338,
			lon1:      -3.71217,
			lat2:      40.42338,
			lon2:      -3.71217,
			expected:  0.0,
			tolerance: 0.001,
		},
		{
			name:      "Plaza de España to nearby point (~350m)",
			lat1:      40.42338,
			lon1:      -3.71217,
			lat2:      40.42650,
			lon2:      -3.71217,
			expected:  0.35,
			tolerance: 0.02,
		},
		{
			name:      "Plaza de España to far point (~5km)",
			lat1:      40.42338,
			lon1:      -3.71217,
			lat2:      40.46838,
			lon2:      -3.71217,
			expected:  5.0,
			tolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("Expected ~%.2f km, got %.2f km", tt.expected, result)
			}
		})
	}
}

func TestWithinRadius(t *testing.T) {
	plazaLat := 40.42338
	plazaLon := -3.71217
	radius := 0.35

	tests := []struct {
		name     string
		lat      float64
		lon      float64
		expected bool
	}{
		{"At plaza", plazaLat, plazaLon, true},
		{"Just inside", 40.42500, -3.71217, true},
		{"Far away", 40.50000, -3.71217, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithinRadius(plazaLat, plazaLon, tt.lat, tt.lon, radius)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v (distance: %.2f km)",
					tt.expected, result,
					HaversineDistance(plazaLat, plazaLon, tt.lat, tt.lon))
			}
		})
	}
}
