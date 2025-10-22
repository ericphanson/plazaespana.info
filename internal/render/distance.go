package render

import "fmt"

// FormatDistance converts a distance in kilometers to a human-readable string.
// Distances less than 1km are shown in meters (e.g., "350m").
// Distances >= 1km are shown in kilometers with one decimal place (e.g., "1.2km").
func FormatDistance(distanceKm float64) string {
	if distanceKm < 1.0 {
		meters := int(distanceKm * 1000)
		return fmt.Sprintf("%dm", meters)
	}
	return fmt.Sprintf("%.1fkm", distanceKm)
}
