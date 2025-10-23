package weather

import (
	"strings"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/render"
)

// BuildWeatherMap creates a map from date string (YYYY-MM-DD) to Weather info
// This can be used to look up weather when converting events to template events
func BuildWeatherMap(forecast *Forecast, basePath string) map[string]*render.Weather {
	if forecast == nil {
		return nil
	}

	weatherMap := make(map[string]*render.Weather)
	for i := range forecast.Prediction.Days {
		day := &forecast.Prediction.Days[i]
		dateStr := extractDate(day.Date)
		weatherMap[dateStr] = buildWeatherForDay(day, basePath)
	}

	return weatherMap
}

// GetWeatherForEvent looks up weather for a specific event from the weather map
// eventDate should be a time.Time representing the event start time
func GetWeatherForEvent(weatherMap map[string]*render.Weather, eventDate time.Time) *render.Weather {
	if weatherMap == nil {
		return nil
	}

	// Format event date as YYYY-MM-DD for lookup
	dateStr := eventDate.Format("2006-01-02")
	return weatherMap[dateStr]
}

// extractDate extracts YYYY-MM-DD from various date formats
func extractDate(dateStr string) string {
	// Handle ISO8601 format: "2025-10-23T00:00:00" -> "2025-10-23"
	if idx := strings.Index(dateStr, "T"); idx > 0 {
		return dateStr[:idx]
	}
	// Handle date-only format: "2025-10-23"
	if len(dateStr) >= 10 {
		return dateStr[:10]
	}
	return dateStr
}

// buildWeatherForDay creates a Weather struct for a specific day
// Uses afternoon/evening period (12-24) as the default for all-day events
func buildWeatherForDay(day *DayForecast, basePath string) *render.Weather {
	// Use afternoon/evening period (12-24) as default
	period := "12-24"

	// Extract sky state for that period
	skyState := extractSkyForPeriod(day.SkyState, period)

	// Extract precipitation probability for that period
	precipProb := extractPrecipProbForPeriod(day.PrecipProbability, period)

	// Extract precipitation amount for that period
	precipAmount := extractPrecipAmountForPeriod(day.Precipitation, period)

	return &render.Weather{
		Date:            extractDate(day.Date),
		TempMax:         day.Temperature.Max,
		TempMin:         day.Temperature.Min,
		PrecipProb:      precipProb,
		PrecipAmount:    precipAmount,
		SkyCode:         skyState.Value,
		SkyDescription:  skyState.Description,
		SkyIconURL:      GetWeatherIconURL(skyState.Value, basePath),
		WeatherCategory: GetWeatherCategory(skyState.Value),
		IsNight:         IsNightCondition(skyState.Value),
	}
}

// extractSkyForPeriod finds the sky state for a given period
func extractSkyForPeriod(skyStates []PeriodValue, period string) PeriodValue {
	// Try to find exact period match
	for _, state := range skyStates {
		if state.Period == period {
			return state
		}
	}

	// Fall back to broader periods
	// If looking for morning (06-12), try 00-12
	if period == "06-12" {
		for _, state := range skyStates {
			if state.Period == "00-12" {
				return state
			}
		}
	}

	// If looking for afternoon/evening (12-18, 18-24), try 12-24
	if period == "12-18" || period == "18-24" {
		for _, state := range skyStates {
			if state.Period == "12-24" {
				return state
			}
		}
	}

	// Fall back to 00-24 (all day)
	for _, state := range skyStates {
		if state.Period == "00-24" {
			return state
		}
	}

	// No data found - return empty
	return PeriodValue{}
}

// extractPrecipProbForPeriod finds the precipitation probability for a given period
func extractPrecipProbForPeriod(probs []PeriodIntValue, period string) int {
	// Try exact match first
	for _, prob := range probs {
		if prob.Period == period && prob.Value != nil {
			return *prob.Value
		}
	}

	// Fall back to broader periods
	if period == "06-12" {
		for _, prob := range probs {
			if prob.Period == "00-12" && prob.Value != nil {
				return *prob.Value
			}
		}
	}

	if period == "12-18" || period == "18-24" {
		for _, prob := range probs {
			if prob.Period == "12-24" && prob.Value != nil {
				return *prob.Value
			}
		}
	}

	// Fall back to all-day
	for _, prob := range probs {
		if prob.Period == "00-24" && prob.Value != nil {
			return *prob.Value
		}
	}

	return 0
}

// extractPrecipAmountForPeriod finds the precipitation amount for a given period
func extractPrecipAmountForPeriod(amounts []PeriodFloatValue, period string) float64 {
	// Try exact match first
	for _, amount := range amounts {
		if amount.Period == period {
			return amount.Value
		}
	}

	// Fall back to broader periods
	if period == "06-12" {
		for _, amount := range amounts {
			if amount.Period == "00-12" {
				return amount.Value
			}
		}
	}

	if period == "12-18" || period == "18-24" {
		for _, amount := range amounts {
			if amount.Period == "12-24" {
				return amount.Value
			}
		}
	}

	// Fall back to all-day
	for _, amount := range amounts {
		if amount.Period == "00-24" {
			return amount.Value
		}
	}

	return 0.0
}
