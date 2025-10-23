package render

import "time"

// TemplateData holds data for HTML template rendering.
type TemplateData struct {
	Lang           string
	CSSHash        string
	LastUpdated    string
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
	DistanceMeters    int    // Distance in meters for CSS filtering (0-1000)
	DistanceBucket    string // Distance bucket for CSS filtering ("0-250", "251-500", "501-750", "751-1000", "1000+")
	AtPlaza           bool   // True if venue name indicates Plaza de España (for distance=0 filter)
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
