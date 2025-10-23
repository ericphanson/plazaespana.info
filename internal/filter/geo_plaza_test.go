package filter

import "testing"

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
