package filter

import "testing"

func TestIsAtPlazaEspana(t *testing.T) {
	tests := []struct {
		name     string
		venue    string
		expected bool
	}{
		// Positive cases - should match
		{
			name:     "Exact match with accents",
			venue:    "Plaza de España",
			expected: true,
		},
		{
			name:     "Uppercase",
			venue:    "PLAZA DE ESPAÑA",
			expected: true,
		},
		{
			name:     "Lowercase",
			venue:    "plaza de españa",
			expected: true,
		},
		{
			name:     "Mixed case",
			venue:    "Plaza De España",
			expected: true,
		},
		{
			name:     "Without de",
			venue:    "Plaza España",
			expected: true,
		},
		{
			name:     "Abbreviated pl. de",
			venue:    "Pl. de España",
			expected: true,
		},
		{
			name:     "Abbreviated pl. without de",
			venue:    "Pl. España",
			expected: true,
		},
		{
			name:     "Abbreviated without period",
			venue:    "Pl España",
			expected: true,
		},
		{
			name:     "Plza abbreviation",
			venue:    "Plza España",
			expected: true,
		},
		{
			name:     "Plza with period",
			venue:    "Plza. España",
			expected: true,
		},
		{
			name:     "Without accent on a",
			venue:    "Plaza de Espana",
			expected: true,
		},
		{
			name:     "Part of longer text",
			venue:    "Evento en Plaza de España, Madrid",
			expected: true,
		},
		{
			name:     "Different accents",
			venue:    "Pláza dé Españá",
			expected: true,
		},

		// Negative cases - should NOT match
		{
			name:     "Empty string",
			venue:    "",
			expected: false,
		},
		{
			name:     "Only Plaza",
			venue:    "Plaza Mayor",
			expected: false,
		},
		{
			name:     "Only España",
			venue:    "Jardines de España",
			expected: false,
		},
		{
			name:     "Different plaza",
			venue:    "Plaza de Cibeles",
			expected: false,
		},
		{
			name:     "Substring but not plaza",
			venue:    "Estación España",
			expected: false,
		},
		{
			name:     "Unrelated venue",
			venue:    "Teatro Real",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAtPlazaEspana(tt.venue)
			if result != tt.expected {
				t.Errorf("IsAtPlazaEspana(%q) = %v, expected %v", tt.venue, result, tt.expected)
			}
		})
	}
}

func TestGetDistanceBucket(t *testing.T) {
	tests := []struct {
		name           string
		distanceMeters int
		expected       string
	}{
		// 0-250m bucket
		{
			name:           "0 meters",
			distanceMeters: 0,
			expected:       "0-250",
		},
		{
			name:           "50 meters",
			distanceMeters: 50,
			expected:       "0-250",
		},
		{
			name:           "250 meters (boundary)",
			distanceMeters: 250,
			expected:       "0-250",
		},

		// 251-500m bucket
		{
			name:           "251 meters",
			distanceMeters: 251,
			expected:       "251-500",
		},
		{
			name:           "400 meters",
			distanceMeters: 400,
			expected:       "251-500",
		},
		{
			name:           "500 meters (boundary)",
			distanceMeters: 500,
			expected:       "251-500",
		},

		// 501-750m bucket
		{
			name:           "501 meters",
			distanceMeters: 501,
			expected:       "501-750",
		},
		{
			name:           "650 meters",
			distanceMeters: 650,
			expected:       "501-750",
		},
		{
			name:           "750 meters (boundary)",
			distanceMeters: 750,
			expected:       "501-750",
		},

		// 751-1000m bucket
		{
			name:           "751 meters",
			distanceMeters: 751,
			expected:       "751-1000",
		},
		{
			name:           "900 meters",
			distanceMeters: 900,
			expected:       "751-1000",
		},
		{
			name:           "1000 meters (boundary)",
			distanceMeters: 1000,
			expected:       "751-1000",
		},

		// 1000+ bucket
		{
			name:           "1001 meters",
			distanceMeters: 1001,
			expected:       "1000+",
		},
		{
			name:           "2000 meters",
			distanceMeters: 2000,
			expected:       "1000+",
		},
		{
			name:           "10000 meters",
			distanceMeters: 10000,
			expected:       "1000+",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDistanceBucket(tt.distanceMeters)
			if result != tt.expected {
				t.Errorf("GetDistanceBucket(%d) = %q, expected %q", tt.distanceMeters, result, tt.expected)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove Spanish accents",
			input:    "España",
			expected: "espana",
		},
		{
			name:     "Remove múltiple accents",
			input:    "Málaga, Córdoba, Cádiz",
			expected: "malaga, cordoba, cadiz",
		},
		{
			name:     "Uppercase to lowercase",
			input:    "MADRID",
			expected: "madrid",
		},
		{
			name:     "Mixed case with accents",
			input:    "Código Postal",
			expected: "codigo postal",
		},
		{
			name:     "Already normalized",
			input:    "plaza mayor",
			expected: "plaza mayor",
		},
		{
			name:     "French accents",
			input:    "Café français",
			expected: "cafe francais",
		},
		{
			name:     "German umlauts",
			input:    "Müller über",
			expected: "muller uber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeText(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
