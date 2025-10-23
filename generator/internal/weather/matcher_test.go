package weather

import (
	"testing"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/render"
)

func TestExtractDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ISO8601 with time",
			input:    "2025-10-23T00:00:00",
			expected: "2025-10-23",
		},
		{
			name:     "Date only",
			input:    "2025-10-23",
			expected: "2025-10-23",
		},
		{
			name:     "Short string",
			input:    "2025-10",
			expected: "2025-10",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDate(tt.input)
			if result != tt.expected {
				t.Errorf("extractDate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildWeatherMap(t *testing.T) {
	// Helper to create int pointers
	intPtr := func(i int) *int { return &i }

	forecast := &Forecast{
		Prediction: Prediction{
			Days: []DayForecast{
				{
					Date: "2025-10-23T00:00:00",
					Temperature: Temperature{
						Max: 25,
						Min: 15,
					},
					SkyState: []PeriodValue{
						{Period: "12-24", Value: "12", Description: "Poco nuboso"},
					},
					PrecipProbability: []PeriodIntValue{
						{Period: "12-24", Value: intPtr(20)},
					},
					Precipitation: []PeriodFloatValue{
						{Period: "12-24", Value: 0.5},
					},
				},
				{
					Date: "2025-10-24T00:00:00",
					Temperature: Temperature{
						Max: 22,
						Min: 12,
					},
					SkyState: []PeriodValue{
						{Period: "12-24", Value: "14", Description: "Nuboso"},
					},
					PrecipProbability: []PeriodIntValue{
						{Period: "12-24", Value: intPtr(60)},
					},
					Precipitation: []PeriodFloatValue{
						{Period: "12-24", Value: 2.0},
					},
				},
			},
		},
	}

	weatherMap := BuildWeatherMap(forecast, "/test")

	if weatherMap == nil {
		t.Fatal("BuildWeatherMap returned nil")
	}

	if len(weatherMap) != 2 {
		t.Fatalf("Expected 2 entries in weather map, got %d", len(weatherMap))
	}

	// Check first day
	w1 := weatherMap["2025-10-23"]
	if w1 == nil {
		t.Fatal("Weather for 2025-10-23 not found")
	}
	if w1.TempMax != 25 {
		t.Errorf("Expected TempMax=25, got %d", w1.TempMax)
	}
	if w1.TempMin != 15 {
		t.Errorf("Expected TempMin=15, got %d", w1.TempMin)
	}
	if w1.SkyCode != "12" {
		t.Errorf("Expected SkyCode=12, got %s", w1.SkyCode)
	}
	if w1.PrecipProb != 20 {
		t.Errorf("Expected PrecipProb=20, got %d", w1.PrecipProb)
	}

	// Check second day
	w2 := weatherMap["2025-10-24"]
	if w2 == nil {
		t.Fatal("Weather for 2025-10-24 not found")
	}
	if w2.TempMax != 22 {
		t.Errorf("Expected TempMax=22, got %d", w2.TempMax)
	}
	if w2.PrecipProb != 60 {
		t.Errorf("Expected PrecipProb=60, got %d", w2.PrecipProb)
	}
}

func TestGetWeatherForEvent(t *testing.T) {
	weatherMap := map[string]*render.Weather{
		"2025-10-23": {
			Date:    "2025-10-23",
			TempMax: 25,
			TempMin: 15,
		},
		"2025-10-24": {
			Date:    "2025-10-24",
			TempMax: 22,
			TempMin: 12,
		},
	}

	madridTZ, _ := time.LoadLocation("Europe/Madrid")

	tests := []struct {
		name      string
		eventDate time.Time
		wantFound bool
		wantMax   int
	}{
		{
			name:      "Event with matching weather",
			eventDate: time.Date(2025, 10, 23, 18, 0, 0, 0, madridTZ),
			wantFound: true,
			wantMax:   25,
		},
		{
			name:      "Event with no matching weather",
			eventDate: time.Date(2025, 10, 25, 18, 0, 0, 0, madridTZ),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather := GetWeatherForEvent(weatherMap, tt.eventDate)
			if tt.wantFound {
				if weather == nil {
					t.Fatal("Expected weather but got nil")
				}
				if weather.TempMax != tt.wantMax {
					t.Errorf("Expected TempMax=%d, got %d", tt.wantMax, weather.TempMax)
				}
			} else {
				if weather != nil {
					t.Errorf("Expected nil weather but got %+v", weather)
				}
			}
		})
	}
}

func TestExtractSkyForPeriod(t *testing.T) {
	skyStates := []PeriodValue{
		{Period: "00-24", Value: "11", Description: "Despejado"},
		{Period: "12-24", Value: "12", Description: "Poco nuboso"},
	}

	tests := []struct {
		name         string
		period       string
		expectedCode string
	}{
		{
			name:         "Exact match",
			period:       "12-24",
			expectedCode: "12",
		},
		{
			name:         "Fallback to all-day",
			period:       "06-12",
			expectedCode: "11",
		},
		{
			name:         "No match - falls back to all-day",
			period:       "99-99",
			expectedCode: "11", // Falls back to 00-24
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSkyForPeriod(skyStates, tt.period)
			if result.Value != tt.expectedCode {
				t.Errorf("extractSkyForPeriod(%q) = %q, want %q", tt.period, result.Value, tt.expectedCode)
			}
		})
	}
}
