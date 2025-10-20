package event

import (
	"math"
	"time"
)

const earthRadiusKm = 6371.0

// CityEvent represents a city/tourism event from esmadrid.com.
// This includes festivals, outdoor events, gaming events, and other
// non-cultural city activities.
type CityEvent struct {
	ID          string
	Title       string
	Description string
	StartDate   time.Time
	EndDate     time.Time
	Venue       string
	Address     string
	Latitude    float64
	Longitude   float64
	Category    string
	Subcategory string
	WebURL      string
	ImageURL    string
	Price       string
}

// EventType returns the type of this event.
func (e CityEvent) EventType() string {
	return "city"
}

// Distance calculates the great-circle distance (in kilometers) from
// this event's location to the given coordinates using the Haversine formula.
func (e CityEvent) Distance(lat, lon float64) float64 {
	return haversineDistance(e.Latitude, e.Longitude, lat, lon)
}

// haversineDistance calculates the great-circle distance between two points
// on Earth's surface (in kilometers) using the Haversine formula.
// This is a copy of internal/filter/geo.go's HaversineDistance to avoid import cycles.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
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
