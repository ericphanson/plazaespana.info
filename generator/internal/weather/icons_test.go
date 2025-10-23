package weather

import "testing"

func TestGetWeatherIconURL(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		basePath string
		expected string
	}{
		{
			name:     "Day code",
			code:     "12",
			basePath: "/base",
			expected: "/base/assets/weather-icons/12.png",
		},
		{
			name:     "Night code with n suffix",
			code:     "12n",
			basePath: "/base",
			expected: "/base/assets/weather-icons/12.png",
		},
		{
			name:     "Empty base path",
			code:     "14",
			basePath: "",
			expected: "/assets/weather-icons/14.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWeatherIconURL(tt.code, tt.basePath)
			if result != tt.expected {
				t.Errorf("GetWeatherIconURL(%q, %q) = %q, want %q", tt.code, tt.basePath, result, tt.expected)
			}
		})
	}
}

func TestGetWeatherCategory(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		// Clear skies (11-13)
		{name: "Clear sky", code: "11", expected: "clear"},
		{name: "Clear sky night", code: "11n", expected: "clear"},
		{name: "Few clouds", code: "12", expected: "clear"},
		{name: "Few clouds night", code: "12n", expected: "clear"},
		{name: "Intervals with high clouds", code: "13", expected: "clear"},

		// Partial clouds (14-15)
		{name: "Partly cloudy", code: "14", expected: "partial"},
		{name: "Cloudy", code: "15", expected: "partial"},

		// Cloudy (16-17)
		{name: "Very cloudy", code: "16", expected: "cloudy"},
		{name: "Overcast", code: "17", expected: "cloudy"},

		// Rain (23-27)
		{name: "Light rain", code: "23", expected: "rain"},
		{name: "Rain", code: "24", expected: "rain"},
		{name: "Heavy rain", code: "25", expected: "rain"},

		// Snow (43-46)
		{name: "Snow showers", code: "43", expected: "snow"},
		{name: "Snow", code: "44", expected: "snow"},

		// Storm (51-53)
		{name: "Storm", code: "51", expected: "storm"},
		{name: "Storm with hail", code: "52", expected: "storm"},
		{name: "Severe storm", code: "53", expected: "storm"},

		// Unknown codes
		{name: "Unknown code 33", code: "33", expected: "unknown"},
		{name: "Unknown code 61", code: "61", expected: "unknown"},
		{name: "Unknown code 99", code: "99", expected: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWeatherCategory(tt.code)
			if result != tt.expected {
				t.Errorf("GetWeatherCategory(%q) = %q, want %q", tt.code, result, tt.expected)
			}
		})
	}
}

func TestIsNightCondition(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{name: "Day code", code: "12", expected: false},
		{name: "Night code", code: "12n", expected: true},
		{name: "Night code uppercase", code: "12N", expected: false}, // HasSuffix is case-sensitive
		{name: "Code with n in middle", code: "1n2", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNightCondition(tt.code)
			if result != tt.expected {
				t.Errorf("IsNightCondition(%q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}
