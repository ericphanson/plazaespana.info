package weather

import (
	"fmt"
	"strings"
)

// GetWeatherIconURL returns the weather icon URL for a sky state code
// Icons are served from our own site (/assets/weather-icons/), copied from fixtures during build
func GetWeatherIconURL(code, basePath string) string {
	// AEMET icons use numeric codes: 11, 12, 13, 14, etc.
	// Some codes have 'n' suffix for night (e.g., "11n")
	// The icon files use just the base code (11.png works for both 11 and 11n)
	baseCode := strings.TrimSuffix(code, "n")
	return fmt.Sprintf("%s/assets/weather-icons/%s.png", basePath, baseCode)
}

// IsNightCondition checks if the code represents a night condition
func IsNightCondition(code string) bool {
	return strings.HasSuffix(code, "n")
}

// GetWeatherCategory returns a simplified category for CSS styling
func GetWeatherCategory(code string) string {
	baseCode := strings.TrimSuffix(code, "n")

	// Handle empty code
	if baseCode == "" {
		return "unknown"
	}

	// Compare lexicographically for code ranges
	switch {
	case baseCode >= "11" && baseCode <= "13":
		return "clear" // Clear/Despejado
	case baseCode == "14" || baseCode == "15":
		return "partial" // Few clouds/Partly cloudy
	case baseCode == "16" || baseCode == "17":
		return "cloudy" // Very cloudy/Overcast
	case baseCode >= "23" && baseCode <= "27":
		return "rain" // Rain
	case baseCode >= "43" && baseCode <= "46":
		return "snow" // Snow
	case baseCode >= "51" && baseCode <= "53":
		return "storm" // Storm
	default:
		return "unknown"
	}
}
