package filter

import "math"

const earthRadiusKm = 6371.0

// HaversineDistance calculates the great-circle distance between two points
// on Earth's surface (in kilometers) using the Haversine formula.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLatRad := (lat2 - lat1) * math.Pi / 180
	deltaLonRad := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLonRad/2)*math.Sin(deltaLonRad/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// WithinRadius returns true if the distance between two points is ≤ radius km.
func WithinRadius(lat1, lon1, lat2, lon2, radiusKm float64) bool {
	return HaversineDistance(lat1, lon1, lat2, lon2) <= radiusKm
}

// GetDistanceBucket returns the distance range bucket for CSS filtering.
// Buckets: "0-250", "251-500", "501-750", "751-1000", "1000+"
func GetDistanceBucket(distanceMeters int) string {
	if distanceMeters <= 250 {
		return "0-250"
	} else if distanceMeters <= 500 {
		return "251-500"
	} else if distanceMeters <= 750 {
		return "501-750"
	} else if distanceMeters <= 1000 {
		return "751-1000"
	}
	return "1000+"
}
