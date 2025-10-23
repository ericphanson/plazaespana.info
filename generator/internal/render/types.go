package render

import "time"

// TemplateData holds data for HTML template rendering.
type TemplateData struct {
	Lang           string
	CSSHash        string
	BasePath       string
	LastUpdated    string
	GitCommit      string // Git commit hash (set at build time)
	CulturalEvents []TemplateEvent
	CityEvents     []TemplateEvent
	TotalEvents    int
}

// TemplateEvent represents an event for template rendering.
type TemplateEvent struct {
	IDEvento          string
	Titulo            string
	StartHuman        string
	StartTime         time.Time // For sorting
	NombreInstalacion string
	ContentURL        string
	Description       string // Truncated description
	EventType         string // "city" or "cultural"
	DistanceHuman     string // Human-readable distance from Plaza de España (e.g., "250m", "1.2km")
	DistanceMeters    int    // Distance in meters (for display/debugging)
	AtPlaza           bool   // True if event is at Plaza de España (for "En Plaza" filter)
	Weather           *Weather // Weather forecast for event date (nil if unavailable)
}

// Weather represents weather information for a specific event date
type Weather struct {
	Date            string  // Forecast date (YYYY-MM-DD)
	TempMax         int     // Max temp (°C)
	TempMin         int     // Min temp (°C)
	PrecipProb      int     // Precipitation probability (%)
	PrecipAmount    float64 // Total precipitation (mm)
	SkyCode         string  // AEMET sky state code (e.g., "12", "15n")
	SkyDescription  string  // Human-readable sky state (Spanish)
	SkyIconURL      string  // Weather icon URL
	WeatherCategory string  // Simplified category for CSS (clear/cloudy/rain/etc)
	IsNight         bool    // True if code ends with 'n'
}

// JSONEvent represents an event in the machine-readable JSON output.
type JSONEvent struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time,omitempty"`
	VenueName  string `json:"venue_name,omitempty"`
	DetailsURL string `json:"details_url,omitempty"`
}

// JSONOutput is the top-level structure for the JSON API output.
type JSONOutput struct {
	CulturalEvents []JSONEvent `json:"cultural_events"`
	CityEvents     []JSONEvent `json:"city_events"`
	Meta           JSONMeta    `json:"meta"`
}

// JSONMeta contains metadata about the JSON output.
type JSONMeta struct {
	UpdateTime    string `json:"update_time"`
	TotalCultural int    `json:"total_cultural"`
	TotalCity     int    `json:"total_city"`
}
